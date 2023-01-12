package main

import (
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"

	"github.com/pfandzelter/go-coap"
)

func startCoAPServer(f *functions) {

	h := coap.FuncHandler(
		func(l *net.UDPConn, a *net.UDPAddr, m *coap.Message) *coap.Message {

			log.Printf("have request: %+v", m)
			log.Printf("is confirmable: %v", m.IsConfirmable())
			log.Printf("path: %s", m.PathString())

			f.RLock()
			defer f.RUnlock()

			p := m.PathString()

			for p != "" && p[0] == '/' {
				p = p[1:]
			}

			handler, ok := f.hosts[p]

			if !ok {
				log.Printf("Function not found: %s", p)
				return &coap.Message{
					Code: coap.NotFound,
					Type: coap.Acknowledgement,
				}
			}

			// call function and return results
			resp, err := http.Get("http://" + handler[rand.Intn(len(handler))] + ":8000/fn")

			if err != nil {
				return &coap.Message{
					Type: coap.Acknowledgement,
					Code: coap.InternalServerError,
				}
			}

			body, err := ioutil.ReadAll(resp.Body)

			if err != nil {
				log.Printf("Error reading body: %s", err)
				return &coap.Message{
					Type: coap.Acknowledgement,
					Code: coap.InternalServerError,
				}
			}

			res := &coap.Message{
				Type:      coap.Acknowledgement,
				Code:      coap.Content,
				MessageID: m.MessageID,
				Token:     m.Token,
				Payload:   []byte(body),
			}

			res.SetOption(coap.ContentFormat, coap.TextPlain)

			log.Printf("response: %+v", res)

			return res
		})

	coap.ListenAndServe("udp", ":6000", h)
}
