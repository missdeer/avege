package local

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"time"

	"common"
	"common/cache"
	"github.com/gin-gonic/gin"
	"github.com/kardianos/osext"
	"inbound"
	"runtime"
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
		if inbound.IsInBoundModeEnabled("redir") {
			go updateRedirFirewallRules()
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
		"Token":  config.Generals.Token,
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
	Statistics.RLock()
	defer Statistics.RUnlock()
	for server := range Statistics.StatisticMap {
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
	Statistics.RLock()
	defer Statistics.RUnlock()
	for server := range Statistics.StatisticMap {
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
		"Msg":    defaultMethod,
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
	switch methodStr {
	case "aes-128-cfb":
	case "aes-192-cfb":
	case "aes-256-cfb":
	case "des-cfb":
	case "bf-cfb":
	case "cast5-cfb":
	case "rc4-md5":
	case "chacha20":
	case "salsa20":
	case "camellia-128-cfb":
	case "camellia-192-cfb":
	case "camellia-256-cfb":
	case "idea-cfb":
	case "rc2-cfb":
	case "seed-cfb":
	default:
		c.JSON(http.StatusOK, gin.H{
			"Result": fmt.Sprintf("unsupported method %s", methodStr),
		})
		return
	}
	defaultMethod = methodStr
	c.JSON(http.StatusOK, gin.H{
		"Result": "OK",
	})
}

func getKeyHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"Result": "OK",
		"Msg":    defaultKey,
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

	defaultKey = keyStr
	c.JSON(http.StatusOK, gin.H{
		"Result": "OK",
	})
}

func getPortHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"Result": "OK",
		"Msg":    defaultPort,
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

	defaultPort = portStr
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
	switch methodStr {
	case "aes-128-cfb":
	case "aes-192-cfb":
	case "aes-256-cfb":
	case "des-cfb":
	case "bf-cfb":
	case "cast5-cfb":
	case "rc4-md5":
	case "chacha20":
	case "salsa20":
	case "camellia-128-cfb":
	case "camellia-192-cfb":
	case "camellia-256-cfb":
	case "idea-cfb":
	case "rc2-cfb":
	case "seed-cfb":
	default:
		c.JSON(http.StatusOK, gin.H{
			"Result": fmt.Sprintf("unsupported method %s", methodStr),
		})
		return
	}

	defaultPort = portStr
	defaultKey = keyStr
	defaultMethod = methodStr
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

func statisticsXMLHandler(c *gin.Context) {
	stats := make(Stats, 0)
	Statistics.RLock()
	for backendInfo, stat := range Statistics.StatisticMap {
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
	Statistics.RUnlock()
	order := c.DefaultQuery("order", "asc")
	orderBy := c.DefaultQuery("orderby", "address")
	switch orderBy {
	case "failedcount":
		if order == "desc" {
			sort.Sort(sort.Reverse(byFailedCount{stats}))
		} else {
			sort.Sort(byFailedCount{stats})
		}
	case "latency":
		if order == "desc" {
			sort.Sort(sort.Reverse(byLatency{stats}))
		} else {
			sort.Sort(byLatency{stats})
		}
	case "download":
		if order == "desc" {
			sort.Sort(sort.Reverse(byTotalDownload{stats}))
		} else {
			sort.Sort(byTotalDownload{stats})
		}
	case "upload":
		if order == "desc" {
			sort.Sort(sort.Reverse(byTotalUpload{stats}))
		} else {
			sort.Sort(byTotalUpload{stats})
		}
	case "highestlasthourbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byHighestLastHourBps{stats}))
		} else {
			sort.Sort(byHighestLastHourBps{stats})
		}
	case "highestlasttenminutesbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byHighestLastTenMinutesBps{stats}))
		} else {
			sort.Sort(byHighestLastTenMinutesBps{stats})
		}
	case "highestlastminutebps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byHighestLastMinuteBps{stats}))
		} else {
			sort.Sort(byHighestLastMinuteBps{stats})
		}
	case "highestlastsecondbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byHighestLastSecondBps{stats}))
		} else {
			sort.Sort(byHighestLastSecondBps{stats})
		}
	case "lasthourbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byLastHourBps{stats}))
		} else {
			sort.Sort(byLastHourBps{stats})
		}
	case "lasttenminutesbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byLastTenMinutesBps{stats}))
		} else {
			sort.Sort(byLastTenMinutesBps{stats})
		}
	case "lastminutebps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byLastMinuteBps{stats}))
		} else {
			sort.Sort(byLastMinuteBps{stats})
		}
	case "lastsecondbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byLastSecondBps{stats}))
		} else {
			sort.Sort(byLastSecondBps{stats})
		}
	case "protocol":
		if order == "desc" {
			sort.Sort(sort.Reverse(byProtocolType{stats}))
		} else {
			sort.Sort(byProtocolType{stats})
		}
	case "address":
		fallthrough
	default:
		if order == "desc" {
			sort.Sort(sort.Reverse(stats))
		} else {
			sort.Sort(stats)
		}
	}
	currentUsing := "nil"
	if smartLastUsedBackendInfo != nil {
		currentUsing = smartLastUsedBackendInfo.address
	}
	c.XML(http.StatusOK, Report{
		Stats:         stats,
		TotalDownload: common.TotalStat.GetDownload(),
		TotalUpload:   common.TotalStat.GetUpload(),
		Quote:         leftQuote,
		CurrentUsing:  currentUsing,
		StartAt:       startAt,
		Uptime:        time.Now().Sub(startAt).String(),
	})
}

func statisticsJSONHandler(c *gin.Context) {
	stats := make(Stats, 0)
	Statistics.RLock()
	for backendInfo, stat := range Statistics.StatisticMap {
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
	Statistics.RUnlock()
	order := c.DefaultQuery("order", "asc")
	orderBy := c.DefaultQuery("orderby", "address")
	switch orderBy {
	case "failedcount":
		if order == "desc" {
			sort.Sort(sort.Reverse(byFailedCount{stats}))
		} else {
			sort.Sort(byFailedCount{stats})
		}
	case "latency":
		if order == "desc" {
			sort.Sort(sort.Reverse(byLatency{stats}))
		} else {
			sort.Sort(byLatency{stats})
		}
	case "download":
		if order == "desc" {
			sort.Sort(sort.Reverse(byTotalDownload{stats}))
		} else {
			sort.Sort(byTotalDownload{stats})
		}
	case "upload":
		if order == "desc" {
			sort.Sort(sort.Reverse(byTotalUpload{stats}))
		} else {
			sort.Sort(byTotalUpload{stats})
		}
	case "highestlasthourbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byHighestLastHourBps{stats}))
		} else {
			sort.Sort(byHighestLastHourBps{stats})
		}
	case "highestlasttenminutesbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byHighestLastTenMinutesBps{stats}))
		} else {
			sort.Sort(byHighestLastTenMinutesBps{stats})
		}
	case "highestlastminutebps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byHighestLastMinuteBps{stats}))
		} else {
			sort.Sort(byHighestLastMinuteBps{stats})
		}
	case "highestlastsecondbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byHighestLastSecondBps{stats}))
		} else {
			sort.Sort(byHighestLastSecondBps{stats})
		}
	case "lasthourbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byLastHourBps{stats}))
		} else {
			sort.Sort(byLastHourBps{stats})
		}
	case "lasttenminutesbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byLastTenMinutesBps{stats}))
		} else {
			sort.Sort(byLastTenMinutesBps{stats})
		}
	case "lastminutebps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byLastMinuteBps{stats}))
		} else {
			sort.Sort(byLastMinuteBps{stats})
		}
	case "lastsecondbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byLastSecondBps{stats}))
		} else {
			sort.Sort(byLastSecondBps{stats})
		}
	case "protocol":
		if order == "desc" {
			sort.Sort(sort.Reverse(byProtocolType{stats}))
		} else {
			sort.Sort(byProtocolType{stats})
		}
	case "address":
		fallthrough
	default:
		if order == "desc" {
			sort.Sort(sort.Reverse(stats))
		} else {
			sort.Sort(stats)
		}
	}
	currentUsing := "nil"
	if smartLastUsedBackendInfo != nil {
		currentUsing = smartLastUsedBackendInfo.address
	}
	c.JSON(http.StatusOK, Report{
		Stats:         stats,
		TotalDownload: common.TotalStat.GetDownload(),
		TotalUpload:   common.TotalStat.GetUpload(),
		Quote:         leftQuote,
		CurrentUsing:  currentUsing,
		StartAt:       startAt,
		Uptime:        time.Now().Sub(startAt).String(),
	})
}

func statisticsHTMLHandler(c *gin.Context) {
	stats := make(Stats, 0)
	Statistics.RLock()
	for backendInfo, stat := range Statistics.StatisticMap {
		s := new(Stat)
		s.Id = backendInfo.id
		s.Address = backendInfo.address
		s.ProtocolType = backendInfo.protocolType
		s.FailedCount = stat.GetFailedCount()
		s.Latency = stat.GetLatency() / 1000000
		s.TotalDownload = stat.GetTotalDownload() / 1000000
		s.TotalUpload = stat.GetTotalUploaded() / 1000000
		s.HighestLastHourBps = stat.GetHighestLastHourBps() / 1000
		s.HighestLastTenMinutesBps = stat.GetHighestLastTenMinutesBps() / 1000
		s.HighestLastMinuteBps = stat.GetHighestLastMinuteBps() / 1000
		s.HighestLastSecondBps = stat.GetHighestLastSecondBps() / 1000
		s.LastHourBps = stat.GetLastHourBps()
		s.LastTenMinutesBps = stat.GetLastTenMinutesBps()
		s.LastMinuteBps = stat.GetLastMinuteBps()
		s.LastSecondBps = stat.GetLastSecondBps()
		stats = append(stats, s)
	}
	Statistics.RUnlock()
	order := c.DefaultQuery("order", "asc")
	orderBy := c.DefaultQuery("orderby", "address")
	switch orderBy {
	case "failedcount":
		if order == "desc" {
			sort.Sort(sort.Reverse(byFailedCount{stats}))
		} else {
			sort.Sort(byFailedCount{stats})
		}
	case "latency":
		if order == "desc" {
			sort.Sort(sort.Reverse(byLatency{stats}))
		} else {
			sort.Sort(byLatency{stats})
		}
	case "download":
		if order == "desc" {
			sort.Sort(sort.Reverse(byTotalDownload{stats}))
		} else {
			sort.Sort(byTotalDownload{stats})
		}
	case "upload":
		if order == "desc" {
			sort.Sort(sort.Reverse(byTotalUpload{stats}))
		} else {
			sort.Sort(byTotalUpload{stats})
		}
	case "highestlasthourbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byHighestLastHourBps{stats}))
		} else {
			sort.Sort(byHighestLastHourBps{stats})
		}
	case "highestlasttenminutesbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byHighestLastTenMinutesBps{stats}))
		} else {
			sort.Sort(byHighestLastTenMinutesBps{stats})
		}
	case "highestlastminutebps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byHighestLastMinuteBps{stats}))
		} else {
			sort.Sort(byHighestLastMinuteBps{stats})
		}
	case "highestlastsecondbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byHighestLastSecondBps{stats}))
		} else {
			sort.Sort(byHighestLastSecondBps{stats})
		}
	case "lasthourbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byLastHourBps{stats}))
		} else {
			sort.Sort(byLastHourBps{stats})
		}
	case "lasttenminutesbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byLastTenMinutesBps{stats}))
		} else {
			sort.Sort(byLastTenMinutesBps{stats})
		}
	case "lastminutebps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byLastMinuteBps{stats}))
		} else {
			sort.Sort(byLastMinuteBps{stats})
		}
	case "lastsecondbps":
		if order == "desc" {
			sort.Sort(sort.Reverse(byLastSecondBps{stats}))
		} else {
			sort.Sort(byLastSecondBps{stats})
		}
	case "protocol":
		if order == "desc" {
			sort.Sort(sort.Reverse(byProtocolType{stats}))
		} else {
			sort.Sort(byProtocolType{stats})
		}
	case "address":
		fallthrough
	default:
		if order == "desc" {
			sort.Sort(sort.Reverse(stats))
		} else {
			sort.Sort(stats)
		}
	}
	currentUsing := "nil"
	if smartLastUsedBackendInfo != nil {
		currentUsing = smartLastUsedBackendInfo.address
	}
	var newOrder string
	if order == "asc" {
		newOrder = "desc"
	} else {
		newOrder = "asc"
	}

	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title":         "avege, a powerful anti-GFW toolset",
		"stats":         stats,
		"currentUsing":  currentUsing,
		"totalDownload": common.TotalStat.GetDownload(),
		"totalUpload":   common.TotalStat.GetUpload(),
		"quote":         leftQuote,
		"startAt":       startAt,
		"uptime":        time.Now().Sub(startAt).String(),
		"order":         newOrder,
		"next":          url.QueryEscape(fmt.Sprintf("orderby=%s&order=%s", orderBy, order)),
	})
}
