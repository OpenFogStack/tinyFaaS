package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"

	"github.com/OpenFogStack/tinyFaaS/pkg/coap"
	"github.com/OpenFogStack/tinyFaaS/pkg/fastcoap"
	"github.com/OpenFogStack/tinyFaaS/pkg/grpc"
	tfhttp "github.com/OpenFogStack/tinyFaaS/pkg/http"
	"github.com/OpenFogStack/tinyFaaS/pkg/rproxy"
)

func main() {
	// add cpu profile
	f, e := os.Create("cpu.prof")

	if e != nil {
		log.Fatalf("could not create cpu profile: %s", e)
	}

	defer f.Close()

	e = pprof.StartCPUProfile(f)

	if e != nil {
		log.Fatalf("could not start cpu profile: %s", e)
	}

	defer pprof.StopCPUProfile()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("rproxy: ")

	if len(os.Args) <= 3 {
		fmt.Println("Usage: ./rproxy <listen-addr> [<protocol>:<listen-addr>]")
		os.Exit(1)
	}

	rproxyListenAddress := os.Args[1]

	listenAddrs := make(map[string]string)

	for _, arg := range os.Args[2:] {
		prot, listenAddr, ok := strings.Cut(arg, ":")

		if !ok {
			fmt.Println("Usage: ./rproxy <listen-addr> <protocol>:<listen-addr>")
			os.Exit(1)
		}

		prot = strings.ToLower(prot)
		listenAddr = strings.ToLower(listenAddr)

		log.Printf("adding %s listener on %s", prot, listenAddr)
		listenAddrs[prot] = listenAddr
	}

	if len(listenAddrs) == 0 {
		return // nothing to do
	}

	r := rproxy.New()

	// CoAP
	if listenAddr, ok := listenAddrs["coap"]; ok {
		log.Printf("starting coap server on %s", listenAddr)
		go coap.Start(r, listenAddr)
	}

	// Fast CoAP
	if listenAddr, ok := listenAddrs["fastcoap"]; ok {
		log.Printf("starting coap server on %s", listenAddr)
		go fastcoap.Start(r, listenAddr)
	}

	// HTTP
	if listenAddr, ok := listenAddrs["http"]; ok {
		log.Printf("starting http server on %s", listenAddr)
		go tfhttp.Start(r, listenAddr)
	}
	// GRPC
	if listenAddr, ok := listenAddrs["grpc"]; ok {
		log.Printf("starting grpc server on %s", listenAddr)
		go grpc.Start(r, listenAddr)
	}
	// FastHTTP
	if listenAddr, ok := listenAddrs["fasthttp"]; ok {
		log.Printf("starting fasthttp server on %s", listenAddr)
		go tfhttp.StartFastHTTP(r, listenAddr)
	}

	server := http.NewServeMux()

	server.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		log.Printf("have request: %+v", req)

		buf := new(bytes.Buffer)
		buf.ReadFrom(req.Body)
		newStr := buf.String()

		log.Printf("have body: %s", newStr)

		var def struct {
			FunctionResource   string   `json:"name"`
			FunctionContainers []string `json:"ips"`
		}

		err := json.Unmarshal([]byte(newStr), &def)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Printf("have definition: %+v", def)

		if def.FunctionResource[0] == '/' {
			def.FunctionResource = def.FunctionResource[1:]
		}

		if len(def.FunctionContainers) > 0 {
			// "ips" field not empty: add function
			log.Printf("adding %s", def.FunctionResource)
			err = r.Add(def.FunctionResource, def.FunctionContainers)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		} else {

			log.Printf("deleting %s", def.FunctionResource)
			err = r.Del(def.FunctionResource)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return

			}
		}
	})

	go func() {
		log.Printf("listening on %s", rproxyListenAddress)
		err := http.ListenAndServe(rproxyListenAddress, server)

		if err != nil {
			log.Printf("%s", err)
		}
	}()

	s := make(chan os.Signal, 1)

	signal.Notify(s, os.Interrupt)

	<-s

	log.Printf("exiting")
	return
}
