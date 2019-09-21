package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var (
	account  string
	password string
)

func getMiaoPSCookie() string {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// login
	data := `email=` + account + `%40qq.com&passwd=` + password + `&code=&remember_me=week`
	req, err := http.NewRequest("POST", "https://xn--i2ru8q2qg.com/auth/login", bytes.NewBufferString(data))
	if err != nil {
		fmt.Println("Could not parse login request:", err)
		return ""
	}

	req.Header.Set("Referer", "https://xn--i2ru8q2qg.com/auth/login")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2684.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Origin", `https://xn--i2ru8q2qg.com`)
	req.Header.Set("X-Requested-With", `XMLHttpRequest`)
	req.Header.Set("Content-Type", `application/x-www-form-urlencoded; charset=UTF-8`)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Could not post login request:", err)
		return ""
	}

	defer resp.Body.Close()

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("can't read login response", err)
	}

	cookies := resp.Cookies()
	var cookie []string
	for _, v := range cookies {
		ss := strings.Split(v.String(), ";")
		cookie = append(cookie, ss[0])
	}

	return strings.Join(cookie, "; ")
}

func getMiaoPSNodePCConf(cookie string) []byte {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://xn--i2ru8q2qg.com/user/getpcconf", nil)
	if err != nil {
		fmt.Println("Could not parse get all servers request:", err)
		return []byte("")
	}

	req.Header.Set("Referer", "https://xn--i2ru8q2qg.com/user")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2684.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-encoding", `gzip, deflate, sdch`)
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
	req.Header.Set("upgrade-insecure-requests", `1`)
	req.Header.Set("authority", `xn--i2ru8q2qg.com`)
	req.Header.Set("cookie", cookie)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Could not send get all servers request:", err)
		return []byte("")
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("can't read pc conf")
		return []byte("")
	}
	return data
}

func checkinMiaoPS(cookie string) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", "https://xn--i2ru8q2qg.com/user/checkin", nil)
	if err != nil {
		fmt.Println("Could not parse checkin request:", err)
		return
	}

	req.Header.Set("Origin", `https://xn--i2ru8q2qg.com`)
	req.Header.Set("X-Requested-With", `XMLHttpRequest`)
	req.Header.Set("Referer", "https://xn--i2ru8q2qg.com/user")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2684.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("accept-encoding", `gzip, deflate, sdch`)
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
	req.Header.Set("content-length", `0`)
	req.Header.Set("authority", `xn--i2ru8q2qg.com`)
	req.Header.Set("cookie", cookie)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Could not post checkin request:", err)
		return
	}

	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("can't read checkin response", err)
		return
	}
}

type MoeConfig struct {
	Server string `json:"server"`
}

type Moe struct {
	Config []*MoeConfig `json:"configs"`
}

func getMiaoPSNodeServers(conf []byte) *Moe {
	pcconf := &Moe{}
	if json.Unmarshal(conf, pcconf) != nil {
		fmt.Println("unmarshalling failed", string(conf))
		return nil
	}
	return pcconf
}

func main() {
	flag.StringVar(&account, "account", "93414321", "the account aka QQ number")
	flag.StringVar(&password, "password", "QDaSTxeeuRKG9HdM", "the password")
	flag.Parse()
	cookie := getMiaoPSCookie()
	checkinMiaoPS(cookie)

	// conf := getMiaoPSNodePCConf(cookie)
	// pcconf := getMiaoPSNodeServers(conf)
	// var names []string
	// for _, v := range pcconf.Config {
	// 	r := regexp.MustCompile(`([a-z0-9\-]+)\.xt\.lianjiang.moe`)
	// 	ss := r.FindAllStringSubmatch(v.Server, -1)
	// 	for _, s := range ss {
	// 		names = append(names, string(s[1]))
	// 	}
	// }
	// fmt.Println(names)
}
