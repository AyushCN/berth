package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

func main() {
	id := os.Args[1]
	token := os.Args[2]
	url := "ws://localhost:8080/ws/sandbox/" + id + "?token=" + token
	
	header := http.Header{}
	header.Add("Origin", "http://localhost:3000")
	
	c, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	err = c.WriteMessage(websocket.TextMessage, []byte("ls -la\r"))
	if err != nil {
		log.Println("write:", err)
		return
	}

	for i := 0; i < 5; i++ {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		log.Printf("recv: %s", string(message))
	}
}
