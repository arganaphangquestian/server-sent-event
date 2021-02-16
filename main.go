package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SSEChannel struct
type SSEChannel struct {
	Clients  []chan string
	Notifier chan string
}

type body struct {
	Message string `json:"message"`
}

var sseChannel SSEChannel

func main() {
	sseChannel = SSEChannel{
		Clients:  make([]chan string, 0),
		Notifier: make(chan string),
	}
	done := make(chan interface{})
	defer close(done)
	go broadcaster(done)
	http.HandleFunc("/sse", handleSSE)
	http.HandleFunc("/send", sendMessage)

	fmt.Println("Running at http://127.0.0.1:8080")
	http.ListenAndServe(":8080", nil)
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Connection doesnot support streaming", http.StatusBadRequest)
		return
	}

	sseChan := make(chan string)
	sseChannel.Clients = append(sseChannel.Clients, sseChan)

	d := make(chan interface{})
	defer close(d)
	defer fmt.Println("Closing channel.")

	for {
		select {
		case <-d:
			close(sseChan)
			return
		case data := <-sseChan:
			fmt.Fprintf(w, "data: %v \n\n", data)
			flusher.Flush()
		}
	}
}

func sendMessage(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var res body
	err := decoder.Decode(&res)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"data":    nil,
		})
	}

	sseChannel.Notifier <- res.Message

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    res.Message,
	})
}

func broadcaster(done <-chan interface{}) {
	for {
		select {
		case <-done:
			return
		case data := <-sseChannel.Notifier:
			for _, channel := range sseChannel.Clients {
				channel <- data
			}
		}
	}
}
