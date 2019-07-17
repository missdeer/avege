#!/bin/bash
sysctl -w net.ipv4.ip_forward=1
sysctl -w fs.file-max=51200
sysctl -w net.core.rmem_max=67108864
sysctl -w net.core.wmem_max=67108864
sysctl -w net.core.netdev_max_backlog=250000
sysctl -w net.core.somaxconn=4096
sysctl -w net.ipv4.tcp_syncookies=1
sysctl -w net.ipv4.tcp_tw_reuse=1
sysctl -w net.ipv4.tcp_tw_recycle=0
sysctl -w net.ipv4.tcp_fin_timeout=30
sysctl -w net.ipv4.tcp_keepalive_time=1200
sysctl -w "net.ipv4.ip_local_port_range=10000 65000"
sysctl -w net.ipv4.tcp_max_syn_backlog=8192
sysctl -w net.ipv4.tcp_max_tw_buckets=5000
sysctl -w net.ipv4.tcp_fastopen=3
sysctl -w "net.ipv4.tcp_rmem=4096 87380 67108864"
sysctl -w "net.ipv4.tcp_wmem=4096 65536 67108864"
sysctl -w net.ipv4.tcp_mtu_probing=1
cd /home/pi/redis-2.8.18
src/redis-server redis.conf
cd /home/pi/avege
iptables-restore < ./rules.v4.updated
#inotifywait -mrq --format '%w%f' -e close_write /home/pi/avege/rules.v4.latest | while read file
#do
#   iptables-restore < $file
#   cp $file /home/pi/avege/rules.v4.updated
#done &
./avege 2> out &
./ngrok -log=stdout -proto=http -config=./ngrok.cfg -subdomain= 58098 &
