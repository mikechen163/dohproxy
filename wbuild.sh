# compile for openwrt intel x86 structure 
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags="-s -w"
