package main

import (
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka/internal/consumer"
)

func BenchmarkPS_UpdateState(b *testing.B) {
	topic := "test_topic"

	cm := consumer.KafkaConsumer{}
	for i := 0; b.Loop(); i++ {
		offset := kafka.Offset(i)
		tp := &kafka.TopicPartition{
			Topic:     &topic,
			Partition: 0,
			Offset:    offset,
		}

		cm.UpdateState(tp, consumer.MsgState_Pending)
	}
}

func BenchmarkPS_FindLatestToCommit(b *testing.B) {
	topic := "test_topic"

	tp := &kafka.TopicPartition{
		Topic:     &topic,
		Partition: 0,
		Offset:    0,
	}

	cm := consumer.NewTestKafkaConsumer(topic, tp)
	ps, err := cm.GetPartitionState(0)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; b.Loop(); i++ {
		offset := kafka.Offset(i)
		tp.Offset = offset
		if i%10_000 == 0 {
			cm.UpdateState(tp, consumer.MsgState_Pending)
			continue
		}

		cm.UpdateState(tp, consumer.MsgState_Success)

		if i%100 == 0 {
			for i := 0; i < 10; i++ {
				latestTP, err := ps.FindLatestToCommit()
				if err != nil {
					panic(err)
				}

				cm.UpdateState(latestTP, consumer.MsgState_Success)
				// fmt.Printf("updated state for %d to SUCCESS\n", latestTP.Offset)
			}
		}

	}

}
