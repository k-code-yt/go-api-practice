package transport

import (
	"github.com/k-code-yt/go-api-practice/protocol-playground/invoicer/ports"
	grpcserver "github.com/k-code-yt/go-api-practice/protocol-playground/invoicer/transport/grpc"
	httpserver "github.com/k-code-yt/go-api-practice/protocol-playground/invoicer/transport/http"
	"github.com/k-code-yt/go-api-practice/protocol-playground/shared"
)

func NewServerTransport(serverType shared.TransportType) ports.ServerTransport {
	if serverType == shared.Invoicer_GRPCTransportType {
		return grpcserver.NewGRPCServer()
	}

	if serverType == shared.Invoicer_HTTPTransportType {
		return httpserver.NewHTTPServer()
	}
	return nil
}
