package main

import (
	"io/ioutil"
	"math/rand"
	"net/http"
)

func startHTTPServer(f *functions) {

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		f.RLock()
		defer f.RUnlock()

		p := r.URL.Path

		for p != "" && p[0] == '/' {
			p = p[1:]
		}

		handler, ok := f.hosts[p]

		if ok {
			// call function and return results
			resp, err := http.Get("http://" + handler[rand.Intn(len(handler))] + ":8000")

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			body, err := ioutil.ReadAll(resp.Body)

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Write(body)
		}
	})

	http.ListenAndServe(":7000", mux)
}
