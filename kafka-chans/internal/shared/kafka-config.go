package shared

type KafkaConfig struct {
	DefaultTopic             string
	Host                     string
	ConsumerGroup            string
	ParititionAssignStrategy string
	NumPartitions            int
}

func NewKafkaConfig() *KafkaConfig {
	return &KafkaConfig{
		ParititionAssignStrategy: "cooperative-sticky",
		// ParititionAssignStrategy: "roundrobin",
		DefaultTopic:  "local_topic_chans5",
		Host:          "localhost",
		ConsumerGroup: "local_cg3",
		NumPartitions: 2,
	}
}
