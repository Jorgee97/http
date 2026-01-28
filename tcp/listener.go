package tcp

import (
	"fmt"
	"net"
	"log"
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

	buf := make([]byte, 1024)
	n, err := c.Read(buf)
	if err != nil {
		log.Println("read error:", err)
		return
	}
	fmt.Printf("Read %d bytes:\n%s\n", n, buf[:n])
}
