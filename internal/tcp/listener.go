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

	buf := []byte{}
	closeCon := false
	for  {
		closeCon, buf = handleRequest(c, buf)
		if closeCon {
			return
		}
	}

}

// Return true represents that we need to close the connection
func handleRequest(c net.Conn, buf []byte) (bool, []byte) {

	// Minimal state machine
	var (
		headerParsed   bool
		contentLength  int
		bodyStartIndex int
	)
	request := Request{
		startLine: StartLine{},
	}

	shouldClose := false

	for {
		tb := make([]byte, maxHeaderSize)
		n, err := c.Read(tb)
		if err != nil {
			log.Println(err)
			return true, buf
		}

		buf = append(buf, tb[:n]...)

		if !headerParsed && len(buf) > maxHeaderSize {
			_ = writeResponse(c, 431, "Request Header Fields Too Large", true, nil)
			return true, buf
		}

		if !headerParsed {
			headerEnd := bytes.Index(buf, []byte("\r\n\r\n"))
			if headerEnd != -1 {
				headerParsed = true
				bodyStartIndex = headerEnd + 4

				startLineEnd := bytes.Index(buf[:headerEnd], []byte("\r\n"))
				if startLineEnd == -1 {
					_ = writeResponse(c, 400, "Bad Request", true, nil)
					return true, buf
				}
				sl, err := extractRequestStartLine(buf[:startLineEnd])
				if err != nil {
					_ = writeResponse(c, 400, "Bad Request", true, nil)
					return true, buf
				}
				request.startLine = sl

				if request.startLine.Protocol == "HTTP/1.0" {
					shouldClose = true
				}

				headerPart := buf[startLineEnd+2 : headerEnd]
				request.headers = headerPart
				for _, l := range bytes.Split(headerPart, []byte("\r\n")) {
					line := bytes.SplitN(l, []byte(":"), 2)
					if len(line) != 2 {
						_ = writeResponse(c, 400, "Bad Request", true, nil)
						return true, buf
					}
					k, v := bytes.TrimSpace(line[0]), bytes.TrimSpace(line[1])

					if bytes.EqualFold(k, []byte("Content-Length")) {
						contentLength, _ = strconv.Atoi(string(bytes.TrimSpace(v)))
					}

					if bytes.EqualFold(k, []byte("Connection")) {
						if bytes.EqualFold(v, []byte("close")) {
							shouldClose = true
						}

						if request.startLine.Protocol == "HTTP/1.0" && bytes.EqualFold(v, []byte("keep-alive")) {
							shouldClose = false
						}

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

	_ = writeResponse(c, 200, "OK", shouldClose, []byte("Hello from MeServer\n"))

	return shouldClose, buf

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

func writeResponse(c net.Conn, statusCode int, statusText string, closeCon bool, body []byte) error {
	if body == nil {
		body = make([]byte, 0)
	}

	w := bufio.NewWriter(c)
	fmt.Fprintf(w, "HTTP/1.1 %d %s\r\n", statusCode, statusText)
	if closeCon {
		w.WriteString("Connection: close\r\n")
	} else {
		w.WriteString("Connection: keep-alive\r\n")
	}
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
