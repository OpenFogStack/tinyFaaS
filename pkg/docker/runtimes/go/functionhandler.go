package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Path == "/health" {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "OK")
				log.Println("reporting health: OK")
				return
			}
			w.WriteHeader(http.StatusNotFound)
			return

		case http.MethodPost:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, err)
				return
			}

			headers := make(map[string]string)
			for k, v := range r.Header {
				headers[k] = v[0]
			}

			result, err := fn(string(body), headers) // returns string, err
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, err = w.Write([]byte(result))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, err)
			}

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})

	log.Printf("Server starting on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
