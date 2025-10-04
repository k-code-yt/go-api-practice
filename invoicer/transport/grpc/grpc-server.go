package transport

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
	shared.UnimplementedGetterInvoiceTransportServiceServer
	shared.UnimplementedStreamingTransportSerivceServer
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
	shared.RegisterGetterInvoiceTransportServiceServer(server, s)
	shared.RegisterStreamingTransportSerivceServer(server, s)
	logrus.Infof("Registered GRPC Server on port = %s, info = %v\n", s.port, server.GetServiceInfo())

	return server.Serve(l)
}

func (s *GRPCServer) GetInvoice(ctx context.Context, req *shared.GetInvoiceRequest) (*shared.GetInvoiceResponse, error) {
	id := req.GetID()
	inv, err := s.svc.GetInvoice(id)
	if inv == nil {
		return nil, fmt.Errorf("invoice does not exist")
	}

	return &shared.GetInvoiceResponse{
		Invoice: &shared.InvoiceProto{
			ID:       inv.ID,
			Amount:   inv.Amount,
			Category: string(inv.Category),
		},
	}, err
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
		"success":   true,
		"InvoiceID": i.ID,
	}).Infof("GRPC:SaveInvoice")

	return &shared.SaveInvoiceResponse{
		Success: true,
		Msg:     string(i.Category),
	}, nil
}

func (s *GRPCServer) SensorDataStream(stream grpc.BidiStreamingServer[shared.SensorDataRequest, shared.SensorDataResponse]) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			logrus.Errorf("error receiveing GRPC data %v", err)
			return err
		}
		data := req.GetData()
		err = s.svc.SensorDataStream(data.GetID(), data.GetLat(), data.GetLng())
		if err != nil {
			fmt.Println(err)
		}

		respData := shared.SensorDataResponse{
			Data: data,
		}
		logrus.WithFields(logrus.Fields{
			"ID": data.GetID(),
		}).Info("sending response to the client")

		err = stream.Send(&respData)
		if err != nil {
			logrus.Errorf("error sending GRPC data %v", err)
		}
	}

}
