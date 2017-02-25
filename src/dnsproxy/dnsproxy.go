package dnsproxy

import (
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	"common"
	"common/domain"
	iputil "common/ip"
	"common/netutil"
	"config"
	"github.com/miekg/dns"
)

var (
	clients           map[string]*dns.Client
	servers           []*dns.Server
	mutexServers      sync.RWMutex
	externalIPAddress string
)

func exchange(s *config.DNSConfig, c *dns.Client, r *dns.Msg, resp chan *dns.Msg) {
	if s.EDNSClientSubnetEnabled {
		req := *r
		r = &req
		ednsClientSubnetFilter(r)
	}
	if rs, _, err := c.Exchange(r, s.Address); err == nil {
		resp <- rs
	} else {
		resp <- nil
		if err == dns.ErrTruncated && s.EDNSClientSubnetEnabled == true {
			s.EDNSClientSubnetEnabled = false
		}
		common.Warningf("query dns %s from %s failed, %+v\n", r.Question[0].Name, s.Address, err)
	}
}

func dropResponse(r *dns.Msg) (rs *dns.Msg) {
	rs = &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:                 r.Id,
			Response:           true,
			RecursionDesired:   true,
			RecursionAvailable: true,
		},
		Question: r.Question,
		Answer:   make([]dns.RR, 1),
	}
	if r.Question[0].Qtype == dns.TypeA {
		rs.Answer[0], _ = dns.NewRR(fmt.Sprintf("%s 3600 IN A 127.0.0.1", r.Question[0].Name))
	} else if r.Question[0].Qtype == dns.TypeAAAA {
		rs.Answer[0], _ = dns.NewRR(fmt.Sprintf("%s 3600 IN AAAA ::1", r.Question[0].Name))
	}
	return rs
}

func querySpecificServer(r *dns.Msg) (rs *dns.Msg) {
	doQuery := false
	for _, d := range config.Configurations.DNSProxy.Server.Domains {
		if strings.Contains(r.Question[0].Name, d) {
			doQuery = true
			break
		}
	}
	if doQuery == false {
		return nil
	}

	length := len(config.Configurations.DNSProxy.Server.Servers)
	resp := make(chan *dns.Msg, length)
	for _, v := range config.Configurations.DNSProxy.Server.Servers {
		if c, ok := clients[v.Protocol]; ok {
			go exchange(v, c, r, resp)
		}
	}

	timeout := config.Configurations.DNSProxy.Timeout
	if timeout < 15*time.Second {
		timeout = 15 * time.Second
	}
	timeoutTicker := time.NewTicker(timeout)
	defer timeoutTicker.Stop()
	for {
		select {
		case <-timeoutTicker.C:
			return nil
		case rs = <-resp:
			return rs
		}
	}
}

func processSearchDomain(r *dns.Msg) {
	if len(config.Configurations.DNSProxy.SearchDomain) > 0 {
		for i, v := range r.Question {
			fields := strings.Split(v.Name, ".")
			if len(fields) == 1 {
				r.Question[i].Name = v.Name + config.Configurations.DNSProxy.SearchDomain + "."
			}
		}
	}
}

func finalResponse(w dns.ResponseWriter, r *dns.Msg, rs *dns.Msg, fromCache bool, cacheKey string) {
	valid := true
	if rs == nil {
		rs = &dns.Msg{
			MsgHdr: dns.MsgHdr{
				Id: r.Id,
			},
		}
		valid = false
	}
	for i := len(rs.Answer) - 1; i >= 0; i-- {
		if rs.Answer[i] == nil {
			common.Warningf("found nil answer in %s's %V\nresult: %V\n%s", r.Question[0].Name, r, rs, rs)
			rs.Answer = append(rs.Answer[:i], rs.Answer[i+1:]...)
		}
	}
	if m, err := rs.Pack(); err == nil {
		w.Write(m)
		if !fromCache && valid {
			saveToCache(r, rs, cacheKey, m)
		}
	}
}

func serveDNS(w dns.ResponseWriter, r *dns.Msg) {
	var rs *dns.Msg
	fromCache := false
	cacheKey := fmt.Sprintf("dns:%s:cachekey", r.Question[0].Name)
	defer func() {
		finalResponse(w, r, rs, fromCache, cacheKey)
	}()

	if len(r.Question) == 0 {
		return
	}

	processSearchDomain(r)

	// query from cache
	if rs = hitCache(r, cacheKey); rs != nil {
		fromCache = true
		return
	}

	// server specific query
	if rs = querySpecificServer(r); rs != nil {
		return
	}

	// block domain names
	if rs = isBlocked(r); rs != nil {
		return
	}

	var respChina chan *dns.Msg
	var respAbroad chan *dns.Msg

	abroadOnly := isAbroadOnly(r, cacheKey)
	chinaOnly := isChinaOnly(r, cacheKey) && !abroadOnly

	respChina = queryFromChinaServers(r, !abroadOnly)
	respAbroad = queryFromAbroadServers(r, !chinaOnly)

	giveUpChinaResult := false
	timeout := config.Configurations.DNSProxy.Timeout
	if timeout < 15*time.Second {
		timeout = 15 * time.Second
	}
	timeoutTicker := time.NewTicker(timeout)
	defer timeoutTicker.Stop()
	var rc *dns.Msg
	for {
		select {
		case <-timeoutTicker.C:
			rs = rc
			common.Debug(r.Question[0].Name, "too long waited, just abort")
			return
		case rr := <-respChina:
			shouldReturn := true
			if rs, rc, shouldReturn = chinaResponseHandler(r, rr, chinaOnly, &giveUpChinaResult, cacheKey); shouldReturn {
				return
			}
		case rr := <-respAbroad:
			shouldReturn := true
			if rs, shouldReturn = abroadResponseHandler(r, rr, abroadOnly, &giveUpChinaResult, cacheKey); shouldReturn {
				return
			}
		}
	}
}

func listenAndServe(address string, protocol string) {
	go func() {
		server := &dns.Server{Addr: address, Net: protocol, Handler: nil}
		mutexServers.Lock()
		servers = append(servers, server)
		mutexServers.Unlock()
		for {
			if err := server.ListenAndServe(); err != nil {
				common.Error("dns server listen and serve failed on", address, protocol, err)
			}
		}
	}()
}

func createClients() {
	clients = map[string]*dns.Client{
		"tcp": &dns.Client{
			Net:          "tcp",
			ReadTimeout:  config.Configurations.DNSProxy.ReadTimeout,
			WriteTimeout: config.Configurations.DNSProxy.WriteTimeout,
		},
		"udp": &dns.Client{
			Net:          "udp",
			ReadTimeout:  config.Configurations.DNSProxy.ReadTimeout,
			WriteTimeout: config.Configurations.DNSProxy.WriteTimeout,
		},
		"tcp-tls": &dns.Client{
			Net:          "tcp-tls",
			ReadTimeout:  config.Configurations.DNSProxy.ReadTimeout,
			WriteTimeout: config.Configurations.DNSProxy.WriteTimeout,
			TLSConfig:    &tls.Config{InsecureSkipVerify: true},
		},
	}
}

func Start() {
	createClients()
	dns.HandleFunc(".", serveDNS)
	for _, v := range config.Configurations.DNSProxy.Local {
		common.Debug("starting dns on", v, v.Protocol)
		listenAndServe(v.Address, v.Protocol)
	}

	go iputil.LoadBogusNXDomain()
	go iputil.LoadIPBlacklist()
	go iputil.LoadChinaIPList(false)
	go domain.LoadDomainNameInChina()
	go domain.LoadDomainNameToBlock()
	go domain.LoadDomainNameGFWed()
	go func() {
		for i := 0; externalIPAddress == "" && i < 10; i++ {
			externalIPAddress, _ = netutil.GetExternalIPAddress()
		}
	}()
}
