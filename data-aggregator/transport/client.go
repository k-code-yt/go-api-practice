package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/k-code-yt/go-api-practice/shared"

	"github.com/sirupsen/logrus"
)

type TransportClient interface {
	SaveInvoice(distance shared.Distance) error
}

type HTTPClient struct {
	Endpoint string
}

func NewHTTPClient(endpoint string) *HTTPClient {
	return &HTTPClient{
		Endpoint: endpoint,
	}
}

func (c *HTTPClient) SaveInvoice(distance shared.Distance) error {
	logrus.Info("sending request to save invoice")
	b, err := json.Marshal(distance)
	if err != nil {
		return err
	}

	resp, err := http.Post(c.Endpoint, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	logrus.Infof("received response from invoicer %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response is non 200 status, received %d", resp.StatusCode)
	}
	return nil
}

type GRPCClient struct {
	Endpoint string
}
