package tcp

import (
	"fmt"
	"log"
	"net"
	"strings"
)

func Listen(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	fmt.Printf("Server listening on port: %d...", port)
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

	// TODO: size of the buffer must not be static since we don't really know the size of the request
	buf := make([]byte, 1024)
	n, err := c.Read(buf)
	if err != nil {
		log.Println("read error:", err)
		return
	}
	fmt.Printf("Read %d bytes:\n%s\n", n, buf[:n])
	r := strings.Split(fmt.Sprintf("%s", buf[:n]), "\r\n\r\n")
	parts := strings.Split(r[0], "\r\n")

	startLine := parts[0]
	headers := parts[1:]
	body := r[1]

	fmt.Println("Parts: ", len(parts))
	fmt.Println("Start line:", startLine)
	fmt.Println("Headers: ", headers)
	fmt.Println("Body: ", body)
}
