package debezium

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ConnectorConfig struct {
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config"`
}

// TODO -> how to run deb migrations?
func RegisterConnector(connectURL, connectorName string) error {
	config := ConnectorConfig{
		Name: connectorName,
		Config: map[string]interface{}{
			"connector.class":       "io.debezium.connector.postgresql.PostgresConnector",
			"tasks.max":             "1",
			"database.hostname":     "postgres-go-api",
			"database.port":         "5432",
			"database.user":         "user",
			"database.password":     "pass",
			"database.dbname":       "kafka_cdc",
			"database.server.name":  "payment_server",
			"table.include.list":    "public.payment, public.event_inbox",
			"plugin.name":           "pgoutput",
			"topic.prefix":          "cdc",
			"slot.name":             "debezium_slot",
			"heartbeat.interval.ms": "10000",
			"decimal.handling.mode": "string",
		},
	}

	body, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := waitForKafkaConnect(connectURL); err != nil {
		return err
	}

	resp, err := http.Get(fmt.Sprintf("%s/connectors/%s", connectURL, connectorName))
	if err == nil && resp.StatusCode == http.StatusOK {
		fmt.Printf("Connector %s already exists\n", connectorName)
		return nil
	}
	resp, err = http.Post(
		fmt.Sprintf("%s/connectors", connectURL),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return fmt.Errorf("failed to create connector: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create connector: status=%d, body=%s", resp.StatusCode, string(bodyBytes))
	}

	fmt.Printf("Connector %s created successfully\n", connectorName)
	return nil
}

func waitForKafkaConnect(connectURL string) error {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(fmt.Sprintf("%s/connectors", connectURL))
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("kafka connect not ready after %d retries", maxRetries)
}

// "transforms":                                    "outbox",
// "transforms.outbox.type":                        "io.debezium.transforms.outbox.EventRouter",
// "transforms.outbox.table.field.event.id":        "event_id",
// "transforms.outbox.table.field.event.type":      "event_type",
// "transforms.outbox.table.field.event.key":       "parent_id",
// "transforms.outbox.table.field.event.payload":   "parent_metadata",
// "transforms.outbox.table.field.event.timestamp": "timestamp",
// "transforms.outbox.route.topic.replacement":     "events.${routedByValue}",
// "transforms.outbox.route.by.field":              "event_type",
// "predicates":                                    "isEventsTable",
// "predicates.isEventsTable.type":                 "org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
// "predicates.isEventsTable.pattern":              "cdc.public.events",
// "transforms.outbox.predicate":                   "isEventsTable",
