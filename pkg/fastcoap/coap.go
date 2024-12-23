package fastcoap

import (
	"bytes"
	"log"

	"github.com/OpenFogStack/tinyFaaS/pkg/rproxy"
	coap "github.com/plgd-dev/go-coap/v3"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/mux"
)

const async = false

func Start(r *rproxy.RProxy, listenAddr string) {

	h := mux.HandlerFunc(
		func(w mux.ResponseWriter, m *mux.Message) {

			p, err := m.Path()

			if err != nil {
				log.Printf("error getting path: %v", err)
				return
			}

			for p != "" && p[0] == '/' {
				p = p[1:]
			}

			payload, err := m.ReadBody()

			if err != nil {
				log.Printf("error reading body: %v", err)
				return
			}

			s, res := r.Call(p, payload, async, nil)

			switch s {
			case rproxy.StatusOK:
				err = w.SetResponse(codes.Content, message.TextPlain, bytes.NewReader(res))
			case rproxy.StatusAccepted:
				err = w.SetResponse(codes.Created, message.TextPlain, bytes.NewReader(res))
			case rproxy.StatusNotFound:
				err = w.SetResponse(codes.NotFound, message.TextPlain, bytes.NewReader(res))
			case rproxy.StatusError:
				err = w.SetResponse(codes.InternalServerError, message.TextPlain, bytes.NewReader(res))
			}

			if err != nil {
				log.Printf("error setting response: %v", err)
				return
			}

		})

	log.Printf("Starting fast CoAP server on %s", listenAddr)

	coap.ListenAndServe("udp", listenAddr, h)
}
