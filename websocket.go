package websocket

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

type Websocket struct {
	context context.Context

	Origin              string
	Cache               string
	UserAgent           string
	WebsocketKey        string
	WebsocketExtensions []string

	Handler func([]byte) (Opcode, []byte, error)
	RW      *bufio.ReadWriter
}

func (ws *Websocket) Hijack(w http.ResponseWriter) (net.Conn, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		panic("cannot hijack: hijacker not present")
	}
	conn, readerWriter, err := hijacker.Hijack()
	if err != nil {
		return nil, err
	}
	ws.RW = readerWriter
	return conn, err
}

func (ws *Websocket) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := ws.Parse(r)
	if err != nil {
		log.Println(err)
		panic(err)
	}

	conn, err := ws.Hijack(w)
	if err != nil {
		log.Println(err)
		panic(err)
	}
	defer conn.Close()

	ws.Accept()
	for {
		select {
		case <-ws.context.Done():
		default:
			frame1, err := ws.RW.ReadByte()
			if err != nil {
				if err == io.EOF {
					return
				}
				panic(err)
			}

			log.Println("frame1", frame1)

			opcode := GetOpcode(frame1)
			switch opcode {
			case Close:
				log.Println("closing and returning")
				return
			case Ping:
				// TODO: Send Pong
			case Text, Binary:
				ws.HandleIncomingBinary()
			}

			incomming, err := ws.ReadData()
			if err != nil {
				panic(err)
			}

			log.Println(incomming)
			resType, result, err := ws.Handler(incomming)
			if resType != Text && resType != Binary {
				panic("Unknown response type")
			}

			if err != nil {
				panic(err)
			}

			switch resType {
			case Binary:
				ws.SendHeader(true, Binary)
			case Text:
				ws.SendHeader(true, Text)
			}
			ws.SendData(result)
		}
	}
}

func (ws *Websocket) HandleIncomingBinary() error {
	return nil
}

func (ws *Websocket) SendHeader(finished bool, op Opcode) {
	frame1 := uint8(0)
	if finished {
		frame1 |= uint8(1 << 7)
	}

	frame1 |= uint8(op)
	ws.RW.WriteByte(frame1)
}

func (ws *Websocket) SendData(message []byte) error {
	length := uint64(len(message))
	// NOTE: the length is send in the network order (Big endian)
	// NOTE: Server must not mask any frame it sends to the client
	if length < 126 {
		ws.RW.WriteByte(uint8(length))
	} else if length < 65536 {
		ws.RW.WriteByte(126)
		ws.RW.WriteByte(uint8(length >> 8))
		ws.RW.WriteByte(uint8(length))
	} else {
		ws.RW.WriteByte(127)
		ws.RW.WriteByte(uint8(length >> 24))
		ws.RW.WriteByte(uint8(length >> 16))
		ws.RW.WriteByte(uint8(length >> 8))
		ws.RW.WriteByte(uint8(length))
	}

	ws.RW.Write(message)
	return ws.RW.Flush()
}

func (ws *Websocket) ReadData() ([]byte, error) {
	var size uint64
	ib, err := ws.RW.ReadByte()
	if err != nil {
		return nil, err
	}

	// TODO: Handle FIN bit

	b := uint8(ib)
	mask := b >> 7

	b &= ((1 << 7) - 1) // Seven bit set
	switch b {
	case 126:
		b1, err := ws.RW.ReadByte()
		if err != nil {
			return nil, err
		}
		b2, err := ws.RW.ReadByte()
		if err != nil {
			return nil, err
		}

		size = uint64(uint64(b1)<<8 + uint64(b2))
		// TODO: Test the size matches or not
	case 127:
		for i := 0; i < 8; i++ {
			bi, err := ws.RW.ReadByte()
			if err != nil {
				return nil, err
			}
			b := uint64(bi)
			size <<= 8
			size += b
		}

	default:
		size = uint64(b)
	}

	log.Println("size", size)
	maskKey := [4]byte{}

	if mask == 1 {
		for i := 0; i < 4; i++ {
			bi, err := ws.RW.ReadByte()
			if err != nil {
				return nil, err
			}
			maskKey[i] = bi
		}
	}

	data := make([]byte, size)
	for i := 0; i < int(size); i++ {
		b, err := ws.RW.ReadByte()
		if err != nil {
			return nil, err
		}
		data[i] += maskKey[i%4] ^ b
	}

	return data, nil
}

func (ws *Websocket) Parse(r *http.Request) error {
	header := r.Header

	if header.Get("Connection") != "Upgrade" {
		return errors.New("Unknown connection type")
	}

	if header.Get("Upgrade") != "websocket" {
		return errors.New("Unknown upgrade type")
	}

	extentionsString := header.Get("Sec-WebSocket-Extensions")
	var extenstionList []string
	if extentionsString != "" {
		extenstionList = strings.Split(extentionsString, ";")
	}

	wsKey := header.Get("Sec-WebSocket-Key")
	if wsKey == "" {
		return errors.New("Unknown websocket key")
	}

	ws.context = r.Context()
	ws.Origin = header.Get("Origin")
	ws.Cache = header.Get("Cache-Control")
	ws.UserAgent = header.Get("User-Agent")
	ws.WebsocketKey = wsKey
	ws.WebsocketExtensions = extenstionList

	return nil
}

func (ws *Websocket) Accept() error {
	// Defined in the standard
	keyGUID := []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")
	h := sha1.New()
	h.Write([]byte(ws.WebsocketKey))
	h.Write([]byte(keyGUID))
	ws.WebsocketKey = base64.StdEncoding.EncodeToString(h.Sum(nil))

	ws.RW.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	ws.RW.WriteString("Upgrade: websocket\r\n")
	ws.RW.WriteString("Connection: Upgrade\r\n")
	ws.RW.WriteString("Access-Control-Allow-Origin: *\r\n")
	ws.RW.WriteString("Sec-WebSocket-Accept: ")
	ws.RW.WriteString(ws.WebsocketKey)
	ws.RW.WriteString("\r\n\r\n")
	return ws.RW.Flush()
}

func NewWebsocketHandler(fn func([]byte) (Opcode, []byte, error)) http.Handler {
	return &Websocket{
		Handler: fn,
	}
}
