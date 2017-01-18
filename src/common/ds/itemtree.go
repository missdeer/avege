package ds

import (
	"bufio"
	"os"
	"sort"
	"strings"
	"sync"

	"common"
	"common/fs"
)

type itemNode struct {
	next itemTree
}

type itemTree map[string]*itemNode

type ItemTree struct {
	fileName         string
	autoApplyChanges bool
	sync.RWMutex
	itemTree
}

func NewItemTree(file string, autoApplyChanges bool) *ItemTree {
	il := &ItemTree{
		fileName:         file,
		autoApplyChanges: autoApplyChanges,
	}

	if exists, _ := fs.IsFileExists(file); exists == true && autoApplyChanges == true {
		il.MonitorFileChange()
	}

	return il
}

func (this *ItemTree) MonitorFileChange() {
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

func (this *ItemTree) Hit(item string) bool {
	subs := strings.Split(item, ".")
	if len(subs) == 0 {
		return false
	}

	this.RLock()
	defer this.RUnlock()
	node := this.itemTree
	for i := len(subs) - 1; i >= 0; i-- {
		sub := subs[i]
		theNode, ok := node[sub]
		if !ok {
			return false
		}
		if i == 0 {
			if _, ok := theNode.next["nil"]; ok {
				return true
			}
		}

		node = theNode.next
	}

	return false
}

func (this *ItemTree) join(prefix string, key string, tree *itemTree, keys *[]string) {
	if _, ok := (*tree)["nil"]; ok {
		strings.Join([]string{prefix, key}, ".")
		*keys = append(*keys, key)
		return
	}
	for k, v := range *tree {
		this.join(strings.Join([]string{prefix, key}, "."), k, &v.next, keys)
	}
}

func (this *ItemTree) Save() bool {
	outFile, err := os.OpenFile(this.fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		common.Errorf("opening file %s for writing failed %v", this.fileName, err)
		return false
	}
	defer outFile.Close()

	this.RLock()
	var keys []string
	for k, v := range this.itemTree {
		this.join("", k, &v.next, &keys)
	}
	this.RUnlock()
	sort.Strings(keys)
	for _, k := range keys {
		outFile.WriteString(k + "\n")
	}
	return true
}

func (this *ItemTree) Load() bool {
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
	defer this.Unlock()

	this.itemTree = make(itemTree)

	for scanner.Scan() {
		item := scanner.Text()

		subs := strings.Split(item, ".")
		if len(subs) == 0 {
			continue
		}
		node := this.itemTree
		for i := len(subs) - 1; i >= 0; i-- {
			sub := subs[i]
			if len(sub) == 0 {
				continue
			}
			theNode, ok := node[sub]
			if !ok {
				node[sub] = &itemNode{
					next: make(itemTree),
				}
				theNode = node[sub]
			}
			node = theNode.next
		}
		node["nil"] = nil
	}
	return true
}

func (this *ItemTree) Clear() {
	this.Lock()
	defer this.Unlock()
	this.itemTree = make(itemTree)
}

func (this *ItemTree) AddItem(key string) {
	this.Lock()
	defer this.Unlock()
	subs := strings.Split(key, ".")
	if len(subs) == 0 {
		return
	}
	node := this.itemTree
	for i := len(subs) - 1; i >= 0; i-- {
		sub := subs[i]
		if len(sub) == 0 {
			continue
		}
		theNode, ok := node[sub]
		if !ok {
			node[sub] = &itemNode{
				next: make(itemTree),
			}
			theNode = node[sub]
		}
		node = theNode.next
	}
	node["nil"] = nil
}

func (this *ItemTree) IsEmpty() bool {
	this.RLock()
	defer this.RUnlock()
	return len(this.itemTree) == 0
}
