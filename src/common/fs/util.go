package fs

import (
	"errors"
	"os"
	"path"

	"common"
	"github.com/go-fsnotify/fsnotify"
	"github.com/kardianos/osext"
)

// IsFileExists check if the file exists
func IsFileExists(path string) (bool, error) {
	stat, err := os.Stat(path)
	if err == nil {
		if stat.Mode()&os.ModeType == 0 {
			return true, nil
		}
		return false, errors.New(path + " exists but is not regular file")
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// GetConfigPath find the configuration file in some paths
func GetConfigPath(fileName string) (filePath string, err error) {
	configFile := fileName
	var binDir string
	if exists, _ := IsFileExists(configFile); !exists {
		if binDir, err = os.Getwd(); err == nil {
			oldConfig := configFile
			configFile = path.Join(binDir, fileName)
			common.Warningf("%s not found, try file %s\n", oldConfig, configFile)
		}
	}

	if exists, _ := IsFileExists(configFile); !exists {
		oldConfig := configFile
		configFile = path.Join(binDir, "conf", fileName)
		common.Warningf("%s not found, try file %s\n", oldConfig, configFile)
	}

	if exists, _ := IsFileExists(configFile); !exists {
		if executable, err := osext.Executable(); err == nil {
			binDir = path.Dir(executable)
			oldConfig := configFile
			configFile = path.Join(binDir, fileName)
			common.Warningf("%s not found, try file %s\n", oldConfig, configFile)
		}
	}

	if exists, _ := IsFileExists(configFile); !exists {
		oldConfig := configFile
		configFile = path.Join(binDir, "conf", fileName)
		common.Warningf("%s not found, try file %s\n", oldConfig, configFile)
	}

	if exists, _ := IsFileExists(configFile); !exists {
		return "", errors.New("File not found")
	}
	return configFile, nil
}

// MonitorFileChanegs notify the channel if the file has been changed
func MonitorFileChanegs(path string, changed chan bool) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		common.Fatal(err)
		return
	}

	err = watcher.Add(path)
	if err != nil {
		common.Fatal(err)
		return
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event := <-watcher.Events:
				if (event.Op&fsnotify.Write == fsnotify.Write) ||
					(event.Op&fsnotify.Rename == fsnotify.Rename) {
					common.Debug("modified file:", event.Name)
					changed <- true
				}
			case err := <-watcher.Errors:
				common.Error("error:", err)
			}
		}
	}()
}
