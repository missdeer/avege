package common

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path"

	"github.com/go-fsnotify/fsnotify"
	"github.com/kardianos/osext"
)

func PrintVersion() {
	const version = "1.0.0"
	fmt.Println("avege version", version)
}

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

func GetConfigPath(fileName string) (filePath string, err error) {
	configFile := fileName
	var binDir string
	if exists, _ := IsFileExists(configFile); !exists {
		if binDir, err = os.Getwd(); err == nil {
			oldConfig := configFile
			configFile = path.Join(binDir, fileName)
			Warningf("%s not found, try file %s\n", oldConfig, configFile)
		}
	}

	if exists, _ := IsFileExists(configFile); !exists {
		oldConfig := configFile
		configFile = path.Join(binDir, "conf", fileName)
		Warningf("%s not found, try file %s\n", oldConfig, configFile)
	}

	if exists, _ := IsFileExists(configFile); !exists {
		if executable, err := osext.Executable(); err == nil {
			binDir = path.Dir(executable)
			oldConfig := configFile
			configFile = path.Join(binDir, fileName)
			Warningf("%s not found, try file %s\n", oldConfig, configFile)
		}
	}

	if exists, _ := IsFileExists(configFile); !exists {
		oldConfig := configFile
		configFile = path.Join(binDir, "conf", fileName)
		Warningf("%s not found, try file %s\n", oldConfig, configFile)
	}

	if exists, _ := IsFileExists(configFile); !exists {
		return "", errors.New("File not found")
	}
	return configFile, nil
}

func MonitorFileChanegs(path string, changed chan bool) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		Fatal(err)
		return
	}

	err = watcher.Add(path)
	if err != nil {
		Fatal(err)
		return
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event := <-watcher.Events:
				if (event.Op&fsnotify.Write == fsnotify.Write) ||
					(event.Op&fsnotify.Rename == fsnotify.Rename) {
					Debug("modified file:", event.Name)
					changed <- true
				}
			case err := <-watcher.Errors:
				Error("error:", err)
			}
		}
	}()
}

func DownloadRemoteContent(remoteLink string) (io.ReadCloser, error) {
	response, err := http.Get(remoteLink)
	if err != nil {
		Error("Error while downloading", remoteLink, err)
		return nil, err
	}

	return response.Body, nil
}

func DownloadRemoteFile(remoteLink string, saveToFile string) error {
	response, err := http.Get(remoteLink)
	if err != nil {
		Error("Error while downloading", remoteLink, err)
		return err
	}
	defer response.Body.Close()

	output, err := os.Create(saveToFile)
	if err != nil {
		Error("Error while creating", saveToFile, err)
		return err
	}
	defer output.Close()

	if _, err := io.Copy(output, response.Body); err != nil {
		Error("Error while reading response", remoteLink, err)
		return err
	}

	return nil
}

func GenerateRandomString(length int) (res string) {
	baseStr := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	for i := 0; i < length; i++ {
		index := rand.Intn(len(baseStr))
		res = res + string(baseStr[index])
	}
	return
}
