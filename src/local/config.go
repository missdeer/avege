package local

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"

	"common"
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

type GeneralConfig struct {
	LoadBalance              string `json:"load_balance"`
	API                      string `json:"api"`
	Token                    string `json:"token"`
	CacheService             string `json:"cache_service"`
	ProtectSocketPathPrefix  string `json:"protect_socket_path_prefix"`
	MaxOpenFiles             uint64 `json:"max_openfiles"`
	LogLevel                 int    `json:"log_level"`
	Timeout                  int    `json:"timeout"`
	PProfEnabled             bool   `json:"pprof"`
	GenRelease               bool   `json:"gen_release"`
	UDPEnabled               bool   `json:"udp_enabled"`
	APIEnabled               bool   `json:"api_enabled"`
	BroadcastEnabled         bool   `json:"broadcast_enabled"`
	Tun2SocksEnabled         bool   `json:"tun2socks_enabled"`
	PriorityInterfaceEnabled bool   `json:"priority_interface_enabled"`
	PriorityInterfaceAddress string `json:"priority_interface_address"`
	ConsoleHost              string `json:"console_host"`
	ConsoleVersion           string `json:"console_version"`
	ConsoleWebSocketURL      string `json:"console_websocket_url"`
}

type AllowDeny struct {
	Allow string `json:"allow"`
	Deny  string `json:"deny"`
}

type ACL struct {
	Port AllowDeny `json:"port"`
	IP   AllowDeny `json:"ip"`
}

type DNSConfig struct {
	Address  string `json:"address"`
	Protocol string `json:"protocol"`
}

type DNSServerSpecific struct {
	Domains []string     `json:"domains"`
	Servers []*DNSConfig `json:"servers"`
}

type DNS struct {
	Enabled           bool              `json:"enabled"`
	CacheEnabled      bool              `json:"cache"`
	CacheTTL          bool              `json:"cache_ttl"`
	CacheTimeout      int               `json:"cache_timeout"`
	Timeout           int               `json:"timeout"`
	ReadTimeout       int               `json:"read_timeout"`
	WriteTimeout      int               `json:"write_timeout"`
	SearchDomain      string            `json:"search_domain"`
	ChinaServerCount  string            `json:"china_server_count"`
	AbroadServerCount string            `json:"abroad_server_count"`
	AbroadProtocol    string            `json:"abroad_protocol"`
	Local             []*DNSConfig      `json:"local"`
	China             []*DNSConfig      `json:"china"`
	Abroad            []*DNSConfig      `json:"abroad"`
	Server            DNSServerSpecific `json:"server"`
}

type OutBound struct {
	Address string `json:"address"`
	// Key shadowsocks only key used to encrypting
	Key string `json:"key"`
	// Method shadowsocks encrypting algorithm, eg. rc4-md5, aes-256-cfb etc.
	Method string `json:"method"`
	// Type protocol type, http/https/socks4/socks4a/socks5/shadowsocks are supported
	Type string `json:"type"`
	// Protocol shadowsocks only obfs protocol
	Protocol string `json:"protocol"`
	// ProtocolParam shadowsocks only obfs protocol parameter
	ProtocolParam string `json:"pparam"`
	// Obfs shadowsocks only obfs
	Obfs string `json:"obfs"`
	// ObfsParam shadowsocks only obfs parameter
	ObfsParam string `json:"oparam"`
	// Username auth for http/https/socks
	Username string `json:"username"`
	// Password auth for http/https/socks
	Password string `json:"password"`
	// TLSInsecureSkipVerify  https only
	TLSInsecureSkipVerify bool `json:"insecureskipverify"`
	// TLSDomain https only
	TLSDomain string `json:"domain"`
	// Timeout connecting timeout
	Timeout int `json:"timeout"`
	// Restrict == true if only 80/443 ports are allowed, otherwise all ports are allowed
	Restrict bool `json:"restrict"`
	// Local == true if this configuration item is from local config file, otherwise it's from remote console server's pushing
	Local bool `json:"local"`
	// TCPFastOpen == true if this backend supports TCP Fast Open
	TCPFastOpen bool `json:"tcpfastopen"`
}

func (o *OutBound) UnmarshalJSON(b []byte) error {
	type xob OutBound
	xo := &xob{
		Obfs:     "plain",
		Protocol: "origin",
		Type:     "shadowsocks",
	}
	if err := json.Unmarshal(b, xo); err != nil {
		return err
	}
	*o = OutBound(*xo)
	return nil
}

type InBound struct {
	Type    string `json:"type"`
	Address string `json:"address"`
	Port    int    `json:"port"`
	Timeout int    `json:"timeout"`
}

type LocalConfig struct {
	Generals        *GeneralConfig `json:"general"`
	DNSProxy        *DNS           `json:"dns"`
	Target          *ACL           `json:"target"`
	InBoundConfig   *InBound       `json:"inbound"`
	OutBoundsConfig []*OutBound    `json:"outbounds"`
}

func changeKeyMethod() {
	cipher, err := ss.NewCipher(defaultMethod, defaultKey)
	if err != nil {
		common.Error("Failed generating ciphers:", err)
		return
	}

	Backends.Lock()
	for _, backendInfo := range Backends.BackendsInformation {
		if backendInfo.local == false {
			backendInfo.cipher = cipher
		}
	}
	Backends.Unlock()
}

func changePort() {
	Backends.Lock()
	for _, backendInfo := range Backends.BackendsInformation {
		if backendInfo.local == false {
			host, _, _ := net.SplitHostPort(backendInfo.address)
			backendInfo.address = net.JoinHostPort(host, defaultPort)
		}
	}
	Backends.Unlock()
}

func removeServer(address string) {
	for i, backendInfo := range Backends.BackendsInformation {
		host, _, _ := net.SplitHostPort(backendInfo.address)
		if host == address && backendInfo.local == false {
			// remove this element from Backends array
			Statistics.Delete(Backends.Get(i))
			Backends.Remove(i)
			break
		}
	}

	for i, outbound := range config.OutBoundsConfig {
		host, _, _ := net.SplitHostPort(outbound.Address)
		if host == address && outbound.Local == false {
			config.OutBoundsConfig = append(config.OutBoundsConfig[:i], config.OutBoundsConfig[i + 1:]...)
			// save to redis
			break
		}
	}
}

func addServer(address string) {
	cipher, err := ss.NewCipher(defaultMethod, defaultKey)
	if err != nil {
		common.Error("Failed generating ciphers:", err)
		return
	}

	// don't append directly, scan the existing elements and update them
	find := false
	for _, backendInfo := range Backends.BackendsInformation {
		host, _, _ := net.SplitHostPort(backendInfo.address)
		if host == address && backendInfo.local == false {
			backendInfo.protocolType = "shadowsocks"
			backendInfo.cipher = cipher
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
			obfs:         "plain",
			protocol:     "origin",
			cipher:       cipher,
			timeout:      config.Generals.Timeout,
		}
		Backends.Append(bi)

		stat := common.NewStatistic()
		Statistics.Insert(bi, stat)

		outbound := &OutBound{
			Address: net.JoinHostPort(address, defaultPort),
			Key:     defaultKey,
			Method:  defaultMethod,
			Type:    "shadowsocks",
		}
		config.OutBoundsConfig = append(config.OutBoundsConfig, outbound)
		// save to redis
	}
}

func parseMultiServersConfig(data []byte) error {
	if err := json.Unmarshal(data, config); err != nil {
		common.Error("Failed unmarshalling config file", err, len(data))
		return err
	}

	if len(config.Generals.CacheService) == 0 {
		config.Generals.CacheService = "redis"
	}

	ss.ProtectSocketPathPrefix = config.Generals.ProtectSocketPathPrefix
	consoleHost = config.Generals.ConsoleHost
	consoleVer = config.Generals.ConsoleVersion
	consoleWSUrl = config.Generals.ConsoleWebSocketURL

	if config.InBoundConfig.Timeout == 0 {
		config.InBoundConfig.Timeout = config.Generals.Timeout
	}

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

	// remove the ones that is not included in new config
	for i := 0; i < Backends.Len(); {
		backendInfo := Backends.Get(i)
		find := false
		for _, s := range config.OutBoundsConfig {
			if backendInfo.address == s.Address {
				find = true
				break
			}
		}

		if !find {
			// remove this element from Backends array
			Statistics.Delete(Backends.Get(i))
			Backends.Remove(i)
			i = 0
		} else {
			i++
		}
	}

	// add or update the ones that is included in the config
	for _, outboundConfig := range config.OutBoundsConfig {
		var cipher *ss.Cipher
		var err error
		if outboundConfig.Type == "shadowsocks" || outboundConfig.Type == "ss" {
			cipher, err = ss.NewCipher(outboundConfig.Method, outboundConfig.Key)
			if err != nil {
				common.Error("Failed generating ciphers:", err)
				continue
			}
		}

		// don't append directly, scan the existing elements and update them
		find := false
		for _, backendInfo := range Backends.BackendsInformation {
			if backendInfo.address == outboundConfig.Address {
				backendInfo.protocolType = outboundConfig.Type
				backendInfo.cipher = cipher
				if outboundConfig.Timeout != 0 {
					backendInfo.timeout = outboundConfig.Timeout
				} else {
					backendInfo.timeout = config.Generals.Timeout
				}

				find = true
				break
			}
		}

		if !find {
			// append directly
			backendInfo := &BackendInfo{
				id:                 common.GenerateRandomString(4),
				address:            outboundConfig.Address,
				protocolType:       outboundConfig.Type,
				cipher:             cipher,
				obfs:               outboundConfig.Obfs,
				obfsParam:          outboundConfig.ObfsParam,
				protocol:           outboundConfig.Protocol,
				protocolParam:      outboundConfig.ProtocolParam,
				username:           outboundConfig.Username,
				password:           outboundConfig.Password,
				insecureSkipVerify: outboundConfig.TLSInsecureSkipVerify,
				domain:             outboundConfig.TLSDomain,
				tcpFastOpen:        outboundConfig.TCPFastOpen,
			}
			if outboundConfig.Timeout != 0 {
				backendInfo.timeout = outboundConfig.Timeout
			} else {
				backendInfo.timeout = config.Generals.Timeout
			}
			backendInfo.restrict = outboundConfig.Restrict
			backendInfo.local = outboundConfig.Local

			Backends.Append(backendInfo)

			stat := common.NewStatistic()
			Statistics.Insert(backendInfo, stat)
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
	common.Debugf("servers in config: %V\n", Backends)

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
