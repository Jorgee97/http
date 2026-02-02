package tcp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
)

type Request struct {
	startLine StartLine
	headers   []byte
	body      []byte
}

type StartLine struct {
	Protocol      string
	Method        string
	RequestTarget string
}

type CloseConn bool

var (
	maxHeaderSize = 20 << 20
)

var ErrWrongStartLineFormat = errors.New("StartLine doesn't have 3 parts")

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

	closeCon := false

	for closeCon == false {
		closeCon = handleRequest(c)
	}

}

// Return true represents that we need to close the connection
func handleRequest(c net.Conn) bool {
	var buf []byte

	// Minimal state machine
	var (
		headerParsed   bool
		contentLength  int
		bodyStartIndex int
	)
	request := Request{
		startLine: StartLine{},
	}

	for {
		tb := make([]byte, 1024)
		n, err := c.Read(tb)
		if err != nil {
			log.Println(err)
			return true
		}

		buf = append(buf, tb[:n]...)

		if !headerParsed && len(buf) > maxHeaderSize {
			_ = writeResponse(c, 431, "Request Header Fields Too Large", nil)
			return true
		}

		if !headerParsed {
			headerEnd := bytes.Index(buf, []byte("\r\n\r\n"))
			if headerEnd != -1 {
				headerParsed = true
				bodyStartIndex = headerEnd + 4

				startLineEnd := bytes.Index(buf[:headerEnd], []byte("\r\n"))
				if startLineEnd == -1 {
					_ = writeResponse(c, 400, "Bad Request", nil)
					return true
				}
				sl, err := extractRequestStartLine(buf[:startLineEnd])
				if err != nil {
					_ = writeResponse(c, 400, "Bad Request", nil)
					return true
				}
				request.startLine = sl

				headerPart := buf[startLineEnd+2 : headerEnd]
				request.headers = headerPart
				for _, l := range bytes.Split(headerPart, []byte("\r\n")) {
					line := bytes.SplitN(l, []byte(":"), 2)
					if len(line) != 2 {
						_ = writeResponse(c, 400, "Bad Request", nil)
						return true
					}
					k, v := bytes.TrimSpace(line[0]), bytes.TrimSpace(line[1])

					if bytes.EqualFold(k, []byte("Content-Length")) {
						contentLength, _ = strconv.Atoi(string(bytes.TrimSpace(v)))
					}

					if bytes.EqualFold(k, []byte("Connection")) && bytes.EqualFold(v, []byte("close")) {
						return true
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

	fmt.Println("Request: ", request.startLine)
	fmt.Println("Headers: ", string(request.headers))
	fmt.Println("Body: ", string(request.body))

	_ = writeResponse(c, 200, "OK", []byte("Hello from MeServer\n"))

	if request.startLine.Protocol == "HTTP/1.0" {
		return true
	}

	return false

}

func extractRequestStartLine(sl []byte) (StartLine, error) {
	parts := bytes.SplitN(sl, []byte(" "), 3)
	if len(parts) != 3 {
		return StartLine{}, ErrWrongStartLineFormat
	}
	return StartLine{
		Method:        string(bytes.TrimSpace(parts[0])),
		RequestTarget: string(bytes.TrimSpace(parts[1])),
		Protocol:      string(bytes.TrimSpace(parts[2])),
	}, nil
}

func writeResponse(c net.Conn, statusCode int, statusText string, body []byte) error {
	if body == nil {
		body = make([]byte, 0)
	}

	w := bufio.NewWriter(c)
	fmt.Fprintf(w, "HTTP/1.1 %d %s\r\n", statusCode, statusText)
	w.WriteString("Connection: keep-alive\r\n")
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
