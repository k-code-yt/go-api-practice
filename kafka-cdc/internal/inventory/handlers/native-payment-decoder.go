package handlers

// import (
// 	"encoding/binary"
// 	"fmt"

// 	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/debezium"
// 	pkgkafka "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/kafka"
// )

// func ParseDebeziumPayload(cfg *pkgkafka.KafkaConfig, topic string, data []byte) (*debezium.Payload[CDCPayment], error) {
// 	if len(data) < 5 {
// 		return nil, fmt.Errorf("data too short")
// 	}
// 	if data[0] != 0x00 {
// 		return nil, fmt.Errorf("invalid Avro magic byte")
// 	}

// 	schemaID := int(binary.BigEndian.Uint32(data[1:5]))
// 	avroData := data[5:]

// 	codec, err := cfg.AvroSchemaCache.GetCodec(cfg, topic, schemaID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	native, _, err := codec.NativeFromBinary(avroData)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decode avro: %w", err)
// 	}

// 	avroMap, ok := native.(map[string]interface{})
// 	if !ok {
// 		return nil, fmt.Errorf("unexpected avro data type: %T", native)
// 	}

// 	payload := &debezium.Payload[CDCPayment]{}

// 	if op, ok := avroMap["op"].(string); ok {
// 		payload.Op = op
// 	}

// 	if ts, ok := avroMap["ts_ms"].(int64); ok {
// 		payload.Timestamp = &ts
// 	}

// 	if before := avroMap["before"]; before != nil {
// 		if beforeMap, ok := before.(map[string]interface{}); ok {
// 			payment, err := mapToPayment(beforeMap)
// 			if err != nil {
// 				return nil, fmt.Errorf("failed to parse before: %w", err)
// 			}
// 			payload.Before = &payment
// 		}
// 	}

// 	if after := avroMap["after"]; after != nil {
// 		if afterMap, ok := after.(map[string]interface{}); ok {
// 			value, ok := afterMap["cdc.public.payment.Value"].(map[string]interface{})
// 			if !ok {
// 				panic("cdc.public.payment.Value does not exist")
// 			}
// 			payment, err := mapToPayment(value)
// 			if err != nil {
// 				return nil, fmt.Errorf("failed to parse after: %w", err)
// 			}
// 			payload.After = &payment
// 		}
// 	}

// 	if source, ok := avroMap["source"].(map[string]interface{}); ok {
// 		payload.Source = mapToSource(source)
// 	}

// 	// if txn := avroMap["transaction"]; txn != nil {
// 	// 	if txnMap, ok := txn.(map[string]interface{}); ok {
// 	// 		transaction := mapToTransaction(txnMap)
// 	// 		payload.Transaction = &transaction
// 	// 	}
// 	// }

// 	return payload, nil
// }

// func mapToPayment(m map[string]interface{}) (CDCPayment, error) {
// 	payment := CDCPayment{}

// 	// if id, ok := m["id"].(int32); ok {
// 	// 	payment.ID = int(id)
// 	// } else if id, ok := m["id"].(int64); ok {
// 	// 	payment.ID = int(id)
// 	// }

// 	// if orderNum, ok := m["order_number"].(string); ok {
// 	// 	payment.OrderNumber = orderNum
// 	// }

// 	// if amount, ok := m["amount"].(string); ok {
// 	// 	payment.Amount = amount
// 	// }

// 	// if status, ok := m["status"].(string); ok {
// 	// 	payment.Status = status
// 	// }

// 	// if createdAt, ok := m["created_at"].(int64); ok {
// 	// 	payment.CreatedAt = createdAt
// 	// }

// 	// if updatedAt, ok := m["updated_at"].(int64); ok {
// 	// 	payment.UpdatedAt = updatedAt
// 	// }

// 	return payment, nil
// }

// func mapToSource(m map[string]interface{}) debezium.DebeziumSource {
// 	source := debezium.DebeziumSource{}

// 	// if v, ok := m["version"].(string); ok {
// 	// 	source.Version = v
// 	// }
// 	// if v, ok := m["connector"].(string); ok {
// 	// 	source.Connector = v
// 	// }
// 	// if v, ok := m["name"].(string); ok {
// 	// 	source.Name = v
// 	// }
// 	// if v, ok := m["ts_ms"].(int64); ok {
// 	// 	source.Timestamp = v
// 	// }
// 	// if v, ok := m["snapshot"].(string); ok {
// 	// 	source.Snapshot = v
// 	// }
// 	// if v, ok := m["db"].(string); ok {
// 	// 	source.Db = v
// 	// }
// 	// if v, ok := m["sequence"].(string); ok {
// 	// 	source.Sequence = v
// 	// }
// 	// if v, ok := m["schema"].(string); ok {
// 	// 	source.Schema = v
// 	// }
// 	// if v, ok := m["table"].(string); ok {
// 	// 	source.Table = v
// 	// }
// 	// if v, ok := m["txId"].(int64); ok {
// 	// 	source.TxId = v
// 	// }
// 	// if v, ok := m["lsn"].(int64); ok {
// 	// 	source.Lsn = v
// 	// }

// 	return source
// }
