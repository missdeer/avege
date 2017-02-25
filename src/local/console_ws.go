package local

import (
	"encoding/json"
	"net"
	"time"

	"common"
	"config"
	"github.com/gorilla/websocket"
	"inbound"
	"outbound"
	"outbound/ss"
	"rule"
)

var (
	consoleWSUrl = "wss://www.console.com/v1/ws"
	writeWait    = 60 * time.Second
	pingWait     = 10 * time.Second
	wsDialer     = websocket.Dialer{
		Subprotocols:    []string{"p1", "p2"},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type Connection struct {
	conn *websocket.Conn
	msg  chan []byte
	done chan bool
}

func (c *Connection) pingHandler(s string) error {
	c.conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(pingWait))
	return nil
}

func (c *Connection) handleWS(msg []byte) []byte {
	var m common.WebsocketMessage
	err := json.Unmarshal(msg, &m)
	if err != nil {
		res := common.WebsocketMessage{
			Cmd: common.CMD_ERROR,
		}
		r, _ := json.Marshal(res)
		return r
	}

	switch m.Cmd {
	case common.CMD_START_REVERSE_SSH:
		m.Cmd = common.CMD_REVERSE_SSH_STARTED
	case common.CMD_STOP_REVERSE_SSH:
		m.Cmd = common.CMD_REVERSE_SSH_STOPPED
	case common.CMD_NEW_RULES:
		if inbound.IsModeEnabled("redir") {
			go rule.UpdateRedirFirewallRules()
		}
		m.Cmd = common.CMD_RESPONSE
		m.WParam = "ok"
	case common.CMD_ADD_SERVER:
		addServer(m.WParam)
		m.Cmd = common.CMD_RESPONSE
		m.WParam = "ok"
	case common.CMD_DEL_SERVER:
		removeServer(m.WParam)
		m.Cmd = common.CMD_RESPONSE
		m.WParam = "ok"
	case common.CMD_SET_PORT:
		config.DefaultPort = m.WParam
		changePort()
		m.Cmd = common.CMD_RESPONSE
		m.WParam = "ok"
	case common.CMD_SET_KEY:
		config.DefaultKey = m.WParam
		changeKeyMethod()
		m.Cmd = common.CMD_RESPONSE
		m.WParam = "ok"
	case common.CMD_SET_METHOD:
		config.DefaultMethod = m.WParam
		changeKeyMethod()
		m.Cmd = common.CMD_RESPONSE
		m.WParam = "ok"
	}

	r, _ := json.Marshal(m)
	return r
}

func (c *Connection) readWS() (err error) {
	var p []byte
	for err == nil {
		_, p, err = c.conn.ReadMessage()
		// don't worry, if write goroutine exits abnormally,
		// the connection will be closed,
		// then this read operation will be interrupted abnormally
		if err != nil {
			common.Error("websocket reading message failed", err)
			c.done <- true
			break
		}
		c.msg <- p
	}
	return
}

func (c *Connection) writeWS() (err error) {
	for err == nil {
		select {
		case t := <-c.msg:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			err = c.conn.WriteMessage(websocket.BinaryMessage, c.handleWS(t))
			if err != nil {
				common.Error("websocket writing message failed", err)
				break
			}
			common.Debug("websocket message sent")
		case <-c.done:
			common.Debug("websocket done")
			err = websocket.ErrCloseSent
			break
		}
	}
	return
}

func (c *Connection) SendMsg(cmd int, wParam string, lParam string) {
	msg := &common.WebsocketMessage{
		Cmd:    cmd,
		WParam: wParam,
		LParam: lParam,
	}
	m, err := json.Marshal(msg)
	if err != nil {
		common.Error("marshalling message failed", err)
		return
	}

	c.msg <- m
}

func (c *Connection) connectWS() {
	var err error
	c.conn, _, err = wsDialer.Dial(consoleWSUrl, nil)
	if err != nil {
		common.Error("websocket dialing failed to", consoleWSUrl, err)
		return
	}
	defer c.conn.Close()
	c.conn.SetPingHandler(c.pingHandler)
	common.Debug("websocket connected to", consoleWSUrl)

	c.SendMsg(common.CMD_AUTH, config.Configurations.Generals.Token, "")

	go c.readWS()
	c.writeWS()
}

func consoleWS() {
	c := &Connection{
		msg:  make(chan []byte, 1),
		done: make(chan bool),
	}

	for {
		c.connectWS()
		time.Sleep(10 * time.Second)
	}
}

func changeKeyMethod() {
	_, err := ss.NewStreamCipher(config.DefaultMethod, config.DefaultKey)
	if err != nil {
		common.Error("Failed generating ciphers:", err)
		return
	}

	backends.Lock()
	for _, backendInfo := range backends.BackendsInformation {
		if backendInfo.local == false {
			backendInfo.encryptMethod = config.DefaultMethod
			backendInfo.encryptPassword = config.DefaultKey
		}
	}
	backends.Unlock()
}

func changePort() {
	backends.Lock()
	for _, backendInfo := range backends.BackendsInformation {
		if backendInfo.local == false {
			host, _, _ := net.SplitHostPort(backendInfo.address)
			backendInfo.address = net.JoinHostPort(host, config.DefaultPort)
		}
	}
	backends.Unlock()
}

func removeServer(address string) {
	for i, backendInfo := range backends.BackendsInformation {
		host, _, _ := net.SplitHostPort(backendInfo.address)
		if host == address && backendInfo.local == false {
			// remove this element from backends array
			statistics.Delete(backends.Get(i))
			backends.Remove(i)
			break
		}
	}

	for i, outbound := range config.Configurations.OutboundsConfig {
		host, _, _ := net.SplitHostPort(outbound.Address)
		if host == address && outbound.Local == false {
			config.Configurations.OutboundsConfig = append(config.Configurations.OutboundsConfig[:i], config.Configurations.OutboundsConfig[i+1:]...)
			// save to redis
			break
		}
	}
}

func addServer(address string) {
	_, err := ss.NewStreamCipher(config.DefaultMethod, config.DefaultKey)
	if err != nil {
		common.Error("Failed generating ciphers:", err)
		return
	}

	// don't append directly, scan the existing elements and update them
	find := false
	for _, backendInfo := range backends.BackendsInformation {
		host, _, _ := net.SplitHostPort(backendInfo.address)
		if host == address && backendInfo.local == false {
			backendInfo.protocolType = "shadowsocks"
			//backendInfo.cipher = cipher
			backendInfo.encryptMethod = config.DefaultMethod
			backendInfo.encryptPassword = config.DefaultKey
			backendInfo.timeout = config.Configurations.Generals.Timeout

			find = true
			break
		}
	}

	if !find {
		// append directly
		bi := &BackendInfo{
			id:           common.GenerateRandomString(4),
			address:      net.JoinHostPort(address, config.DefaultPort),
			protocolType: "shadowsocks",
			timeout:      config.Configurations.Generals.Timeout,
			SSRInfo: SSRInfo{
				obfs:     "plain",
				protocol: "origin",
				SSInfo: SSInfo{
					//cipher: cipher,
					encryptMethod:   config.DefaultMethod,
					encryptPassword: config.DefaultKey,
				},
			},
		}
		backends.Append(bi)

		stat := common.NewStatistic()
		statistics.Insert(bi, stat)

		outbound := &outbound.Outbound{
			Address: net.JoinHostPort(address, config.DefaultPort),
			SSRInfo: outbound.SSRInfo{
				SSInfo: outbound.SSInfo{
					Key:    config.DefaultKey,
					Method: config.DefaultMethod,
				},
			},
			Type: "shadowsocks",
		}
		config.Configurations.OutboundsConfig = append(config.Configurations.OutboundsConfig, outbound)
		// save to redis
	}
}
