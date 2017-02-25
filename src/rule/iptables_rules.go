// +build linux

package rule

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"common"
	"common/fs"
	"config"
)

var (
	template               string
	ruleFile               = `rules.v4.latest`
	oneUpdateIptablesRules *sync.Once
)

func init() {
	var err error
	if ruleFile, err = fs.GetConfigPath(ruleFile); err != nil {
		ruleFile = `rules.v4.latest`
	}

	templateFile := `rules.v4.template`
	if templateFile, err = fs.GetConfigPath(templateFile); err != nil {
		templateFile = `rules.v4.template`
	}

	if b, err := ioutil.ReadFile(templateFile); err == nil {
		template = string(b)
		monitorFileChange(templateFile)
	}
}

func monitorFileChange(fileName string) {
	configFileChanged := make(chan bool)
	go func() {
		for {
			select {
			case <-configFileChanged:
				common.Debug(fileName, "changes, reload now...")
				if b, err := ioutil.ReadFile(fileName); err == nil {
					template = string(b)
				}
			}
		}
	}()
	go fs.MonitorFileChanegs(fileName, configFileChanged)
}

func UpdateRedirFirewallRules() {
	if oneUpdateIptablesRules == nil {
		oneUpdateIptablesRules = &sync.Once{}
	}
	oneUpdateIptablesRules.Do(doUpdateRules)
}

func getSSServerDomainList() (res []string) {
	retry := 0
doRequest:
	req, err := http.NewRequest("GET", config.Configurations.Generals.ConsoleHost+"/admin/servers", nil)
	if err != nil {
		common.Error("Could not parse get ss server list request:", err)
		return
	}

	req.Header.Set("Authorization", "PqFVgV6InD")

	client := &http.Client{
		Timeout: 300 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		common.Error("Could not send get ss server list request:", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		common.Error("get ss server list request not 200")
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		common.Error("cannot read get ss server list content", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
	}

	var suffixes []string
	if err = json.Unmarshal(content, &suffixes); err != nil {
		common.Error("unmarshalling ss server list failed", err)
		return
	}

	for _, v := range suffixes {
		res = append(res, fmt.Sprintf("%s.dfordsoft.com", v))
	}
	return res
}

func getExceptionDomainList() (res []string) {
	exceptionFile, err := fs.GetConfigPath(`exception.txt`)
	if err != nil {
		exceptionFile = `exception.txt`
	}
	inFile, err := os.Open(exceptionFile)
	if err != nil {
		common.Error("opening exception.txt failed", err)
	} else {
		defer inFile.Close()
		scanner := bufio.NewScanner(inFile)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			rec := scanner.Text()
			res = append(res, rec)
		}
	}
	return
}

func resolveIPFromDomainName(host string) (res []string) {
	if ips, err := net.LookupIP(host); err == nil {
		for _, ip := range ips {
			if ip.To4() != nil {
				res = append(res, ip.String())
			}
		}
	}
	return
}

func addAbroadDNSServerIPs(encountered map[string]bool) (records []string) {
	// abroad DNS servers IPs
	record := "-A SS -d %s/32 -j RETURN"
	records = append(records, "# skip DNS server out of China")
	for _, v := range config.Configurations.DNSProxy.Abroad {
		if v.Protocol != "tcp" {
			// only TCP is NATed
			continue
		}
		val := fmt.Sprintf(record, v.Address[:strings.Index(v.Address, ":")])
		if _, ok := encountered[val]; ok {
			// don't insert duplicated items
			continue
		}
		encountered[val] = true
		records = append(records, val)
	}
	return
}

func addProxyServerIPs(encountered map[string]bool) (records []string) {
	// ss servers ip
	record := "-A SS -d %s/32 -j RETURN"
	var ss []string
	for len(ss) == 0 {
		ss = getSSServerDomainList()
	}
	// exception domains
	exception := getExceptionDomainList()
	// resolve
	dl := append(ss, exception...)
	for _, host := range dl {
		ips := resolveIPFromDomainName(host)
		if len(ips) > 0 {
			records = append(records, fmt.Sprintf("# skip ip for %s", host))
		}
		for _, v := range ips {
			val := fmt.Sprintf(record, v)
			if _, ok := encountered[val]; !ok {
				// don't insert duplicated items
				encountered[val] = true
				records = append(records, val)
			}
		}
	}
	return
}

func addChinaIPs(encountered map[string]bool) (records []string) {
	// china IPs
	apnicFile, err := fs.GetConfigPath("apnic.txt")
	if err != nil {
		apnicFile = "apnic.txt"
	}
	inFile, err := os.Open(apnicFile)
	if err != nil {
		common.Error("opening apnic.txt failed", err)
	} else {
		defer inFile.Close()
		scanner := bufio.NewScanner(inFile)
		scanner.Split(bufio.ScanLines)

		records = append(records, "# skip ip in China")
		record := "-A SS -d %s/%d -j RETURN"
		for scanner.Scan() {
			rec := scanner.Text()
			s := strings.Split(rec, "|")
			if len(s) == 7 && s[0] == "apnic" && s[1] == "CN" && s[2] == "ipv4" {
				if v, err := strconv.ParseFloat(s[4], 64); err != nil {
					common.Errorf("converting string %s to float64 failed\n", s[4])
					continue
				} else {
					mask := 32 - math.Log2(v)
					records = append(records, fmt.Sprintf(record, s[3], int(mask)))
				}
			}
		}
	}
	return
}

func addCurrentRunningServerIPs(encountered map[string]bool) (res []string) {
	// current running server addresses
	record := "-A SS -d %s/32 -j RETURN"
	for _, outbound := range config.Configurations.OutboundsConfig {
		host, _, _ := net.SplitHostPort(outbound.Address)
		ips, err := net.LookupIP(host)
		if err != nil {
			continue
		}
		for _, ip := range ips {
			if ip.To4() == nil {
				// invalid IPv4 address
				continue
			}
			val := fmt.Sprintf(record, ip.String())
			if _, ok := encountered[val]; ok {
				// don't insert duplicated items
				continue
			}
			encountered[val] = true
			res = append(res, val)
		}
	}
	return
}

func saveToRuleFile(res []string, records []string) {
	if file, err := os.OpenFile(ruleFile, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644); err != nil {
		common.Error("open rule file failed", err)
	} else {
		file.WriteString(fmt.Sprintf(template, strings.Join(res, "\n"), strings.Join(records, "\n")))
		file.Close()
		common.Debug("rule file has been updated")
	}
}

func doUpdateRules() {
	encountered := make(map[string]bool)
	records := addAbroadDNSServerIPs(encountered)
	records = append(records, addProxyServerIPs(encountered)...)
	records = append(records, addChinaIPs(encountered)...)
	res := addCurrentRunningServerIPs(encountered)
	saveToRuleFile(res, records)
	oneUpdateIptablesRules = nil
}
