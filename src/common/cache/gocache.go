package cache

import (
	"errors"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

var (
	ErrObjectNotFound  = errors.New("not found")
	ErrObjectNotNumber = errors.New("not a number")
	goCache            *GoCache
)

type GoCache struct {
	i *gocache.Cache
}

func goCacheInit() *GoCache {
	goCache = &GoCache{
		i: gocache.New(5*time.Minute, 30*time.Minute),
	}
	return goCache
}

func (g *GoCache) Get(key string) (interface{}, error) {
	r, b := g.i.Get(key)
	if b {
		return r, nil
	}
	return nil, ErrObjectNotFound
}

func (g *GoCache) GetMulti(keys []string) []interface{} {
	var res []interface{}
	for _, key := range keys {
		r, b := g.i.Get(key)
		if b {
			res = append(res, r)
		}
	}
	return res
}

func (g *GoCache) PutWithTimeout(key string, val interface{}, timeout int64) error {
	g.i.Set(key, val, time.Duration(timeout)*time.Second)
	return nil
}

func (g *GoCache) Put(key string, val interface{}) error {
	g.i.Set(key, val, gocache.NoExpiration)
	return nil
}

func (g *GoCache) Delete(key string) error {
	g.i.Delete(key)
	return nil
}

func (g *GoCache) IsExist(key string) bool {
	_, b := g.i.Get(key)
	return b
}

func (g *GoCache) Add(key string, delta int64) error {
	r, b := g.i.Get(key)
	if !b {
		return ErrObjectNotFound
	}

	if d, ok := r.(int64); ok {
		d += delta
		g.i.Set(key, d, gocache.NoExpiration)
		return nil
	}
	return ErrObjectNotNumber
}

func (g *GoCache) Incr(key string) error {
	return g.Add(key, 1)
}

func (g *GoCache) Decr(key string) error {
	return g.Add(key, -1)
}

func (g *GoCache) ClearAll() error {
	return nil
}
