package shared

import (
	"encoding/json"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-chans/internal/repo"
)

type MsgState = int32

const (
	MsgState_Pending MsgState = iota
	MsgState_Success MsgState = iota
	MsgState_Error   MsgState = iota
)

type Message struct {
	Metadata *kafka.TopicPartition
	Event    *repo.Event
}

func NewMessage(metadata *kafka.TopicPartition, data []byte) *Message {
	e := &repo.Event{}
	err := json.Unmarshal(data, e)
	if err != nil {
		panic(fmt.Sprintf("err unmarshalling event = %v\n", err))
	}
	return &Message{
		Metadata: metadata,
		Event:    e,
	}
}
