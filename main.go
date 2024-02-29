package main

import (
	"log"
	"net/http"
	"wechat/core"
)

func main() {
	http.HandleFunc("/webhook", core.HandleWebhook)
	log.Fatal(http.ListenAndServe(":8086", nil))
}

//alert配置  send_resolved: true and resolve_timeout是5m 意思是五分钟没有在收到报警 才会发告警恢复
//CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./webhook -ldflags '-s -w' main.go
