package socks

import (
	"net"
	"common"
	iputil "common/ip"
)

func HandleInbound(conn *net.TCPConn, outboundHander common.OutboundHandler) {
	common.Debugf("socks connect from %s\n", conn.RemoteAddr().String())
	conf := &SocksServerConfig{}
	s, err := NewSocks5Server(conf)
	if err != nil {
		common.Error("creating socks5 server failed", err)
		return
	}
	req, err := s.GetRequest(conn)
	if err != nil {
		common.Error("getting socks5 request failed", err)
		return
	}

	// \attention: IPv4 only!!!
	rawaddr := make([]byte, 7)
	// address type, 1 - IPv4, 4 - IPv6, 3 - hostname, only IPv4 is supported now
	rawaddr[0] = 1
	// raw IP address, 4 bytes for IPv4 or 16 bytes for IPv6, only IPv4 is supported now
	copy(rawaddr[1:5], req.DestAddr.IP)
	// port
	rawaddr[5] = byte(req.DestAddr.Port / 256)
	rawaddr[6] = byte(req.DestAddr.Port % 256)

	if rawaddr[0] == 1 && iputil.IPv4InChina(rawaddr[1:5]) {
		// ipv4 connect directly
		defer conn.Close()
		s.HandleRequest(req, conn)
		return
	}
	// Sending connection established message immediately to client.
	// This some round trip time for creating socks connection with the client.
	// But if connection failed, the client will get connection reset error.
	_, err = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x43})
	if err != nil {
		common.Debug("send connection confirmation:", err)
		return
	}
	addr := req.DestAddr.Address()
	outboundHander(conn, rawaddr, addr)
}
