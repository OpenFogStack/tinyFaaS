package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
)

func main() {
	port := ":8000"

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
			data, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, err)
				return
			}
			cmd := exec.Command("./fn.sh")
			cmd.Stdin = bytes.NewReader(data)
			output, err := cmd.CombinedOutput()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write(output)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})

	log.Printf("Server listening on port %s\n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
