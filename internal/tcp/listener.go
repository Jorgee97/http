package tcp

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
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

	for {
		tb := make([]byte, 1024)
		n, err := c.Read(tb)
		if err != nil {
			log.Println(err)
		}

		buf = append(buf, tb[:n]...)

		if !headerParsed {
			if i := bytes.Index(buf, []byte("\r\n\r\n")); i != -1 {
				headerParsed = true
				bodyStartIndex = i + 4

				headerPart := string(buf[:i])
				lines := strings.Split(headerPart, "\r\n")

				// lines[0] is the startLine
				// Then comes the headers part of the request
				// I guess we can or want to do this part without using strings
				for _, l := range lines[1:] {
					parts := strings.SplitN(l, ":", 2)
					if len(parts) == 2 && strings.TrimSpace(parts[0]) == "Content-Length" {
						contentLength, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
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

	headerEnd := bodyStartIndex - 4
	headerText := string(buf[:headerEnd])
	body := buf[bodyStartIndex : bodyStartIndex+contentLength]

	fmt.Println("Headers: ", headerText)
	fmt.Println("Body: ", body)
}
