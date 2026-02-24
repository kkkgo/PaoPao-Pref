#!/bin/sh
cd dnscrypt_resolver || exit 
rm -rf check&&go mod init check&& go get -u&&go build -ldflags="-s -w" -o check check.go
./check
rm -rf go.mod go.sum check
sha256sum ban_list.txt > ban_list.txt.sha256sum
sha256sum server.toml > server.toml.sha256sum