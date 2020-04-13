package main

import (
	"log"

	"github.com/missdeer/avege/common/netutil"
	flag "github.com/spf13/pflag"

	"github.com/missdeer/avege/common"
	"github.com/missdeer/avege/common/fs"
	"github.com/missdeer/avege/config"
	"github.com/missdeer/avege/rule"
)

const (
	apnicURL       = `http://ftp.apnic.net/apnic/stats/apnic/delegated-apnic-latest`
	chinaIPListURL = `https://cdn.jsdelivr.net/gh/17mon/china_ip_list@master/china_ip_list.txt`
)

func main() {
	configFile := "config.json"
	updateAPNIC := false
	updateChinaIPList := false
	keepNodeUnresolved := false
	nodePolicy := "all"
	var uniqueNode, hkNode, sgNode, twNode, usNode, ruNode, euNode, cnNode, jpNode, krNode string

	flag.StringVarP(&configFile, "config", "c", configFile, "Specify config file")
	flag.BoolVarP(&updateAPNIC, "apnic", "n", updateAPNIC, "update APNIC list")
	flag.BoolVarP(&updateChinaIPList, "china", "i", updateChinaIPList, "update China IP List")
	flag.BoolVarP(&keepNodeUnresolved, "unresolved", "u", keepNodeUnresolved, "keep node domain name unresolved")
	flag.StringVarP(&nodePolicy, "policy", "p", nodePolicy, "node policy: all, area, unique")
	flag.StringVarP(&uniqueNode, "unique", "", uniqueNode, "unique node address")
	flag.StringVarP(&hkNode, "hk", "", hkNode, "hk node address")
	flag.StringVarP(&sgNode, "sg", "", sgNode, "sg node address")
	flag.StringVarP(&twNode, "tw", "", twNode, "tw node address")
	flag.StringVarP(&usNode, "us", "", usNode, "us node address")
	flag.StringVarP(&ruNode, "ru", "", ruNode, "ru node address")
	flag.StringVarP(&euNode, "eu", "", euNode, "eu node address")
	flag.StringVarP(&jpNode, "jp", "", jpNode, "jp node address")
	flag.StringVarP(&krNode, "kr", "", krNode, "kr node address")
	flag.StringVarP(&cnNode, "cn", "", cnNode, "cn node address")

	flag.Parse()

	if updateAPNIC {
		apnicFile, err := fs.GetConfigPath("apnic.txt")
		if err != nil {
			apnicFile = "apnic.txt"
		}
		err = netutil.DownloadRemoteFile(apnicURL, apnicFile)
		if err != nil {
			log.Fatal(err)
		}
	}

	if updateChinaIPList {
		chinaIPListFile, err := fs.GetConfigPath("china_ip_list.txt")
		if err != nil {
			chinaIPListFile = "china_ip_list.txt"
		}
		err = netutil.DownloadRemoteFile(chinaIPListURL, chinaIPListFile)
		if err != nil {
			log.Fatal(err)
		}
	}

	config.Properties["keep-node-unresolved"] = keepNodeUnresolved
	config.Properties["policy"] = nodePolicy
	config.Properties["unique"] = uniqueNode
	config.Properties["hk"] = hkNode
	config.Properties["sg"] = sgNode
	config.Properties["tw"] = twNode
	config.Properties["us"] = usNode
	config.Properties["ru"] = ruNode
	config.Properties["eu"] = euNode
	config.Properties["jp"] = jpNode
	config.Properties["kr"] = krNode
	config.Properties["cn"] = cnNode

	// read config file
	var err error
	configFile, err = fs.GetConfigPath(configFile)
	if err != nil {
		common.Panic("config file not found")
	}

	if err = config.ParseMultiServersConfigFile(configFile); err != nil {
		common.Panic("parsing multi servers config file failed: ", err)
	}

	rule.UpdateRedirFirewallRules()
}
