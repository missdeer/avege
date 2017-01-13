package common

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
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

func HmacMD5(key []byte, data []byte) []byte {
	hmacMD5 := hmac.New(md5.New, key)
	hmacMD5.Write(data)
	return hmacMD5.Sum(nil)[:10]
}

func HmacSHA1(key []byte, data []byte) []byte {
	hmacSHA1 := hmac.New(sha1.New, key)
	hmacSHA1.Write(data)
	return hmacSHA1.Sum(nil)[:10]
}

func GenerateRandomString(length int) (res string) {
	baseStr := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	for i := 0; i < length; i++ {
		index := rand.Intn(len(baseStr))
		res = res + string(baseStr[index])
	}
	return
}

func Md5Sum(d []byte) []byte {
	h := md5.New()
	h.Write(d)
	return h.Sum(nil)
}

func EVPBytesToKey(password string, keyLen int) (key []byte) {
	const md5Len = 16

	cnt := (keyLen-1)/md5Len + 1
	m := make([]byte, cnt * md5Len)
	copy(m, Md5Sum([]byte(password)))

	// Repeatedly call md5 until bytes generated is enough.
	// Each call to md5 uses data: prev md5 sum + password.
	d := make([]byte, md5Len + len(password))
	start := 0
	for i := 1; i < cnt; i++ {
		start += md5Len
		copy(d, m[start - md5Len:start])
		copy(d[md5Len:], password)
		copy(m[start:], Md5Sum(d))
	}
	return m[:keyLen]
}
