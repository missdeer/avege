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
	regLevel12         = regexp.MustCompile(`([a-zA-Z]{2,2})\-[a-z0-9]{1,2}\.[a-z\-]{12,12}\.com`)
	regLevel3          = regexp.MustCompile(`[a-z0-9]{6,6}\.[a-z\-]{12,12}\.com`)
	prefixLocalPortMap = make(PrefixPortMap)
	keepNodeUnresolved = false

	sortedPrefixes = []string{
		"us", "jp", "hk", "sg", "tw", "kr", "eu", "ru",
		"cn",
	}
	level3LocationsPrefixMap = map[string]string{
		`美国`:  `us`,
		`日本`:  `jp`,
		`台湾`:  `tw`,
		`俄罗斯`: `ru`,
		`新加坡`: `sg`,
		`澳门`:  `kr`,
		//`香港`:`hk`,
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
	content, err := base64.RawURLEncoding.DecodeString(sr)
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
		if len(ss) <= 2 {
			common.Error("unexpected output:", output)
			continue
		}
		host := ss[0]
		common.Info("output:", output, host)

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
		if regLevel12.MatchString(host) {
			if _, ok := level12HostRemarksMap[host]; !ok {
				level12HostRemarksMap[host] = remarks
				res = append(res, host)
			}
		}
		if regLevel3.MatchString(host) {
			if _, ok := level3HostRemarksMap[host]; !ok {
				level3HostRemarksMap[host] = remarks
				res = append(res, host)
			}
		}
	}

	common.Info("level12 map:", level12HostRemarksMap)
	common.Info("level3 map:", level3HostRemarksMap)

	level12PrefixesExistMap := make(map[string]placeholder)
	for host := range level12HostRemarksMap {
		ss := regLevel12.FindAllStringSubmatch(host, -1)
		if len(ss) > 0 && len(ss[0]) == 2 {
			level12PrefixesExistMap[ss[0][1]] = placeholder{}
		}
	}
	level3PrefixesExistMap := make(map[string]placeholder)
	for _, remarks := range level3HostRemarksMap {
		matched := false
		for location, prefix := range level3LocationsPrefixMap {
			if strings.Contains(remarks, location) {
				level3PrefixesExistMap[prefix] = placeholder{}
				matched = true
			}
		}
		if !matched {
			level3PrefixesExistMap[`hk`] = placeholder{}
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

func getAllHostsByPrefix(prefix string, hostRemarksMap map[string]string) (hosts []string) {
	for host, remarks := range hostRemarksMap {
		if !strings.HasPrefix(host, prefix) && regLevel12.MatchString(host) {
			continue
		}
		var location string
		notHK := false
		for l, p := range level3LocationsPrefixMap {
			if prefix == `hk` && strings.Contains(remarks, l) {
				notHK = true
			}
			if p == prefix {
				location = l
				break
			}
		}
		if (!strings.Contains(remarks, location) || notHK) && regLevel3.MatchString(host) {
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
	return
}

func getHostsByPrefix(prefix string) (hosts []string) {
	if v, ok := config.Properties[prefix]; ok {
		if host, ok := v.(string); ok {
			// resolve host name to IP
			ips, err := net.LookupIP(host)
			if err != nil {
				common.Error("lookup IP failed", err)
				return []string{host}
			}
			for _, ip := range ips {
				if ip.To4() == nil {
					// invalid IPv4 address
					return []string{host}
				}
				return []string{ip.String()}
			}
		}
	}
	return
}

func generateHAProxyMixedConfiguration(hostRemarksMap map[string]string, prefixes []string, saveFile string) {
	port := `59237`
	if v, ok := config.Properties["port"]; ok {
		if port, ok = v.(string); !ok {
			port = `59237`
		}
	}
	d := struct {
		Port     string
		Prefixes []string
		Hosts    [][]string
	}{
		Port:     port,
		Prefixes: prefixes,
	}
	nodePolicy := `all`
	if v, ok := config.Properties["policy"]; ok {
		if nodePolicy, ok = v.(string); !ok {
			nodePolicy = `all`
		}
	}
	switch strings.ToLower(nodePolicy) {
	case "all":
		for _, prefix := range d.Prefixes {
			hosts := getAllHostsByPrefix(prefix, hostRemarksMap)
			d.Hosts = append(d.Hosts, hosts)
		}
	case "area":
		for _, prefix := range d.Prefixes {
			hosts := getHostsByPrefix(prefix)
			d.Hosts = append(d.Hosts, hosts)
		}
	case "unique":
		hosts := getHostsByPrefix("unique")
		for range d.Prefixes {
			d.Hosts = append(d.Hosts, hosts)
		}
	default:
		log.Fatal("unsupported node policy")
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
