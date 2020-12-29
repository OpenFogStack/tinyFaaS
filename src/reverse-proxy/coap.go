package main

import (
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"

	"github.com/pfandzelter/go-coap"
)

func startCoAPServer(f *functions) {

	mux := coap.NewServeMux()

	mux.Handle("/", coap.FuncHandler(
		func(l *net.UDPConn, a *net.UDPAddr, m *coap.Message) *coap.Message {

			if m.IsConfirmable() {
				f.RLock()
				defer f.RUnlock()

				p := m.PathString()

				for p != "" && p[0] == '/' {
					p = p[1:]
				}

				handler, ok := f.hosts[p]

				if ok {
					// call function and return results
					resp, err := http.Get("http://" + handler[rand.Intn(len(handler))] + ":8000")

					if err != nil {
						return &coap.Message{
							Type: coap.Acknowledgement,
							Code: coap.InternalServerError,
						}
					}

					body, err := ioutil.ReadAll(resp.Body)

					if err != nil {
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

					return res
				}
				return &coap.Message{
					Type: coap.Acknowledgement,
					Code: coap.NotFound,
				}

			}

			return nil
		}))

	coap.ListenAndServe("udp", ":6000", mux)
}
