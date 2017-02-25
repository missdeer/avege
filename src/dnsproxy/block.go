package dnsproxy

import (
	"strings"

	"common/domain"
	"github.com/miekg/dns"
)

var (
	prefixPatterns = map[string]bool{
		"ad":         true,
		"ads":        true,
		"banner":     true,
		"banners":    true,
		"creatives":  true,
		"oas":        true,
		"oascentral": true,
		"stats":      true,
		"tag":        true,
		"telemetry":  true,
		"tracker":    true,
	}
	suffixPatterns = map[string]bool{
		"lan":         true,
		"local":       true,
		"localdomain": true,
		"workgroup":   true,
	}
)

func hitPattern(s string, patterns map[string]bool) bool {
	_, ok := patterns[s]
	return ok
}

func isBlocked(r *dns.Msg) (rs *dns.Msg) {
	for _, v := range r.Question {
		vv := strings.Split(v.Name, ".")
		if len(vv) > 1 {
			if hitPattern(vv[len(vv)-1], suffixPatterns) ||
				hitPattern(vv[0], prefixPatterns) ||
				domain.ToBlock(strings.Join(vv, ".")) {
				return dropResponse(r)
			}
		}
	}
	return nil
}
