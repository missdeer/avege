package rule

import (
	"bytes"
	"os"
	"text/template"

	"github.com/missdeer/avege/common"
)

var (
	iptablesRuleTemplateFile = `rules.v4.template`
	ruleFile                 = `rules.v4.latest`
	ipsetTemplateFile        = `ipset.tmpl`
	ipsetFile                = `ipset.txt`
)

func saveToIptablesRuleFile(recordMap map[string][]string) {
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
		if _, ok := recordMap[prefix]; ok && prefix != "cn" {
			d.Prefixes = append(d.Prefixes, Prefix{Prefix: prefix, Port: port})
		}
	}
	for _, prefix := range sortedPrefixes {
		if port, ok := prefixLocalPortMap[prefix]; ok {
			d.DefaultPort = port
			break
		}
	}

	t, err := template.New("rules.v4.template").ParseFiles(iptablesRuleTemplateFile)
	if err != nil {
		common.Error("parse rules template failed", err)
		return
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
	d := struct {
		Prefixes []string
		Nets     [][]string
	}{}

	for _, prefix := range sortedPrefixes {
		_, ok1 := prefixLocalPortMap[prefix]
		records, ok2 := recordMap[prefix]
		if ok1 && ok2 {
			d.Prefixes = append(d.Prefixes, prefix)
			d.Nets = append(d.Nets, records)
		}
	}

	t, err := template.New("ipset.tmpl").ParseFiles(ipsetTemplateFile)
	if err != nil {
		common.Error("parse ipset template failed", err)
		return
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
