package main

import (
	"context"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"

	"github.com/pfandzelter/tinyFaaS/cmd/rproxy/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCServer is the grpc endpoint for this tinyFaaS instance.
type GRPCServer struct {
	f *functions
}

// Request handles a request to the GRPC endpoint of the reverse-proxy of this tinyFaaS instance.
func (s *GRPCServer) Request(ctx context.Context, d *api.Data) (*api.Response, error) {

	s.f.RLock()
	defer s.f.RUnlock()

	handler, ok := s.f.hosts[d.FunctionIdentifier]

	if !ok {
		log.Printf("Function not found: %s", d.FunctionIdentifier)
		return nil, status.Errorf(codes.NotFound,
			"No such function")
	}

	req_body := d.Data

	// call function and return results
	resp, err := http.Post("http://"+handler[rand.Intn(len(handler))]+":8000/fn", "application/binary", strings.NewReader(req_body))

	if err != nil {
		log.Printf("Error calling function: %s", err)
		return nil, status.Errorf(codes.Unavailable,
			"Invalid response from function handler")
	}

	defer resp.Body.Close()
	res_body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Error reading response from function: %s", err)
		return nil, status.Errorf(codes.Unavailable,
			"Invalid response from function handler")

	}

	log.Printf("have response: %s", res_body)

	return &api.Response{
		Response: string(res_body),
	}, nil
}

func startGRPCServer(f *functions, listenAddr string) {
	s := grpc.NewServer()

	api.RegisterTinyFaaSServer(s, &GRPCServer{
		f: f,
	})

	lis, err := net.Listen("tcp", listenAddr)

	if err != nil {
		log.Fatal("Failed to listen")
	}

	defer s.GracefulStop()
	s.Serve(lis)
}
