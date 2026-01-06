package config

type KafkaConfig struct {
	DefaultTopic             string
	Host                     string
	ConsumerGroup            string
	ParititionAssignStrategy string
	NumPartitions            int
}

func NewKafkaConfig() *KafkaConfig {
	return &KafkaConfig{
		// ParititionAssignStrategy: "cooperative-sticky",
		ParititionAssignStrategy: "range",
		DefaultTopic:             "local_topic_range",
		Host:                     "localhost",
		ConsumerGroup:            "local_cg",
		NumPartitions:            4,
	}
}
