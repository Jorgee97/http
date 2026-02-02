package tcp

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strconv"
)

type Request struct {
	startLine []byte
	headers []byte
	body []byte
}

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
		}

		buf = append(buf, tb[:n]...)

		if !headerParsed {
			headerEnd := bytes.Index(buf, []byte("\r\n\r\n"))
			if headerEnd != -1 {
				headerParsed = true
				bodyStartIndex = headerEnd + 4

				startLineEnd := bytes.Index(buf[:headerEnd], []byte("\r\n"))
				if startLineEnd == -1 {
					// TODO: if this even happend this request my be broken
					c.Write([]byte("HTTP/1.1 400 Bad Request\r\nServer: MeServer"))
					c.Close()
					continue
				}
				request.startLine = buf[:startLineEnd]

				headerPart := buf[startLineEnd+2:headerEnd]
				request.headers = headerPart
				for _, l := range bytes.Split(headerPart, []byte("\r\n")) {
					line := bytes.SplitN(l, []byte(":"), 2)
					k, v := line[0], line[1]

					if  bytes.Equal(k, []byte("Content-Length")) {
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

	request.body = buf[bodyStartIndex : bodyStartIndex+contentLength]

	fmt.Println("Request: ", string(request.startLine))
	fmt.Println("Headers: ", string(request.headers))
	fmt.Println("Body: ", string(request.body))

}
