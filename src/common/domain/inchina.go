package domain

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"time"

	"common"
	"common/ds"
	"common/netutil"
)

var (
	//domainNameInChina    = ds.NewItemTree("china-domain.lst", true)
	domainNameInChina    = ds.NewItemMapWithCap("china-domain.lst", true, 30000)
	domainNameInChinaUrl = "https://raw.githubusercontent.com/felixonmars/dnsmasq-china-list/master/accelerated-domains.china.conf"
)

func InChina(dn string) bool {
	return domainNameInChina.Hit(dn[:len(dn)-1])
}

func LoadDomainNameInChina() {
	if domainNameInChina.Load() == false {
		domainNameInChina.Clear()
		go UpdateDomainNameInChina()
	}
}

func UpdateDomainNameInChina() {
	var content io.ReadCloser
	for err := os.ErrNotExist; err != nil; time.Sleep(5 * time.Second) {
		common.Warning("try to download content from", domainNameInChinaUrl)
		content, err = netutil.DownloadRemoteContent(domainNameInChinaUrl)
	}
	defer content.Close()

	regex, _ := regexp.Compile(`server=\/([^\/]+)`)
	scanner := bufio.NewScanner(content)
	scanner.Split(bufio.ScanLines)
	domainNameInChina.Lock()
	for scanner.Scan() {
		ss := regex.FindStringSubmatch(scanner.Text())
		if len(ss) > 1 {
			domainNameInChina.AddItem(ss[1])
		}
	}
	domainNameInChina.Unlock()
	common.Debugf("domain name in China from %s loaded", domainNameInChinaUrl)
	domainNameInChina.Save()
}
