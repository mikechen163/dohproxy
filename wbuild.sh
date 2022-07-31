<<<<<<< HEAD
# compile for openwrt 
=======
# compile for openwrt intel x86 structure 
>>>>>>> d0f15915e8883866aa5b3e5a293a39f4a8e1283c
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags="-s -w"
