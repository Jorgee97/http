package tcp

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strconv"
	"bufio"
)

type Request struct {
	startLine []byte
	headers []byte
	body []byte
}

var (
	maxHeaderSize = 20 << 20

)

func Listen(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	fmt.Printf("Server listening on port: %d...\n", port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept error: ", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(c net.Conn) {
	defer c.Close()

	var buf []byte

	// Minimal state machine
	var (
		headerParsed   bool
		contentLength  int
		bodyStartIndex int
	)
	request := Request{
		startLine: nil,
	}

	for {
		tb := make([]byte, 1024)
		n, err := c.Read(tb)
		if err != nil {
			log.Println(err)
			return
		}

		buf = append(buf, tb[:n]...)

		if !headerParsed && len(buf) > maxHeaderSize {
			_ = writeResponse(c, 431, "Request Header Fields Too Large", nil)
			return
		}

		if  !headerParsed {
			headerEnd := bytes.Index(buf, []byte("\r\n\r\n"))
			if headerEnd != -1 {
				headerParsed = true
				bodyStartIndex = headerEnd + 4

				startLineEnd := bytes.Index(buf[:headerEnd], []byte("\r\n"))
				if startLineEnd == -1 {
					_ = writeResponse(c, 400, "Bad Request", nil)
					return
				}
				request.startLine = buf[:startLineEnd]

				headerPart := buf[startLineEnd+2:headerEnd]
				request.headers = headerPart
				for _, l := range bytes.Split(headerPart, []byte("\r\n")) {
					line := bytes.SplitN(l, []byte(":"), 2)
					if len(line) != 2 {
						_ = writeResponse(c, 400, "Bad Request", nil)
						return
					}
					k, v := bytes.TrimSpace(line[0]), bytes.TrimSpace(line[1])

					if  bytes.EqualFold(k, []byte("Content-Length")) {
						contentLength, _ = strconv.Atoi(string(bytes.TrimSpace(v)))
					}
				}
			}

		}

		if headerParsed {
			bodyBytes := len(buf) - bodyStartIndex
			if bodyBytes >= contentLength {
				break
			}

		}
	}

	if contentLength > 0 {
		request.body = buf[bodyStartIndex : bodyStartIndex+contentLength]
	}

	fmt.Println("Request: ", string(request.startLine))
	fmt.Println("Headers: ", string(request.headers))
	fmt.Println("Body: ", string(request.body))

	_ = writeResponse(c, 200, "OK", []byte("Hello from MeServer\n"))
}

func writeResponse(c net.Conn, statusCode int, statusText string, body []byte) error {
	if body == nil {
		body = make([]byte, 0)
	}
	
	w := bufio.NewWriter(c)
	fmt.Fprintf(w, "HTTP/1.1 %d %s\r\n", statusCode, statusText)
	w.WriteString("Connection: close\r\n")
	fmt.Fprintf(w, "Content-Length: %d\r\n", len(body))
	w.WriteString("\r\n")
	
	if _, err := w.Write(body); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}

	return nil
}
