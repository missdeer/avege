package local

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
	"outbound/ss"
)

var (
	config            = new(LocalConfig)
	leftQuote   int64 = 1 * 1024 * 1024 // 1MB initially
	allowedPort       = make(map[int]bool)
	deniedPort        = make(map[int]bool)
	allowedIP         = make(map[uint32]bool) // IPv4 only
	deniedIP          = make(map[uint32]bool) // IPv4 only
	serverIP          = make(map[uint32]int)  // IPv4 only

	defaultPort   string
	defaultKey    string
	defaultMethod string
)

// GeneralConfig represents the general config section in configuration file
type GeneralConfig struct {
	LoadBalance              string `json:"load_balance"`
	API                      string `json:"api"`
	Token                    string `json:"token"`
	CacheService             string `json:"cache_service"`
	ProtectSocketPathPrefix  string `json:"protect_socket_path_prefix"`
	MaxOpenFiles             uint64 `json:"max_openfiles"`
	LogLevel                 int    `json:"log_level"`
	Timeout                  time.Duration
	InboundTimeout           time.Duration
	PProfEnabled             bool   `json:"pprof"`
	GenRelease               bool   `json:"gen_release"`
	UDPEnabled               bool   `json:"udp_enabled"`
	APIEnabled               bool   `json:"api_enabled"`
	BroadcastEnabled         bool   `json:"broadcast_enabled"`
	Tun2SocksEnabled         bool   `json:"tun2socks_enabled"`
	PriorityInterfaceEnabled bool   `json:"priority_interface_enabled"`
	PriorityInterfaceAddress string `json:"priority_interface_address"`
	ConsoleReportEnabled     bool   `json:"console_report_enabled"`
	ConsoleHost              string `json:"console_host"`
	ConsoleVersion           string `json:"console_version"`
	ConsoleWebSocketURL      string `json:"console_websocket_url"`
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

// DNS represents the DNS section in configuration file
type DNS struct {
	Enabled                bool `json:"enabled"`
	CacheEnabled           bool `json:"cache"`
	CacheTTL               bool `json:"cache_ttl"`
	CacheTimeout           time.Duration
	Timeout                time.Duration
	ReadTimeout            time.Duration
	WriteTimeout           time.Duration
	SearchDomain           string            `json:"search_domain"`
	EDNSClientSubnetPolicy string            `json:"edns_client_subnet_policy"`
	EDNSClientSubnetIP     string            `json:"edns_client_subnet_ip"`
	ChinaServerCount       string            `json:"china_server_count"`
	AbroadServerCount      string            `json:"abroad_server_count"`
	AbroadProtocol         string            `json:"abroad_protocol"`
	Local                  []*DNSConfig      `json:"local"`
	China                  []*DNSConfig      `json:"china"`
	Abroad                 []*DNSConfig      `json:"abroad"`
	Server                 DNSServerSpecific `json:"server"`
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

func changeKeyMethod() {
	_, err := ss.NewStreamCipher(defaultMethod, defaultKey)
	if err != nil {
		common.Error("Failed generating ciphers:", err)
		return
	}

	backends.Lock()
	for _, backendInfo := range backends.BackendsInformation {
		if backendInfo.local == false {
			backendInfo.encryptMethod = defaultMethod
			backendInfo.encryptPassword = defaultKey
		}
	}
	backends.Unlock()
}

func changePort() {
	backends.Lock()
	for _, backendInfo := range backends.BackendsInformation {
		if backendInfo.local == false {
			host, _, _ := net.SplitHostPort(backendInfo.address)
			backendInfo.address = net.JoinHostPort(host, defaultPort)
		}
	}
	backends.Unlock()
}

func removeServer(address string) {
	for i, backendInfo := range backends.BackendsInformation {
		host, _, _ := net.SplitHostPort(backendInfo.address)
		if host == address && backendInfo.local == false {
			// remove this element from backends array
			statistics.Delete(backends.Get(i))
			backends.Remove(i)
			break
		}
	}

	for i, outbound := range config.OutboundsConfig {
		host, _, _ := net.SplitHostPort(outbound.Address)
		if host == address && outbound.Local == false {
			config.OutboundsConfig = append(config.OutboundsConfig[:i], config.OutboundsConfig[i+1:]...)
			// save to redis
			break
		}
	}
}

func addServer(address string) {
	_, err := ss.NewStreamCipher(defaultMethod, defaultKey)
	if err != nil {
		common.Error("Failed generating ciphers:", err)
		return
	}

	// don't append directly, scan the existing elements and update them
	find := false
	for _, backendInfo := range backends.BackendsInformation {
		host, _, _ := net.SplitHostPort(backendInfo.address)
		if host == address && backendInfo.local == false {
			backendInfo.protocolType = "shadowsocks"
			//backendInfo.cipher = cipher
			backendInfo.encryptMethod = defaultMethod
			backendInfo.encryptPassword = defaultKey
			backendInfo.timeout = config.Generals.Timeout

			find = true
			break
		}
	}

	if !find {
		// append directly
		bi := &BackendInfo{
			id:           common.GenerateRandomString(4),
			address:      net.JoinHostPort(address, defaultPort),
			protocolType: "shadowsocks",
			timeout:      config.Generals.Timeout,
			SSRInfo: SSRInfo{
				obfs:     "plain",
				protocol: "origin",
				SSInfo: SSInfo{
					//cipher: cipher,
					encryptMethod:   defaultMethod,
					encryptPassword: defaultKey,
				},
			},
		}
		backends.Append(bi)

		stat := common.NewStatistic()
		statistics.Insert(bi, stat)

		outbound := &outbound.Outbound{
			Address: net.JoinHostPort(address, defaultPort),
			Key:     defaultKey,
			Method:  defaultMethod,
			Type:    "shadowsocks",
		}
		config.OutboundsConfig = append(config.OutboundsConfig, outbound)
		// save to redis
	}
}

func handleCacheService() {
	if len(config.Generals.CacheService) == 0 {
		config.Generals.CacheService = "gocache"
	}
}

func handleInboundTimeout() {
	if config.Generals.InboundTimeout == 0 {
		config.Generals.InboundTimeout = config.Generals.Timeout
	}
}

func handlePortACL() {
	if len(config.Target.Port.Allow) > 0 && config.Target.Port.Allow != "all" {
		ports := strings.Split(config.Target.Port.Allow, ",")
		for _, port := range ports {
			if p, err := strconv.Atoi(port); err != nil {
				common.Error("converting allowed port failed", err)
			} else {
				allowedPort[p] = true
			}
		}
	}
	if len(config.Target.Port.Deny) > 0 && config.Target.Port.Deny != "all" {
		ports := strings.Split(config.Target.Port.Deny, ",")
		for _, port := range ports {
			if p, err := strconv.Atoi(port); err != nil {
				common.Error("converting denied port failed", err)
			} else {
				deniedPort[p] = true
			}
		}
	}
}

func handleIPACL() {
	if len(config.Target.IP.Allow) > 0 && config.Target.IP.Allow != "all" {
		ips := strings.Split(config.Target.IP.Allow, ",")
		for _, ip := range ips {
			if v := net.ParseIP(ip); v == nil {
				common.Errorf("converting allowed IP %s failed", ip)
			} else {
				ipv4 := []byte(v.To4()) // IPv4 only
				ipAddr := uint32(ipv4[3]) + uint32(ipv4[2])<<8 + uint32(ipv4[1])<<16 + uint32(ipv4[0])<<24
				allowedIP[ipAddr] = true
			}
		}
	}
	if len(config.Target.IP.Deny) > 0 && config.Target.IP.Deny != "all" {
		ips := strings.Split(config.Target.IP.Deny, ",")
		for _, ip := range ips {
			if v := net.ParseIP(ip); v == nil {
				common.Errorf("converting denied IP %s failed", ip)
			} else {
				ipv4 := []byte(v.To4()) // IPv4 only
				ipAddr := uint32(ipv4[3]) + uint32(ipv4[2])<<8 + uint32(ipv4[1])<<16 + uint32(ipv4[0])<<24
				deniedIP[ipAddr] = true
			}
		}
	}
}

func removeDeprecatedServers() {
	// remove the ones that is not included in new config
	for i := 0; i < backends.Len(); {
		backendInfo := backends.Get(i)
		find := false
		for _, s := range config.OutboundsConfig {
			if backendInfo.address == s.Address {
				find = true
				break
			}
		}

		if !find {
			// remove this element from backends array
			statistics.Delete(backends.Get(i))
			backends.Remove(i)
			i = 0
		} else {
			i++
		}
	}
}

func updateExistsOutboundConfig(outboundConfig *outbound.Outbound) bool {
	for _, backendInfo := range backends.BackendsInformation {
		if backendInfo.address == outboundConfig.Address {
			backendInfo.protocolType = outboundConfig.Type
			backendInfo.encryptMethod = outboundConfig.Method
			backendInfo.encryptPassword = outboundConfig.Key
			if outboundConfig.Timeout != 0 {
				backendInfo.timeout = outboundConfig.Timeout
			} else {
				backendInfo.timeout = config.Generals.Timeout
			}

			return true
		}
	}
	return false
}

func addNewOutboundConfig(outboundConfig *outbound.Outbound) {
	// append directly
	backendInfo := &BackendInfo{
		id:           common.GenerateRandomString(4),
		address:      outboundConfig.Address,
		protocolType: outboundConfig.Type,
		SSRInfo: SSRInfo{
			obfs:          outboundConfig.Obfs,
			obfsParam:     outboundConfig.ObfsParam,
			protocol:      outboundConfig.Protocol,
			protocolParam: outboundConfig.ProtocolParam,
			SSInfo: SSInfo{
				encryptMethod:   outboundConfig.Method,
				encryptPassword: outboundConfig.Key,
				tcpFastOpen:     outboundConfig.TCPFastOpen,
			},
		},

		HTTPSProxyInfo: HTTPSProxyInfo{
			insecureSkipVerify: outboundConfig.TLSInsecureSkipVerify,
			domain:             outboundConfig.TLSDomain,
			CommonProxyInfo: CommonProxyInfo{
				username: outboundConfig.Username,
				password: outboundConfig.Password,
			},
		},
	}
	if outboundConfig.Timeout != 0 {
		backendInfo.timeout = outboundConfig.Timeout
	} else {
		backendInfo.timeout = config.Generals.Timeout
	}
	backendInfo.local = outboundConfig.Local

	backends.Append(backendInfo)

	stat := common.NewStatistic()
	statistics.Insert(backendInfo, stat)
}

func updateNewServers() {
	// add or update the ones that is included in the config
	for _, outboundConfig := range config.OutboundsConfig {
		if outboundConfig.Type == "shadowsocks" || outboundConfig.Type == "ss" {
			_, err := ss.NewStreamCipher(outboundConfig.Method, outboundConfig.Key)
			if err != nil {
				common.Error("Failed generating ciphers:", err)
				continue
			}
		}

		// don't append directly, scan the existing elements and update them
		if !updateExistsOutboundConfig(outboundConfig) {
			addNewOutboundConfig(outboundConfig)
		}

		if len(defaultKey) == 0 {
			defaultKey = outboundConfig.Key
		}
		if len(defaultPort) == 0 {
			_, defaultPort, _ = net.SplitHostPort(outboundConfig.Address)
		}
		if len(defaultMethod) == 0 {
			defaultMethod = outboundConfig.Method
		}
	}
}

func parseMultiServersConfig(data []byte) error {
	if err := json.Unmarshal(data, config); err != nil {
		common.Error("Failed unmarshalling config file", err, len(data))
		return err
	}

	handleCacheService()

	ss.ProtectSocketPathPrefix = config.Generals.ProtectSocketPathPrefix
	consoleHost = config.Generals.ConsoleHost
	consoleVer = config.Generals.ConsoleVersion
	consoleWSUrl = config.Generals.ConsoleWebSocketURL

	handleInboundTimeout()

	handlePortACL()

	handleIPACL()

	removeDeprecatedServers()

	updateNewServers()

	common.Debugf("servers in config: %V\n", backends)

	return nil
}

func parseMultiServersConfigFile(path string) error {
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

	return parseMultiServersConfig(data)
}
