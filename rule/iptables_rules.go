// +build linux

package rule

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/missdeer/avege/common"
	"github.com/missdeer/avege/common/fs"
)

var (
	rulesTemplate          string
	ruleFile               = `rules.v4.latest`
	oneUpdateIptablesRules *sync.Once
)

func monitorFileChange(fileName string) {
	configFileChanged := make(chan bool)
	go func() {
		for {
			select {
			case <-configFileChanged:
				common.Debug(fileName, "changes, reload now...")
				if b, err := ioutil.ReadFile(fileName); err == nil {
					rulesTemplate = string(b)
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

func saveToRuleFile(res []string, records []string) {
	file, err := os.OpenFile(ruleFile, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		common.Error("open rule file failed", err)
		return
	}
	file.WriteString(fmt.Sprintf(rulesTemplate, strings.Join(res, "\n"), strings.Join(records, "\n")))
	file.Close()
	common.Debug("rule file has been updated")
}

func doUpdateRules() {
	var err error
	if ruleFile, err = fs.GetConfigPath(ruleFile); err != nil {
		ruleFile = `rules.v4.latest`
	}

	templateFile := `rules.v4.template`
	if templateFile, err = fs.GetConfigPath(templateFile); err != nil {
		templateFile = `rules.v4.template`
	}

	if b, err := ioutil.ReadFile(templateFile); err == nil {
		rulesTemplate = string(b)
		monitorFileChange(templateFile)
	}
	encountered := make(map[string]placeholder)
	records := addAbroadDNSServerIPs(encountered)
	records = append(records, addProxyServerIPs(encountered)...)
	records = append(records, filterSpecialIPs(encountered, prefixLocalPortMap)...)
	res := addCurrentRunningServerIPs(encountered)
	saveToRuleFile(res, records)
	oneUpdateIptablesRules = nil
}
