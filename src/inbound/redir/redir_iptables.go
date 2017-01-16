// +build linux

package redir

import (
	"common"
	"errors"
	"fmt"
	"net"
	"syscall"
)

const (
	SO_ORIGINAL_DST      = 80
	IP6T_SO_ORIGINAL_DST = 80
)

func getOriginalDst(clientConn *net.TCPConn) (rawaddr []byte, host string, newTCPConn *net.TCPConn, err error) {
	if clientConn == nil {
		common.Errorf("copy(): oops, dst is nil!")
		err = errors.New("ERR: clientConn is nil")
		return
	}

	// test if the underlying fd is nil
	remoteAddr := clientConn.RemoteAddr()
	if remoteAddr == nil {
		common.Errorf("getOriginalDst(): oops, clientConn.fd is nil!")
		err = errors.New("ERR: clientConn.fd is nil")
		return
	}

	srcipport := fmt.Sprintf("%v", clientConn.RemoteAddr())

	newTCPConn = nil
	// net.TCPConn.File() will cause the receiver's (clientConn) socket to be placed in blocking mode.
	// The workaround is to take the File returned by .File(), do getsockopt() to get the original
	// destination, then create a new *net.TCPConn by calling net.Conn.FileConn().  The new TCPConn
	// will be in non-blocking mode.  What a pain.
	clientConnFile, err := clientConn.File()
	if err != nil {
		common.Errorf("GETORIGINALDST|%v->?->FAILEDTOBEDETERMINED|ERR: could not get a copy of the client connection's file object", srcipport)
		return
	} else {
		clientConn.Close()
	}

	// Get original destination
	// this is the only syscall in the Golang libs that I can find that returns 16 bytes
	// Example result: &{Multiaddr:[2 0 31 144 206 190 36 45 0 0 0 0 0 0 0 0] Interface:0}
	// port starts at the 3rd byte and is 2 bytes long (31 144 = port 8080)
	ipv6 := false
	// IPv6 version, didn't find a way to detect network family
	addr, err := syscall.GetsockoptIPv6Mreq(int(clientConnFile.Fd()), syscall.IPPROTO_IPV6, IP6T_SO_ORIGINAL_DST)
	if err == nil {
		ipv6 = true
	} else {
		// IPv4 address starts at the 5th byte, 4 bytes long (206 190 36 45)
		addr, err = syscall.GetsockoptIPv6Mreq(int(clientConnFile.Fd()), syscall.IPPROTO_IP, SO_ORIGINAL_DST)
	}
	common.Debugf("getOriginalDst(): SO_ORIGINAL_DST=%+v\n", addr)
	if err != nil {
		common.Errorf("GETORIGINALDST|%v->?->FAILEDTOBEDETERMINED|ERR: getsocketopt(SO_ORIGINAL_DST) failed: %v", srcipport, err)
		return
	}
	newConn, err := net.FileConn(clientConnFile)
	if err != nil {
		common.Errorf("GETORIGINALDST|%v->?->%v|ERR: could not create a FileConn fron clientConnFile=%+v: %v", srcipport, addr, clientConnFile, err)
		return
	}
	if _, ok := newConn.(*net.TCPConn); ok {
		newTCPConn = newConn.(*net.TCPConn)
		clientConnFile.Close()
	} else {
		errmsg := fmt.Sprintf("ERR: newConn is not a *net.TCPConn, instead it is: %T (%v)", newConn, newConn)
		common.Errorf("GETORIGINALDST|%v->?->%v|%s", srcipport, addr, errmsg)
		err = errors.New(errmsg)
		return
	}

	if ipv6 {
		rawaddr = make([]byte, 19)
		// address type, 1 - IPv4, 4 - IPv6, 3 - hostname
		rawaddr[0] = 4
		//! \attention seemly won't work
		// raw IP address, 4 bytes for IPv4 or 16 bytes for IPv6
		copy(rawaddr[1:], addr.Multiaddr[4:])
		// port
		copy(rawaddr[1 + 16:], addr.Multiaddr[2:2 + 2])
	} else {
		rawaddr = make([]byte, 7)
		// address type, 1 - IPv4, 4 - IPv6, 3 - hostname
		rawaddr[0] = 1
		// raw IP address, 4 bytes for IPv4 or 16 bytes for IPv6
		copy(rawaddr[1:], addr.Multiaddr[4:4 + 4])
		// port
		copy(rawaddr[1 + 4:], addr.Multiaddr[2:2 + 2])

		host = fmt.Sprintf("%d.%d.%d.%d:%d",
			addr.Multiaddr[4],
			addr.Multiaddr[5],
			addr.Multiaddr[6],
			addr.Multiaddr[7],
			uint16(addr.Multiaddr[2])<<8+uint16(addr.Multiaddr[3]))
	}

	return
}

func HandleInbound(conn *net.TCPConn, outboundHander common.OutboundHandler) {
	common.Debugf("redir connect from %s, Linux is detected, use iptables now.\n",
		conn.RemoteAddr().String())

	if conn == nil {
		common.Errorf("handleRedirInbound(): oops, conn is nil")
		return
	}

	// test if the underlying fd is nil
	remoteAddr := conn.RemoteAddr()
	if remoteAddr == nil {
		common.Errorf("handleRedirInbound(): oops, conn.fd is nil!")
		return
	}

	rawaddr, addr, conn, err := getOriginalDst(conn)
	if err != nil {
		common.Errorf("handleRedirInbound(): can not handle this connection, error occurred in getting original destination ip address/port: %+v\n", err)
		return
	}

	outboundHander(conn, rawaddr, addr)
}
