package main

import (
	"flag"

	"github.com/missdeer/avege/common"
	"github.com/missdeer/avege/common/fs"
	"github.com/missdeer/avege/config"
	"github.com/missdeer/avege/rule"
)

func main() {
	var configFile string

	flag.StringVar(&configFile, "c", "config.Configurations.json", "(s)pecify config file")

	flag.Parse()
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
