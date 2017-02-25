package local

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"common"
	"config"
	"github.com/RouterScript/ProxyClient"
	"outbound/ss"
)

// CommonProxyInfo fields that auth for http/https/socks
type CommonProxyInfo struct {
	username string
	password string
}

// HTTPSProxyInfo fields that https used only
type HTTPSProxyInfo struct {
	CommonProxyInfo
	insecureSkipVerify bool
	domain             string
}

// SSInfo fields that shadowsocks/shadowsocksr used only
type SSInfo struct {
	encryptMethod   string
	encryptPassword string
	tcpFastOpen     bool
}

// SSRInfo fields that shadowsocksr used only
type SSRInfo struct {
	SSInfo
	obfs          string
	obfsParam     string
	obfsData      interface{}
	protocol      string
	protocolParam string
	protocolData  interface{}
}

// BackendInfo all fields that a backend used
type BackendInfo struct {
	id                 string
	address            string
	protocolType       string
	timeout            time.Duration
	local              bool
	firewalled         bool
	ipv6               bool
	lastCheckTimePoint time.Time
	ips                []net.IP
	HTTPSProxyInfo
	SSRInfo
}

func (bi *BackendInfo) testLatency(rawaddr []byte) {
	startTime := time.Now()
	remote, err := bi.connect(rawaddr)
	if err == nil {
		if remote != nil {
			defer remote.Close()
		}

		bi.firewalled = false
	}
	bi.lastCheckTimePoint = time.Now()
	endTime := time.Now()
	if stat, ok := statistics.Get(bi); ok {
		if err != nil {
			stat.IncreaseFailedCount()
			if stat.GetFailedCount() > 10 {
				stat.SetLatency(math.MaxInt64)
			}
		} else {
			stat.SetLatency(endTime.Sub(startTime).Nanoseconds())
			stat.ClearFailedCount()
		}
	}
}

func (bi *BackendInfo) pipe(local net.Conn, remote net.Conn, buffer *common.Buffer) (inboundSideError bool, err error) {
	sig := make(chan bool)
	result := make(chan error)
	stat, ok := statistics.Get(bi)
	if !ok || stat == nil {
		return false, errors.New("invalid statistics")
	}

	go func() {
		pp := pipeParam{
			local,
			remote,
			config.Configurations.Generals.InboundTimeout,
			bi.timeout,
			stat,
			sig,
		}
		result <- PipeInboundToOutbound(pp, &buffer)
	}()
	pp := pipeParam{
		remote,
		local,
		bi.timeout,
		config.Configurations.Generals.InboundTimeout,
		stat,
		sig,
	}
	err = PipeOutboundToInbound(pp)
	if err == ErrWrite {
		inboundSideError = true
	}
	if err == ErrRead || err == ErrNoSignal || err == ErrSignalFalse {
		statistics.StatisticMap[bi].IncreaseFailedCount()
		common.Errorf("piping outbound to inbound error: %v, at %v\n", err, bi)
		go func() {
			// clear the channel
			<-result
		}()
		return
	}

	if neterr, ok := err.(net.Error); ok {
		common.Error("piping outbound to inbound unknown error: ", neterr)
	}

	err = <-result
	if err == ErrRead || err == ErrWrite || err == ErrNoSignal || err == ErrSignalFalse {
		statistics.StatisticMap[bi].IncreaseFailedCount()
		common.Errorf("piping inbound to outbound error: %v, at %v\n", err, bi)
		if err == ErrRead {
			inboundSideError = true
		}
		return
	}

	if neterr, ok := err.(net.Error); ok {
		common.Error("piping inbound to outbound unknown error: ", neterr)
	}

	return false, nil
}

func connectToProxy(u *url.URL, bi *BackendInfo) (remote net.Conn, err error) {
	dialer := net.Dial
	if bi.timeout != 0 {
		dialer = proxyclient.DialWithTimeout(bi.timeout)
	}
	p, err := proxyclient.NewClientWithDial(u, dialer)

	if err != nil {
		common.Error("creating proxy client failed", *u, err)
		return
	}

	remote, err = p("tcp", u.Host)
	if err != nil {
		common.Error("connecting to target failed.", *u, err)
	}
	return
}

func tlsConnect(_ []byte, bi *BackendInfo) (remote net.Conn, err error) {
	u := &url.URL{
		Scheme: bi.protocolType,
		User:   url.UserPassword(bi.username, bi.password),
		Host:   bi.address,
	}
	v := u.Query()
	v.Set("tls-insecure-skip-verify", strconv.FormatBool(bi.insecureSkipVerify))
	v.Set("tls-domain", bi.domain)
	u.RawQuery = v.Encode()

	return connectToProxy(u, bi)
}

func plainConnect(_ []byte, bi *BackendInfo) (remote net.Conn, err error) {
	u := &url.URL{
		Scheme: bi.protocolType,
		User:   url.UserPassword(bi.username, bi.password),
		Host:   bi.address,
	}
	return connectToProxy(u, bi)
}

func ssConnect(rawaddr []byte, bi *BackendInfo) (remote net.Conn, err error) {
	u := &url.URL{
		Scheme: bi.protocolType,
		Host:   bi.address,
	}

	v := u.Query()
	v.Set("priority-interface-enabled", strconv.FormatBool(config.Configurations.Generals.PriorityInterfaceEnabled))
	v.Set("priority-interface-address", config.Configurations.Generals.PriorityInterfaceAddress)
	v.Set("encrypt-method", bi.encryptMethod)
	v.Set("encrypt-key", bi.encryptPassword)
	v.Set("obfs", bi.obfs)
	v.Set("obfs-param", bi.obfsParam)
	v.Set("protocol", bi.protocol)
	v.Set("protocol-param", bi.protocolParam)
	u.RawQuery = v.Encode()

	if c, e := connectToProxy(u, bi); e == nil {
		return setSSRData(rawaddr, bi, c)
	}
	return nil, errors.New("connecting to ss server failed")
}

func setSSRData(rawaddr []byte, bi *BackendInfo, c net.Conn) (remote net.Conn, err error) {
	ssconn, ok := c.(*ss.SSTCPConn)
	if !ok {
		return nil, errors.New("not a *SSTCPConn")
	}
	if bi.obfsData == nil {
		bi.obfsData = ssconn.IObfs.GetData()
	}
	ssconn.IObfs.SetData(bi.obfsData)

	if bi.protocolData == nil {
		bi.protocolData = ssconn.IProtocol.GetData()
	}
	ssconn.IProtocol.SetData(bi.protocolData)

	if _, err := ssconn.Write(rawaddr); err != nil {
		ssconn.Close()
		return nil, err
	}
	return ssconn, nil
}

type connector func([]byte, *BackendInfo) (net.Conn, error)

var (
	connectorMap = map[string]connector{
		"https":        tlsConnect,
		"socks5+tls":   tlsConnect,
		"http":         plainConnect,
		"socks4":       plainConnect,
		"socks4a":      plainConnect,
		"socks5":       plainConnect,
		"ss":           ssConnect,
		"ssr":          ssConnect,
		"shadowsocks":  ssConnect,
		"shadowsocksr": ssConnect,
	}
)

func (bi *BackendInfo) connect(rawaddr []byte) (remote net.Conn, err error) {
	ctr, ok := connectorMap[strings.ToLower(bi.protocolType)]
	if !ok {
		return nil, fmt.Errorf("Unknown backend protocol type: %s", bi.protocolType)
	}

	return ctr(rawaddr, bi)
}
