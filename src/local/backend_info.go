package local

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"common"
	"common/semaphore"
	"github.com/RouterScript/ProxyClient"
	"outbound/ss"
	"outbound/ss/obfs"
	"outbound/ss/protocol"
	"outbound/ss/ssr"
)

var (
	ERR_NOT_FILTERED = errors.New("server is not filtered out by firewall")
)

type BackendInfo struct {
	id                 string
	address            string
	protocolType       string
	username           string      // auth for http/https/socks
	password           string      // auth for http/https/socks
	insecureSkipVerify bool        // https only
	domain             string      // https only
	obfs               string      // shadowsocksr only
	obfsParam          string      // shadowsocksr only
	obfsData           interface{} // shadowsocksr only
	protocol           string      // shadowsocksr only
	protocolParam      string      // shadowsocksr only
	protocolData       interface{} // shadowsocksr only
	cipher             *ss.Cipher  // shadowsocks/shadowsocksr only
	tcpFastOpen        bool        // shadowsocks/shadowsocksr only
	timeout            int
	restrict           bool
	local              bool
	firewalled         bool
	lastCheckTimePoint time.Time
	ips                []net.IP
}

func (bi *BackendInfo) testLatency(rawaddr []byte, addr string, wg *sync.WaitGroup, sem *semaphore.Semaphore) {
	sem.Acquire()
	defer func() {
		sem.Release()
		wg.Done()
	}()
	startTime := time.Now()
	remote, err := bi.connect(rawaddr, addr)
	if err == nil {
		if remote != nil {
			defer remote.Close()
		}

		bi.firewalled = false
	}
	bi.lastCheckTimePoint = time.Now()
	endTime := time.Now()
	if stat, ok := Statistics.Get(bi); ok {
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

func (bi *BackendInfo) pipe(local net.Conn, remote net.Conn, buffer *common.Buffer) (err error, inboundSideError bool) {
	sig := make(chan bool)
	result := make(chan error)
	stat, ok := Statistics.Get(bi)
	if !ok || stat == nil {
		return errors.New("invalid statistics"), false
	}

	go func() {
		result <- PipeInboundToOutbound(local,
			remote,
			time.Duration(config.InBoundConfig.Timeout)*time.Second,
			time.Duration(bi.timeout)*time.Second,
			stat,
			sig,
			&buffer)
	}()
	err = PipeOutboundToInbound(remote,
		local,
		time.Duration(bi.timeout)*time.Second,
		time.Duration(config.InBoundConfig.Timeout)*time.Second,
		stat,
		sig)
	if err == ERR_WRITE {
		inboundSideError = true
	}
	if err == ERR_READ || err == ERR_NOSIG || err == ERR_SIGFALSE {
		Statistics.StatisticMap[bi].IncreaseFailedCount()
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
	if err == ERR_READ || err == ERR_WRITE || err == ERR_NOSIG || err == ERR_SIGFALSE {
		Statistics.StatisticMap[bi].IncreaseFailedCount()
		common.Errorf("piping inbound to outbound error: %v, at %v\n", err, bi)
		if err == ERR_READ {
			inboundSideError = true
		}
		return
	}

	if neterr, ok := err.(net.Error); ok {
		common.Error("piping inbound to outbound unknown error: ", neterr)
	}

	return nil, false
}

func (bi *BackendInfo) connectToProxy(u string, addr string) (remote net.Conn, err error) {
	dialer := net.Dial
	if bi.timeout != 0 {
		dialer = proxyclient.DialWithTimeout(time.Duration(bi.timeout) * time.Second)
	}
	p, err := proxyclient.NewProxyClientWithDial(u, dialer)

	if err != nil {
		common.Error("creating proxy client failed", u, err)
		return
	}

	remote, err = p("tcp", addr)
	if err != nil {
		common.Error("connecting to target failed.", u, addr, err)
	}
	return
}

func (bi *BackendInfo) connect(rawaddr []byte, addr string) (remote net.Conn, err error) {
	switch bi.protocolType {
	case "https", "socks5+tls":
		u := url.URL{
			Scheme:bi.protocolType,
			User:  url.UserPassword(bi.username, bi.password),
			Host:  bi.address,
		}
		v := u.Query()
		if bi.insecureSkipVerify {
			v.Set("tls-insecure-skip-verify", "true")
		}
		if len(bi.domain) > 0 {
			v.Set("tls-domain", bi.domain)
		}
		if len(v) > 0 {
			u.RawQuery = v.Encode()
		}
		return bi.connectToProxy(u.String(), addr)
	case "http", "socks4", "socks4a", "socks5":
		u := url.URL{
			Scheme:bi.protocolType,
			User:  url.UserPassword(bi.username, bi.password),
			Host:  bi.address,
		}
		return bi.connectToProxy(u.String(), addr)
	case "shadowsocks", "ss":
		if bi.firewalled == true && time.Now().Sub(bi.lastCheckTimePoint) < 1*time.Hour {
			err = ERR_NOT_FILTERED
			common.Warningf("server %s is not filtered out by firewall.\n", bi.address)
			return
		}

		var ssconn *ss.SSTCPConn
		priorityInterfaceAddress := config.Generals.PriorityInterfaceAddress
		if !config.Generals.PriorityInterfaceEnabled {
			priorityInterfaceAddress = ""
		}

		if ssconn, err = ss.DialShadowsocks(bi.address, bi.cipher.Copy(), priorityInterfaceAddress); err != nil {
			return nil, err
		}
		ss.ProtectSocket(ssconn)

		// should initialize obfs/protocol now
		rs := strings.Split(ssconn.RemoteAddr().String(), ":")
		port, _ := strconv.Atoi(rs[1])

		ssconn.IObfs = obfs.NewObfs(bi.obfs)
		obfsServerInfo := &ssr.ServerInfoForObfs{
			Host:       rs[0],
			Port:       uint16(port),
			TcpMss:     1460,
			Param:      bi.obfsParam,
		}
		ssconn.IObfs.SetServerInfo(obfsServerInfo)
		if bi.obfsData == nil {
			bi.obfsData = ssconn.IObfs.GetData()
		}
		ssconn.IObfs.SetData(bi.obfsData)

		ssconn.IProtocol = protocol.NewProtocol(bi.protocol)
		protocolServerInfo := &ssr.ServerInfoForObfs{
			Host:       rs[0],
			Port:       uint16(port),
			TcpMss:     1460,
			Param:      bi.protocolParam,
		}
		ssconn.IProtocol.SetServerInfo(protocolServerInfo)
		if bi.protocolData == nil {
			bi.protocolData = ssconn.IProtocol.GetData()
		}
		ssconn.IProtocol.SetData(bi.protocolData)

		if _, err = ssconn.Write(rawaddr); err != nil {
			ssconn.Close()
			return nil, err
		}
		remote = ssconn
	default:
		return nil, fmt.Errorf("Unknown backend protocol type: %s", bi.protocolType)
	}
	return
}

type BackendsInformation []*BackendInfo

func (slice BackendsInformation) Len() int {
	return len(slice)
}

func (slice BackendsInformation) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

type ByLastSecondBps struct{ BackendsInformation }

func (slice ByLastSecondBps) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetLastSecondBps() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetLastSecondBps()
}

type ByLastMinuteBps struct{ BackendsInformation }

func (slice ByLastMinuteBps) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetLastMinuteBps() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetLastMinuteBps()
}

type ByLastTenMinutesBps struct{ BackendsInformation }

func (slice ByLastTenMinutesBps) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetLastTenMinutesBps() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetLastTenMinutesBps()
}

type ByLastHourBps struct{ BackendsInformation }

func (slice ByLastHourBps) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetLastHourBps() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetLastHourBps()
}

type ByHighestLastSecondBps struct{ BackendsInformation }

func (slice ByHighestLastSecondBps) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetHighestLastSecondBps() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetHighestLastSecondBps()
}

type ByHighestLastMinuteBps struct{ BackendsInformation }

func (slice ByHighestLastMinuteBps) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetHighestLastMinuteBps() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetHighestLastMinuteBps()
}

type ByHighestLastTenMinutesBps struct{ BackendsInformation }

func (slice ByHighestLastTenMinutesBps) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetHighestLastTenMinutesBps() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetHighestLastTenMinutesBps()
}

type ByHighestLastHourBps struct{ BackendsInformation }

func (slice ByHighestLastHourBps) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetHighestLastHourBps() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetHighestLastHourBps()
}

type ByLatency struct{ BackendsInformation }

func (slice ByLatency) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	if Statistics.StatisticMap[slice.BackendsInformation[i]].GetLatency() == 0 {
		return false
	}
	if Statistics.StatisticMap[slice.BackendsInformation[j]].GetLatency() == 0 {
		return true
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetLatency() < Statistics.StatisticMap[slice.BackendsInformation[j]].GetLatency()
}

type ByFailedCount struct{ BackendsInformation }

func (slice ByFailedCount) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetFailedCount() < Statistics.StatisticMap[slice.BackendsInformation[j]].GetFailedCount()
}

type ByTotalUpload struct{ BackendsInformation }

func (slice ByTotalUpload) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetTotalUploaded() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetTotalUploaded()
}

type ByTotalDownload struct{ BackendsInformation }

func (slice ByTotalDownload) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetTotalDownload() < Statistics.StatisticMap[slice.BackendsInformation[j]].GetTotalDownload()
}

type BackendsInformationWrapper struct {
	sync.RWMutex
	BackendsInformation
}

func NewBackendsInformationWrapper() *BackendsInformationWrapper {
	biw := &BackendsInformationWrapper{}
	biw.BackendsInformation = make(BackendsInformation, 0)
	return biw
}

func (biw *BackendsInformationWrapper) Append(bi *BackendInfo) {
	biw.Lock()
	defer biw.Unlock()
	biw.BackendsInformation = append(biw.BackendsInformation, bi)
}

func (biw *BackendsInformationWrapper) Remove(i int) {
	biw.Lock()
	defer biw.Unlock()
	biw.BackendsInformation = append(biw.BackendsInformation[:i], biw.BackendsInformation[i + 1:]...)
}

func (biw *BackendsInformationWrapper) Get(i int) *BackendInfo {
	biw.RLock()
	defer biw.RUnlock()
	if i >= len(biw.BackendsInformation) {
		return nil
	}
	return biw.BackendsInformation[i]
}

func (biw *BackendsInformationWrapper) Len() int {
	biw.RLock()
	defer biw.RUnlock()
	return len(biw.BackendsInformation)
}
