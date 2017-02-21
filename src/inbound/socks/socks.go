package socks

import (
	"encoding/binary"
	"net"

	"common"
	"common/domain"
	iputil "common/ip"
	"inbound"
)

func handleTCPInbound(conn *net.TCPConn, outboundHandler common.TCPOutboundHandler) error {
	common.Debugf("socks connect from %s\n", conn.RemoteAddr().String())
	conf := &SocksServerConfig{}
	s, err := NewSocks5Server(conf)
	if err != nil {
		common.Error("creating socks5 server failed", err)
		return err
	}
	req, err := s.GetRequest(conn)
	if err != nil {
		common.Error("getting socks5 request failed", err)
		return err
	}

	var rawaddr []byte
	if req.DestAddr.IP.To4() != nil {
		// IPv4
		rawaddr = make([]byte, 7)
		// address type, 1 - IPv4, 4 - IPv6, 3 - hostname
		rawaddr[0] = 1
		// raw IP address, 4 bytes for IPv4 or 16 bytes for IPv6
		copy(rawaddr[1:5], req.DestAddr.IP)
		// port
		binary.BigEndian.PutUint16(rawaddr[5:], uint16(req.DestAddr.Port))

		if rawaddr[0] == 1 && iputil.IPv4InChina(rawaddr[1:5]) {
			// ipv4 connect directly
			defer conn.Close()
			s.HandleRequest(req, conn)
			return nil
		}
	} else if req.DestAddr.IP.To16() != nil {
		// IPv6
		rawaddr = make([]byte, 19)
		rawaddr[0] = 4
		copy(rawaddr[1:1+16], req.DestAddr.IP)
		binary.BigEndian.PutUint16(rawaddr[17:], uint16(req.DestAddr.Port))
	} else {
		// variant length domain name
		host, _, _ := net.SplitHostPort(req.DestAddr.Address())
		if domain.ToBlock(host) {
			conn.Close()
			return nil
		}
		rawaddr = make([]byte, 1+1+len(host)+2)
		rawaddr[0] = 3
		rawaddr[1] = byte(len(host))
		copy(rawaddr[2:2+len(host)], []byte(host))
		binary.BigEndian.PutUint16(rawaddr[2+len(host):], uint16(req.DestAddr.Port))
	}
	// Sending connection established message immediately to client.
	// This some round trip time for creating socks connection with the client.
	// But if connection failed, the client will get connection reset error.
	_, err = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x43})
	if err != nil {
		common.Debug("send connection confirmation:", err)
		return err
	}
	addr := req.DestAddr.Address()
	outboundHandler(conn, rawaddr, addr)
	return nil
}

func GetTCPInboundHandler(inbound *inbound.Inbound) inbound.TCPInboundHandler {
	return handleTCPInbound
}

func GetUDPInboundHandler(inbound *inbound.Inbound) inbound.UDPInboundHandler {
	return func(conn net.PacketConn, outboundHandler common.UDPOutboundHandler) error {
		common.Debugf("socks5 connect from %s\n", conn.LocalAddr().String())
		return nil
	}
}
