#!/bin/sh
cd dnscrypt_resolver || exit 
docker build -t check .
docker run --rm -v $(pwd):/data check