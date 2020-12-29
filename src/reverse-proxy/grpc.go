package main

import (
	"context"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"

	"github.com/OpenFogStack/tinyFaaS/reverse-proxy/api"
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

	if ok {
		// call function and return results
		resp, err := http.Get("http://" + handler[rand.Intn(len(handler))] + ":8000")

		if err != nil {
			return nil, status.Errorf(codes.Unavailable,
				"Invalid response from function handler")
		}

		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return nil, status.Errorf(codes.Unavailable,
				"Invalid response from function handler")

		}

		return &api.Response{
			Response: string(body),
		}, nil
	}

	return nil, status.Errorf(codes.NotFound,
		"No such function")

}

func startGRPCServer(f *functions) {
	s := grpc.NewServer()

	api.RegisterTinyFaaSServer(s, &GRPCServer{
		f: f,
	})

	lis, err := net.Listen("tcp", ":8000")

	if err != nil {
		log.Fatal("Failed to listen")
	}

	defer s.GracefulStop()
	s.Serve(lis)
}
