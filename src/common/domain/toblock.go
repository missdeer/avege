package domain

import (
	"bufio"
	"io"
	"os"
	"time"

	"common"
	"common/ds"
	"common/netutil"
)

const toBlockURL = "https://yii.li/dnblock"

var (
	//toBlock    = ds.NewItemTree("toblock.lst", true)
	toBlock = ds.NewItemMapWithCap("toblock.lst", true, 200000)
)

func ToBlock(dn string) bool {
	return toBlock.Hit(dn[:len(dn)-1])
}

func LoadDomainNameToBlock() {
	if toBlock.Load() == false {
		toBlock.Clear()
		go UpdateDomainNameToBlock()
	}
}

func UpdateDomainNameToBlock() {
	var content io.ReadCloser
	for err := os.ErrNotExist; err != nil; time.Sleep(5 * time.Second) {
		common.Warning("try to download content from", toBlockURL)
		content, err = netutil.DownloadRemoteContent(toBlockURL)
	}
	defer content.Close()

	scanner := bufio.NewScanner(content)
	scanner.Split(bufio.ScanLines)
	toBlock.Lock()
	for scanner.Scan() {
		toBlock.AddItem(scanner.Text())
	}
	toBlock.Unlock()
	common.Debugf("domain name to be blocked from %s loaded", toBlockURL)
	toBlock.Save()
}
