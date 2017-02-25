package local

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"common"
	"config"
)

var (
	consoleHost = "https://www.console.com"
	consoleVer  = "v1"
	client      = &http.Client{
		Timeout: 30 * time.Second,
	}
)

func updateConsoleConfigurations() {
	consoleHost = config.Configurations.Generals.ConsoleHost
	consoleVer = config.Configurations.Generals.ConsoleVersion
	consoleWSUrl = config.Configurations.Generals.ConsoleWebSocketURL
}

func getQuote() {
	retried := false
DO_GET:

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s/quote", consoleHost, consoleVer), nil)
	req.Header.Set("Authorization", config.Configurations.Generals.Token)
	resp, err := client.Do(req)
	if err != nil {
		if !retried {
			common.Error("getting quote failed", err, " try again")
			retried = true
			goto DO_GET
		}
		common.Error("getting quote failed", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		common.Error("getting quote response", resp.Status)
		if resp.StatusCode == http.StatusUnauthorized {
			config.LeftQuote = 0
		}
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		common.Error("getting quote response failed", err)
		return
	}

	var resQuote map[string]int64
	err = json.Unmarshal(body, &resQuote)
	if err != nil {
		var res map[string]string
		err = json.Unmarshal(body, &res)
		if err == nil {
			common.Error("getting quote result:", res["result"])
		} else {
			common.Error("getting quote unknown error, can't unmashal response:", string(body))
		}
		return
	}
	if q, ok := resQuote["quote"]; ok {
		common.Debug("left quote:", q)
		config.LeftQuote = q
	}
}

func uploadStatistic() {
	dl := common.DeltaStat.ResetDownload()
	ul := common.DeltaStat.ResetUpload()
	if dl == 0 && ul == 0 {
		common.Debug("no consume")
		getQuote()
		return
	}

	retried := false
DO_POST:
	postValues := url.Values{
		"download": {fmt.Sprintf("%d", dl)},
		"upload":   {fmt.Sprintf("%d", ul)},
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s/used", consoleHost, consoleVer), strings.NewReader(postValues.Encode()))
	req.Header.Set("Authorization", config.Configurations.Generals.Token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		if !retried {
			common.Error("Posting statistic failed", err, " try again")
			retried = true
			goto DO_POST
		}
		common.Error("Posting statistic failed", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		common.Error("getting statistic response", resp.Status)
		if resp.StatusCode == http.StatusUnauthorized {
			config.LeftQuote = 0
		}
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		common.Error("getting statistic response failed", err)
		return
	}
	var resQuote map[string]interface{}
	if err = json.Unmarshal(body, &resQuote); err != nil {
		common.Error("posting statistic unknown error, can't unmashal response:", err, string(body))
		return
	}
	if res, ok := resQuote["result"]; ok {
		if res != "OK" {
			common.Error("posting statistic error:", res)
			return
		}
	}
	if q, ok := resQuote["quote"]; ok {
		if quote, ok := q.(int64); ok {
			common.Debug("left quote:", quote)
			config.LeftQuote = quote
		}
	}
}
