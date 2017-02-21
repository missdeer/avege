package ss

import (
	"net"
	"time"

	"common"
)

var (
	tcpAddr *net.TCPAddr

	dualStackDialer = net.Dialer{
		DualStack: true,
	}

	lastCheckTimestamp time.Time
)

func setLocalAddress(priorityInterfaceAddress string) {
	now := time.Now()
	if now.Sub(lastCheckTimestamp) < 10*time.Minute {
		return
	}
	lastCheckTimestamp = now

	if tcpAddr != nil || len(priorityInterfaceAddress) == 0 {
		return
	}

	_, priorityInterfaceAddressIPNet, e := net.ParseCIDR(priorityInterfaceAddress)
	if e != nil {
		common.Error("parsing priority interface address failed", priorityInterfaceAddress, e)
		return
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		common.Warning("can't get interface addresses", err)
		return
	}

	for _, addr := range addrs {
		if ip, _, e := net.ParseCIDR(addr.String()); e == nil && priorityInterfaceAddressIPNet.Contains(ip) {
			tcpAddr = &net.TCPAddr{
				IP: ip,
			}
			break
		}
	}
}

// rawaddr shoud contain part of the data in socks request, starting from the
// ATYP field. (Refer to rfc1928 for more information.)
func Dial(host string, cipher *StreamCipher, priorityInterfaceAddress string) (c *SSTCPConn, err error) {
	setLocalAddress(priorityInterfaceAddress)

	var conn net.Conn
	if tcpAddr != nil {
		dialer := net.Dialer{
			LocalAddr: tcpAddr,
			DualStack: true,
		}

		if conn, err = dialer.Dial("tcp", host); err == nil {
			return NewSSTCPConn(conn, cipher), nil
		}
		common.Warning("dialing on the interface with priorityInterfaceAddress", priorityInterfaceAddress, err)
		tcpAddr = nil
	}

	if conn, err = dualStackDialer.Dial("tcp", host); err == nil {
		return NewSSTCPConn(conn, cipher), nil
	}

	return
}
