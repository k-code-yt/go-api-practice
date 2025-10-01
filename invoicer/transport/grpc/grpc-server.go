package grpcserver

import (
	"context"
	"fmt"
	"net"

	"github.com/k-code-yt/go-api-practice/invoicer/ports"
	"github.com/k-code-yt/go-api-practice/shared"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type GRPCServer struct {
	port string
	svc  ports.Invoicer
	shared.UnimplementedInvoiceTransportServiceServer
}

func NewGRPCServer() ports.ServerTransport {
	return &GRPCServer{
		port: shared.HTTPPortInvoice,
	}
}

func (s *GRPCServer) Listen(svc ports.Invoicer) error {
	l, err := net.Listen("tcp", s.port)
	if err != nil {
		return err
	}
	s.svc = svc
	server := grpc.NewServer(grpc.EmptyServerOption{})
	shared.RegisterInvoiceTransportServiceServer(server, s)
	logrus.Infof("Registered GRPC Server on port = %s, info = %v\n", s.port, server.GetServiceInfo())

	return server.Serve(l)
}

func (s *GRPCServer) SaveInvoice(ctx context.Context, req *shared.SaveInvoiceRequest) (*shared.SaveInvoiceResponse, error) {
	d := shared.Distance{
		ID:        req.Distance.GetID(),
		Value:     req.Distance.GetValue(),
		Timestamp: int64(req.GetDistance().Timestamp),
	}
	i := s.svc.SaveInvoice(&d)
	if i == nil {
		errMsg := "Error saving invoice"
		logrus.WithFields(logrus.Fields{
			"success": false,
		}).Errorf("GPRC:SaveInvoice:ERROR")

		return &shared.SaveInvoiceResponse{
			Success: false,
			Msg:     errMsg,
		}, fmt.Errorf("%s", errMsg)
	}

	logrus.WithFields(logrus.Fields{
		"success": true,
	}).Infof("GRPC:SaveInvoice")

	return &shared.SaveInvoiceResponse{
		Success: true,
		Msg:     string(i.Category),
	}, nil
}
