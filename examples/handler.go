package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/yushaolong10/gorpm"
	"io/ioutil"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

var (
	fileNameInfo = "http_goreplay.log.info"
	fileNameErr  = "http_goreplay.log.error"

	fwOk  *bufio.Writer
	fwErr *bufio.Writer
	qps   int64
)

func createFile() error {
	//info
	f1, err := os.OpenFile(fileNameInfo, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("create file:%s err:%s", fileNameInfo, err.Error())
	}
	fwOk = bufio.NewWriter(f1)
	//error
	f2, err := os.OpenFile(fileNameErr, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("create file:%s err:%s", fileNameErr, err.Error())
	}
	fwErr = bufio.NewWriter(f2)
	return nil
}

func closeFile() {
	//刷入缓存
	fwOk.Flush()
	fwErr.Flush()
	time.Sleep(time.Second)
}

func monitor() {
	for {
		time.Sleep(time.Second * 10)
		curQps := atomic.LoadInt64(&qps)
		gorpm.LogStdErrOutput("http monitor qps:%d", curQps)
		atomic.StoreInt64(&qps, 0)
	}
}

type HttpMessage struct {
	Code           int           `json:"code"`
	Message        string        `json:"message"`
	ReqTime        string        `json:"request_time"`
	ReqTimestamp   int64         `json:"request_timestamp"`
	MessageId      string        `json:"message_id"`
	Method         string        `json:"method"`
	URL            string        `json:"url"`
	RawResponse    *HttpResponse `json:"raw_response"`
	ReplayResponse *HttpResponse `json:"replay_response"`
	Reason         []string      `json:"reason,omitempty"`
}

func handleMessage(req *gorpm.GorMessage, resp *gorpm.GorMessage, reply *gorpm.GorMessage) {
	atomic.AddInt64(&qps, 1)
	respStatus, _ := gorpm.ParseHttpStatus(string(resp.HTTP))
	replayStatus, _ := gorpm.ParseHttpStatus(string(reply.HTTP))
	if respStatus != replayStatus {
		gorpm.LogStdErrOutput("replay status %s not equal response status %s", replayStatus, respStatus)
		return
	}
	httpReq, err := gorpm.ParseHttpRequest(req.HTTP)
	if err != nil {
		gorpm.LogStdErrOutput("ParseHttpRequest err:%s", err.Error())
		return
	}
	rawRet, err := gorpm.ParseHttpResponse(resp.HTTP)
	if err != nil {
		gorpm.LogStdErrOutput("ParseHttpResponse raw response err:%s", err.Error())
		return
	}
	replayRet, err := gorpm.ParseHttpResponse(reply.HTTP)
	if err != nil {
		gorpm.LogStdErrOutput("ParseHttpResponse replay response err:%s", err.Error())
		return
	}
	rawBody, _ := ioutil.ReadAll(rawRet.Body)
	replayBody, _ := ioutil.ReadAll(replayRet.Body)
	diffMessage(req.ID, httpReq, rawBody, replayBody)
}

type HttpResponse struct {
	RequestId string                 `json:"request_id"`
	Code      int                    `json:"code"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data"`
}

func diffMessage(messageId string, httpReq *http.Request, rawBody, replayBody []byte) {
	rawRet := &HttpResponse{}
	replayRet := &HttpResponse{}
	_ = json.Unmarshal(rawBody, rawRet)
	_ = json.Unmarshal(replayBody, replayRet)

	httpMsg := &HttpMessage{
		ReqTime:        time.Now().Format("2006-01-02 15:04:05"),
		ReqTimestamp:   time.Now().Unix(),
		MessageId:      messageId,
		Method:         httpReq.Method,
		URL:            httpReq.URL.RequestURI(),
		RawResponse:    rawRet,
		ReplayResponse: replayRet,
	}
	if ok, reason := gorpm.Compare(rawRet.Data, replayRet.Data); ok {
		httpMsg.Code = 0
		httpMsg.Message = "ok"
		gorpm.LogFileOutput(fwOk, "%s", gorpm.JsonEncode(httpMsg))
	} else {
		httpMsg.Code = 1
		httpMsg.Message = "response data not equal"
		httpMsg.Reason = reason
		gorpm.LogFileOutput(fwErr, "%s", gorpm.JsonEncode(httpMsg))
	}
}
