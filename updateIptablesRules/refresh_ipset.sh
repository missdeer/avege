#!/bin/bash
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root" 
   exit 1
fi

ipset list | grep route | while read line; 
do 
   name=`echo $line | awk -F ' ' '{print $2}'`; 
   ipset flush $name; 
done

# generate other route
sed -i '/create/d' ipset.txt
sed -i '/cnroute/d' ipset.txt
ipset restore -f ipset.txt

# generate server list
rm -f server.txt
grep -r -a  -o '[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}' haproxy.mixed.cfg | sort -n | uniq | while read line
do
   echo "add serverlist $line/32" >> server.txt
done
ipset flush serverlist
ipset restore -f server.txt

# generate cnroute
rm -f cnroute.txt
curl https://cdn.jsdelivr.net/gh/17mon/china_ip_list@master/china_ip_list.txt -sSL | while read line 
do 
   echo "add cnroute $line" >> cnroute.txt; 
done
ipset flush cnroute
ipset restore -f cnroute.txt