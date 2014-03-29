package web

import (
	"log"
	"net/http"
	"sync"

	"code.google.com/p/go.net/websocket"
)

type Message interface{}

type Hub struct {
	In   chan Message
	outs map[chan Message]bool
	sync.Mutex
}

func NewHub() *Hub {
	h := &Hub{outs: make(map[chan Message]bool)}
	h.In = make(chan Message)
	go func() {
		for m := range h.In {
			for out, _ := range h.outs {
				select {
				case out <- m:
				default:
					log.Println("could not broadcast tweet:", m)
				}
			}
		}
	}()

	return h
}

func (h *Hub) Add(out chan Message) {
	h.Lock()
	h.outs[out] = true
	h.Unlock()
}

func (h *Hub) Remove(out chan Message) {
	h.Lock()
	delete(h.outs, out)
	h.Unlock()
}

func (h *Hub) Handler() http.Handler {
	return websocket.Handler(func(ws *websocket.Conn) {
		in := make(chan Message)
		h.Add(in)
		go func() {
			for {
				var m Message
				if err := websocket.JSON.Receive(ws, &m); err == nil {
					//out <- m
				} else {
					log.Println("Message Websocket receive err:", err)
					return
				}
			}
		}()

		for m := range in {
			if err := websocket.JSON.Send(ws, &m); err != nil {
				log.Println("Message Websocket send err:", err)
				break
			}
		}
		h.Remove(in)
		ws.Close()
	})
}
