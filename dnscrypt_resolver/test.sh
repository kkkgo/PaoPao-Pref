#!/bin/sh
docker build -t check -f dockerfile .
docker run --rm -v $(pwd):/data check