package main

import (
	"fmt"
	"log"
	"net/http"
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
