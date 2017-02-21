package domain

import (
	"bufio"
	"encoding/base64"
	"io"
	"os"
	"regexp"
	"time"

	"common"
	"common/ds"
	"common/netutil"
)

const gfwlistURL = "https://yii.li/gfwlist"

var (
	//gfwlist    = ds.NewItemTree("gfwlist.lst", true)
	gfwlist = ds.NewItemMapWithCap("gfwlist.lst", true, 4000)
)

func IsGFWed(dn string) bool {
	return gfwlist.Hit(dn[:len(dn)-1])
}

func LoadDomainNameGFWed() {
	if gfwlist.Load() == false {
		gfwlist.Clear()
		if !gfwlist.IsEmpty() {
			common.Fatal("unexpect behavior that item map should be empty")
		}
		go UpdateGFWList()
	}
}

func UpdateGFWList() {
	var content io.ReadCloser
	for err := os.ErrNotExist; err != nil; time.Sleep(5 * time.Second) {
		common.Warning("try to download content from", gfwlistURL)
		content, err = netutil.DownloadRemoteContent(gfwlistURL)
	}
	defer content.Close()

	decoder := base64.NewDecoder(base64.StdEncoding, content)

	commentPattern, _ := regexp.Compile(`^\!|\[|^@@|^\d+\.\d+\.\d+\.\d+`)
	domainPattern, _ := regexp.Compile(`([\w\-\_]+\.[\w\.\-\_]+)[\/\*]*`)
	scanner := bufio.NewScanner(decoder)
	scanner.Split(bufio.ScanLines)
	gfwlist.Lock()
	for scanner.Scan() {
		t := scanner.Text()
		if commentPattern.MatchString(t) {
			continue
		}
		ss := domainPattern.FindStringSubmatch(t)
		if len(ss) > 1 {
			gfwlist.AddItem(ss[1])
		}
	}
	gfwlist.Unlock()
	common.Debugf("gfwlist from %s loaded", gfwlistURL)
	gfwlist.Save()
}
