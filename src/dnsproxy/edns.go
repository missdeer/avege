package dnsproxy

import (
	"net"

	"config"
	"github.com/miekg/dns"
)

func addEDNSClientSubnet(r *dns.Msg, ip string) {
	if len(ip) == 0 {
		return
	}

	edns0Subnet := &dns.EDNS0_SUBNET{
		Code:          dns.EDNS0SUBNET,
		Address:       net.ParseIP(ip),
		Family:        1,  // 1 for IPv4 source address, 2 for IPv6
		SourceNetmask: 32, // 32 for IPV4, 128 for IPv6
		SourceScope:   0,
	}
	if edns0Subnet.Address.To4() == nil {
		edns0Subnet.Family = 2          // 1 for IPv4 source address, 2 for IPv6
		edns0Subnet.SourceNetmask = 128 // 32 for IPV4, 128 for IPv6
	}

	option := &dns.OPT{
		Hdr: dns.RR_Header{
			Name:   ".",
			Rrtype: dns.TypeOPT,
		},
		Option: []dns.EDNS0{edns0Subnet},
	}

	r.Extra = append(r.Extra, option)
}

func ednsClientSubnetFilter(r *dns.Msg) {
	switch config.Configurations.DNSProxy.EDNSClientSubnetPolicy {
	case "custom":
		addEDNSClientSubnet(r, config.Configurations.DNSProxy.EDNSClientSubnetIP)
	case "auto":
		addEDNSClientSubnet(r, externalIPAddress)
	case "disable":
	default:
	}
}
