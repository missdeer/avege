package tunnel

import (
	"encoding/binary"
	"net"
	"strconv"

	"common"
	"inbound"
)

func parseAddr(inbound *inbound.Inbound) (rawaddr []byte, addr string) {
	host, p, err := net.SplitHostPort(inbound.Parameter)
	if err != nil {
		common.Error("incorrect inbound parameter format", inbound.Parameter, err)
		return
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		common.Error("can't convert port string", p, err)
		return
	}
	ip := net.ParseIP(host)
	if ip == nil {
		// variant length domain name
		rawaddr = make([]byte, 1+1+len(host)+2)
		rawaddr[0] = 3
		rawaddr[1] = byte(len(host))
		copy(rawaddr[2:2+len(host)], []byte(host))
		binary.BigEndian.PutUint16(rawaddr[2+len(host):], uint16(port))
	} else if ip.To4() != nil {
		// IPv4
		rawaddr = make([]byte, 7)
		// address type, 1 - IPv4, 4 - IPv6, 3 - hostname
		rawaddr[0] = 1
		// raw IP address, 4 bytes for IPv4 or 16 bytes for IPv6
		copy(rawaddr[1:5], ip.To4())
		// port
		binary.BigEndian.PutUint16(rawaddr[5:], uint16(port))
	} else if ip.To16() != nil {
		// IPv6
		rawaddr = make([]byte, 19)
		rawaddr[0] = 4
		copy(rawaddr[1:1+16], ip.To16())
		binary.BigEndian.PutUint16(rawaddr[17:], uint16(port))
	} else {

	}
	addr = inbound.Parameter
	return
}

func GetUDPInboundHandler(inbound *inbound.Inbound) inbound.UDPInboundHandler {
	rawaddr, addr := parseAddr(inbound)
	return func(c net.PacketConn, outboundHandler common.UDPOutboundHandler) error {
		common.Debug("tunnel connect from", c.LocalAddr().String())
		return outboundHandler(c, rawaddr, addr)
	}
}

func GetTCPInboundHandler(inbound *inbound.Inbound) inbound.TCPInboundHandler {
	rawaddr, addr := parseAddr(inbound)
	return func(conn *net.TCPConn, outboundHandler common.TCPOutboundHandler) error {
		common.Debugf("tunnel connect from %s\n", conn.RemoteAddr().String())
		return outboundHandler(conn, rawaddr, addr)
	}
}
