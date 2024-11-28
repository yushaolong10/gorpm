package gorpm

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	MessageRequest  = "1"
	MessageResponse = "2"
	MessageReply    = "3"
)

// GorMessage stores data and parsed information in a incoming request
type GorMessage struct {
	ID      string
	Type    string
	Meta    [][]byte // Meta is an array size of 4, containing: request type, uuid, timestamp, latency
	RawMeta []byte   //
	HTTP    []byte   // Raw HTTP payload
}

// gorMessagePack is temporary for request/response/reply message, will be clear when timeout or complete
type gorMessagePack struct {
	Request  *GorMessage
	Response *GorMessage
	Reply    *GorMessage
	created  time.Time
}

func (pack *gorMessagePack) Ready() bool {
	return pack.Request != nil && pack.Response != nil && pack.Reply != nil
}

// Gor is the middleware itself
type Gor struct {
	lock       *sync.Mutex
	msgPackMap map[string]*gorMessagePack
	input      chan string
	parsed     chan *GorMessage
	stderr     io.Writer
	//callback when request ok, you can modify replay request body here
	onRequestFunc func(req *GorMessage) *GorMessage
	//callback when all message complete
	onCompleteFunc func(req, resp, reply *GorMessage)
}

// CreateGor creates a Gor object
func CreateGor() *Gor {
	gor := &Gor{
		lock:       new(sync.Mutex),
		msgPackMap: make(map[string]*gorMessagePack),
		input:      make(chan string),
		parsed:     make(chan *GorMessage),
		stderr:     os.Stderr,
	}
	return gor
}

func (gor *Gor) OnRequest(msgReqFunc func(req *GorMessage) *GorMessage) {
	gor.onRequestFunc = msgReqFunc
}

func (gor *Gor) OnComplete(msgCompleteFunc func(req, resp, reply *GorMessage)) {
	gor.onCompleteFunc = msgCompleteFunc
}

func (gor *Gor) getMsgPack(id string) *gorMessagePack {
	gor.lock.Lock()
	defer gor.lock.Unlock()
	pack, ok := gor.msgPackMap[id]
	if ok {
		return pack
	}
	pack = &gorMessagePack{created: time.Now()}
	gor.msgPackMap[id] = pack
	return pack
}

func (gor *Gor) deleteMsgPack(id string) {
	gor.lock.Lock()
	defer gor.lock.Unlock()
	delete(gor.msgPackMap, id)
}

// emit triggers the registered event callback when receiving certain GorMessage
func (gor *Gor) emit(msg *GorMessage) error {
	pack := gor.getMsgPack(msg.ID)
	switch msg.Type {
	case MessageRequest:
		pack.Request = msg
		if gor.onRequestFunc != nil {
			msg = gor.onRequestFunc(msg)
		}
		fmt.Fprintf(os.Stdout, gor.hexData(msg))
	case MessageResponse:
		pack.Response = msg
	case MessageReply:
		pack.Reply = msg
	default:
		return fmt.Errorf("invalid message type: %s", msg.Type)
	}
	if !pack.Ready() {
		return nil
	}
	if gor.onCompleteFunc != nil {
		gor.onCompleteFunc(pack.Request, pack.Response, pack.Reply)
	}
	gor.deleteMsgPack(msg.ID)
	return nil
}

// HexData translates a GorMessage into middleware dataflow string
func (gor *Gor) hexData(msg *GorMessage) string {
	encodeList := [3][]byte{msg.RawMeta, []byte("\n"), msg.HTTP}
	encodedList := make([]string, 3)
	for i, val := range encodeList {
		encodedList[i] = hex.EncodeToString(val)
	}
	encodedList = append(encodedList, "\n")
	return strings.Join(encodedList, "")
}

// parseMessage parses string middleware dataflow into a GorMessage
func (gor *Gor) parseMessage(line string) (*GorMessage, error) {
	payload, err := hex.DecodeString(strings.TrimSpace(line))
	if err != nil {
		return nil, err
	}
	metaPos := bytes.Index(payload, []byte("\n"))
	metaRaw := payload[:metaPos]
	metaArr := bytes.Split(metaRaw, []byte(" "))
	msgType, pid := metaArr[0], string(metaArr[1])
	httpPayload := payload[metaPos+1:]
	return &GorMessage{
		ID:      pid,
		Type:    string(msgType),
		Meta:    metaArr,
		RawMeta: metaRaw,
		HTTP:    httpPayload,
	}, nil
}

func (gor *Gor) clearTimeoutMsgPack(interval int) {
	ticker := time.NewTicker(time.Second * 1)
	for range ticker.C {
		gor.lock.Lock()
		for id, pack := range gor.msgPackMap {
			if time.Since(pack.created) > time.Duration(interval) {
				delete(gor.msgPackMap, id)
			}
		}
		gor.lock.Unlock()
	}
}

func (gor *Gor) preProcessor() {
	for {
		line := <-gor.input
		if msg, err := gor.parseMessage(line); err != nil {
			gor.stderr.Write([]byte(err.Error()))
		} else {
			gor.parsed <- msg
		}
	}
}

func (gor *Gor) receiver() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		gor.input <- scanner.Text()
	}
}

func (gor *Gor) processor() {
	for {
		msg := <-gor.parsed
		gor.emit(msg)
	}
}

func (gor *Gor) shutdown() {
}

func (gor *Gor) handleSignal(sigChan chan os.Signal) {
	for {
		s := <-sigChan
		gor.stderr.Write([]byte(fmt.Sprintf("receive a signal %s\n", s.String())))
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGSTOP, syscall.SIGINT:
			gor.shutdown()
			return
		default:
			return
		}
	}
}

// Run is entrypoint of Gor
func (gor *Gor) Run() {
	go gor.receiver()
	go gor.preProcessor()
	go gor.processor()
	go gor.clearTimeoutMsgPack(30)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM,
		syscall.SIGINT, syscall.SIGSTOP)
	gor.handleSignal(c)
}
