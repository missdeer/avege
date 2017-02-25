package dnsproxy

import (
	"strings"

	"common"
	"common/cache"
	"config"
	"github.com/miekg/dns"
)

func cacheDNSResultLocation(cacheKey string, inChina bool) {
	cache.Instance.PutWithTimeout(cacheKey+"ca", inChina, 30*24*3600) // for 1 month
}

func hitCache(r *dns.Msg, cacheKey string) (rs *dns.Msg) {
	if config.Configurations.DNSProxy.Enabled && cache.Instance.IsExist(cacheKey) {
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
	if config.Configurations.DNSProxy.CacheEnabled {
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
			if config.Configurations.DNSProxy.CacheTTL {
				if ttl <= 60 {
					cache.Instance.PutWithTimeout(cacheKey, m, ttl)
				} else {
					cache.Instance.PutWithTimeout(cacheKey, m, 60)
				}
			} else {
				cache.Instance.PutWithTimeout(cacheKey, m, int64(config.Configurations.DNSProxy.CacheTimeout))
			}
		}
	}
}
