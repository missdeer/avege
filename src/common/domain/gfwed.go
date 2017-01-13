package domain

import (
	"bufio"
	"common"
	"encoding/base64"
	"io"
	"os"
	"regexp"
	"time"
)

var (
	//gfwlist    = common.NewItemTree("gfwlist.lst", true)
	gfwlist    = common.NewItemMapWithCap("gfwlist.lst", true, 4000)
	gfwlistUrl = "https://raw.githubusercontent.com/gfwlist/gfwlist/master/gfwlist.txt"
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
		common.Warning("try to download content from", gfwlistUrl)
		content, err = common.DownloadRemoteContent(gfwlistUrl)
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
	common.Debugf("gfwlist from %s loaded", gfwlistUrl)
	gfwlist.Save()
}
