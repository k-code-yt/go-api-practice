package main

import (
	"fmt"
	"math/rand"
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

func BenchmarkPS_ReadHeavy(b *testing.B) {
	topic := "test_topic"
	lastMsgOffset := 10_000

	tp := &kafka.TopicPartition{
		Topic:     &topic,
		Partition: 0,
		Offset:    kafka.Offset(lastMsgOffset),
	}

	ps := consumer.NewTestPartitionState(tp)

	for offset := range ps.MaxReceived.Offset {
		msg := consumer.NewUpdateStateMsg(offset, shared.MsgState_Pending)
		ps.UpdateStateCH <- msg
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		randomOffset := kafka.Offset(rand.Intn(lastMsgOffset))
		ps.ReadOffsetReqCH <- randomOffset
		<-ps.ReadOffsetRespCH
	}
}

func BenchmarkPS_Balanced(b *testing.B) {
	topic := "test_topic"
	lastMsgOffset := 10_000

	tp := &kafka.TopicPartition{
		Topic:     &topic,
		Partition: 0,
		Offset:    kafka.Offset(lastMsgOffset),
	}

	ps := consumer.NewTestPartitionState(tp)

	for offset := range ps.MaxReceived.Offset {
		msg := consumer.NewUpdateStateMsg(offset, shared.MsgState_Pending)
		ps.UpdateStateCH <- msg
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		if i%10 < 5 {
			randomOffset := kafka.Offset(rand.Intn(lastMsgOffset))
			ps.ReadOffsetReqCH <- randomOffset
			<-ps.ReadOffsetRespCH
		} else {
			randomOffset := kafka.Offset(rand.Intn(lastMsgOffset))
			state := shared.MsgState_Success
			msg := consumer.NewUpdateStateMsg(randomOffset, state)
			ps.UpdateStateCH <- msg
		}
	}
}

// func BenchmarkPS_KafkaSim(b *testing.B) {
// 	topic := "test_topic"
// 	tp := &kafka.TopicPartition{
// 		Topic:     &topic,
// 		Partition: 0,
// 		Offset:    0,
// 	}

// 	latestMsgOffset := 1000
// 	cm := consumer.NewTestKafkaConsumer(topic, tp)
// 	consumer.NewTestPartitionState(cm, tp, latestMsgOffset)

// 	ps, err := cm.GetPartitionState(0)
// 	if err != nil {
// 		b.Fatal(err)
// 	}

// 	nextOffset := kafka.Offset(latestMsgOffset)

// 	b.ResetTimer()
// 	for i := 0; b.Loop(); i++ {
// 		newTp := &kafka.TopicPartition{
// 			Topic:     &topic,
// 			Partition: 0,
// 			Offset:    nextOffset,
// 		}
// 		cm.UpdateState(newTp, consumer.MsgState_Pending)

// 		ps.Mu.Lock()
// 		if nextOffset > ps.MaxReceived.Offset {
// 			ps.MaxReceived.Offset = nextOffset
// 		}
// 		ps.Mu.Unlock()

// 		nextOffset++

// 		result, err := ps.FindLatestToCommit()
// 		if err == nil && result != nil {
// 			ps.Mu.Lock()
// 			ps.LastCommited = result.Offset
// 			ps.Mu.Unlock()

// 			if i%3 == 0 {
// 				for offset := result.Offset - kafka.Offset(rand.Intn(10)); offset < result.Offset; offset++ {
// 					tp := &kafka.TopicPartition{
// 						Topic:     &topic,
// 						Partition: 0,
// 						Offset:    offset,
// 					}
// 					cm.UpdateState(tp, consumer.MsgState_Success)
// 				}
// 			}
// 		}
// 	}
// }
