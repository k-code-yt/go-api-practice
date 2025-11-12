package shared

import "fmt"

type TransportType string

const (
	HTTPPortAggregator            string        = ":3000"
	HTTPPortInvoice               string        = ":3100"
	WSPort                        string        = ":30000"
	Kafka_DefaultConsumerGroup    string        = "sensor_data_consumer_group"
	Kafka_DefaultHost             string        = "localhost:9092"
	Invoicer_GRPCTransportType    TransportType = "GRPC"
	Invoicer_HTTPTransportType    TransportType = "HTTP"
	Invoicer_DefaultTransportType TransportType = Invoicer_GRPCTransportType
)

var (
	Kafka_DefaultTopic string = "sensor_data3"
	WSEndpoint         string = fmt.Sprintf("ws://127.0.0.1%s", WSPort)
)
