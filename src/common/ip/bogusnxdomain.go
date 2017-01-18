package ip

import (
	"common/ds"
)

var (
	bogusNXDomain = ds.NewItemMapWithCap("bogus-nxdomain.lst", true, 64)
)

// IsBogusNXDomain returns true if the given IP is bogus nxdomain
func IsBogusNXDomain(ip string) bool {
	return bogusNXDomain.Hit(ip)
}

// LoadBogusNXDomain loads the bogus nxdomain list from file
func LoadBogusNXDomain() {
	bogusNXDomain.Load()
}
