# Avege

Socks5/Transparent Proxy Client

[![Build Status](https://travis-ci.org/avege/avege.svg?branch=master)](https://travis-ci.org/missdeer/avege)
[![GitHub release](https://img.shields.io/github/release/missdeer/avege.svg?maxAge=2592000)](https://github.com/missdeer/avege/releases) 
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/missdeer/avege/master/LICENSE) 
[![Github Releases Downloads Total](https://img.shields.io/github/downloads/missdeer/avege/total.svg)](https://github.com/missdeer/avege/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/missdeer/avege)](https://goreportcard.com/report/github.com/missdeer/avege)
[![codebeat badge](https://codebeat.co/badges/1cf94e1d-9b1a-4834-b9ae-a99df311273e)](https://codebeat.co/projects/github-com-missdeer-avege-master)

## Feature

* Windows/macOS/Linux/variant BSDs supported
* socks5 frontend, IPv4/IPv6/remote DNS resolving supported
* redir mode frontend on Linux (iptables compatible), IPv4/IPv6(not tested) supported
* tunnel mode frontend, IPv4/IPv6 supported
* http/https backend
* socks4/socks4a/socks5 backend
* Shadowsocks(R) backend
* DNS proxy that protects user against DNS poisoning but is CDN friendly in China

#### SS Encrypting algorithm

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

* UDP forwarding, include original Shadowsocks compatible UDP relay and ShadowsocksR compatible UDP over TCP relay
* tun based system wide proxy
* Adblock Plus rules based filter for http
* TCP Fast Open on Linux with 3.7+ kernel
* Transparent proxy aka. redir mode on Mac OS X and variant BSDs(ipfw/pf mode)

## Build

#### Dependencies

```shell
go get -u -f -v github.com/op/go-logging
go get -u -f -v github.com/garyburd/redigo/redis
go get -u -f -v github.com/fsnotify/fsnotify
go get -u -f -v github.com/kardianos/osext
go get -u -f -v github.com/gin-gonic/gin
go get -u -f -v github.com/gorilla/websocket
go get -u -f -v github.com/DeanThompson/ginpprof
go get -u -f -v github.com/miekg/dns
go get -u -f -v github.com/Yawning/chacha20
go get -u -f -v github.com/dgryski/go-camellia
go get -u -f -v github.com/dgryski/go-idea
go get -u -f -v github.com/dgryski/go-rc2
go get -u -f -v github.com/patrickmn/go-cache
go get -u -f -v github.com/RouterScript/ProxyClient
go get -u -f -v github.com/ftrvxmtrx/fd
```

#### Steps

for macOS/Linux/BSDs

```shell
git clone https://github.com/missdeer/avege.git
cd avege/src/avege
GOPATH=$GOPATH:$PWD/../.. go build 
```

## Usage

#### Dependencies

* redis-server (not necessary if you use other cache service such as **gocache** instead)

#### Steps

for macOS/Linux/BSDs, open terminal as you like, input:

```shell
cd avege/src/avege
cp config-sample.json config.json
# modify config.json as you like
./avege
```

for Windows, open cmd.exe, input:

```shell
cd avege\src\avege
copy config-sample.json config.json
# modify config.json as you like
avege.exe
```

## Reference

* [redsocks](https://github.com/darkk/redsocks)
* [Shadowsocks-go](https://github.com/shadowsocks/shadowsocks-go)
* [ShadowsocksR](https://github.com/breakwa11/shadowsocks-csharp)
* [go-any-proxy](https://github.com/freskog/go-any-proxy)
* [go-socks5](https://github.com/armon/go-socks5)
