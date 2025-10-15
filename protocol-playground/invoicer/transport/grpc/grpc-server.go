package transport

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/k-code-yt/go-api-practice/protocol-playground/invoicer/ports"
	"github.com/k-code-yt/go-api-practice/protocol-playground/shared"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TODO:
// ws ping/pong logic
// use-cases for birectional grpc
// review metadata in the headers
type GRPCServer struct {
	port      string
	svc       ports.Invoicer
	uploadDir string
	shared.UnimplementedInvoiceTransportServiceServer
	shared.UnimplementedGetterInvoiceTransportServiceServer
	shared.UnimplementedStreamingTransportSerivceServer
}

func NewGRPCServer() ports.ServerTransport {
	return &GRPCServer{
		port:      shared.HTTPPortInvoice,
		uploadDir: "./",
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func StreamAuthInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	logrus.Info("triggered StreamAuthInterceptor")
	md, ok := metadata.FromIncomingContext(ss.Context())
	if !ok {
		return status.Error(codes.InvalidArgument, "missing metadata")

	}

	token := md.Get("authorization")
	tenantID := md.Get("tenant-id")
	reqID := md.Get("request-id")

	logrus.WithFields(logrus.Fields{
		"auth_token": token,
		"tenant_id":  tenantID,
		"req_id":     reqID,
		"method":     info.FullMethod,
	}).Info("Stream started with headers")

	ctx := context.WithValue(ss.Context(), "userId", token)

	wrapped := &wrappedServerStream{
		ctx:          ctx,
		ServerStream: ss,
	}

	return handler(srv, wrapped)
}

func (s *GRPCServer) Listen(svc ports.Invoicer) error {
	l, err := net.Listen("tcp", s.port)
	if err != nil {
		return err
	}
	s.svc = svc
	server := grpc.NewServer()
	// server := grpc.NewServer(grpc.StreamInterceptor(StreamAuthInterceptor))
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
		const errMsg = "Error saving invoice"
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
		// HEADERS

		data := req.GetData()
		stream.SetHeader(metadata.New(map[string]string{
			"ID": data.GetID(),
		}))
		stream.SetTrailer(metadata.New(map[string]string{
			"IDTrailer": data.GetID(),
		}))

		// ---------------

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

func (s *GRPCServer) UploadFile(stream grpc.ClientStreamingServer[shared.FileChunk, shared.FileUploadResponse]) error {
	var (
		file     *os.File
		filename string
		bytesRec int64
		filePath string
	)

	logrus.Info("Starting file stream receive")

	for {
		chunk, err := stream.Recv()

		if err == io.EOF {
			if file != nil {
				err := file.Close()
				if err != nil {
					return err
				}
				file = nil
			}

			return stream.SendAndClose(&shared.FileUploadResponse{
				Success:       true,
				Message:       "file uploaded",
				SavePath:      filePath,
				BytesReceived: bytesRec,
			})
		}

		if err != nil {
			const errMsg = "error receiveing file chunks"
			logrus.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error(errMsg)
			return status.Errorf(codes.Internal, errMsg)
		}

		if file == nil {
			filename = chunk.GetFilename()
			ts := time.Now().Format("20250101_100100")
			ext := filepath.Ext(filename)
			fileNameNotExt := strings.Split(filename, ".")[0]
			filename = fmt.Sprintf("%s_%s%s", fileNameNotExt, ts, ext)

			filePath = filepath.Join(s.uploadDir, filename)

			file, err = os.Create(filePath)
			if err != nil {
				const errMsg = "couldn't create file"
				logrus.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error(errMsg)
				return status.Errorf(codes.Internal, errMsg)

			}

		}
		n, err := file.Write(chunk.Content)
		if err != nil {
			const errMsg = "error writing file chunk"
			logrus.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error(errMsg)
			return status.Errorf(codes.Internal, errMsg)
		}

		bytesRec += int64(n)

	}
}
