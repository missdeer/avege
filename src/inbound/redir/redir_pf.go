// +build openbsd netbsd darwin

package redir

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"syscall"
	"unsafe"

	"common"
	"inbound"
)

type Natlook struct {
	saddr     [16]byte
	daddr     [16]byte
	rsaddr    [16]byte
	rdaddr    [16]byte
	sxport    [4]byte
	dxport    [4]byte
	rxsport   [4]byte
	rxdport   [4]byte
	af        uint8
	proto     uint8
	direction uint8
}

func doNatLook(data unsafe.Pointer) (err syscall.Errno) {
	pfdev, _ := syscall.Open("/dev/pf", syscall.O_RDONLY, 0666)

	_, _, err = syscall.RawSyscall(syscall.SYS_IOCTL, uintptr(pfdev), uintptr(3226747927), uintptr(data))
	if err != 0 {
		syscall.Close(pfdev)
		common.Errorf("got error: %T(%v) = %d", err, err, err)
		return
	}
	syscall.Close(pfdev)
	return
}

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

	remoteHost, remotePortStr, err := net.SplitHostPort(clientConn.RemoteAddr().String())
	remotePortInt, _ := strconv.Atoi(remotePortStr)
	localHost, localPortStr, _ := net.SplitHostPort(clientConn.LocalAddr().String())
	localPortInt, _ := strconv.Atoi(localPortStr)

	natlook := Natlook{}
	natlook.af = syscall.AF_INET
	natlook.saddr[0] = net.ParseIP(remoteHost)[12]
	natlook.saddr[1] = net.ParseIP(remoteHost)[13]
	natlook.saddr[2] = net.ParseIP(remoteHost)[14]
	natlook.saddr[3] = net.ParseIP(remoteHost)[15]
	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, uint32(remotePortInt))

	natlook.sxport[0] = bs[2]
	natlook.sxport[1] = bs[3]

	natlook.daddr[0] = net.ParseIP(localHost)[12]
	natlook.daddr[1] = net.ParseIP(localHost)[13]
	natlook.daddr[2] = net.ParseIP(localHost)[14]
	natlook.daddr[3] = net.ParseIP(localHost)[15]
	bs2 := make([]byte, 4)
	binary.BigEndian.PutUint32(bs2, uint32(localPortInt))
	natlook.dxport[0] = bs2[2]
	natlook.dxport[1] = bs2[3]
	natlook.proto = syscall.IPPROTO_TCP
	natlook.direction = 3
	common.Errorf("before(natlook): %v", natlook)

	doNatLook(unsafe.Pointer(&natlook))

	newConn, err := net.FileConn(clientConnFile)
	if err != nil {
		common.Errorf("GETORIGINALDST|%v->?->%v|ERR: could not create a FileConn fron clientConnFile=%+v: %v", srcipport, natlook, clientConnFile, err)
		return
	}
	if _, ok := newConn.(*net.TCPConn); ok {
		newTCPConn = newConn.(*net.TCPConn)
		clientConnFile.Close()
	} else {
		errmsg := fmt.Sprintf("ERR: newConn is not a *net.TCPConn, instead it is: %T (%v)", newConn, newConn)
		common.Errorf("GETORIGINALDST|%v->?->%v|%s", srcipport, natlook, errmsg)
		err = errors.New(errmsg)
		return
	}

	rawaddr = append(rawaddr, byte(1))
	rawaddr = append(rawaddr, natlook.rdaddr[0])
	rawaddr = append(rawaddr, natlook.rdaddr[1])
	rawaddr = append(rawaddr, natlook.rdaddr[2])
	rawaddr = append(rawaddr, natlook.rdaddr[3])
	rawaddr = append(rawaddr, natlook.rxdport[0])
	rawaddr = append(rawaddr, natlook.rxdport[1])

	dportBytes := make([]byte, 2)
	dportBytes[1] = natlook.rxdport[0]
	dportBytes[0] = natlook.rxdport[1]
	var port uint16
	binary.Read(bytes.NewBuffer(dportBytes[:]), binary.LittleEndian, &port)

	host = fmt.Sprintf("%d.%d.%d.%d:%d",
		natlook.rdaddr[0],
		natlook.rdaddr[1],
		natlook.rdaddr[2],
		natlook.rdaddr[3],
		port)
	return
}

func handleTCPInbound(conn *net.TCPConn, outboundHandler common.TCPOutboundHandler) error {
	common.Debugf("redir connect from %s, BSD is detected, use pf now.\n",
		conn.RemoteAddr().String())

	if conn == nil {
		common.Errorf("handleRedirInbound(): oops, conn is nil")
		return errors.New("input conn is nil")
	}

	// test if the underlying fd is nil
	remoteAddr := conn.RemoteAddr()
	if remoteAddr == nil {
		common.Errorf("handleRedirInbound(): oops, conn.fd is nil!")
		return errors.New("input conn.fd is nil")
	}

	rawaddr, addr, conn, err := getOriginalDst(conn)
	if err != nil {
		common.Errorf("handleRedirInbound(): can not handle this connection, error occurred in getting original destination ip address/port: %+v\n", err)
		return err
	}

	outboundHandler(conn, rawaddr, addr)
	return nil
}

func GetTCPInboundHandler(inbound *inbound.Inbound) inbound.TCPInboundHandler {
	return handleTCPInbound
}

func GetUDPInboundHandler(inbound *inbound.Inbound) inbound.UDPInboundHandler {
	return func(conn net.PacketConn, outboundHandler common.UDPOutboundHandler) error {
		common.Debugf("redir connect from %s\n", conn.LocalAddr().String())
		return nil
	}
}
