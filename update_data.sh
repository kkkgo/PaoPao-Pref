#!/bin/sh
curl -sLo top-1m.csv.zip http://s3-us-west-1.amazonaws.com/umbrella-static/top-1m.csv.zip
unzip top-1m.csv.zip
cut -d"," -f2 top-1m.csv >domains.txt
rm top-1m.csv.zip top-1m.csv
sudo apt-get -qq -y install golang
go build -ldflags "-s -w" -trimpath -o ./paopao-pref
export FILE_OUTPUT=yes
export DNS_LIMIT=9
export DNS_SLEEP=0ms
export DNS_TIMEOUT=3s
touch domains_ok.txt
chmod +x ./paopao-pref
./paopao-pref
count=$(cat domains_ok.txt | wc -l)
if [ "$count" -gt 100000 ]; then
    mv domains_ok.txt domains.txt
else
    rm domains_ok.txt
fi
rm paopao-pref
