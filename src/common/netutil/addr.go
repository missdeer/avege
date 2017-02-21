package netutil

import (
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"

	"common"
)

func GetExternalIPAddress() (string, error) {
	client := &http.Client{}
	u := "https://if.yii.li"
	req, err := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", "curl/7.41.0")
	resp, err := client.Do(req)
	if err != nil {
		common.Errorf("request %s failed", u)
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		common.Error("reading ifconfig response failed")
		return "", err
	}

	for i := len(body); i > 0 && (body[i-1] < '0' || body[i-1] > '9'); i = len(body) {
		body = body[:i-1]
	}

	if matched, err := regexp.Match(`^([0-9]{1,3}\.){3,3}[0-9]{1,3}$`, body); err == nil && matched == true {
		return string(body), nil
	}

	return "", errors.New("invalid IP address")
}
