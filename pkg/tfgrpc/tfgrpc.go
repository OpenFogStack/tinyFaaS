package tfgrpc

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/pfandzelter/tinyFaaS/pkg/rproxy"
	"github.com/pfandzelter/tinyFaaS/pkg/tfgrpc/api"
	"google.golang.org/grpc"
)

// GRPCServer is the grpc endpoint for this tinyFaaS instance.
type GRPCServer struct {
	r *rproxy.RProxy
}

// Request handles a request to the GRPC endpoint of the reverse-proxy of this tinyFaaS instance.
func (gs *GRPCServer) Request(ctx context.Context, d *api.Data) (*api.Response, error) {

	log.Printf("have request for path: %s (async: %v)", d.FunctionIdentifier, false)

	s, res := gs.r.Call(d.FunctionIdentifier, []byte(d.Data), false)

	switch s {
	case rproxy.StatusOK:
		return &api.Response{
			Response: string(res),
		}, nil
	case rproxy.StatusAccepted:
		return &api.Response{}, nil
	case rproxy.StatusNotFound:
		return nil, fmt.Errorf("function %s not found", d.FunctionIdentifier)
	case rproxy.StatusError:
		return nil, fmt.Errorf("error calling function %s", d.FunctionIdentifier)
	}
	return &api.Response{
		Response: string(res),
	}, nil
}

func Start(r *rproxy.RProxy, listenAddr string) {
	gs := grpc.NewServer()

	api.RegisterTinyFaaSServer(gs, &GRPCServer{
		r: r,
	})

	lis, err := net.Listen("tcp", listenAddr)

	if err != nil {
		log.Fatal("Failed to listen")
	}

	log.Printf("Starting GRPC server on %s", listenAddr)
	defer gs.GracefulStop()
	gs.Serve(lis)
}
