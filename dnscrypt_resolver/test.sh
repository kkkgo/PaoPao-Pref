#!/bin/sh
cd dnscrypt_resolver || exit 
go mod init check&& go get -u&&go build -ldflags="-s -w" -o check check.go
./check
rm -rf go.mod go.sum