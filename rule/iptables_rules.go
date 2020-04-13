// +build linux

package rule

import (
	"sync"

	"github.com/missdeer/avege/common"
	"github.com/missdeer/avege/common/fs"
	"github.com/missdeer/avege/config"
)

var (
	oneUpdateIptablesRules *sync.Once
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
	if v, ok := config.Properties["keep-node-unresolved"]; ok {
		keepNodeUnresolved, ok = v.(bool)
	}
	if oneUpdateIptablesRules == nil {
		oneUpdateIptablesRules = &sync.Once{}
	}
	oneUpdateIptablesRules.Do(doUpdateRules)
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

	saveToIptablesRuleFile(recordMap)
	saveToIPSetFile(recordMap)
	oneUpdateIptablesRules = nil
}
