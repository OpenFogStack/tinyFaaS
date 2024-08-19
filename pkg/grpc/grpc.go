package grpc

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/OpenFogStack/tinyFaaS/pkg/grpc/tinyfaas"
	"github.com/OpenFogStack/tinyFaaS/pkg/rproxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// GRPCServer is the grpc endpoint for this tinyFaaS instance.
type GRPCServer struct {
	r *rproxy.RProxy
}

// Request handles a request to the GRPC endpoint of the reverse-proxy of this tinyFaaS instance.
func (gs *GRPCServer) Request(ctx context.Context, d *tinyfaas.Data) (*tinyfaas.Response, error) {

	log.Printf("have request for path: %s (async: %v)", d.FunctionIdentifier, false)

	// Extract metadata from the gRPC context
	md, ok := metadata.FromIncomingContext(ctx)
	headers := make(map[string]string)
	if ok {
		// Convert metadata to map[string]string
		for k, v := range md {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}
	} else {
		log.Print("failed to extract metadata from context, using empty headers GRPC request")
	}

	s, res := gs.r.Call(d.FunctionIdentifier, []byte(d.Data), false, headers)

	switch s {
	case rproxy.StatusOK:
		return &tinyfaas.Response{
			Response: string(res),
		}, nil
	case rproxy.StatusAccepted:
		return &tinyfaas.Response{}, nil
	case rproxy.StatusNotFound:
		return nil, fmt.Errorf("function %s not found", d.FunctionIdentifier)
	case rproxy.StatusError:
		return nil, fmt.Errorf("error calling function %s", d.FunctionIdentifier)
	}
	return &tinyfaas.Response{
		Response: string(res),
	}, nil
}

func Start(r *rproxy.RProxy, listenAddr string) {
	gs := grpc.NewServer()

	tinyfaas.RegisterTinyFaaSServer(gs, &GRPCServer{
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
