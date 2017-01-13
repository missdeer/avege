package ip

import (
	"common"
)

var (
	ipBlacklist = common.NewItemMapWithCap("ipblacklist.lst", true, 512)
)

// InBlacklist returns true if the given IP is in the black list
func InBlacklist(ip string) bool {
	return ipBlacklist.Hit(ip)
}

// LoadIPBlacklist loads the IP black list from file
func LoadIPBlacklist() {
	ipBlacklist.Load()
}
