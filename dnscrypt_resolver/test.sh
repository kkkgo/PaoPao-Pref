#!/bin/sh
cd dnscrypt_resolver || exit 
docker build -t check -f dockerfile .
docker run --rm -v $(pwd):/data check