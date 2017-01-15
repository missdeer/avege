package local

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"common"
	"common/cache"
	"common/domain"
	iputil "common/ip"
	"github.com/garyburd/redigo/redis"
	"github.com/miekg/dns"
)

var (
	tcpClient    *dns.Client
	tcpTLSClient *dns.Client
	udpClient    *dns.Client
	servers      []*dns.Server
	mutexServers sync.RWMutex
)

func exchange(server string, c *dns.Client, r *dns.Msg, resp chan *dns.Msg) {
	if rs, _, err := c.Exchange(r, server); err == nil {
		resp <- rs
	} else {
		resp <- nil
		common.Errorf("query dns %s from %s failed, %+v\n", r.Question[0].Name, server, err)
	}
}

func isAbroadOnly(r *dns.Msg, cacheKey string) bool {
	if key := cacheKey + "ca"; cache.Instance.IsExist(key) {
		if b, err := redis.Bool(cache.Instance.Get(key)); err == nil {
			return !b
		}
	}
	for _, v := range r.Question {
		vv := strings.Split(v.Name, ".")
		count := len(vv)
		for i := 0; i < count-1; i++ {
			if domain.IsGFWed(strings.Join(vv[i:], ".")) == true {
				return true
			}
		}
	}
	return false
}

func isChinaOnly(r *dns.Msg, cacheKey string) bool {
	if key := cacheKey + "ca"; cache.Instance.IsExist(key) {
		if b, err := redis.Bool(cache.Instance.Get(key)); err == nil {
			return b
		}
	}
	for _, v := range r.Question {
		vv := strings.Split(v.Name, ".")
		count := len(vv)
		for i := 0; i < count-1; i++ {
			if domain.InChina(strings.Join(vv[i:], ".")) == true {
				return true
			}
		}
	}
	return false
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

func isBlocked(r *dns.Msg) (rs *dns.Msg) {
	for _, v := range r.Question {
		vv := strings.Split(v.Name, ".")
		if domain.ToBlock(strings.Join(vv, ".")) {
			return dropResponse(r)
		}
	}
	return nil
}

func hitCache(r *dns.Msg, cacheKey string) (rs *dns.Msg) {
	if config.DNSProxy.CacheEnabled && cache.Instance.IsExist(cacheKey) {
		if cacheValue, err := cache.Instance.Get(cacheKey); err == nil {
			if b, ok := cacheValue.([]byte); ok {
				rs = &dns.Msg{}
				if err = rs.Unpack(b); err == nil {
					common.Debug(r.Question[0].Name, "extracted from cache")
					rs.Id = r.Id
					return rs
				}
			}
		}
	}
	return nil
}

func saveToCache(r *dns.Msg, rs *dns.Msg, cacheKey string, m []byte) {
	if config.DNSProxy.CacheEnabled {
		valid := false
		var ttl int64
		for _, v := range rs.Answer {
			if v.Header().Rrtype == dns.TypeA || v.Header().Rrtype == dns.TypeAAAA {
				ss := strings.Split(v.String(), "\t")
				if len(ss) == 5 {
					valid = true
					ttl = int64(v.Header().Ttl)
					break
				}
			}
		}
		if valid {
			common.Debug(r.Question[0].Name, "not from cache, save to cache")
			if config.DNSProxy.CacheTTL {
				if ttl <= 60 {
					cache.Instance.PutWithTimeout(cacheKey, m, ttl)
				} else {
					cache.Instance.PutWithTimeout(cacheKey, m, 60)
				}
			} else {
				cache.Instance.PutWithTimeout(cacheKey, m, int64(config.DNSProxy.CacheTimeout))
			}
		}
	}
}

func queryFromChinaServers(r *dns.Msg, do bool) (resp chan *dns.Msg) {
	if do == false {
		return make(chan *dns.Msg)
	}
	length := len(config.DNSProxy.China)
	if config.DNSProxy.ChinaServerCount != "all" {
		length, _ = strconv.Atoi(config.DNSProxy.ChinaServerCount)
	}
	resp = make(chan *dns.Msg, length)
	count := 0
	for _, v := range config.DNSProxy.China {
		switch v.Protocol {
		case "tcp":
			count++
			go exchange(v.Address, tcpClient, r, resp)
		case "tcp-tls":
			count++
			go exchange(v.Address, tcpTLSClient, r, resp)
		default:
			count++
			go exchange(v.Address, udpClient, r, resp)
		}

		if count == length {
			break
		}
	}
	return
}

func queryFromAbroadServers(r *dns.Msg, do bool) (resp chan *dns.Msg) {
	if do == false {
		return make(chan *dns.Msg)
	}
	length := len(config.DNSProxy.Abroad)
	if config.DNSProxy.AbroadServerCount != "all" {
		length, _ = strconv.Atoi(config.DNSProxy.AbroadServerCount)
	}
	resp = make(chan *dns.Msg, length)
	count := 0
	for _, v := range config.DNSProxy.Abroad {
		if v.Protocol == "tcp" && (config.DNSProxy.AbroadProtocol == "tcp" || config.DNSProxy.AbroadProtocol == "all") {
			count++
			go exchange(v.Address, tcpClient, r, resp)
		}
		if v.Protocol == "tcp-tls" && (config.DNSProxy.AbroadProtocol == "tcp-tls" || config.DNSProxy.AbroadProtocol == "all") {
			count++
			go exchange(v.Address, tcpTLSClient, r, resp)
		}
		if v.Protocol == "udp" && (config.DNSProxy.AbroadProtocol == "udp" || config.DNSProxy.AbroadProtocol == "all") {
			count++
			go exchange(v.Address, udpClient, r, resp)
		}
		if count == length {
			break
		}
	}
	return
}

func querySpecificServer(r *dns.Msg) (rs *dns.Msg) {
	doQuery := false
	for _, d := range config.DNSProxy.Server.Domains {
		if strings.Contains(r.Question[0].Name, d) {
			doQuery = true
			break
		}
	}
	if doQuery == false {
		return nil
	}

	length := len(config.DNSProxy.Server.Servers)
	resp := make(chan *dns.Msg, length)
	for _, v := range config.DNSProxy.Server.Servers {
		switch v.Protocol {
		case "tcp":
			go exchange(v.Address, tcpClient, r, resp)
		case "tcp-tls":
			go exchange(v.Address, tcpTLSClient, r, resp)
		default:
			go exchange(v.Address, udpClient, r, resp)
		}
	}

	timeout := config.DNSProxy.Timeout
	if timeout < 15 {
		timeout = 15
	}
	timeoutTicker := time.NewTicker(time.Duration(timeout) * time.Second)
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

func cacheDNSResultLocation(cacheKey string, inChina bool) {
	cache.Instance.PutWithTimeout(cacheKey+"ca", inChina, 30*24*3600) // for 1 month
}

func serveDNS(w dns.ResponseWriter, r *dns.Msg) {
	var rs *dns.Msg
	fromCache := false
	cacheKey := fmt.Sprintf("dns:%s:cachekey", r.Question[0].Name)
	defer func() {
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
	}()

	if len(r.Question) == 0 {
		return
	}

	if len(config.DNSProxy.SearchDomain) > 0 {
		for i, v := range r.Question {
			fields := strings.Split(v.Name, ".")
			if len(fields) == 1 {
				r.Question[i].Name = v.Name + config.DNSProxy.SearchDomain + "."
			}
		}
	}

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
	timeout := config.DNSProxy.Timeout
	if timeout < 15 {
		timeout = 15
	}
	timeoutTicker := time.NewTicker(time.Duration(timeout) * time.Second)
	defer timeoutTicker.Stop()
	var rc *dns.Msg
	for {
		select {
		case <-timeoutTicker.C:
			rs = rc
			common.Debug(r.Question[0].Name, "too long waited, just abort")
			return
		case rr := <-respChina:
			if giveUpChinaResult == true {
				break
			}
			if rr == nil {
				//common.Error(r.Question[0].Name, "querying from china dns failed")
				break
			}

			if chinaOnly {
				common.Debug(r.Question[0].Name, "use result from China DNS servers only")
				rs = rr
				return
			}

			for _, v := range rr.Answer {
				if v.Header().Rrtype != dns.TypeA {
					continue
				}

				ss := strings.Split(v.String(), "\t")
				if len(ss) == 5 {
					ip := ss[4]
					if iputil.InChina(ip) {
						common.Debug(r.Question[0].Name, "use result from China DNS servers")
						cacheDNSResultLocation(cacheKey, true)
						rs = rr
						return
					}

					if iputil.IsBogusNXDomain(ip) {
						rs = dropResponse(r)
						return
					}

					// always use the result from abroad DNS server if the ip is not in China,
					if iputil.InBlacklist(ip) {
						common.Debug(r.Question[0].Name, "drop all results from China DNS servers due to in blacklist", ip)
					} else {
						common.Debug(r.Question[0].Name, "drop all results from China DNS servers due to out of China", ip)
						rc = rr // candidates
					}

					giveUpChinaResult = true
					break
				}

				common.Debug(r.Question[0].Name, "empty record")
				giveUpChinaResult = true
				break
			}

			if rs != nil {
				// maybe result from abroad DNS server is ok
				common.Debug(r.Question[0].Name, "use result from abroad DNS servers")
				return
			}
		case rr := <-respAbroad:
			if rr == nil {
				//common.Error(r.Question[0].Name, "querying from abroad dns failed")
				break
			}

			if abroadOnly == true {
				common.Debug(r.Question[0].Name, "use result from abroad DNS servers only")
				rs = rr
				return
			}

			for _, v := range rr.Answer {
				if v.Header().Rrtype != dns.TypeA {
					continue
				}

				ss := strings.Split(v.String(), "\t")
				if len(ss) == 5 {
					ip := ss[4]
					// always use the result from abroad DNS server if the ip is not in China,
					if iputil.InBlacklist(ip) {
						common.Debug(r.Question[0].Name, "drop this results from abroad DNS servers due to in blacklist", ip)
						goto continueLoop
					}

					if iputil.IsBogusNXDomain(ip) {
						rs = dropResponse(r)
						return
					}
				}
			}

			rs = rr
			if giveUpChinaResult == true {
				common.Debug(r.Question[0].Name, "use result from abroad DNS servers")
				cacheDNSResultLocation(cacheKey, false)
				return
			}
		continueLoop:
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
	tcpClient = &dns.Client{
		Net:          "tcp",
		ReadTimeout:  time.Duration(config.DNSProxy.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.DNSProxy.WriteTimeout) * time.Second,
	}
	udpClient = &dns.Client{
		Net:          "udp",
		ReadTimeout:  time.Duration(config.DNSProxy.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.DNSProxy.WriteTimeout) * time.Second,
	}
	tcpTLSClient = &dns.Client{
		Net:          "tcp-tls",
		ReadTimeout:  time.Duration(config.DNSProxy.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.DNSProxy.WriteTimeout) * time.Second,
		TLSConfig:    &tls.Config{InsecureSkipVerify: true},
	}
}

func startDNSProxy() {
	if config.DNSProxy.Enabled {
		createClients()
		dns.HandleFunc(".", serveDNS)
		for _, v := range config.DNSProxy.Local {
			common.Debug("starting dns on", v.Address, v.Protocol)
			listenAndServe(v.Address, v.Protocol)
		}

		go iputil.LoadBogusNXDomain()
		go iputil.LoadIPBlacklist()
		go iputil.LoadChinaIPList(false)
		go domain.LoadDomainNameInChina()
		go domain.LoadDomainNameToBlock()
		go domain.LoadDomainNameGFWed()
	}
}
