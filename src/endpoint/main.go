package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

var functions map[string][]string

type functionInfo struct {
	FunctionResource   string   `json:"function_resource"`
	FunctionContainers []string `json:"function_containers"`
}

func main() {
	functions = make(map[string][]string)

	go func() {
		server := http.NewServeMux()

		server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				buf := new(bytes.Buffer)
				buf.ReadFrom(r.Body)
				newStr := buf.String()

				var f functionInfo
				err := json.Unmarshal([]byte(newStr), &f)

				if err != nil {
					return
				}

				if f.FunctionResource[0] == '/' {
					f.FunctionResource = f.FunctionResource[1:]
				}

				if len(f.FunctionContainers) > 0 {
					functions[f.FunctionResource] = f.FunctionContainers
				} else {
					_, ok := functions[f.FunctionResource]
					if ok {
						delete(functions, f.FunctionResource)
					}
				}

				return

			}
		})

		http.ListenAndServe(":80", server)
	}()

	func() {
		server := http.NewServeMux()

		director := func(req *http.Request) {
			handler := ""
			for path := range functions {

				if strings.HasPrefix(req.URL.Path, "/"+path) {
					handler = path
				}
			}

			urls, ok := functions[handler]
			if ok {
				dest, _ := url.Parse("http://" + urls[rand.Intn(len(urls))] + ":8000")
				req.URL.Host = dest.Host
				req.URL.Path = strings.Replace(req.URL.Path, "/"+handler, "", 1)
			}
			req.URL.Scheme = "http"
		}
		proxy := &httputil.ReverseProxy{Director: director}
		server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			proxy.ServeHTTP(w, r)
		})

		err := http.ListenAndServe(":5683", server)
		fmt.Println(err)
	}()

}
