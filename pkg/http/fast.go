package http

import (
	"log"
	"net/http"

	"github.com/OpenFogStack/tinyFaaS/pkg/rproxy"
	"github.com/valyala/fasthttp"
)

type handler struct {
	r *rproxy.RProxy
}

func (h *handler) ServeHTTP(ctx *fasthttp.RequestCtx) {
	p := string(ctx.Path())

	for p != "" && p[0] == '/' {
		p = p[1:]
	}

	async := string(ctx.Request.Header.Peek("X-tinyFaaS-Async")) != ""

	// log.Printf("have request for path: %s (async: %v)", p, async)

	req_body := ctx.PostBody()

	// headers := make(map[string]string)
	// ctx.Request.Header.VisitAll(func(k, v []byte) {
	// 	headers[string(k)] = string(v)
	// })

	s, res := h.r.Call(p, req_body, async, nil)
	// ctx.SetStatusCode(http.StatusOK)
	// return

	switch s {
	case rproxy.StatusOK:
		ctx.SetStatusCode(http.StatusOK)
		ctx.Write(res)
	case rproxy.StatusAccepted:
		ctx.SetStatusCode(http.StatusAccepted)
	case rproxy.StatusNotFound:
		ctx.SetStatusCode(http.StatusNotFound)
	case rproxy.StatusError:
		ctx.SetStatusCode(http.StatusInternalServerError)
	}
}

func StartFastHTTP(r *rproxy.RProxy, listenAddr string) {

	h := &handler{r: r}

	log.Printf("Starting fasthttp server on %s", listenAddr)

	s := fasthttp.Server{
		Handler:     h.ServeHTTP,
		Concurrency: 256 * 1024,
	}

	// err := fasthttp.ListenAndServe(listenAddr, h.ServeHTTP)
	err := s.ListenAndServe(listenAddr)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("fasthttp server stopped")
}
