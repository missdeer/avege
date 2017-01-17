package ss

import (
	"encoding/binary"
	"net"
	"strconv"
	"strings"
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

	pia := strings.Split(priorityInterfaceAddress, "/")
	if len(pia) != 2 {
		common.Warning("unexpected priority interface address format, something like 192.168.1.0/24 is expected.")
		return
	}

	// ipv4 only
	piaddr := net.ParseIP(pia[0])
	if piaddr == nil || piaddr.To4() == nil {
		common.Warning("incorrect IP for priority interface address, only IPv4 is supported currently.", pia[0])
		return
	}

	mask, err := strconv.Atoi(pia[1])
	if err != nil {
		common.Warning("incorrect network address mask, 1-32 is required", pia[1])
		return
	}

	piaddrInt32 := binary.BigEndian.Uint32(piaddr.To4())
	var finalMask uint32
	for i := 0; i < mask; i++ {
		var bit uint32 = 0x80000000
		bit >>= uint(i)
		finalMask |= bit
	}
	piaddrInt32 &= finalMask

	addrs, err := net.InterfaceAddrs();
	if err != nil {
		common.Warning("can't get interface addresses", err)
		return
	}
	for _, addr := range addrs {
		if a := strings.Split(addr.String(), "/"); len(a) == 2 {
			ip := net.ParseIP(a[0])
			if ip == nil || ip.To4() == nil {
				continue
			}

			ipInt32 := binary.BigEndian.Uint32(ip.To4()) & finalMask
			if ipInt32 == piaddrInt32 {
				tcpAddr = &net.TCPAddr{
					IP: ip,
				}
				break
			}
		}
	}
}

// rawaddr shoud contain part of the data in socks request, starting from the
// ATYP field. (Refer to rfc1928 for more information.)
func Dial(host string, cipher *Cipher, priorityInterfaceAddress string) (c *SSTCPConn, err error) {
	setLocalAddress(priorityInterfaceAddress)

	var conn net.Conn
	if tcpAddr != nil {
		dialer := net.Dialer{
			LocalAddr: tcpAddr,
			DualStack: true,
		}

		if conn, err = dialer.Dial("tcp", host); err == nil {
			return NewSSConn(conn, cipher), nil
		}
		common.Warning("dialing on the interface with priorityInterfaceAddress", priorityInterfaceAddress, err)
		tcpAddr = nil
	}

	if conn, err = dualStackDialer.Dial("tcp", host); err == nil {
		return NewSSConn(conn, cipher), nil
	}

	return
}
