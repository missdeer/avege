package local

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"

	"common"
	"common/cache"
	"config"
	"github.com/gin-gonic/gin"
	"github.com/kardianos/osext"
	"inbound"
	"rule"
)

var (
	p       *os.Process
	startAt = time.Now()
)

func pong(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}

func clearDNSCache(c *gin.Context) {
	domain := c.PostForm("domain")
	if len(domain) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"Result": "Error",
			"Error":  "Domain name is required.",
		})
		return
	}
	if err := cache.Instance.Delete(fmt.Sprintf("dns:%s.:cachekey", domain)); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"Result": "Error",
			"Error":  err,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"Result": "OK",
		})
	}
}

func updateIptablesRulesHandler(c *gin.Context) {
	if runtime.GOOS == "linux" {
		if inbound.IsModeEnabled("redir") {
			go rule.UpdateRedirFirewallRules()
			c.JSON(http.StatusOK, gin.H{
				"Result": "OK",
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"Result": "not redir mode, can't generate iptables rules",
			})
		}
	} else {
		c.JSON(http.StatusOK, gin.H{
			"Result": "host is not Linux, can't generate iptables rules",
		})
	}
}

func getTokenHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"Result": "OK",
		"Token":  config.Configurations.Generals.Token,
	})
}

func getSelectServerHandler(c *gin.Context) {
	id := c.Param("id")
	next, _ := url.QueryUnescape(c.Query("next"))
	defer c.Redirect(http.StatusFound, "/h5/?"+next)
	if len(id) == 0 {
		return
	}
	if smartLastUsedBackendInfo != nil && smartLastUsedBackendInfo.id == id {
		return
	}
	statistics.RLock()
	defer statistics.RUnlock()
	for server := range statistics.StatisticMap {
		if server.id == id {
			smartLastUsedBackendInfo = server
			return
		}
	}
}

func postSelectServerHandler(c *gin.Context) {
	id := c.PostForm("id")
	if len(id) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"Result": "id is required",
		})
		return
	}
	if smartLastUsedBackendInfo != nil && smartLastUsedBackendInfo.id == id {
		c.JSON(http.StatusOK, gin.H{
			"Result": fmt.Sprintf("server with id %s does be used now, no need to change", id),
		})
		return
	}
	statistics.RLock()
	defer statistics.RUnlock()
	for server := range statistics.StatisticMap {
		if server.id == id {
			smartLastUsedBackendInfo = server
			c.JSON(http.StatusOK, gin.H{
				"Result": "OK",
				"Msg":    fmt.Sprintf("change current using server to %s", server.address),
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"Result": fmt.Sprintf("can't find server info with id %s", id),
	})
	return
}

func forceUpdateSmartUsedServerInfoHandler(c *gin.Context) {
	forceUpdateSmartLastUsedBackendInfo = true
	c.JSON(http.StatusOK, gin.H{
		"Result": "OK",
	})
}

func getMethodHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"Result": "OK",
		"Msg":    config.DefaultMethod,
	})
}

func setMethodHandler(c *gin.Context) {
	methodStr := c.PostForm("method")
	if len(methodStr) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"Result": "missing method field",
		})
		return
	}
	methodMap := map[string]bool{
		"aes-128-cfb":      true,
		"aes-256-cfb":      true,
		"aes-192-cfb":      true,
		"aes-128-ctr":      true,
		"aes-256-ctr":      true,
		"aes-192-ctr":      true,
		"aes-128-ofb":      true,
		"aes-256-ofb":      true,
		"aes-192-ofb":      true,
		"des-cfb":          true,
		"bf-cfb":           true,
		"cast5-cfb":        true,
		"rc4-md5":          true,
		"chacha20":         true,
		"chacha20-ietf":    true,
		"salsa20":          true,
		"camellia-128-cfb": true,
		"camellia-192-cfb": true,
		"camellia-256-cfb": true,
		"idea-cfb":         true,
		"rc2-cfb":          true,
		"seed-cfb":         true,
	}
	_, ok := methodMap[methodStr]
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"Result": fmt.Sprintf("unsupported method %s", methodStr),
		})
		return
	}
	config.DefaultMethod = methodStr
	c.JSON(http.StatusOK, gin.H{
		"Result": "OK",
	})
}

func getKeyHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"Result": "OK",
		"Msg":    config.DefaultKey,
	})
}

func setKeyHandler(c *gin.Context) {
	keyStr := c.PostForm("key")
	if len(keyStr) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"Result": "missing key field",
		})
		return
	}

	config.DefaultKey = keyStr
	c.JSON(http.StatusOK, gin.H{
		"Result": "OK",
	})
}

func getPortHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"Result": "OK",
		"Msg":    config.DefaultPort,
	})
}

func setPortHandler(c *gin.Context) {
	portStr := c.PostForm("port")
	if len(portStr) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"Result": "missing port field",
		})
		return
	}

	p, e := strconv.Atoi(portStr)
	if e != nil {
		c.JSON(http.StatusOK, gin.H{
			"Result": fmt.Sprintln("invalid port value", e),
		})
		return
	}

	if p < 1024 || p > 65535 {
		c.JSON(http.StatusOK, gin.H{
			"Result": fmt.Sprintf("port value %d out of range[1024, 65535]", p),
		})
		return
	}

	config.DefaultPort = portStr
	c.JSON(http.StatusOK, gin.H{
		"Result": "OK",
	})
}

func addServerFullHandler(c *gin.Context) {
	address := c.PostForm("address")
	if len(address) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"Result": "missing server address",
		})
		return
	}
	portStr := c.PostForm("port")
	if len(portStr) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"Result": "missing port field",
		})
		return
	}
	keyStr := c.PostForm("key")
	if len(keyStr) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"Result": "missing key field",
		})
		return
	}
	methodStr := c.PostForm("method")
	if len(methodStr) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"Result": "missing method field",
		})
		return
	}
	methodMap := map[string]bool{
		"aes-128-cfb":      true,
		"aes-256-cfb":      true,
		"aes-192-cfb":      true,
		"aes-128-ctr":      true,
		"aes-256-ctr":      true,
		"aes-192-ctr":      true,
		"aes-128-ofb":      true,
		"aes-256-ofb":      true,
		"aes-192-ofb":      true,
		"des-cfb":          true,
		"bf-cfb":           true,
		"cast5-cfb":        true,
		"rc4-md5":          true,
		"chacha20":         true,
		"chacha20-ietf":    true,
		"salsa20":          true,
		"camellia-128-cfb": true,
		"camellia-192-cfb": true,
		"camellia-256-cfb": true,
		"idea-cfb":         true,
		"rc2-cfb":          true,
		"seed-cfb":         true,
	}
	_, ok := methodMap[methodStr]
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"Result": fmt.Sprintf("unsupported method %s", methodStr),
		})
		return
	}

	config.DefaultPort = portStr
	config.DefaultKey = keyStr
	config.DefaultMethod = methodStr
	addServer(address)
	c.JSON(http.StatusOK, gin.H{
		"Result": "OK",
	})
}

func addServerHandler(c *gin.Context) {
	address := c.PostForm("address")
	if len(address) > 0 {
		addServer(address)
		c.JSON(http.StatusOK, gin.H{
			"Result": "OK",
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"Result": "missing server address",
		})
	}
}

func removeServerHandler(c *gin.Context) {
	address := c.PostForm("address")
	if len(address) > 0 {
		removeServer(address)
		c.JSON(http.StatusOK, gin.H{
			"Result": "OK",
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"Result": "missing server address",
		})
	}
}

func startSSHReverseHandler(c *gin.Context) {
	executable, _ := osext.Executable()
	binDir := path.Dir(executable)
	exe := path.Join(binDir, "ngrok")
	args := []string{
		exe,
		fmt.Sprintf("-config=%s", path.Join(binDir, "ngrok.cfg")),
		"-log=stdout",
		"start",
		"ssh",
	}

	var procAttr os.ProcAttr
	procAttr.Files = []*os.File{nil, os.Stdout, os.Stderr}
	var err error
	p, err = os.StartProcess(exe, args, &procAttr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"Result": fmt.Sprintln("starting ssh reverse connection failed", err),
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"Result": "OK",
		})
		go p.Wait()
	}
}

func stopSSHReverseHandler(c *gin.Context) {
	if p != nil {
		err := p.Kill()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"Result": fmt.Sprintln("stopping ssh reverse connection failed", err),
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"Result": "OK",
			})
		}
		p = nil
	} else {
		c.JSON(http.StatusOK, gin.H{
			"Result": "ssh reverse connection not started yet",
		})
	}
}

// Report type for statistics report
type Report struct {
	Quote         int64
	TotalDownload uint64
	TotalUpload   uint64
	CurrentUsing  string
	StartAt       time.Time
	Uptime        string
	Stats
}

func createStats(order string, orderBy string) Stats {
	stats := make(Stats, 0)
	statistics.RLock()
	for backendInfo, stat := range statistics.StatisticMap {
		s := new(Stat)
		s.Id = backendInfo.id
		s.Address = backendInfo.address
		s.ProtocolType = backendInfo.protocolType
		s.FailedCount = stat.GetFailedCount()
		s.Latency = stat.GetLatency()
		s.TotalDownload = stat.GetTotalDownload()
		s.TotalUpload = stat.GetTotalUploaded()
		s.HighestLastHourBps = stat.GetHighestLastHourBps()
		s.HighestLastTenMinutesBps = stat.GetHighestLastTenMinutesBps()
		s.HighestLastMinuteBps = stat.GetHighestLastMinuteBps()
		s.HighestLastSecondBps = stat.GetHighestLastSecondBps()
		s.LastHourBps = stat.GetLastHourBps()
		s.LastTenMinutesBps = stat.GetLastTenMinutesBps()
		s.LastMinuteBps = stat.GetLastMinuteBps()
		s.LastSecondBps = stat.GetLastSecondBps()
		stats = append(stats, s)
	}
	statistics.RUnlock()
	orderStats(order, orderBy, stats)
	return stats
}

func createReport(order string, orderBy string) *Report {
	stats := createStats(order, orderBy)
	currentUsing := "nil"
	if smartLastUsedBackendInfo != nil {
		currentUsing = smartLastUsedBackendInfo.address
	}
	return &Report{
		Stats:         stats,
		TotalDownload: common.TotalStat.GetDownload(),
		TotalUpload:   common.TotalStat.GetUpload(),
		Quote:         config.LeftQuote,
		CurrentUsing:  currentUsing,
		StartAt:       startAt,
		Uptime:        time.Now().Sub(startAt).String(),
	}
}

func createGinH(order string, orderBy string) *gin.H {
	stats := createStats(order, orderBy)
	currentUsing := "nil"
	if smartLastUsedBackendInfo != nil {
		currentUsing = smartLastUsedBackendInfo.address
	}
	newOrders := map[string]string{
		"asc":  "desc",
		"desc": "asc",
	}
	return &gin.H{
		"title":         "avege, a powerful anti-GFW toolset",
		"stats":         stats,
		"currentUsing":  currentUsing,
		"totalDownload": common.TotalStat.GetDownload(),
		"totalUpload":   common.TotalStat.GetUpload(),
		"quote":         config.LeftQuote,
		"startAt":       startAt,
		"uptime":        time.Now().Sub(startAt).String(),
		"order":         newOrders[order],
		"next":          url.QueryEscape(fmt.Sprintf("orderby=%s&order=%s", orderBy, order)),
	}
}

func statisticsXMLHandler(c *gin.Context) {
	order := c.DefaultQuery("order", "asc")
	orderBy := c.DefaultQuery("orderby", "address")
	r := createReport(order, orderBy)
	c.XML(http.StatusOK, *r)
}

func statisticsJSONHandler(c *gin.Context) {
	order := c.DefaultQuery("order", "asc")
	orderBy := c.DefaultQuery("orderby", "address")
	r := createReport(order, orderBy)
	c.JSON(http.StatusOK, *r)
}

func statisticsHTMLHandler(c *gin.Context) {
	order := c.DefaultQuery("order", "asc")
	orderBy := c.DefaultQuery("orderby", "address")
	h := createGinH(order, orderBy)
	c.HTML(http.StatusOK, "index.tmpl", *h)
}

func orderStats(order string, orderBy string, stats Stats) {
	if sorter, ok := statsSortMap[orderBy]; ok {
		sorter(order, stats)
	} else {
		orderByAddress(order, stats)
	}
}
