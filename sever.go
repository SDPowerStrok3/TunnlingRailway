package main

import (
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/agent", agentHandler)

	go startPublicTCP(":25565")

	log.Println("Tunnel v2 server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// ================= TCP (Public) =================

func startPublicTCP(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Public TCP listening on", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handlePublicClient(conn)
	}
}

// ================= WebSocket per client =================

func handlePublicClient(client net.Conn) {
	defer client.Close()

	ws, _, err := websocket.DefaultDialer.Dial(
		"ws://localhost:8080/agent",
		nil,
	)
	if err != nil {
		log.Println("Agent WS dial failed:", err)
		return
	}
	defer ws.Close()

	done := make(chan struct{})

	// TCP -> WS
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := client.Read(buf)
			if err != nil {
				close(done)
				return
			}
			ws.WriteMessage(websocket.BinaryMessage, buf[:n])
		}
	}()

	// WS -> TCP
	go func() {
		for {
			_, data, err := ws.ReadMessage()
			if err != nil {
				close(done)
				return
			}
			client.Write(data)
		}
	}()

	<-done
}

// ================= Agent =================

func agentHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	log.Println("Agent WS connected")

	select {}
}
