package main

import (
	"github.com/yushaolong10/gorpm"
)

//OnRequestHandle 可以调整转发http的消息报文
func OnRequestHandle(req *gorpm.GorMessage) *gorpm.GorMessage {
	//do nothing
	return req
}

//OnCompleteHandle 消息完成回调
func OnCompleteHandle(req, resp, reply *gorpm.GorMessage) {
	//when all complete
	handleMessage(req, resp, reply)
}

func main() {
	//创建数据文件
	err := createFile()
	if err != nil {
		gorpm.LogStdErrOutput("create file failed:%s", err.Error())
		return
	}
	defer closeFile()
	//monitor
	go monitor()
	//创建gor
	gor := gorpm.CreateGor()
	//订阅消息事件
	gor.OnRequest(OnRequestHandle)
	gor.OnComplete(OnCompleteHandle)

	gor.Run()
}
