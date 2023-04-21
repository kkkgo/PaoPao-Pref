#!/bin/sh
curl -sLo top-1m.csv.zip http://s3-us-west-1.amazonaws.com/umbrella-static/top-1m.csv.zip
unzip top-1m.csv.zip
cut -d"," -f2  top-1m.csv > domains.txt
rm top-1m.csv.zip top-1m.csv
sudo apt-get -qq -y install golang
go build -ldflags "-s -w" -trimpath -o ./paopao-pref
export FILE_OUTPUT=yes
export DNS_LIMIT=100
touch domains_ok.txt
paopao-pref
mv domains_ok.txt domains.txt