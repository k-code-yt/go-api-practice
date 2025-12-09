package main

import (
	"math/rand"
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

func BenchmarkPS_ReadHeavy(b *testing.B) {
	topic := "test_topic"
	tp := &kafka.TopicPartition{
		Topic:     &topic,
		Partition: 0,
		Offset:    0,
	}

	cm := consumer.NewTestKafkaConsumer(topic, tp)
	consumer.NewTestPartitionState(cm, tp, 10000)

	ps, err := cm.GetPartitionState(0)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		if i%10 < 8 {
			_ = ps.ReadState()
		} else {
			randomOffset := kafka.Offset(rand.Intn(10000))
			tp := &kafka.TopicPartition{
				Topic:     &topic,
				Partition: 0,
				Offset:    randomOffset,
			}
			cm.UpdateState(tp, consumer.MsgState_Success)
		}
	}
}

func BenchmarkPS_Balanced(b *testing.B) {
	topic := "test_topic"
	tp := &kafka.TopicPartition{
		Topic:     &topic,
		Partition: 0,
		Offset:    0,
	}

	cm := consumer.NewTestKafkaConsumer(topic, tp)
	consumer.NewTestPartitionState(cm, tp, 10000)

	ps, err := cm.GetPartitionState(0)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		if i%2 == 0 {
			_ = ps.ReadState()
		} else {
			randomOffset := kafka.Offset(rand.Intn(10000))
			tp := &kafka.TopicPartition{
				Topic:     &topic,
				Partition: 0,
				Offset:    randomOffset,
			}
			state := consumer.MsgState_Pending
			if rand.Intn(5) != 0 {
				state = consumer.MsgState_Success
			}
			cm.UpdateState(tp, state)
		}
	}
}

func BenchmarkPS_KafkaSim(b *testing.B) {
	topic := "test_topic"
	tp := &kafka.TopicPartition{
		Topic:     &topic,
		Partition: 0,
		Offset:    0,
	}

	cm := consumer.NewTestKafkaConsumer(topic, tp)
	consumer.NewTestPartitionState(cm, tp, 1000)

	ps, err := cm.GetPartitionState(0)
	if err != nil {
		b.Fatal(err)
	}

	nextOffset := kafka.Offset(1000)

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		newTp := &kafka.TopicPartition{
			Topic:     &topic,
			Partition: 0,
			Offset:    nextOffset,
		}
		cm.UpdateState(newTp, consumer.MsgState_Pending)

		ps.Mu.Lock()
		if nextOffset > ps.MaxReceived.Offset {
			ps.MaxReceived.Offset = nextOffset
		}
		ps.Mu.Unlock()

		nextOffset++

		result, err := ps.FindLatestToCommit()
		if err == nil && result != nil {
			ps.Mu.Lock()
			ps.LastCommited = result.Offset
			ps.Mu.Unlock()

			if i%3 == 0 {
				for offset := result.Offset - kafka.Offset(rand.Intn(10)); offset < result.Offset; offset++ {
					tp := &kafka.TopicPartition{
						Topic:     &topic,
						Partition: 0,
						Offset:    offset,
					}
					cm.UpdateState(tp, consumer.MsgState_Success)
				}
			}
		}
	}
}
