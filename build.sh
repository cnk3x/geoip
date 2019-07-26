#!/bin/sh
cd $(dirname $0)
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0
go build -ldflags '-s -w' -v -o dist/geoip ./cmd/*.go
docker run --rm -it -w /app -v $(pwd):/app shuxs/upx:latest -9 -k dist/geoip
docker build -t shuxs/geoip:latest .
docker push shuxs/geoip:latest
curl -XPOST https://docker.amzcs.com/api/webhooks/30fb4f8f-795e-4fdd-b01e-f7b5431785e3
# docker run --rm -it --publish 3000:80 shuxs/geoip:latest
