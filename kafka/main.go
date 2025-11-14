package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/k-code-yt/go-api-practice/kafka/consumer"
	"github.com/k-code-yt/go-api-practice/kafka/producer"
)

type Server struct {
	addr     string
	producer *producer.KafkaProducer
	consumer *consumer.KafkaConsumer
}

func NewServer(addr string) *Server {
	return &Server{
		addr:     addr,
		producer: producer.NewKafkaProducer(),
		consumer: consumer.NewKafkaConsumer(),
	}
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	b := r.Body
	defer b.Close()

	// s.producer.Produce(nil)
}

func (s *Server) initListener() {
	mux := http.NewServeMux()
	mux.HandleFunc("", s.handler)
	log.Fatal(http.ListenAndServe(s.addr, nil))
}

func (s *Server) handleMsg(msg string) {
	fmt.Printf("Consumed msg = %+v\n", msg)
	s.consumer.CommitMsg(msg)
}

func main() {
	s := NewServer(":7576")
	// go s.initListener()
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for ts := range ticker.C {
			s.producer.Produce(fmt.Sprintf("hello from Kafka, ts = %v\n", ts.Format("15:04:05")))
		}
	}()

	for msg := range s.consumer.MsgCH {
		go s.handleMsg(msg)
	}
}

func parseTopicPartitionOffset(s string) (topic string, partition int32, offset int64, err error) {
	// Split by '@' to separate topic/partition from offset
	parts := strings.Split(s, "@")
	if len(parts) != 2 {
		return "", 0, 0, fmt.Errorf("invalid format: expected topic[partition]@offset")
	}

	// Parse offset
	offset, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid offset: %w", err)
	}

	// Split topic and partition: "local_topic[0]"
	topicPart := parts[0]
	bracketStart := strings.Index(topicPart, "[")
	bracketEnd := strings.Index(topicPart, "]")

	if bracketStart == -1 || bracketEnd == -1 {
		return "", 0, 0, fmt.Errorf("invalid format: missing brackets")
	}

	topic = topicPart[:bracketStart]
	partitionStr := topicPart[bracketStart+1 : bracketEnd]

	partitionInt, err := strconv.ParseInt(partitionStr, 10, 32)
	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid partition: %w", err)
	}
	partition = int32(partitionInt)

	return topic, partition, offset, nil
}
