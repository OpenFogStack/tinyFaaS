package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
)

type functions struct {
	hosts map[string][]string
	sync.RWMutex
}

func main() {
	f := functions{
		hosts: make(map[string][]string),
	}

	// CoAP
	go startCoAPServer(&f)
	// HTTP
	go startCoAPServer(&f)
	// GRPC
	go startGRPCServer(&f)

	server := http.NewServeMux()

	server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {

			return
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		newStr := buf.String()

		var def struct {
			FunctionResource   string   `json:"function_resource"`
			FunctionContainers []string `json:"function_containers"`
		}

		err := json.Unmarshal([]byte(newStr), &f)

		if err != nil {
			return
		}

		if def.FunctionResource[0] == '/' {
			def.FunctionResource = def.FunctionResource[1:]
		}

		f.Lock()
		defer f.Unlock()
		if len(def.FunctionContainers) > 0 {
			f.hosts[def.FunctionResource] = def.FunctionContainers
		} else {
			_, ok := f.hosts[def.FunctionResource]
			if ok {
				delete(f.hosts, def.FunctionResource)
			}
		}

	})

	http.ListenAndServe(":80", server)

}
