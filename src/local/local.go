package local

import (
	"flag"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"reflect"
	"strconv"
	"time"

	"common"
	"common/cache"
	"common/domain"
	"common/fs"
	iputil "common/ip"
	"github.com/DeanThompson/ginpprof"
	"github.com/gin-gonic/gin"
	"inbound"
	"inbound/redir"
	"inbound/socks"
	"inbound/tunnel"
)

const (
	broadcastAddr = "224.225.236.237:51290"
)

var (
	quit             = make(chan bool)
	udpBroadcastConn *net.UDPConn
)

func serveTCPInbound(ib *inbound.Inbound, ibHandler inbound.TCPInboundHandler) {
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP(ib.Address),
		Port: ib.Port,
		Zone: "",
	})
	if err != nil {
		common.Panic("Failed listening on TCP port", ib.Address, err)
		return
	}

	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			common.Error("accept err: ", err)
			continue
		}
		go ibHandler(conn, handleTCPOutbound)
	}
}

func serveUDPInbound(ib *inbound.Inbound, ibHandler inbound.UDPInboundHandler) {
	c, err := net.ListenPacket("udp", net.JoinHostPort(ib.Address, strconv.Itoa(ib.Port)))
	if err != nil {
		common.Panic("Failed listening on UDP port", ib.Address, err)
		return
	}
	defer c.Close()

	for err == nil {
		err = ibHandler(c, handleUDPOutbound)
	}
}

func runInbound(ib *inbound.Inbound) {
	if leftQuote <= 0 {
		common.Fatal("no quote now, please charge in time")
	}
	switch ib.Type {
	case "socks5", "socks":
		go serveTCPInbound(ib, socks.GetTCPInboundHandler(ib))
		//go serveUDPInbound(ib, socks.GetUDPInboundHandler(ib))
	case "redir":
		go serveTCPInbound(ib, redir.GetTCPInboundHandler(ib))
		//go serveUDPInbound(ib, redir.GetUDPInboundHandler(ib))
	case "tunnel":
		go serveTCPInbound(ib, tunnel.GetTCPInboundHandler(ib))
		go serveUDPInbound(ib, tunnel.GetUDPInboundHandler(ib))
	}
}

func run() {
	if config.InboundConfig != nil {
		inbound.ModeEnable(config.InboundConfig.Type)
		go runInbound(config.InboundConfig)
	}
	for _, i := range config.InboundsConfig {
		inbound.ModeEnable(i.Type)
		go runInbound(i)
	}

	if config.DNSProxy.Enabled {
		startDNSProxy()
	}
	statistics.LoadFromCache()
	if config.Generals.ConsoleReportEnabled {
		go consoleWS()
		go getQuote()
	}
	if inbound.IsModeEnabled("redir") {
		go updateRedirFirewallRules()
	}

	if inbound.Has() {
		go statistics.UpdateLatency()
		go statistics.UpdateServerIP()
	}

	timers()
}

func dialUDP() (conn *net.UDPConn, err error) {
	var addr *net.UDPAddr
	addr, err = net.ResolveUDPAddr("udp", broadcastAddr)
	if err != nil {
		common.Error("Can't resolve broadcast address", err)
		return
	}

	conn, err = net.DialUDP("udp", nil, addr)
	if err != nil {
		common.Error("Dialing broadcast failed", err)
	}
	return
}

func onSecondTicker() {
	if inbound.Has() {
		go statistics.UpdateBps()
	}
	if config.Generals.BroadcastEnabled {
		if udpBroadcastConn == nil {
			common.Warning("broadcast UDP conn is nil")
			udpBroadcastConn, _ = dialUDP()
			if udpBroadcastConn == nil {
				common.Warning("recreating UDP conn failed")
			}
		}
		if _, err := udpBroadcastConn.Write([]byte(config.Generals.Token)); err != nil {
			common.Error("failed to broadcast", err)
			udpBroadcastConn.Close()
			udpBroadcastConn, _ = dialUDP()
		}
	}
}

func onMinuteTicker() {
	if inbound.Has() {
		go statistics.UpdateLatency()
	}
	if config.Generals.ConsoleReportEnabled {
		go uploadStatistic()
	}
}

func onHourTicker() {
	if inbound.Has() {
		go statistics.UpdateServerIP()
	}
}

func onDayTicker() {
	if inbound.IsModeEnabled("redir") {
		go updateRedirFirewallRules()
	}
}

func onWeekTicker() {
	go iputil.LoadChinaIPList(true)
	go domain.UpdateDomainNameInChina()
	go domain.UpdateDomainNameToBlock()
	go domain.UpdateGFWList()
}

func timers() {
	type onTicker func()
	onTickers := []struct {
		*time.Ticker
		onTicker
	}{
		{time.NewTicker(1 * time.Second), onSecondTicker},
		{time.NewTicker(1 * time.Minute), onMinuteTicker},
		{time.NewTicker(1 * time.Hour), onHourTicker},
		{time.NewTicker(24 * time.Hour), onDayTicker},
		{time.NewTicker(7 * 24 * time.Hour), onWeekTicker},
	}

	cases := make([]reflect.SelectCase, len(onTickers)+1)
	for i, v := range onTickers {
		cases[i].Dir = reflect.SelectRecv
		cases[i].Chan = reflect.ValueOf(v.Ticker.C)
	}
	cases[len(onTickers)].Dir = reflect.SelectRecv
	cases[len(onTickers)].Chan = reflect.ValueOf(quit)

	for chosen, _, _ := reflect.Select(cases); chosen < len(onTickers); chosen, _, _ = reflect.Select(cases) {
		onTickers[chosen].onTicker()
	}

	for _, v := range onTickers {
		v.Ticker.Stop()
	}
}

// Main the entry of this program
func Main() {
	var configFile string
	var printVer bool

	flag.BoolVar(&printVer, "version", false, "print (v)ersion")
	flag.StringVar(&configFile, "c", "config.json", "(s)pecify config file")

	flag.Parse()

	if printVer {
		common.PrintVersion()
		os.Exit(0)
	}

	// read config file
	var err error
	configFile, err = fs.GetConfigPath(configFile)
	if err != nil {
		common.Panic("config file not found")
	}

	if err = parseMultiServersConfigFile(configFile); err != nil {
		common.Panic("parsing multi servers config file failed: ", err)
	}

	configFileChanged := make(chan bool)
	go func() {
		for {
			select {
			case <-configFileChanged:
				if err = parseMultiServersConfigFile(configFile); err != nil {
					common.Error("reloading multi servers config file failed: ", err)
				} else {
					common.Debug("reload", configFile)
				}
			}
		}
	}()
	go fs.MonitorFileChanegs(configFile, configFileChanged)
	// end reading config file

	common.DebugLevel = common.DebugLog(config.Generals.LogLevel)

	if config.Generals.APIEnabled {
		if config.Generals.GenRelease {
			gin.SetMode(gin.ReleaseMode)
		}
		r := gin.Default()
		r.LoadHTMLGlob("templates/*")
		v1 := r.Group("/v1")
		{
			v1.GET("/ping", pong) // test purpose
			v1.GET("/statistics.xml", statisticsXMLHandler)
			v1.GET("/statistics.json", statisticsJSONHandler)
			v1.GET("/startSSHReverse", startSSHReverseHandler)
			v1.GET("/stopSSHReverse", stopSSHReverseHandler)
			v1.POST("/startSSHReverse", startSSHReverseHandler)
			v1.POST("/stopSSHReverse", stopSSHReverseHandler)
			v1.POST("/server/add", addServerHandler)
			v1.POST("/server/add/full", addServerFullHandler)
			v1.POST("/server/delete", removeServerHandler)
			v1.POST("/server/select", postSelectServerHandler)
			v1.POST("/server/lastused/update", forceUpdateSmartUsedServerInfoHandler)
			v1.POST("/method", setMethodHandler)
			v1.GET("/method", getMethodHandler)
			v1.POST("/port", setPortHandler)
			v1.GET("/port", getPortHandler)
			v1.POST("/key", setKeyHandler)
			v1.GET("/key", getKeyHandler)
			v1.GET("/token", getTokenHandler)
			v1.POST("/rules/iptables/update", updateIptablesRulesHandler)
			v1.POST("/dns/clear", clearDNSCache)
		}
		h5 := r.Group("/h5")
		{
			h5.GET("/", statisticsHTMLHandler)
			h5.GET("/server/select/:id", getSelectServerHandler)
		}

		r.Static("/assets", "./resources/assets")
		r.StaticFS("/static", http.Dir("./resources/static"))
		r.StaticFile("/favicon.ico", "./resources/favicon.ico")

		if config.Generals.PProfEnabled {
			ginpprof.Wrapper(r)
		}
		go r.Run(config.Generals.API) // listen and serve
	}

	ApplyGeneralConfig()

	switch config.Generals.CacheService {
	case "redis":
		cache.DefaultRedisKey = "avegeClient"
	}
	cache.Init(config.Generals.CacheService)

	run()
}
