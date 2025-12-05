package main

import (
	"fmt"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-chans/internal/consumer"
	"github.com/k-code-yt/go-api-practice/kafka-chans/internal/shared"
)

func BenchmarkPS_UpdateState(b *testing.B) {
	topic := "test_topic"
	tp := &kafka.TopicPartition{
		Topic:     &topic,
		Partition: 0,
		Offset:    0,
	}

	ps := consumer.NewPartitionState(tp)
	defer ps.Cancel()
	for i := 0; b.Loop(); i++ {
		offset := kafka.Offset(i)
		msg := consumer.NewUpdateStateMsg(offset, shared.MsgState_Pending)
		ps.UpdateStateCH <- msg
	}
}

func BenchmarkPS_FindLatestToCommit(b *testing.B) {
	topic := "test_topic"
	tp := &kafka.TopicPartition{
		Topic:     &topic,
		Partition: 0,
		Offset:    0,
	}

	ps := consumer.NewPartitionState(tp)
	defer ps.Cancel()

	state := shared.MsgState_Success
	for i := 0; i < b.N; i++ {
		offset := kafka.Offset(i)
		if i%10_000 == 0 && i != 0 {
			msg := consumer.NewUpdateStateMsg(offset, shared.MsgState_Pending)
			ps.UpdateStateCH <- msg
			continue
		}
		msg := consumer.NewUpdateStateMsg(offset, state)
		ps.UpdateStateCH <- msg

		if i%100 == 0 {
			for i := 0; i < 10; i++ {
				ps.FindLatestToCommitReqCH <- struct{}{}
				latest := <-ps.FindLatestToCommitRespCH
				msg := consumer.NewUpdateStateMsg(latest, state)
				ps.UpdateStateCH <- msg
				fmt.Printf("updated state for %d to SUCCESS\n", latest)

			}
		}

	}
}
