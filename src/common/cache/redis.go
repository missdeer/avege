package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"os"
	"strconv"
	"time"
)

var (
	// the collection name of redis for cache adapter.
	DefaultRedisKey = "avegeConsole"
	RedisServer     = "127.0.0.1:6379"
	rd              *RedisCache
)

func redisInit() *RedisCache {
	rd = newRedisCache()
	if err := rd.StartAndGC(fmt.Sprintf("{\"conn\" : \"%s\"}", RedisServer)); err != nil {
		log.Panic("connecting to redis failed")
		os.Exit(1)
	}
	return rd
}

// Redis cache adapter.
type RedisCache struct {
	p        *redis.Pool // redis connection pool
	connInfo string
	dbNum    int
	key      string
	password string
}

// create new redis cache with default collection name.
func newRedisCache() *RedisCache {
	return &RedisCache{key: DefaultRedisKey}
}

// actually do the redis cmds
func (rc *RedisCache) do(commandName string, args ...interface{}) (reply interface{}, err error) {
	c := rc.p.Get()
	defer c.Close()

	return c.Do(commandName, args...)
}

// Get cache from redis.
func (rc *RedisCache) Get(key string) (interface{}, error) {
	return rc.do("GET", key)
}

// GetMulti get cache from redis.
func (rc *RedisCache) GetMulti(keys []string) []interface{} {
	size := len(keys)
	var rv []interface{}
	c := rc.p.Get()
	defer c.Close()
	var err error
	for _, key := range keys {
		err = c.Send("GET", key)
		if err != nil {
			goto ERROR
		}
	}
	if err = c.Flush(); err != nil {
		goto ERROR
	}
	for i := 0; i < size; i++ {
		if v, err := c.Receive(); err == nil {
			rv = append(rv, v.([]byte))
		} else {
			rv = append(rv, err)
		}
	}
	return rv
ERROR:
	rv = rv[0:0]
	for i := 0; i < size; i++ {
		rv = append(rv, nil)
	}

	return rv
}

// put cache to redis.
func (rc *RedisCache) PutWithTimeout(key string, val interface{}, timeout int64) error {
	var err error
	if _, err = rc.do("SETEX", key, timeout, val); err != nil {
		return err
	}

	if _, err = rc.do("HSET", rc.key, key, true); err != nil {
		return err
	}
	return err
}

func (rc *RedisCache) Put(key string, val interface{}) error {
	var err error
	if _, err = rc.do("SET", key, val); err != nil {
		return err
	}

	if _, err = rc.do("HSET", rc.key, key, true); err != nil {
		return err
	}
	return err
}

// delete cache in redis.
func (rc *RedisCache) Delete(key string) error {
	var err error
	if _, err = rc.do("DEL", key); err != nil {
		return err
	}
	_, err = rc.do("HDEL", rc.key, key)
	return err
}

// check cache's existence in redis.
func (rc *RedisCache) IsExist(key string) bool {
	v, err := redis.Bool(rc.do("EXISTS", key))
	if err != nil {
		return false
	}
	if v == false {
		if _, err = rc.do("HDEL", rc.key, key); err != nil {
			return false
		}
	}
	return v
}

func (rc *RedisCache) Add(key string, delta int64) error {
	_, err := redis.Bool(rc.do("INCRBY", key, delta))
	return err
}

// increase counter in redis.
func (rc *RedisCache) Incr(key string) error {
	_, err := redis.Bool(rc.do("INCRBY", key, 1))
	return err
}

// decrease counter in redis.
func (rc *RedisCache) Decr(key string) error {
	_, err := redis.Bool(rc.do("INCRBY", key, -1))
	return err
}

// clean all cache in redis. delete this redis collection.
func (rc *RedisCache) ClearAll() error {
	cachedKeys, err := redis.Strings(rc.do("HKEYS", rc.key))
	if err != nil {
		return err
	}
	for _, str := range cachedKeys {
		if _, err = rc.do("DEL", str); err != nil {
			return err
		}
	}
	_, err = rc.do("DEL", rc.key)
	return err
}

// start redis cache adapter.
// config is like {"key":"collection key","conn":"connection info","dbNum":"0"}
// the cache item in redis are stored forever,
// so no gc operation.
func (rc *RedisCache) StartAndGC(config string) error {
	var cf map[string]string
	json.Unmarshal([]byte(config), &cf)

	if _, ok := cf["key"]; !ok {
		cf["key"] = DefaultRedisKey
	}
	if _, ok := cf["conn"]; !ok {
		return errors.New("config has no conn key")
	}
	if _, ok := cf["dbNum"]; !ok {
		cf["dbNum"] = "0"
	}
	if _, ok := cf["password"]; !ok {
		cf["password"] = ""
	}
	rc.key = cf["key"]
	rc.connInfo = cf["conn"]
	rc.dbNum, _ = strconv.Atoi(cf["dbNum"])
	rc.password = cf["password"]

	rc.connectInit()

	c := rc.p.Get()
	defer c.Close()

	return c.Err()
}

// connect to redis.
func (rc *RedisCache) connectInit() {
	dialFunc := func() (c redis.Conn, err error) {
		c, err = redis.Dial("tcp", rc.connInfo)
		if err != nil {
			return nil, err
		}

		if len(rc.password) > 0 {
			if _, err := c.Do("AUTH", rc.password); err != nil {
				c.Close()
				return nil, err
			}
		}

		if _, selectErr := c.Do("SELECT", rc.dbNum); selectErr != nil {
			c.Close()
			return nil, selectErr
		}
		return
	}
	// initialize a new pool
	rc.p = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 180 * time.Second,
		Dial:        dialFunc,
	}
}
