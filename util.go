package gorpm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func ParseHttpStatus(payload string) (string, error) {
	payload, err := url.PathUnescape(payload)
	if err != nil {
		return "", err
	}
	pstart := strings.IndexByte(payload, ' ')
	if pstart == -1 {
		return "", fmt.Errorf("invalid not found blank start")
	}
	pend := strings.IndexByte(payload[pstart+1:], ' ')
	if pend == -1 {
		return "", fmt.Errorf("invalid not found blank end")
	}
	return payload[pstart+1 : pstart+pend+1], nil
}

func ParseHttpRequest(buf []byte) (*http.Request, error) {
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(buf)))
	if err != nil {
		return nil, err
	}
	return req, nil
}

func ParseHttpResponse(buf []byte) (*http.Response, error) {
	// 创建一个缓冲区来读取响应字符串
	reader := bufio.NewReader(bytes.NewReader(buf))
	// 解析状态行
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read status line: %w", err)
	}
	statusLine = strings.TrimSpace(statusLine)

	// 解析状态码和状态消息
	var proto, statusCodeStr string
	_, err = fmt.Sscanf(statusLine, "%s %s ", &proto, &statusCodeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse status line: %w", err)
	}
	statusCode, _ := strconv.Atoi(statusCodeStr) // 这里已经确保能转换成功，所以忽略错误
	// 创建一个http.Response对象
	response := &http.Response{
		Proto:      proto,
		ProtoMajor: 1, // HTTP/1.1
		ProtoMinor: 1,
		StatusCode: statusCode,
		Header:     make(http.Header),
	}
	// 解析头部字段
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read header line: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break // 空行表示头部结束
		}
		// 解析头部字段名和值（这里简单处理，不考虑多个值的情况）
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue // 忽略格式不正确的头部字段
		}
		fieldName := strings.TrimSpace(parts[0])
		fieldValue := strings.TrimSpace(parts[1])
		response.Header.Add(fieldName, fieldValue)
	}
	body, _ := ioutil.ReadAll(reader)
	data := bytes.Split(body, []byte("\r\n"))
	if len(data) < 2 {
		response.Body = ioutil.NopCloser(bytes.NewReader(body))
		return response, nil
	}
	response.Body = ioutil.NopCloser(bytes.NewReader(data[1]))
	return response, nil
}

func JsonEncode(data interface{}) string {
	v, _ := json.Marshal(data)
	return string(v)
}

func LogStdErrOutput(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf("%s\n", format), a...)
}

func LogFileOutput(writer *bufio.Writer, format string, a ...interface{}) {
	fmt.Fprintf(writer, fmt.Sprintf("%s\n", format), a...)
}
