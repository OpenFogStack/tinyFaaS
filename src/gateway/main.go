package main

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
  "bytes"

	"github.com/dustin/go-coap"
)

var functions map[string][]string

type function_info struct {
	Function_resource   string `json:"function_resource"`
	Function_containers []string `json:"function_containers"`
}

func handleFunctionCall(l *net.UDPConn, a *net.UDPAddr, m *coap.Message) *coap.Message {

	if m.IsConfirmable() {

		handler, ok := functions[m.PathString()]

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
		} else {
			return &coap.Message{
				Type: coap.Acknowledgement,
				Code: coap.NotFound,
			}
		}

	}

	return nil
}

func main() {
  functions = make(map[string][]string)

  mux := coap.NewServeMux()
  mux.Handle("/functions", coap.FuncHandler(handleFunctionCall))

  go func() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
      if r.Method == "POST" {
        buf := new(bytes.Buffer)
        buf.ReadFrom(r.Body)
        newStr := buf.String()

        var f function_info
        err := json.Unmarshal([]byte(newStr), &f)

        if err != nil {
          return
        }

        if f.Function_resource[0] == '/' {
          f.Function_resource = f.Function_resource[1:]
        }

        functions[f.Function_resource] = f.Function_containers

        mux.Handle(f.Function_resource, coap.FuncHandler(handleFunctionCall))

        return

      }
    })

    http.ListenAndServe(":80", nil)
  }()

  func() {
    coap.ListenAndServe("udp", ":5683", mux)
  }()

}
