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

func queryFromChinaServers(r *dns.Msg, do bool) (resp chan *dns.Msg) {
	if do == false {
		return make(chan *dns.Msg)
	}
	length := len(config.Configurations.DNSProxy.China)
	if config.Configurations.DNSProxy.ChinaServerCount != "all" {
		length, _ = strconv.Atoi(config.Configurations.DNSProxy.ChinaServerCount)
	}
	resp = make(chan *dns.Msg, length)
	count := 0
	for _, v := range config.Configurations.DNSProxy.China {
		if c, ok := clients[v.Protocol]; ok {
			go exchange(v, c, r, resp)
			count++
		}

		if count == length {
			break
		}
	}
	return
}

func chinaResponseHandler(r *dns.Msg, rr *dns.Msg, chinaOnly bool, giveUpChinaResult *bool, cacheKey string) (rs *dns.Msg, rc *dns.Msg, shouldReturn bool) {
	shouldReturn = true
	if *giveUpChinaResult == true {
		shouldReturn = false
		return
	}
	if rr == nil {
		//common.Error(r.Question[0].Name, "querying from china dns failed")
		shouldReturn = false
		return
	}

	if chinaOnly {
		common.Debug(r.Question[0].Name, "use result from China DNS servers only")
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

			*giveUpChinaResult = true
			break
		}

		common.Debug(r.Question[0].Name, "empty record")
		*giveUpChinaResult = true
		break
	}

	if rs != nil {
		// maybe result from abroad DNS server is ok
		common.Debug(r.Question[0].Name, "use result from abroad DNS servers")
		return
	}

	shouldReturn = false
	return
}
