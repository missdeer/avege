#!/bin/bash
curl -H "Authorization: PqFVgV6InD" https://console.yii.li/admin/servers -s | grep -o "[a-z]\{2\}\-[0-9]" | while read sub; do curl -d "address=$sub.dfordsoft.com" 127.0.0.1:58098/v1/server/add; done
