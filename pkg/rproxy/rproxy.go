package rproxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
)

type Status uint32

const (
	StatusOK Status = iota
	StatusAccepted
	StatusNotFound
	StatusError
)

type RProxy struct {
	hosts map[string][]string
	hl    sync.RWMutex
}

func New() *RProxy {
	return &RProxy{
		hosts: make(map[string][]string),
	}
}

func (r *RProxy) Add(name string, ips []string) error {
	if len(ips) == 0 {
		return fmt.Errorf("no ips given")
	}

	r.hl.Lock()
	defer r.hl.Unlock()

	if _, ok := r.hosts[name]; ok {
		return fmt.Errorf("function already exists")
	}

	r.hosts[name] = ips
	return nil
}

func (r *RProxy) Del(name string) error {
	r.hl.Lock()
	defer r.hl.Unlock()

	if _, ok := r.hosts[name]; !ok {
		return fmt.Errorf("function not found")
	}

	delete(r.hosts, name)
	return nil
}

func (r *RProxy) Call(name string, payload []byte, async bool) (Status, []byte) {

	handler, ok := r.hosts[name]

	if !ok {
		log.Printf("function not found: %s", name)
		return StatusNotFound, nil
	}

	log.Printf("have handler: %s", handler)

	if async {
		log.Printf("async request accepted")
		go func() {
			resp, err := http.Post("http://"+handler[rand.Intn(len(handler))]+":8000/fn", "application/binary", bytes.NewBuffer(payload))

			if err != nil {
				return
			}

			resp.Body.Close()

			log.Printf("async request finished")
		}()
		return StatusAccepted, nil
	}

	// call function and return results
	log.Printf("sync request starting")
	resp, err := http.Post("http://"+handler[rand.Intn(len(handler))]+":8000/fn", "application/binary", bytes.NewBuffer(payload))

	if err != nil {
		log.Print(err)
		return StatusError, nil
	}

	log.Printf("sync request finished")

	defer resp.Body.Close()
	res_body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Print(err)
		return StatusError, nil
	}

	log.Printf("have response for sync request: %s", res_body)

	return StatusOK, res_body
}
