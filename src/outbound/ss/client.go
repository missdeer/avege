package ss

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/RouterScript/ProxyClient"
	"outbound/ss/obfs"
	"outbound/ss/protocol"
	"outbound/ss/ssr"
)

func init() {
	proxyclient.RegisterScheme("ssr", newSSRClient)
	proxyclient.RegisterScheme("ss", newSSRClient)
	proxyclient.RegisterScheme("shadowsocks", newSSRClient)
	proxyclient.RegisterScheme("shadowsocksr", newSSRClient)
}

func newSSRClient(u *url.URL, _ proxyclient.Dial) (proxyclient.Dial, error) {
	return func(network, address string) (net.Conn, error) {
		query := u.Query()

		priorityInterfaceAddress := query.Get("priority-interface-address")
		priorityInterfaceEnabled, err := strconv.ParseBool(query.Get("priority-interface-enabled"))
		if err != nil {
			return nil, err
		}
		encryptMethod := query.Get("encrypt-method")
		encryptKey := query.Get("encrypt-key")
		cipher, err := NewStreamCipher(encryptMethod, encryptKey)
		if err != nil {
			return nil, err
		}

		if !priorityInterfaceEnabled {
			priorityInterfaceAddress = ""
		}

		var ssconn *SSTCPConn
		if ssconn, err = Dial(address, cipher, priorityInterfaceAddress); err != nil {
			return nil, err
		}

		if ssconn.Conn, err = ProtectSocket(ssconn.Conn); err != nil {
			return nil, err
		}

		if ssconn.Conn == nil || ssconn.RemoteAddr() == nil {
			return nil, errors.New("nil connection")
		}

		// should initialize obfs/protocol now
		rs := strings.Split(ssconn.RemoteAddr().String(), ":")
		port, _ := strconv.Atoi(rs[1])

		ssconn.IObfs = obfs.NewObfs(query.Get("obfs"))
		obfsServerInfo := &ssr.ServerInfoForObfs{
			Host:   rs[0],
			Port:   uint16(port),
			TcpMss: 1460,
			Param:  query.Get("obfs-param"),
		}
		ssconn.IObfs.SetServerInfo(obfsServerInfo)

		ssconn.IProtocol = protocol.NewProtocol(query.Get("protocol"))
		protocolServerInfo := &ssr.ServerInfoForObfs{
			Host:   rs[0],
			Port:   uint16(port),
			TcpMss: 1460,
			Param:  query.Get("protocol-param"),
		}
		ssconn.IProtocol.SetServerInfo(protocolServerInfo)

		return ssconn, nil
	}, nil
}
