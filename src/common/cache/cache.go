package cache

import (
	"common"
)

type ICache interface {
	Get(key string) (interface{}, error)
	GetMulti(keys []string) []interface{}
	PutWithTimeout(key string, val interface{}, timeout int64) error
	Put(key string, val interface{}) error
	Delete(key string) error
	IsExist(key string) bool
	Add(key string, delta int64) error
	Incr(key string) error
	Decr(key string) error
	ClearAll() error
}

var (
	instances = make(map[string]ICache)
	// Instance default ICache instance
	Instance ICache
)

func Init(cacheName string) {
	switch cacheName {
	case "redis":
		r := redisInit()
		instances["redis"] = r
		Instance = r
	case "gocache":
		g := goCacheInit()
		instances["gocache"] = g
		Instance = g
	default:
		common.Panic("unsupported cache service:", cacheName)
	}
}

func FindCache(cacheName string) ICache {
	return instances[cacheName]
}
