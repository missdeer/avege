package domain

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"common"
	"common/ds"
	"common/netutil"
)

var (
	//domainNameToBlock    = ds.NewItemTree("toblock.lst", true)
	domainNameToBlock    = ds.NewItemMapWithCap("toblock.lst", true, 200000)
	domainNameToBlockURL = "https://raw.githubusercontent.com/missdeer/blocklist/master/toblock.lst"
)

func ToBlock(dn string) bool {
	return domainNameToBlock.Hit(dn[:len(dn)-1])
}

func LoadDomainNameToBlock() {
	if domainNameToBlock.Load() == false {
		domainNameToBlock.Clear()
		go UpdateDomainNameToBlock()
	}
}

func UpdateDomainNameToBlock() {
	var content io.ReadCloser
	for err := os.ErrNotExist; err != nil; time.Sleep(5 * time.Second) {
		common.Warning("try to download content from", domainNameToBlockURL)
		content, err = netutil.DownloadRemoteContent(domainNameToBlockURL)
	}
	defer content.Close()

	regex, _ := regexp.Compile(`127\.0\.0\.1\s+([0-9a-zA-Z\-\.]+)[\r\n|$]?`)
	scanner := bufio.NewScanner(content)
	scanner.Split(bufio.ScanLines)
	domainNameToBlock.Lock()
	for scanner.Scan() {
		ss := regex.FindStringSubmatch(scanner.Text())
		if len(ss) > 1 &&
			!strings.Contains(ss[1], "google-analytics.com") &&
			!strings.Contains(ss[1], "iqiyi") &&
			!strings.Contains(ss[1], "youku") {
			domainNameToBlock.AddItem(ss[1])
		}
	}
	domainNameToBlock.Unlock()
	common.Debugf("domain name to be blocked from %s loaded", domainNameToBlockURL)
	domainNameToBlock.Save()
}
