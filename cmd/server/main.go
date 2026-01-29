package main

import (
	"log"
	"github.com/Jorgee97/http/internal/tcp"
)

func main() {
	port := 8000
	err := tcp.Listen(port)
	if err != nil {
		log.Fatal(err)
	}


}
