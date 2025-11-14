package shared

type KafkaConfig struct {
	DefaultTopic  string
	Host          string
	ConsumerGroup string
}

func NewKafkaConfig() *KafkaConfig {
	return &KafkaConfig{
		DefaultTopic:  "local_topic",
		Host:          "localhost",
		ConsumerGroup: "local_cg",
	}
}
