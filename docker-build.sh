#!/bin/sh
mv domains.txt /cp/
go build -ldflags "-s -w" -trimpath -o /cp/paopao-pref
