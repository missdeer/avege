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
		"us", "jp", "hk", "sg", "tw", "kr", "eu", "ru",
		"cn",
	}
	level3Locations = []string{
		`美国`, `日本`, `台湾`, `俄罗斯`, `新加坡`, `澳门`, //`香港`,
	}
	level3Prefixes = []string{
		`us`, `jp`, `tw`, `ru`, `sg`, `kr`, `hk`,
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

	regLevel12 := regexp.MustCompile(`([a-zA-Z]{2,2})\-[a-z0-9]{1,2}\.[a-z\-]{12,12}\.com`)
	regLevel3 := regexp.MustCompile(`[a-z0-9]{6,6}\.[a-z\-]{12,12}\.com`)

	level12HostRemarksMap := make(map[string]string) // host - remarks pair
	level3HostRemarksMap := make(map[string]string)  // host - remarks pair
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		pos := strings.Index(line, "ssr://")
		if pos < 0 {
			common.Error("unexpected ssr subscription line:", line)
			continue
		}
		input := line[pos+6:]
		common.Info("input:", input)
		output := string(decodeBase64(input))
		if len(output) == 0 {
			common.Error("cannot parse ssr subscription line")
			continue
		}

		ss := strings.Split(output, ":")
		common.Info("output:", output, ss[0])

		var remarks string
		idx := strings.Index(output, `&remarks=`)
		if idx <= 0 {
			common.Error("cannot find remarks field")
			continue
		}
		remarks = output[idx+len(`&remarks=`):]
		idx = strings.Index(remarks, `&`)
		if idx <= 0 {
			common.Error("cannot find remarks end")
			continue
		}
		remarks = string(decodeBase64(remarks[:idx]))
		common.Info("remarks:", remarks)
		if regLevel12.MatchString(ss[0]) {
			if _, ok := level12HostRemarksMap[ss[0]]; !ok {
				level12HostRemarksMap[ss[0]] = remarks
				res = append(res, ss[0])
			}
		}
		if regLevel3.MatchString(ss[0]) {
			if _, ok := level3HostRemarksMap[ss[0]]; !ok {
				level3HostRemarksMap[ss[0]] = remarks
				res = append(res, ss[0])
			}
		}
	}

	level12PrefixesExistMap := make(map[string]placeholder)
	level3PrefixesExistMap := make(map[string]placeholder)
	for host := range level12HostRemarksMap {
		ss := regLevel12.FindAllStringSubmatch(host, -1)
		if len(ss) > 0 && len(ss[0]) == 2 {
			level12PrefixesExistMap[ss[0][1]] = placeholder{}
		}
	}
	for host, remarks := range level3HostRemarksMap {
		if regLevel3.MatchString(host) {
			// it's level3
			for index, location := range level3Locations {
				prefix := `hk`
				if strings.Contains(remarks, location) {
					prefix = level3Prefixes[index]
					level3PrefixesExistMap[prefix] = placeholder{}
				}
			}
			continue
		}
	}
	// sorted
	var level12Prefixes []string
	var level3Prefixes []string
	for _, prefix := range sortedPrefixes {
		if _, ok := level12PrefixesExistMap[prefix]; ok {
			level12Prefixes = append(level12Prefixes, prefix)
		}
		if _, ok := level3PrefixesExistMap[prefix]; ok {
			level3Prefixes = append(level3Prefixes, prefix)
		}
	}
	generateHAProxyMixedConfiguration(level12HostRemarksMap, level12Prefixes, `haproxy.level12.mixed.cfg`)
	generateHAProxyMixedConfiguration(level3HostRemarksMap, level3Prefixes, `haproxy.level3.mixed.cfg`)

	prefixRemotePortMap := make(PrefixPortMap)
	for i, prefix := range level12Prefixes {
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

func generateHAProxyMixedConfiguration(hostRemarksMap map[string]string, prefixes []string, saveFile string) {
	d := struct {
		Prefixes []string
		Hosts    [][]string
	}{
		Prefixes: prefixes,
	}
	for _, prefix := range d.Prefixes {
		var hosts []string
		for host := range hostRemarksMap {
			if !strings.HasPrefix(host, prefix) {
				continue
			}
			// resolve host name to IP
			ips, err := net.LookupIP(host)
			if err != nil {
				common.Error("lookup IP failed", err)
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

	outFile, err := os.OpenFile(saveFile, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		common.Error("can't open", saveFile, err)
		return
	}
	defer outFile.Close()
	_, err = outFile.WriteString(tpl.String())
	if err != nil {
		common.Error("failed to write content to", saveFile, err)
	}
}
