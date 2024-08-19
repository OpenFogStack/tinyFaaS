package http

import (
	"io"
	"log"
	"net/http"

	"github.com/OpenFogStack/tinyFaaS/pkg/rproxy"
)

func Start(r *rproxy.RProxy, listenAddr string) {

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		p := req.URL.Path

		for p != "" && p[0] == '/' {
			p = p[1:]
		}

		async := req.Header.Get("X-tinyFaaS-Async") != ""

		log.Printf("have request for path: %s (async: %v)", p, async)

		req_body, err := io.ReadAll(req.Body)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Print(err)
			return
		}

		headers := make(map[string]string)
		for k, v := range req.Header {
			headers[k] = v[0]
		}

		s, res := r.Call(p, req_body, async, headers)

		switch s {
		case rproxy.StatusOK:
			w.WriteHeader(http.StatusOK)
			w.Write(res)
		case rproxy.StatusAccepted:
			w.WriteHeader(http.StatusAccepted)
		case rproxy.StatusNotFound:
			w.WriteHeader(http.StatusNotFound)
		case rproxy.StatusError:
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	log.Printf("Starting HTTP server on %s", listenAddr)
	err := http.ListenAndServe(listenAddr, mux)

	if err != nil {
		log.Fatal(err)
	}

	log.Print("HTTP server stopped")

}
