// +build linux

package rule

import (
	"bytes"
	"html/template"
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
	t, err := template.New("").Parse(rulesTemplate)
	if err != nil {
		common.Error("parse rules template failed", err)
		return
	}

	d := struct {
		Predefined string
		Servers    string
	}{
		Predefined: strings.Join(res, "\n"),
		Servers:    strings.Join(records, "\n"),
	}
	var tpl bytes.Buffer
	err = t.Execute(&tpl, d)
	if err != nil {
		common.Error("executing ss-redir.tmpl failed", err)
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
