// +build linux

package rule

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/missdeer/avege/common"
	"github.com/missdeer/avege/common/fs"
	"github.com/missdeer/avege/config"
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

func addAbroadDNSServerIPs(encountered map[string]struct{}) (records []string) {
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
		encountered[val] = struct{}{}
		records = append(records, val)
	}
	return
}

func addProxyServerIPs(encountered map[string]struct{}) (records []string) {
	// ss servers ip
	record := "-A SS -d %s/32 -j RETURN"
	var ss []string
	// SSR subscription
	if config.Configurations.Generals.SSRSubscriptionEnabled {
		res := getSSRSubcription()
		if len(res) == 0 {
			// read from history
			if file, err := os.OpenFile(`ssrsub.history`, os.O_RDONLY, 0644); err != nil {
				common.Error("open ssrsub.history file for read failed", err)
			} else {
				r, err := ioutil.ReadAll(file)
				if err != nil {
					common.Error("reading ssrsub.history failed", err)
				} else {
					res = strings.Split(string(r), "\n")
				}
				file.Close()
				common.Debug("ssrsub.history has been read")
			}
		}
		ss = append(ss, res...)
		// save to history
		if file, err := os.OpenFile(`ssrsub.history`, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644); err != nil {
			common.Error("open ssrsub.history file failed", err)
		} else {
			file.WriteString(strings.Join(res, "\n"))
			file.Close()
			common.Debug("ssrsub.history file has been updated")
		}
	}
	// exception domains
	exception := getExceptionDomainList()
	// resolve
	dl := append(ss, exception...)
	for _, host := range dl {
		ips := resolveIPFromDomainName(host)
		for _, v := range ips {
			val := fmt.Sprintf(record, v)
			if _, ok := encountered[val]; !ok {
				// don't insert duplicated items
				encountered[val] = struct{}{}
				records = append(records, fmt.Sprintf("# skip ip for %s", host), val)
			}
		}
	}
	return
}

func addChinaIPs(encountered map[string]struct{}) (records []string) {
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

func addCurrentRunningServerIPs(encountered map[string]struct{}) (res []string) {
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
			encountered[val] = struct{}{}
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
	encountered := make(map[string]struct{})
	records := addAbroadDNSServerIPs(encountered)
	records = append(records, addProxyServerIPs(encountered)...)
	records = append(records, addChinaIPs(encountered)...)
	res := addCurrentRunningServerIPs(encountered)
	saveToRuleFile(res, records)
	oneUpdateIptablesRules = nil
}
