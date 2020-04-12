package rule

import (
	"testing"
)

func TestGenerateSSCommandScript(t *testing.T) {
	prefixes := map[string]placeholder{
		"jp": {},
		"cn": {},
		"kr": {},
		"sg": {},
		"us": {},
		"hk": {},
		"tw": {},
	}
	m := make(map[string]int)
	i := 0
	for prefix := range prefixes {
		m[prefix] = 58090 + i
		i++
	}

	generateSSCommandScript(m)
}

func TestGenerateHAProxyMixedConfiguration(t *testing.T) {
	hostsMap := map[string]string{
		"us-1.xxxxxxxxxxxx.com":  ``,
		"us-2.xxxxxxxxxxxx.com":  ``,
		"us-4a.xxxxxxxxxxxx.com": ``,
		"us-6b.xxxxxxxxxxxx.com": ``,
		"us-7c.xxxxxxxxxxxx.com": ``,
		"us-8d.xxxxxxxxxxxx.com": ``,
		"hk-1.xxxxxxxxxxxx.com":  ``,
		"hk-2.xxxxxxxxxxxx.com":  ``,
		"hk-3a.xxxxxxxxxxxx.com": ``,
		"hk-5b.xxxxxxxxxxxx.com": ``,
		"hk-6c.xxxxxxxxxxxx.com": ``,
		"hk-7d.xxxxxxxxxxxx.com": ``,
		"hk-8e.xxxxxxxxxxxx.com": ``,
		"cn-1.xxxxxxxxxxxx.com":  ``,
		"cn-2a.xxxxxxxxxxxx.com": ``,
		"cn-8b.xxxxxxxxxxxx.com": ``,
		"eu-1.xxxxxxxxxxxx.com":  ``,
		"eu-4.xxxxxxxxxxxx.com":  ``,
		"eu-5a.xxxxxxxxxxxx.com": ``,
		"eu-6b.xxxxxxxxxxxx.com": ``,
		"eu-7c.xxxxxxxxxxxx.com": ``,
		"eu-8d.xxxxxxxxxxxx.com": ``,
		"sg-1.xxxxxxxxxxxx.com":  ``,
		"sg-2.xxxxxxxxxxxx.com":  ``,
		"sg-3a.xxxxxxxxxxxx.com": ``,
		"sg-4b.xxxxxxxxxxxx.com": ``,
		"sg-8d.xxxxxxxxxxxx.com": ``,
		"tw-1.xxxxxxxxxxxx.com":  ``,
		"tw-2.xxxxxxxxxxxx.com":  ``,
		"tw-6a.xxxxxxxxxxxx.com": ``,
		"tw-7b.xxxxxxxxxxxx.com": ``,
		"tw-8c.xxxxxxxxxxxx.com": ``,
		"kr-1.xxxxxxxxxxxx.com":  ``,
		"kr-2.xxxxxxxxxxxx.com":  ``,
		"kr-3a.xxxxxxxxxxxx.com": ``,
		"kr-5b.xxxxxxxxxxxx.com": ``,
		"kr-6c.xxxxxxxxxxxx.com": ``,
		"kr-7d.xxxxxxxxxxxx.com": ``,
		"kr-8e.xxxxxxxxxxxx.com": ``,
		"jp-1.xxxxxxxxxxxx.com":  ``,
		"jp-2.xxxxxxxxxxxx.com":  ``,
		"jp-3a.xxxxxxxxxxxx.com": ``,
		"jp-6b.xxxxxxxxxxxx.com": ``,
		"jp-7c.xxxxxxxxxxxx.com": ``,
		"jp-8d.xxxxxxxxxxxx.com": ``,
	}
	prefixes := map[string]placeholder{
		"jp": {},
		"cn": {},
		"kr": {},
		"sg": {},
		"us": {},
		"hk": {},
		"tw": {},
	}
	var ps []string
	for prefix := range prefixes {
		ps = append(ps, prefix)
	}
	generateHAProxyMixedConfiguration(hostsMap, ps, `haproxy.test.mixed.cfg`)
}
