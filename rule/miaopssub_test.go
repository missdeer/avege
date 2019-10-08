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
	hostsMap := map[string]placeholder{
		"us-1.mitsuha-node.com": {},
		"us-2.mitsuha-node.com": {},
		"us-4.mitsuha-node.com": {},
		"us-6.mitsuha-node.com": {},
		"us-7.mitsuha-node.com": {},
		"us-8.mitsuha-node.com": {},
		"hk-1.mitsuha-node.com": {},
		"hk-2.mitsuha-node.com": {},
		"hk-3.mitsuha-node.com": {},
		"hk-5.mitsuha-node.com": {},
		"hk-6.mitsuha-node.com": {},
		"hk-7.mitsuha-node.com": {},
		"hk-8.mitsuha-node.com": {},
		"cn-1.mitsuha-node.com": {},
		"cn-2.mitsuha-node.com": {},
		"cn-8.mitsuha-node.com": {},
		"eu-1.mitsuha-node.com": {},
		"eu-4.mitsuha-node.com": {},
		"eu-5.mitsuha-node.com": {},
		"eu-6.mitsuha-node.com": {},
		"eu-7.mitsuha-node.com": {},
		"eu-8.mitsuha-node.com": {},
		"sg-1.mitsuha-node.com": {},
		"sg-2.mitsuha-node.com": {},
		"sg-3.mitsuha-node.com": {},
		"sg-4.mitsuha-node.com": {},
		"sg-8.mitsuha-node.com": {},
		"tw-1.mitsuha-node.com": {},
		"tw-2.mitsuha-node.com": {},
		"tw-6.mitsuha-node.com": {},
		"tw-7.mitsuha-node.com": {},
		"tw-8.mitsuha-node.com": {},
		"kr-1.mitsuha-node.com": {},
		"kr-2.mitsuha-node.com": {},
		"kr-3.mitsuha-node.com": {},
		"kr-5.mitsuha-node.com": {},
		"kr-6.mitsuha-node.com": {},
		"kr-7.mitsuha-node.com": {},
		"kr-8.mitsuha-node.com": {},
		"jp-1.mitsuha-node.com": {},
		"jp-2.mitsuha-node.com": {},
		"jp-3.mitsuha-node.com": {},
		"jp-6.mitsuha-node.com": {},
		"jp-7.mitsuha-node.com": {},
		"jp-8.mitsuha-node.com": {},
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
	generateHAProxyMixedConfiguration(hostsMap, ps)
}
