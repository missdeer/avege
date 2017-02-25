package local

import (
	"encoding/binary"
	"errors"
	"net"
	"strconv"
	"strings"

	"common"
	"config"
)

var (
	ErrDeniedPort        = errors.New("try to access denied port")
	ErrNotAllowedPort    = errors.New("try to access not allowed port")
	ErrDeniedIP          = errors.New("try to access denied IP")
	ErrNotAllowedIP      = errors.New("try to access not allowed IP")
	ErrLoopProxy         = errors.New("try to access proxy server")
	loadBalancePolicyMap = map[string]func(){
		"smart":      prepareSmartLoadBalance,
		"roundrobin": prepareRoundRobinLoadBalance,
		"none":       prepareNoneLoadBalance,
	}
	serverIP = make(map[uint32]int) // IPv4 only
)

func filterIPv4TCPOutbound(conn net.Conn, rawaddr []byte) error {
	targetIP := net.IPv4(rawaddr[1], rawaddr[2], rawaddr[3], rawaddr[4])
	port := int(binary.BigEndian.Uint16(rawaddr[5:]))
	ipAddr := binary.BigEndian.Uint32(rawaddr[1:5])
	if _, ok := config.DeniedPort[port]; ok {
		common.Warning(conn.RemoteAddr(), "is trying to access denied port", port)
		return ErrDeniedPort
	}
	if config.Configurations.Target.Port.Deny == "all" {
		if _, ok := config.AllowedPort[port]; !ok {
			common.Warning(conn.RemoteAddr(), "is trying to access not allowed port", port)
			return ErrNotAllowedPort
		}
	}
	if _, ok := config.DeniedIP[ipAddr]; ok {
		common.Warning(conn.RemoteAddr(), "is trying to access denied IP", targetIP)
		return ErrDeniedIP
	}
	if config.Configurations.Target.IP.Deny == "all" {
		if _, ok := config.AllowedIP[ipAddr]; !ok {
			common.Warning(conn.RemoteAddr(), "is trying to access not allowed IP", targetIP)
			return ErrNotAllowedIP
		}
	}
	if p, ok := serverIP[ipAddr]; ok && port == p {
		common.Warningf("%v is trying to access proxy server %v:%d",
			conn.RemoteAddr(), targetIP, port)
		backends.RLock()
		defer backends.RUnlock()
		for _, si := range backends.BackendsInformation {
			for _, ip := range si.ips {
				if ip.Equal(targetIP) {
					si.firewalled = true
					break // may be more than one IPs, so continue looping
				}
			}
		}
		return ErrLoopProxy
	}
	common.Debug("try to access:", targetIP, port)
	return nil
}

func prepareSmartLoadBalance() {
	outboundLoadBalanceHandler = smartLoadBalance
}

func prepareRoundRobinLoadBalance() {
	outboundLoadBalanceHandler = roundRobinLoadBalance
}

func prepareNoneLoadBalance() {
	outboundIndex = 0
	outboundLoadBalanceHandler = indexSpecifiedLoadBalance
}

func prepareDefaultLoadBalance() {
	if strings.Index(config.Configurations.Generals.LoadBalance, "index:") == 0 {
		if index, err := strconv.Atoi(config.Configurations.Generals.LoadBalance[6:]); err == nil {
			outboundIndex = index
			outboundLoadBalanceHandler = indexSpecifiedLoadBalance
		} else {
			common.Error("wrong index specified load balance method format, use smart method now")
			outboundLoadBalanceHandler = smartLoadBalance
		}
	}
}

func handleTCPOutbound(conn net.Conn, rawaddr []byte, _ string) error {
	defer conn.Close()
	switch rawaddr[0] {
	case 1:
		// IPv4
		if err := filterIPv4TCPOutbound(conn, rawaddr); err != nil {
			return err
		}
		//case 3:
		// variant length domain name
		//case 4:
		// IPv6
	}

	if outboundLoadBalanceHandler == nil {
		if prepareLoadBalance, ok := loadBalancePolicyMap[config.Configurations.Generals.LoadBalance]; ok {
			prepareLoadBalance()
		} else {
			prepareDefaultLoadBalance()
		}
	}

	outboundLoadBalanceHandler(conn, rawaddr)
	return nil
}
