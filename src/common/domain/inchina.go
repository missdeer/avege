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

const dnChinaURL = "https://yii.li/dnchina"

var (
	//dnChina    = ds.NewItemTree("china-domain.lst", true)
	dnChina = ds.NewItemMapWithCap("china-domain.lst", true, 30000)
)

func InChina(dn string) bool {
	return dnChina.Hit(dn[:len(dn)-1])
}

func LoadDomainNameInChina() {
	if dnChina.Load() == false {
		dnChina.Clear()
		go UpdateDomainNameInChina()
	}
}

func UpdateDomainNameInChina() {
	var content io.ReadCloser
	for err := os.ErrNotExist; err != nil; time.Sleep(5 * time.Second) {
		common.Warning("try to download content from", dnChinaURL)
		content, err = netutil.DownloadRemoteContent(dnChinaURL)
	}
	defer content.Close()

	regex, _ := regexp.Compile(`server=\/([^\/]+)`)
	scanner := bufio.NewScanner(content)
	scanner.Split(bufio.ScanLines)
	dnChina.Lock()
	for scanner.Scan() {
		ss := regex.FindStringSubmatch(scanner.Text())
		if len(ss) > 1 {
			dnChina.AddItem(ss[1])
		}
	}
	dnChina.Unlock()
	common.Debugf("domain name in China from %s loaded", dnChinaURL)
	dnChina.Save()
}
