package main

import (
	"bytes"
	"io"
	"log"
	"math/rand"
	"net/http"
)

func startHTTPServer(f *functions, listenAddr string) {

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		f.RLock()
		defer f.RUnlock()

		p := r.URL.Path

		for p != "" && p[0] == '/' {
			p = p[1:]
		}

		log.Printf("have request for path: %s", p)

		handler, ok := f.hosts[p]

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			log.Printf("function not found: %s", p)
			return
		}

		log.Printf("have handler: %s", handler)

		async := r.Header.Get("X-tinyFaaS-Async") != ""

		log.Printf("is async: %v", async)

		req_body, err := io.ReadAll(r.Body)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Print(err)
			return
		}

		if async {
			log.Printf("async request accepted")
			w.WriteHeader(http.StatusAccepted)
			go func() {
				resp, err := http.Post("http://"+handler[rand.Intn(len(handler))]+":8000/fn", "application/binary", bytes.NewBuffer(req_body))

				if err != nil {
					return
				}

				resp.Body.Close()

				log.Printf("async request finished")
			}()
			return
		}

		// call function and return results
		log.Printf("sync request starting")
		resp, err := http.Post("http://"+handler[rand.Intn(len(handler))]+":8000/fn", "application/binary", bytes.NewBuffer(req_body))

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Print(err)
			return
		}

		log.Printf("sync request finished")

		defer resp.Body.Close()
		res_body, err := io.ReadAll(resp.Body)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Print(err)
			return
		}

		log.Printf("have response for sync request: %s", res_body)

		w.Write(res_body)
	})

	log.Printf("Starting HTTP server on %s", listenAddr)
	err := http.ListenAndServe(listenAddr, mux)

	if err != nil {
		log.Fatal(err)
	}

	log.Print("HTTP server stopped")

}
