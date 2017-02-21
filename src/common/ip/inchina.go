package ip

import (
	"bufio"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"common"
	"common/fs"
	"common/netutil"
)

const apnic = "https://yii.li/apnic"

type ipField struct {
	mask byte
	next ipFieldList
}

type ipFieldList map[byte]*ipField // value - ip field

var (
	ipInChina      ipFieldList
	mutexIPInChina sync.RWMutex
)

func IPv4InChina(ipv4 net.IP) bool {
	mutexIPInChina.RLock()
	defer mutexIPInChina.RUnlock()
	current := &ipInChina
	finalHit := false
	var i, j byte
	for i = 0; i < 4; i++ {
		var mask byte = 32
		b := ipv4[i]
		hit := false
		for j = 0; j < 7 && hit == false; j++ {
			if field, ok := (*current)[b]; !ok {
				if b == 0 || b == 128 {
					return false
				}
				b &= (0xFF ^ (1<<(j+1) - 1))
			} else {
				mask = field.mask
				current = &(field.next)
				hit = true
			}
		}

		if mask <= (i+1)*8 {
			finalHit = hit
			break
		}
	}

	if finalHit {
		common.Debug(ipv4, "is in China")
	}
	return finalHit
}

// InChina returns true if the given IP is in China
func InChina(ip string) bool {
	ipv4 := net.ParseIP(ip).To4()
	if ipv4 == nil || len(ipv4) < 4 {
		common.Error("wrong input ip", ip)
		return false
	}
	return IPv4InChina(ipv4)
}

// LoadChinaIPList loads china IP list from file
func LoadChinaIPList(forceDownload bool) {
	apnicFile, err := fs.GetConfigPath("apnic.txt")
	if err != nil {
		apnicFile = "apnic.txt"
	}
	if err != nil || forceDownload {
		for err = os.ErrNotExist; err != nil; time.Sleep(5 * time.Second) {
			common.Warning(apnicFile, "not found, try to download from", apnic)
			err = netutil.DownloadRemoteFile(apnic, apnicFile)
		}

		common.Debug(apnic, "is downloaded")
	}

	inFile, _ := os.Open(apnicFile)
	defer inFile.Close()
	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)

	mutexIPInChina.Lock()
	defer mutexIPInChina.Unlock()
	ipInChina = make(ipFieldList)
	for scanner.Scan() {
		rec := scanner.Text()
		s := strings.Split(rec, "|")
		if len(s) != 7 || s[0] != "apnic" || s[1] != "CN" || s[2] != "ipv4" {
			continue
		}
		v, err := strconv.ParseFloat(s[4], 64)
		if err != nil {
			common.Errorf("converting string %s to float64 failed\n", s[4])
			continue
		}
		mask := byte(32 - math.Log2(v))
		ipv4 := net.ParseIP(s[3]).To4()
		next := &ipInChina
		for i := 0; i < 4; i++ {
			node, ok := (*next)[ipv4[i]]
			if !ok {
				node = &ipField{mask: mask}
				if i < 3 {
					node.next = make(ipFieldList)
				}
				(*next)[ipv4[i]] = node
			}
			next = &(node.next)
		}
	}
}
