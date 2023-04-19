#!/bin/sh
mkdir -p /cp
unzip top-1m.csv.zip
ls -lah
cut -d"," -f2  top-1m.csv > /cp/domains.txt
# build PaoPao-Pref
go build -ldflags "-s -w" -trimpath -o /cp/paopao-pref
