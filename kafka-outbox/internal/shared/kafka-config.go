package shared

type KafkaConfig struct {
	DefaultTopic             string
	Host                     string
	ConsumerGroup            string
	ParititionAssignStrategy string
}

func NewKafkaConfig() *KafkaConfig {
	return &KafkaConfig{
		ParititionAssignStrategy: "cooperative-sticky",
		DefaultTopic:             "local_topic_sticky",
		Host:                     "localhost",
		ConsumerGroup:            "local_cg",
	}
}
