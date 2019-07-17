package outbound

import (
	"encoding/json"
	"time"
)

type SSInfo struct {
	// Key shadowsocks only key used to encrypting
	Key string `json:"key"`
	// Method shadowsocks encrypting algorithm, eg. rc4-md5, aes-256-cfb etc.
	Method string `json:"method"`
	// TCPFastOpen == true if this backend supports TCP Fast Open
	TCPFastOpen bool `json:"tcpfastopen"`
}

type SSRInfo struct {
	SSInfo
	// Protocol shadowsocks only obfs protocol
	Protocol string `json:"protocol"`
	// ProtocolParam shadowsocks only obfs protocol parameter
	ProtocolParam string `json:"pparam"`
	// Obfs shadowsocks only obfs
	Obfs string `json:"obfs"`
	// ObfsParam shadowsocks only obfs parameter
	ObfsParam string `json:"oparam"`
}

type CommonProxyInfo struct {
	// Username auth for http/https/socks
	Username string `json:"username"`
	// Password auth for http/https/socks
	Password string `json:"password"`
}

type HTTPSProxyInfo struct {
	CommonProxyInfo
	// TLSInsecureSkipVerify  https only
	TLSInsecureSkipVerify bool `json:"insecureskipverify"`
	// TLSDomain https only
	TLSDomain string `json:"domain"`
}

// Outbound configuration struct that represents the outbound
type Outbound struct {
	// Address outbound network address, in Host:Port format
	Address string `json:"address"`
	// Type protocol type, http/https/socks4/socks4a/socks5/shadowsocks are supported
	Type string `json:"type"`
	// Timeout connecting timeout
	Timeout time.Duration `json:"timeout"`
	// Local == true if this configuration item is from local config file, otherwise it's from remote console server's pushing
	Local bool `json:"local"`
	HTTPSProxyInfo
	SSRInfo
}

// UnmarshalJSON override the json unmarshal method, so that some fields could be initialized correctly
func (o *Outbound) UnmarshalJSON(b []byte) error {
	type Alias Outbound
	aux := &struct {
		Timeout string `json:"timeout"`
		*Alias
	}{
		Alias: (*Alias)(o),
	}
	aux.Obfs = "plain"
	aux.Protocol = "origin"
	aux.Type = "shadowsocks"
	err := json.Unmarshal(b, &aux)
	if err != nil {
		return err
	}
	o.Timeout, _ = time.ParseDuration(aux.Timeout)
	return nil
}
