// +build linux darwin netbsd openbsd solaris

package local

import (
	"common"
	"syscall"
)

func ApplyGeneralConfig() {
	if config.Generals.MaxOpenFiles > 1024 {
		var rLimit syscall.Rlimit
		err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			common.Error("getting Rlimit failed", err)
		} else {
			rLimit.Max = config.Generals.MaxOpenFiles
			rLimit.Cur = config.Generals.MaxOpenFiles
			if err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
				common.Error("setting Rlimit failed", err)
			}
		}
	}
}
