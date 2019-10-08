// +build linux

package rule

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"sync"

	"github.com/missdeer/avege/common"
	"github.com/missdeer/avege/common/fs"
)

var (
	rulesTemplate            string
	iptablesRuleTemplateFile = `rules.v4.template`
	ruleFile                 = `rules.v4.latest`
	ipsetTemplateFile        = `ipset.tmpl`
	ipsetFile                = `ipset.txt`
	oneUpdateIptablesRules   *sync.Once
)

func monitorFileChange(fileName string) {
	configFileChanged := make(chan bool)
	go func() {
		for {
			select {
			case <-configFileChanged:
				common.Debug(fileName, "changes, reload now...")
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

func saveToRuleFile() {
	t, err := template.New(filepath.Base(iptablesRuleTemplateFile)).Parse(iptablesRuleTemplateFile)
	if err != nil {
		common.Error("parse rules template failed", err)
		return
	}

	type Prefix struct {
		Prefix string
		Port   int
	}
	d := struct {
		Prefixes    []Prefix
		DefaultPort int
	}{
		Prefixes:    []Prefix{},
		DefaultPort: 58090,
	}

	for prefix, port := range prefixLocalPortMap {
		d.Prefixes = append(d.Prefixes, Prefix{Prefix: prefix, Port: port})
	}
	for _, prefix := range sortedPrefixes {
		if port, ok := prefixLocalPortMap[prefix]; ok {
			d.DefaultPort = port
			break
		}
	}

	var tpl bytes.Buffer
	err = t.Execute(&tpl, d)
	if err != nil {
		common.Error("executing rules.v4.template failed", err)
		return
	}

	file, err := os.OpenFile(ruleFile, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		common.Error("open rule file failed", err)
		return
	}
	defer file.Close()
	_, err = file.WriteString(tpl.String())
	if err != nil {
		common.Error("write content to rule file failed", err)
		return
	}

	common.Debug(ruleFile, " has been updated")
}

func saveToIPSetFile(recordMap map[string][]string) {
	t, err := template.New(filepath.Base(ipsetTemplateFile)).Parse(ipsetTemplateFile)
	if err != nil {
		common.Error("parse ipset template failed", err)
		return
	}

	type Net struct {
	}

	d := struct {
		Prefixes []string
		Nets     [][]string
	}{}

	for _, prefix := range sortedPrefixes {
		if _, ok := prefixLocalPortMap[prefix]; ok {
			d.Prefixes = append(d.Prefixes, prefix)
		}
	}

	for _, prefix := range d.Prefixes {
		if records, ok := recordMap[prefix]; ok {
			d.Nets = append(d.Nets, records)
		}
	}

	var tpl bytes.Buffer
	err = t.Execute(&tpl, d)
	if err != nil {
		common.Error("executing ipset.tmpl failed", err)
		return
	}

	file, err := os.OpenFile(ipsetFile, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		common.Error("open ipset file failed", err)
		return
	}
	defer file.Close()
	_, err = file.WriteString(tpl.String())
	if err != nil {
		common.Error("write content to ipset file failed", err)
		return
	}
	common.Debug(ipsetFile, " has been updated")
}

func doUpdateRules() {
	var err error
	if iptablesRuleTemplateFile, err = fs.GetConfigPath(iptablesRuleTemplateFile); err != nil {
		common.Error("can't find iptables rule template file", err)
		return
	}
	if ruleFile, err = fs.GetConfigPath(ruleFile); err != nil {
		ruleFile = `rules.v4.latest`
	}
	if ipsetTemplateFile, err = fs.GetConfigPath(ipsetTemplateFile); err != nil {
		common.Error("can't find ipset template file", err)
		return
	}
	if ipsetFile, err = fs.GetConfigPath(ipsetFile); err != nil {
		ipsetFile = `ipset.txt`
	}
	monitorFileChange(iptablesRuleTemplateFile)

	encountered := make(map[string]placeholder)
	cnroutes := addProxyServerIPs(encountered)
	cn2, recordMap := filterSpecialIPs(encountered)
	cn3 := addCurrentRunningServerIPs(encountered)
	cnroutes = append(cnroutes, cn2...)
	cnroutes = append(cnroutes, cn3...)
	if len(cnroutes) > 0 {
		recordMap["cn"] = cnroutes
	}

	saveToRuleFile()
	saveToIPSetFile(recordMap)
	oneUpdateIptablesRules = nil
}
