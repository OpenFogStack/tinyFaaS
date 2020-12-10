package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func startHTTPServer(f *functions) {
	server := http.NewServeMux()

	director := func(req *http.Request) {
		handler := ""

		f.RLock()
		defer f.RUnlock()

		for path := range f.hosts {

			if strings.HasPrefix(req.URL.Path, "/"+path) && len(handler) <= len(path) {
				handler = path
			}
		}

		urls, ok := f.hosts[handler]
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

	err := http.ListenAndServe(":7000", server)
	fmt.Println(err)
}
