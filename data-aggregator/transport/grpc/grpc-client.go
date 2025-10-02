package client

import (
	"context"
	"fmt"

	"github.com/k-code-yt/go-api-practice/shared"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCClient struct {
	Endpoint     string
	conn         *grpc.ClientConn
	client       shared.InvoiceTransportServiceClient
	getterClient shared.GetterInvoiceTransportServiceClient
}

func NewGRPCClient(endpoint string) (*GRPCClient, error) {
	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := shared.NewInvoiceTransportServiceClient(conn)
	getterClient := shared.NewGetterInvoiceTransportServiceClient(conn)
	logrus.Infof("Registered GRPC client on port %s\n", endpoint)

	return &GRPCClient{
		Endpoint:     endpoint,
		conn:         conn,
		client:       client,
		getterClient: getterClient,
	}, nil
}

func (c *GRPCClient) GetInvoice(id string) (*shared.Invoice, error) {

	resp, err := c.getterClient.GetInvoice(context.Background(), &shared.GetInvoiceRequest{
		ID: id,
	})

	if err != nil {
		logrus.Errorf("Error GetInvoice GRPC %v", err)
		return nil, err
	}

	return &shared.Invoice{
		ID:       resp.Invoice.GetID(),
		Amount:   resp.Invoice.GetAmount(),
		Category: shared.InvoiceCategory(resp.Invoice.Category),
	}, nil

}

func (c *GRPCClient) SaveInvoice(distance shared.Distance) error {
	dp := shared.DistanceProto{
		ID:        distance.ID,
		Value:     distance.Value,
		Timestamp: int32(distance.Timestamp),
	}
	in := shared.SaveInvoiceRequest{
		Distance: &dp,
	}
	resp, err := c.client.SaveInvoice(context.Background(), &in)
	if err != nil {
		logrus.Errorf("Error SaveInvoice GRPC %v", err)
		return err
	}
	if !resp.Success {
		logrus.Errorf("Error SaveInvoice GRPC %v", err)
		return fmt.Errorf("%v", resp.Msg)
	}
	logrus.WithFields(logrus.Fields{
		"msg":     resp.Msg,
		"success": resp.Success,
	}).Infof("GRPC:SaveInvoice")

	return nil
}
