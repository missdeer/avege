package rule

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/missdeer/avege/common"
	"github.com/missdeer/avege/config"
)

var (
	prefixLocalPortMap = make(PrefixPortMap)
)

type PrefixPortMap map[string]int

func (m PrefixPortMap) Contains(s string) bool {
	_, ok := m[s]
	return ok
}

func (m PrefixPortMap) Value(s string) int {
	return m[s]
}

type placeholder struct{}

func decodeBase64(s string) []byte {
	sr := s
startDecoding:
	content, err := base64.StdEncoding.DecodeString(sr)
	if err != nil && len(sr) > 0 {
		sr = sr[:len(sr)-1]
		goto startDecoding
	}
	return content
}

func getSSRSubcription() (res []string) {
	retry := 0
doRequest:
	req, err := http.NewRequest("GET", config.Configurations.Generals.SSRSubscription, nil)
	if err != nil {
		common.Error("Could not parse get ssr subscription request:", err)
		return
	}

	client := &http.Client{
		Timeout: 300 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		common.Error("Could not send get ssr subscription request:", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		common.Error("get ssr subscription request not 200")
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}

	rawcontent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("cannot read ssr subscription content:", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
	}

	content := decodeBase64(string(rawcontent))
	if len(content) == 0 {
		common.Error("cannot parse ssr subscription as base64 content")
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
	}

	rm := make(map[string]placeholder)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		pos := strings.Index(line, "://")
		if pos > 0 {
			input := line[pos+3:]
			pos = strings.Index(input, "_")
			if pos > 0 {
				input = input[:pos]
			}
			input = input + "/"
			common.Info(input)
			output := decodeBase64(input)
			if len(output) == 0 {
				common.Error("cannot parse ssr subscription line")
				continue
			}
			common.Info(string(output))
			ss := strings.Split(string(output), ":")
			common.Info(ss[0])
			if _, ok := rm[ss[0]]; !ok {
				rm[ss[0]] = placeholder{}
				res = append(res, ss[0])
			}
		}
	}
	reg := regexp.MustCompile(`([a-zA-Z]{2,2})\-[a-z0-9A-Z]+\.mitsuha\-node\.com`)
	prefixes := make(map[string]placeholder)
	for host := range rm {
		ss := reg.FindAllStringSubmatch(host, -1)
		if len(ss) > 0 && len(ss[0]) == 2 {
			prefixes[ss[0][1]] = placeholder{}
		}
	}
	var ps []string
	for prefix := range prefixes {
		ps = append(ps, prefix)
	}
	generateHAProxyMixedConfiguration(rm, ps)

	prefixRemotePortMap := make(PrefixPortMap)
	for i, prefix := range ps {
		prefixRemotePortMap[prefix] = 50543 + i*1000
	}
	generateSSCommandScript(prefixRemotePortMap)

	prefixes["all"] = placeholder{}
	for prefix := range prefixes {
		generateHAProxyConfigurations(rm, prefix)
	}
	return
}

func generateSSCommandScript(prefixRemotePortMap PrefixPortMap) {
	type Item struct {
		Port int
	}
	type S struct {
		Items []Item
	}
	d := S{
		Items: []Item{},
	}
	localPort := 58090
	for prefix, remotePort := range prefixRemotePortMap {
		d.Items = append(d.Items, Item{Port: remotePort})
		prefixLocalPortMap[prefix] = localPort
		localPort++
	}

	t, err := template.New("ss-redir.tmpl").ParseFiles(`ss-redir.tmpl`)
	if err != nil {
		common.Error("parsing ss-redir.tmpl failed", err)
		return
	}

	var tpl bytes.Buffer
	err = t.Execute(&tpl, d)
	if err != nil {
		common.Error("executing ss-redir.tmpl failed", err)
		return
	}

	outFile, err := os.OpenFile(`ss-redir.sh`, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		common.Error("can't open ss-redir.sh", err)
		return
	}
	defer outFile.Close()
	_, err = outFile.WriteString(tpl.String())
	if err != nil {
		common.Error("write content to ss-redir.sh failed", err)
	}
}

func generateHAProxyMixedConfiguration(rm map[string]placeholder, prefixes []string) {
	type S struct {
		Prefixes []string
		Hosts    [][]string
	}
	d := S{
		Prefixes: prefixes,
	}
	for _, prefix := range d.Prefixes {
		var hosts []string
		for host := range rm {
			if strings.HasPrefix(host, prefix) {
				// resolve host name to IP
				ips, err := net.LookupIP(host)
				if err != nil {
					continue
				}
				for _, ip := range ips {
					if ip.To4() == nil {
						// invalid IPv4 address
						continue
					}
					hosts = append(hosts, ip.String())
				}
			}
		}
		d.Hosts = append(d.Hosts, hosts)
	}

	t, err := template.New("haproxy.mixed.cfg.tmpl").ParseFiles(`haproxy.mixed.cfg.tmpl`)

	var tpl bytes.Buffer
	err = t.Execute(&tpl, d)
	if err != nil {
		common.Error("executing ss-redir.tmpl failed", err)
		return
	}

	outFile, err := os.OpenFile(`haproxy.mixed.cfg`, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		common.Error("can't open haproxy.mixed.cfg", err)
		return
	}
	defer outFile.Close()
	_, err = outFile.WriteString(tpl.String())
	if err != nil {
		common.Error("write content to haproxy.cfg failed", err)
	}
}

func generateHAProxyConfigurations(rm map[string]placeholder, prefix string) {
	var be543, be443, be80 string
	count := 0
	for host := range rm {
		if strings.HasPrefix(host, prefix) || prefix == "all" {
			count++
			be543 += fmt.Sprintf("    server %s %s:543 check\n", strings.Split(host, ".")[0], host)
			be443 += fmt.Sprintf("    server %s %s:443 check\n", strings.Split(host, ".")[0], host)
			be80 += fmt.Sprintf("    server %s %s:80 check\n", strings.Split(host, ".")[0], host)
		}
	}
	if count < 5 {
		count = 5
	}
	haproxyCfgTemplate := `
global 
    daemon  
    maxconn 10240 
    pidfile /home/pi/avege/haproxy.pid 

defaults 
    mode tcp
    balance roundrobin
    timeout connect 10000ms  
    timeout client 50000ms  
    timeout server 50000ms  
    log 127.0.0.1 local0 err

listen admin_stats  
    bind 0.0.0.0:8099
    mode http
    option httplog
    maxconn 10  
    stats refresh 30s
    stats uri /stats  

frontend ssr543 
    bind *:58543  
    default_backend miaops543
	
frontend ssr443
    bind *:58443  
    default_backend miaops443  

frontend ssr80 
    bind *:58080  
    default_backend miaops80
	
backend miaops543
    option log-health-checks
    default-server inter %ds fall 3 rise 2
%s 	

backend miaops443
    option log-health-checks
    default-server inter %ds fall 3 rise 2
%s 	

backend miaops80
    option log-health-checks
    default-server inter %ds fall 3 rise 2
%s
`
	outFile, err := os.OpenFile(`haproxy.cfg.`+prefix, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		common.Error("can't open haproxy.cfg", err)
		return
	}
	defer outFile.Close()
	_, err = outFile.WriteString(fmt.Sprintf(haproxyCfgTemplate, count, be543, count, be443, count, be80))
	if err != nil {
		common.Error("write content to haproxy.cfg failed", err)
	}
}
