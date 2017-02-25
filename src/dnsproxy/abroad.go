package dnsproxy

import (
	"strconv"
	"strings"

	"common"
	"common/cache"
	"common/domain"
	iputil "common/ip"
	"config"
	"github.com/garyburd/redigo/redis"
	"github.com/miekg/dns"
)

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

func queryFromAbroadServers(r *dns.Msg, do bool) (resp chan *dns.Msg) {
	if do == false {
		return make(chan *dns.Msg)
	}
	length := len(config.Configurations.DNSProxy.Abroad)
	if config.Configurations.DNSProxy.AbroadServerCount != "all" {
		length, _ = strconv.Atoi(config.Configurations.DNSProxy.AbroadServerCount)
	}
	resp = make(chan *dns.Msg, length)
	count := 0
	for _, v := range config.Configurations.DNSProxy.Abroad {
		if config.Configurations.DNSProxy.AbroadProtocol == v.Protocol || config.Configurations.DNSProxy.AbroadProtocol == "all" {
			if c, ok := clients[v.Protocol]; ok {
				go exchange(v, c, r, resp)
				count++
			}
		}
		if count == length {
			break
		}
	}
	return
}

func abroadResponseHandler(r *dns.Msg, rr *dns.Msg, abroadOnly bool, giveUpChinaResult *bool, cacheKey string) (rs *dns.Msg, shouldReturn bool) {
	shouldReturn = true
	if rr == nil {
		//common.Error(r.Question[0].Name, "querying from abroad dns failed")
		shouldReturn = false
		return
	}

	if abroadOnly == true {
		common.Debug(r.Question[0].Name, "use result from abroad DNS servers only")
		rs = rr
		return
	}

	for _, v := range rr.Answer {
		if v.Header().Rrtype != dns.TypeA && v.Header().Rrtype != dns.TypeAAAA {
			continue
		}

		ss := strings.Split(v.String(), "\t")
		if len(ss) == 5 {
			ip := ss[4]
			// always use the result from abroad DNS server if the ip is not in China,
			if iputil.InBlacklist(ip) {
				common.Debug(r.Question[0].Name, "drop this results from abroad DNS servers due to in blacklist", ip)
				shouldReturn = false
				return
			}

			if iputil.IsBogusNXDomain(ip) {
				rs = dropResponse(r)
				return
			}
		}
	}

	rs = rr
	if *giveUpChinaResult == true {
		common.Debug(r.Question[0].Name, "use result from abroad DNS servers")
		cacheDNSResultLocation(cacheKey, false)
		return
	}

	shouldReturn = false
	return
}
