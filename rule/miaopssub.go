package rule

import (
	"bytes"
	"encoding/base64"
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
	"github.com/missdeer/avege/common/fs"
	"github.com/missdeer/avege/config"
)

var (
	prefixLocalPortMap = make(PrefixPortMap)

	sortedPrefixes = []string{
		"us",
		"jp",
		"hk",
		"sg",
		"tw",
		"kr",
		"eu",
		"ru",
		"cn",
	}
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

func getSubscriptionContent(u string) (content []byte) {
	req, err := http.NewRequest("GET", u, nil)
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
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		common.Error("get ssr subscription request not 200")
		return
	}

	rawcontent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("cannot read ssr subscription content:", err)
		return
	}

	return decodeBase64(string(rawcontent))
}

func getSSRSubcription() (res []string) {
	content := getSubscriptionContent(config.Configurations.Generals.SSRSubscription)
	if len(content) == 0 {
		common.Error("cannot parse ssr subscription as base64 content")
		return
	}

	hostsMap := make(map[string]placeholder)
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
			if _, ok := hostsMap[ss[0]]; !ok {
				hostsMap[ss[0]] = placeholder{}
				res = append(res, ss[0])
			}
		}
	}
	regLevel12 := regexp.MustCompile(`([a-zA-Z]{2,2})\-[a-z0-9A-Z]+\.mitsuha\-node\.com`)
	prefixes := make(map[string]placeholder)
	for host := range hostsMap {
		ss := regLevel12.FindAllStringSubmatch(host, -1)
		if len(ss) > 0 && len(ss[0]) == 2 {
			prefixes[ss[0][1]] = placeholder{}
		}
	}
	// sorted
	var ps []string
	for _, prefix := range sortedPrefixes {
		if _, ok := prefixes[prefix]; ok {
			ps = append(ps, prefix)
		}
	}
	generateHAProxyMixedConfiguration(hostsMap, ps)

	prefixRemotePortMap := make(PrefixPortMap)
	for i, prefix := range ps {
		prefixRemotePortMap[prefix] = 50543 + i*1000
	}
	generateSSCommandScript(prefixRemotePortMap)

	return
}

func generateSSCommandScript(prefixRemotePortMap PrefixPortMap) {
	type Item struct {
		Port int
	}
	d := struct {
		Items []Item
	}{
		Items: []Item{},
	}
	localPort := 58090

	// sorted
	for index, prefix := range sortedPrefixes {
		if remotePort, ok := prefixRemotePortMap[prefix]; ok {
			d.Items = append(d.Items, Item{Port: remotePort})
			prefixLocalPortMap[prefix] = localPort + index
		}
	}

	tmpl, err := fs.GetConfigPath(`ss-redir.tmpl`)
	if err != nil {
		common.Error("can't get ss-redir.tmpl", err)
		return
	}
	t, err := template.New("ss-redir.tmpl").ParseFiles(tmpl)
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

func generateHAProxyMixedConfiguration(hostsMap map[string]placeholder, prefixes []string) {
	d := struct {
		Prefixes []string
		Hosts    [][]string
	}{
		Prefixes: prefixes,
	}
	for _, prefix := range d.Prefixes {
		var hosts []string
		for host := range hostsMap {
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

	tmpl, err := fs.GetConfigPath(`haproxy.mixed.cfg.tmpl`)
	if err != nil {
		common.Error("can't get haproxy.mixed.cfg.tmpl", err)
		return
	}
	t, err := template.New("haproxy.mixed.cfg.tmpl").ParseFiles(tmpl)
	if err != nil {
		common.Error("parsing template failed", err)
		return
	}

	var tpl bytes.Buffer
	err = t.Execute(&tpl, d)
	if err != nil {
		common.Error("executing haproxy.mixed.cfg.tmpl failed", err)
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
