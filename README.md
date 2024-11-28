# gorpm
[goreplay](https://github.com/buger/goreplay) middleware implemented by golang.

### How middleware works
Referred to the goreplay code [token_modifier.go](https://github.com/buger/goreplay/blob/master/examples/middleware/token_modifier.go) :
```bash
                   Original request      +--------------+
+-------------+----------STDIN---------->+              |
|  Gor input  |                          |  Middleware  |
+-------------+----------STDIN---------->+              |
                   Original response     +------+---+---+
                                                |   ^
+-------------+    Modified request             v   |
| Gor output  +<---------STDOUT-----------------+   |
+-----+-------+                                     |
      |                                             |
      |            Replayed response                |
      +------------------STDIN----------------->----+
```



### Installation

```bash
go get -u github.com/yushaolong10/gorpm
```

### Examples

(1) see [examples](https://github.com/yushaolong10/gorpm/tree/master/examples) ,code implementation:
```golang
package main

import (
	"github.com/yushaolong10/gorpm"
)

//OnRequestHandle you can modify request body
func OnRequestHandle(req *gorpm.GorMessage) *gorpm.GorMessage {
	//do nothing
	return req
}

//OnCompleteHandle callback when req,resp,reply message complete 
func OnCompleteHandle(req, resp, reply *gorpm.GorMessage) {
	//when all complete
}

func main() {
	//create
	gor := gorpm.CreateGor()
	//register
	gor.OnRequest(OnRequestHandle)
	gor.OnComplete(OnCompleteHandle)
	//run 
	gor.Run()
}
```
(2) compile source code into binary files:
```bash
#generate file: goreplay_middle_bin
bash examples/build.sh
```
(3) execute the gor command on the target machine, specifying the middleware binary file:
```bash
/usr/local/bin/gor --input-raw ":8080"  \
--http-allow-url "/openapi/yourapi"     \
--output-http="http://demo:8080"        \
--output-http-track-response            \
--input-raw-track-response              \
--input-raw-allow-incomplete            \
--middleware ./goreplay_middle_bin 
```


