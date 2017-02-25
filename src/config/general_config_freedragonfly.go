// +build freebsd dragonfly

package config

import (
	"common"
	"syscall"
)

func ApplyGeneralConfig() {
	if Configurations.Generals.MaxOpenFiles > 1024 {
		var rLimit syscall.Rlimit
		err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			common.Error("getting Rlimit failed", err)
		} else {
			rLimit.Max = int64(Configurations.Generals.MaxOpenFiles)
			rLimit.Cur = int64(Configurations.Generals.MaxOpenFiles)
			if err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
				common.Error("setting Rlimit failed", err)
			}
		}
	}
}
