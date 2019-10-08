package rule

import (
	"testing"

	"github.com/missdeer/avege/common"
	"github.com/missdeer/avege/common/fs"
	"github.com/missdeer/avege/config"
)

func TestSaveToIptablesRuleFile(t *testing.T) {
	configFile, err := fs.GetConfigPath("config.json")
	if err != nil {
		common.Panic("config file not found")
	}

	if err = config.ParseMultiServersConfigFile(configFile); err != nil {
		common.Panic("parsing multi servers config file failed: ", err)
	}

	if iptablesRuleTemplateFile, err = fs.GetConfigPath(iptablesRuleTemplateFile); err != nil {
		common.Panic("can't find iptables rule template file", err)
	}
	if ruleFile, err = fs.GetConfigPath(ruleFile); err != nil {
		ruleFile = `rules.v4.latest`
	}

	encountered := make(map[string]placeholder)
	addProxyServerIPs(encountered)
	filterSpecialIPs(encountered)
	addCurrentRunningServerIPs(encountered)

	saveToIptablesRuleFile()
}

func TestSaveToIPSetFile(t *testing.T) {
	configFile, err := fs.GetConfigPath("config.json")
	if err != nil {
		common.Panic("config file not found")
	}

	if err = config.ParseMultiServersConfigFile(configFile); err != nil {
		common.Panic("parsing multi servers config file failed: ", err)
	}
	if ipsetTemplateFile, err = fs.GetConfigPath(ipsetTemplateFile); err != nil {
		common.Panic("can't find ipset template file", err)
	}
	if ipsetFile, err = fs.GetConfigPath(ipsetFile); err != nil {
		ipsetFile = `ipset.txt`
	}

	encountered := make(map[string]placeholder)
	cnroutes := addProxyServerIPs(encountered)
	cn2, recordMap := filterSpecialIPs(encountered)
	cn3 := addCurrentRunningServerIPs(encountered)
	cnroutes = append(cnroutes, cn2...)
	cnroutes = append(cnroutes, cn3...)
	if len(cnroutes) > 0 {
		recordMap["cn"] = cnroutes
	}

	saveToIPSetFile(recordMap)
}
