package main

import (
	"bytes"
	"io/ioutil"
	"log"
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

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			log.Printf("Function not found: %s", p)
			return
		}

		async := r.Header.Get("X-tinyFaaS-Async") != ""

		req_body, err := ioutil.ReadAll(r.Body)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Print(err)
			return
		}

		if async {
			w.WriteHeader(http.StatusAccepted)
			go func() {
				resp, err := http.Post("http://"+handler[rand.Intn(len(handler))]+":8000/fn", "application/binary", bytes.NewBuffer(req_body))

				if err != nil {
					return
				}

				resp.Body.Close()
			}()
			return
		}

		// call function and return results
		resp, err := http.Post("http://"+handler[rand.Intn(len(handler))]+":8000/fn", "application/binary", bytes.NewBuffer(req_body))

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Print(err)
			return
		}

		defer resp.Body.Close()
		res_body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Print(err)
			return
		}

		w.Write(res_body)

	})

	http.ListenAndServe(":7000", mux)
}
