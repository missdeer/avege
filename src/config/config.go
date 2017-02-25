package config

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"common"
	"inbound"
	"outbound"
)

var (
	Configurations       = new(LocalConfig)
	LeftQuote      int64 = 1 * 1024 * 1024 // 1MB initially
	AllowedPort          = make(map[int]bool)
	DeniedPort           = make(map[int]bool)
	AllowedIP            = make(map[uint32]bool) // IPv4 only
	DeniedIP             = make(map[uint32]bool) // IPv4 only

	DefaultPort   string
	DefaultKey    string
	DefaultMethod string
)

type ConsoleConfiguration struct {
	ConsoleReportEnabled bool   `json:"console_report_enabled"`
	ConsoleHost          string `json:"console_host"`
	ConsoleVersion       string `json:"console_version"`
	ConsoleWebSocketURL  string `json:"console_websocket_url"`
}

type PriorityInterfaceConfiguration struct {
	PriorityInterfaceEnabled bool   `json:"priority_interface_enabled"`
	PriorityInterfaceAddress string `json:"priority_interface_address"`
}

type APIConfiguration struct {
	API        string `json:"api"`
	APIEnabled bool   `json:"api_enabled"`
}

type DebuggingConfiguration struct {
	LogLevel     int  `json:"log_level"`
	PProfEnabled bool `json:"pprof"`
	GenRelease   bool `json:"gen_release"`
}

type ProxyPolicyConfiguration struct {
	LoadBalance      string `json:"load_balance"`
	UDPEnabled       bool   `json:"udp_enabled"`
	Tun2SocksEnabled bool   `json:"tun2socks_enabled"`
}

// GeneralConfig represents the general config section in configuration file
type GeneralConfig struct {
	Token                   string `json:"token"`
	CacheService            string `json:"cache_service"`
	ProtectSocketPathPrefix string `json:"protect_socket_path_prefix"`
	MaxOpenFiles            uint64 `json:"max_openfiles"`
	BroadcastEnabled        bool   `json:"broadcast_enabled"`
	Timeout                 time.Duration
	InboundTimeout          time.Duration
	ProxyPolicyConfiguration
	DebuggingConfiguration
	APIConfiguration
	PriorityInterfaceConfiguration
	ConsoleConfiguration
}

// UnmarshalJSON override the json unmarshal method, so that some fields could be initialized correctly
func (g *GeneralConfig) UnmarshalJSON(b []byte) error {
	type Alias GeneralConfig
	aux := &struct {
		Timeout        string `json:"timeout"`
		InBoundTimeout string `json:"inbound_timeout"`
		*Alias
	}{
		Alias: (*Alias)(g),
	}
	err := json.Unmarshal(b, &aux)
	if err != nil {
		return err
	}
	g.Timeout, err = time.ParseDuration(aux.Timeout)
	if err != nil {
		return err
	}
	g.InboundTimeout, err = time.ParseDuration(aux.InBoundTimeout)
	if err != nil {
		return err
	}
	return nil
}

// AllowDeny items for ACL
type AllowDeny struct {
	Allow string `json:"allow"`
	Deny  string `json:"deny"`
}

// ACL access control list
type ACL struct {
	Port AllowDeny `json:"port"`
	IP   AllowDeny `json:"ip"`
}

// DNSConfig represents each DNS server configuration
type DNSConfig struct {
	Address                 string `json:"address"`
	Protocol                string `json:"protocol"`
	EDNSClientSubnetEnabled bool   `json:"edns_client_subnet_enabled"`
}

// DNSServerSpecific some domain names should be resolved by some special DNS servers
type DNSServerSpecific struct {
	Domains []string     `json:"domains"`
	Servers []*DNSConfig `json:"servers"`
}

type DNSCacheConfiguration struct {
	CacheEnabled bool `json:"cache"`
	CacheTTL     bool `json:"cache_ttl"`
	CacheTimeout time.Duration
}

type DNSTimeoutConfiguration struct {
	Timeout      time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DNSAbroadServerConfiguration struct {
	AbroadServerCount string       `json:"abroad_server_count"`
	AbroadProtocol    string       `json:"abroad_protocol"`
	Abroad            []*DNSConfig `json:"abroad"`
}

type DNSChinaServerConfiguration struct {
	ChinaServerCount string       `json:"china_server_count"`
	China            []*DNSConfig `json:"china"`
}

type DNSEdnsClientSubnetConfiguration struct {
	EDNSClientSubnetPolicy string `json:"edns_client_subnet_policy"`
	EDNSClientSubnetIP     string `json:"edns_client_subnet_ip"`
}

// DNS represents the DNS section in configuration file
type DNS struct {
	DNSCacheConfiguration
	DNSTimeoutConfiguration
	DNSAbroadServerConfiguration
	DNSChinaServerConfiguration
	DNSEdnsClientSubnetConfiguration
	Enabled      bool              `json:"enabled"`
	SearchDomain string            `json:"search_domain"`
	Local        []*DNSConfig      `json:"local"`
	Server       DNSServerSpecific `json:"server"`
}

// UnmarshalJSON override the json unmarshal method, so that some fields could be initialized correctly
func (g *DNS) UnmarshalJSON(b []byte) error {
	type Alias DNS
	aux := &struct {
		CacheTimeout string `json:"cache_timeout"`
		Timeout      string `json:"timeout"`
		ReadTimeout  string `json:"read_timeout"`
		WriteTimeout string `json:"write_timeout"`
		*Alias
	}{
		Alias: (*Alias)(g),
	}
	err := json.Unmarshal(b, &aux)
	if err != nil {
		return err
	}
	g.Timeout, err = time.ParseDuration(aux.Timeout)
	if err != nil {
		return err
	}
	g.CacheTimeout, err = time.ParseDuration(aux.CacheTimeout)
	if err != nil {
		return err
	}
	g.ReadTimeout, err = time.ParseDuration(aux.ReadTimeout)
	if err != nil {
		return err
	}
	g.WriteTimeout, err = time.ParseDuration(aux.WriteTimeout)
	if err != nil {
		return err
	}
	return nil
}

// LocalConfig represents the whole configuration file struct
type LocalConfig struct {
	Generals        *GeneralConfig       `json:"general"`
	DNSProxy        *DNS                 `json:"dns"`
	Target          *ACL                 `json:"target"`
	InboundConfig   *inbound.Inbound     `json:"inbound"`
	InboundsConfig  []*inbound.Inbound   `json:"inbounds"`
	OutboundsConfig []*outbound.Outbound `json:"outbounds"`
}

func handleCacheService() {
	if len(Configurations.Generals.CacheService) == 0 {
		Configurations.Generals.CacheService = "gocache"
	}
}

func handleInboundTimeout() {
	if Configurations.Generals.InboundTimeout == 0 {
		Configurations.Generals.InboundTimeout = Configurations.Generals.Timeout
	}
}

func handlePortACL() {
	if len(Configurations.Target.Port.Allow) > 0 && Configurations.Target.Port.Allow != "all" {
		ports := strings.Split(Configurations.Target.Port.Allow, ",")
		for _, port := range ports {
			if p, err := strconv.Atoi(port); err != nil {
				common.Error("converting allowed port failed", err)
			} else {
				AllowedPort[p] = true
			}
		}
	}
	if len(Configurations.Target.Port.Deny) > 0 && Configurations.Target.Port.Deny != "all" {
		ports := strings.Split(Configurations.Target.Port.Deny, ",")
		for _, port := range ports {
			if p, err := strconv.Atoi(port); err != nil {
				common.Error("converting denied port failed", err)
			} else {
				DeniedPort[p] = true
			}
		}
	}
}

func handleIPACL() {
	if len(Configurations.Target.IP.Allow) > 0 && Configurations.Target.IP.Allow != "all" {
		ips := strings.Split(Configurations.Target.IP.Allow, ",")
		for _, ip := range ips {
			if v := net.ParseIP(ip); v == nil {
				common.Errorf("converting allowed IP %s failed", ip)
			} else {
				ipv4 := []byte(v.To4()) // IPv4 only
				ipAddr := uint32(ipv4[3]) + uint32(ipv4[2])<<8 + uint32(ipv4[1])<<16 + uint32(ipv4[0])<<24
				AllowedIP[ipAddr] = true
			}
		}
	}
	if len(Configurations.Target.IP.Deny) > 0 && Configurations.Target.IP.Deny != "all" {
		ips := strings.Split(Configurations.Target.IP.Deny, ",")
		for _, ip := range ips {
			if v := net.ParseIP(ip); v == nil {
				common.Errorf("converting denied IP %s failed", ip)
			} else {
				ipv4 := []byte(v.To4()) // IPv4 only
				ipAddr := uint32(ipv4[3]) + uint32(ipv4[2])<<8 + uint32(ipv4[1])<<16 + uint32(ipv4[0])<<24
				DeniedIP[ipAddr] = true
			}
		}
	}
}

func ParseMultiServersConfigFile(path string) error {
	file, err := os.Open(path) // For read access.
	if err != nil {
		common.Error("Failed opening config file", err)
		return err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		common.Error("Failed reading config file", err)
		return err
	}

	if err := json.Unmarshal(data, Configurations); err != nil {
		common.Error("Failed unmarshalling config file", err, len(data))
		return err
	}

	handleCacheService()
	handleInboundTimeout()

	handlePortACL()

	handleIPACL()

	return nil
}
