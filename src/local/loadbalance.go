package local

import (
	"errors"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"common"
)

const (
	baseFailCount = 50
	maxFailCount  = 30
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type loadBalanceMethod func(local net.Conn, rawaddr []byte, addr string)

var (
	// Backends collection contains remote server information
	Backends = NewBackendsInformationWrapper()
	// Statistics collections contains all remote servers statistics
	Statistics                          = NewStatisticWrapper()
	outboundLoadBalanceHandler          loadBalanceMethod
	outboundIndex                       int
	smartLastUsedBackendInfo            *BackendInfo
	forceUpdateSmartLastUsedBackendInfo bool
)

func smartCreateServerConn(local net.Conn, rawaddr []byte, addr string, buffer *common.Buffer) (err error) {
	needChangeUsedServerInfo := (smartLastUsedBackendInfo == nil)
	port := uint16(rawaddr[5])<<8 + uint16(rawaddr[6])
	if forceUpdateSmartLastUsedBackendInfo {
		forceUpdateSmartLastUsedBackendInfo = false
		needChangeUsedServerInfo = true
	} else {
		if smartLastUsedBackendInfo != nil {
			if smartLastUsedBackendInfo.restrict == true && port != 80 && port != 443 {
				common.Warning("restrict policy not matched")
				goto pick_server
			}

			if smartLastUsedBackendInfo.firewalled == true && time.Now().Sub(smartLastUsedBackendInfo.lastCheckTimePoint) < 1*time.Hour {
				common.Warning("firewall dropped")
				goto pick_server
			}
			stat, ok := Statistics.Get(smartLastUsedBackendInfo)
			if !ok || stat == nil {
				common.Warning("no statistic record")
				needChangeUsedServerInfo = true
				goto pick_server
			}

			if stat.GetFailedCount() == maxFailCount {
				common.Warning("too many failed count")
				needChangeUsedServerInfo = true
				goto pick_server
			}

			if remote, err := smartLastUsedBackendInfo.connect(rawaddr, addr); err == nil {
				if err, inboundSideError := smartLastUsedBackendInfo.pipe(local, remote, buffer); err == nil {
					return nil
				} else if inboundSideError {
					common.Info("inbound side error")
					return errors.New("Inbound side error")
				}
				common.Warning("piping failed")
			} else {
				common.Warning("connecting failed")
			}
			if stat.GetFailedCount() < maxFailCount {
				stat.IncreaseFailedCount()
				return errors.New("Smart last used server connecting failed")
			}
		}
	}
pick_server:
	ordered := make(BackendsInformation, 0)
	skipped := make(BackendsInformation, 0)
	{
		Backends.RLock()
		for _, bi := range Backends.BackendsInformation {
			if bi == smartLastUsedBackendInfo {
				continue
			}

			if bi.restrict == true && port != 80 && port != 443 {
				continue
			}

			if bi.firewalled == true && time.Now().Sub(bi.lastCheckTimePoint) < 1*time.Hour {
				continue
			}
			ordered = append(ordered, bi)
		}
		Backends.RUnlock()
	}

	sort.Sort(ByHighestLastSecondBps{ordered})
	for _, bi := range ordered {
		// skip failed server, but try it with some probability
		stat, ok := Statistics.Get(bi)
		if !ok || stat == nil {
			continue
		}
		if stat.GetLatency() < 10000000 {
			skipped = append(skipped, bi)
			common.Debugf("too small latency, skip %s\n", bi.address)
			continue
		}
		if stat.GetFailedCount() == maxFailCount || (stat.GetFailedCount() > 0 && rand.Intn(int(stat.GetFailedCount() + baseFailCount)) != 0) {
			skipped = append(skipped, bi)
			common.Debugf("too large failed count, skip %s\n", bi.address)
			continue
		}
		common.Debugf("try %s with failed count %d, %v\n", bi.address, stat.GetFailedCount(), bi)

		if remote, err := bi.connect(rawaddr, addr); err == nil {
			if err, inboundSideError := bi.pipe(local, remote, buffer); err == nil || inboundSideError {
				if needChangeUsedServerInfo {
					smartLastUsedBackendInfo = bi
				}
				return nil
			}
		}
		if stat.GetFailedCount() < maxFailCount {
			stat.IncreaseFailedCount()
		}
		common.Debug("try another available server")
	}

	// last resort, try skipped servers, not likely to succeed
	if len(skipped) > 0 {
		sort.Sort(ByLatency{skipped})
		for _, bi := range skipped {
			stat, ok := Statistics.Get(bi)
			if !ok || stat == nil {
				continue
			}
			common.Debugf("try %s with failed count %d for an additional optunity, %v\n", bi.address, stat.GetFailedCount(), bi)
			if remote, err := bi.connect(rawaddr, addr); err == nil {
				if err, inboundSideError := bi.pipe(local, remote, buffer); err == nil || inboundSideError {
					if needChangeUsedServerInfo {
						smartLastUsedBackendInfo = bi
					}
					return nil
				}
			}
			if stat.GetFailedCount() < maxFailCount {
				stat.IncreaseFailedCount()
			}
			common.Debug("try another skipped server")
		}
	}

	return errors.New("all servers worked abnormally")
}

func smartLoadBalance(local net.Conn, rawaddr []byte, addr string) {
	var buffer *common.Buffer
	maxTryCount := Backends.Len()
	for i := 0; i < maxTryCount; i++ {
		err := smartCreateServerConn(local, rawaddr, addr, buffer)
		if err != nil {
			continue
		}

		break
	}
	common.Debug("closed connection to", addr)
}

func indexSpecifiedCreateServerConn(local net.Conn, rawaddr []byte, addr string) (remote net.Conn, si *BackendInfo, err error) {
	if Backends.Len() == 0 {
		common.Error("no outbound available")
		err = errors.New("no outbound available")
		return
	}
	if outboundIndex >= Backends.Len() {
		//common.Warning("the specified index are out of range, use index 0 now")
		outboundIndex = 0
	}
	s := Backends.Get(outboundIndex)
	stat, ok := Statistics.Get(s)
	if !ok || stat == nil {
		return
	}
	common.Debugf("try %s with failed count %d, %v\n", s.address, stat.GetFailedCount(), s)
	if remote, err = s.connect(rawaddr, addr); err == nil {
		si = s
		return
	}
	if stat.GetFailedCount() < maxFailCount {
		stat.IncreaseFailedCount()
	}
	common.Debug("try another available server")
	return
}

func indexSpecifiedLoadBalance(local net.Conn, rawaddr []byte, addr string) {
	var buffer *common.Buffer
	remote, bi, err := indexSpecifiedCreateServerConn(local, rawaddr, addr)
	if err != nil {
		return
	}
	bi.pipe(local, remote, buffer)
	common.Debug("closed connection to", addr)
}

func roundRobinCreateServerConn(local net.Conn, rawaddr []byte, addr string) (remote net.Conn, si *BackendInfo, err error) {
	outboundIndex++
	return indexSpecifiedCreateServerConn(local, rawaddr, addr)
}

func roundRobinLoadBalance(local net.Conn, rawaddr []byte, addr string) {
	var buffer *common.Buffer
	remote, bi, err := roundRobinCreateServerConn(local, rawaddr, addr)
	if err != nil {
		return
	}
	bi.pipe(local, remote, buffer)
	common.Debug("closed connection to", addr)
}

func handleOutbound(conn net.Conn, rawaddr []byte, addr string) {
	defer conn.Close()
	targetIP := net.IPv4(rawaddr[1], rawaddr[2], rawaddr[3], rawaddr[4])
	port := int(rawaddr[5])<<8 + int(rawaddr[6])
	ipAddr := uint32(rawaddr[4]) + uint32(rawaddr[3])<<8 + uint32(rawaddr[2])<<16 + uint32(rawaddr[1])<<24
	if _, ok := deniedPort[port]; ok {
		common.Warning(conn.RemoteAddr(), "is trying to access denied port", port)
		return
	}
	if config.Target.Port.Deny == "all" {
		if _, ok := allowedPort[port]; !ok {
			common.Warning(conn.RemoteAddr(), "is trying to access not allowed port", port)
			return
		}
	}
	if _, ok := deniedIP[ipAddr]; ok {
		common.Warning(conn.RemoteAddr(), "is trying to access denied IP", targetIP)
		return
	}
	if config.Target.IP.Deny == "all" {
		if _, ok := allowedIP[ipAddr]; !ok {
			common.Warning(conn.RemoteAddr(), "is trying to access not allowed IP", targetIP)
			return
		}
	}
	if p, ok := serverIP[ipAddr]; ok && port == p {
		common.Warningf("%v is trying to access proxy server %v:%d",
			conn.RemoteAddr(), targetIP, port)
		Backends.RLock()
		defer Backends.RUnlock()
		for _, si := range Backends.BackendsInformation {
			for _, ip := range si.ips {
				if ip.Equal(targetIP) {
					si.firewalled = true
					break
				}
			}
		}
		return
	}
	common.Debug("try to access:", targetIP, port)

	if outboundLoadBalanceHandler == nil {
		switch config.Generals.LoadBalance {
		case "smart":
			outboundLoadBalanceHandler = smartLoadBalance
		case "roundrobin":
			outboundLoadBalanceHandler = roundRobinLoadBalance
		case "none":
			outboundIndex = 0
			outboundLoadBalanceHandler = indexSpecifiedLoadBalance
		default:
			if strings.Index(config.Generals.LoadBalance, "index:") == 0 {
				if index, err := strconv.Atoi(config.Generals.LoadBalance[6:]); err == nil {
					outboundIndex = index
					outboundLoadBalanceHandler = indexSpecifiedLoadBalance
				} else {
					common.Error("wrong index specified load balance method format, use smart method now")
					outboundLoadBalanceHandler = smartLoadBalance
				}
			}
		}
	}

	outboundLoadBalanceHandler(conn, rawaddr, addr)
}
