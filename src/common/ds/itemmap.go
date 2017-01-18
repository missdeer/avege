package ds

import (
	"bufio"
	"os"
	"sort"
	"sync"

	"common"
	"common/fs"
)

type itemMap map[string]bool

type ItemMap struct {
	fileName         string
	autoApplyChanges bool
	cap              int
	sync.RWMutex
	itemMap
}

func NewItemMapWithCap(file string, autoApplyChanges bool, cap int) *ItemMap {
	il := &ItemMap{
		fileName:         file,
		autoApplyChanges: autoApplyChanges,
		cap:              cap,
	}

	if exists, _ := fs.IsFileExists(file); exists == true && autoApplyChanges == true {
		il.MonitorFileChange()
	}

	return il
}

func NewItemMap(file string, autoApplyChanges bool) *ItemMap {
	il := &ItemMap{
		fileName:         file,
		autoApplyChanges: autoApplyChanges,
	}

	if exists, _ := fs.IsFileExists(file); exists == true && autoApplyChanges == true {
		il.MonitorFileChange()
	}

	return il
}

func (this *ItemMap) MonitorFileChange() {
	configFileChanged := make(chan bool)
	go func() {
		for {
			select {
			case <-configFileChanged:
				common.Debug(this.fileName, "changes, reload now...")
				this.Load()
			}
		}
	}()
	go fs.MonitorFileChanegs(this.fileName, configFileChanged)
}

func (this *ItemMap) Hit(item string) bool {
	this.RLock()
	_, ok := this.itemMap[item]
	this.RUnlock()
	return ok
}

func (this *ItemMap) Save() bool {
	outFile, err := os.OpenFile(this.fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		common.Errorf("opening file %s for writing failed %v", this.fileName, err)
		return false
	}
	defer outFile.Close()

	this.RLock()
	var keys []string
	for k := range this.itemMap {
		keys = append(keys, k)
	}
	this.RUnlock()
	sort.Strings(keys)
	for _, k := range keys {
		outFile.WriteString(k + "\n")
	}
	return true
}

func (this *ItemMap) Load() bool {
	configFile, err := fs.GetConfigPath(this.fileName)

	if err != nil {
		common.Errorf("%s not found, give up loading content", this.fileName)
		return false
	}

	common.Debugf("%s exists, loading...", configFile)
	if this.fileName != configFile {
		this.fileName = configFile
	}
	inFile, _ := os.Open(configFile)
	defer inFile.Close()

	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)
	this.Lock()
	defer func() {
		this.Unlock()
	}()

	if this.cap != 0 {
		this.itemMap = make(itemMap, this.cap)
	} else {
		this.itemMap = make(itemMap)
	}
	for scanner.Scan() {
		this.itemMap[scanner.Text()] = true
	}

	return len(this.itemMap) != 0
}

func (this *ItemMap) Clear() {
	this.Lock()
	this.itemMap = make(itemMap)
	this.Unlock()
}

func (this *ItemMap) AddItem(key string) {
	if _, ok := this.itemMap[key]; !ok {
		this.itemMap[key] = true
	}
}

func (this *ItemMap) IsEmpty() bool {
	this.RLock()
	isEmpty := (len(this.itemMap) == 0)
	this.RUnlock()
	return isEmpty
}
