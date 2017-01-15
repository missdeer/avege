# Avege

Socks5/Transparent Proxy Client

## Feature

* socks5 frontend
* redir mode frontend on Linux (iptables compatible)
* http/https backend
* socks4/socks4a/socks5 backend
* Shadowsocks(R) backend

#### Encrypting algorithm

* aes-128-cfb
* aes-192-cfb
* aes-256-cfb
* aes-128-ctr
* aes-192-ctr
* aes-256-ctr
* aes-128-ofb
* aes-192-ofb
* aes-256-ofb
* des-cfb
* bf-cfb
* cast5-cfb
* rc4-md5
* chacha20
* chacha20-ietf
* salsa20
* camellia-128-cfb
* camellia-192-cfb
* camellia-256-cfb
* idea-cfb
* rc2-cfb
* seed-cfb

#### SSR Obfs

* plain
* http_simple
* http_post
* random_head
* tls1.2_ticket_auth

#### SSR Protocol

* origin
* verify_sha1 aka. one time auth(OTA)
* auth_sha1_v4
* auth_aes128_md5
* auth_aes128_sha1

## Todo (help wanted)

* UDP forwarding
* IPv6 supported
* tun based system wide proxy
* Adblock Plus rules based filter for http
* TCP Fast Open on Linux with 3.7+ kernel
* Transparent proxy on Mac OS X and variant BSDs(ipfw/pf/fw mode)

## Build

#### Dependencies

```shell
go get github.com/op/go-logging
go get github.com/garyburd/redigo/redis
go get github.com/go-fsnotify/fsnotify
go get github.com/kardianos/osext
go get github.com/gin-gonic/gin
go get github.com/gorilla/websocket
go get github.com/DeanThompson/ginpprof
go get github.com/miekg/dns
go get github.com/aead/chacha20
go get github.com/codahale/chacha20
go get github.com/dgryski/go-camellia
go get github.com/dgryski/go-idea
go get github.com/dgryski/go-rc2
go get github.com/patrickmn/go-cache
```

#### Steps

```shell
git clone https://github.com/missdeer/avege.git
cd avege/src/avege
GOPATH=$GOPATH:$PWD/../.. go build 
```

## Usage

#### Dependencies

* redis-server

#### Steps

```shell
cd avege/src/avege
cp config-sample.json config.json
# modify config.json as you like
./avege
```

## Reference

* [redsocks](https://github.com/darkk/redsocks)
* [Shadowsocks-go](https://github.com/shadowsocks/shadowsocks-go)
* [ShadowsocksR](https://github.com/breakwa11/shadowsocks-csharp)
* [ProxyClient](https://github.com/GameXG/ProxyClient)
* [go-any-proxy](https://github.com/freskog/go-any-proxy)
* [go-socks5](https://github.com/armon/go-socks5)
