package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/application"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/handlers"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/infra/msg"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/infra/repo"
	pkgconstants "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/constants"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/db/postgres"
	pkgkafka "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/kafka"
)

type Server struct {
	service *application.InventoryService
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewServer(inboxRepo *repo.InboxEventRepo, invRepo *repo.InventoryRepo, DBName string) *Server {
	// TODO -> revert inventory deb register
	// err := debezium.RegisterConnector("http://localhost:8083", debezium.GetDebPaymentDBConnectorName(DBName), DBName, fmt.Sprintf("public.%s", repo.DBTableName_Inventory))
	// if err != nil {
	// 	fmt.Printf("err on deb-m conn %v\n", err)
	// 	panic(err)
	// }

	s := application.NewInventoryService(inboxRepo, invRepo)
	kafkaConsumer := pkgkafka.NewKafkaConsumer([]string{msg.DebPaymentTopic})

	msgRouter := handlers.NewMsgRouter(kafkaConsumer)
	paymCreateHandler := handlers.NewPaymentCreatedHandler(s)
	msgRouter.AddHandler(paymCreateHandler.Handler, pkgconstants.EventType_PaymentCreated)
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		service: s,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Unable to get current file path")
	}

	dir := filepath.Dir(filename)
	envPath := filepath.Join(dir, ".env")

	if err := godotenv.Load(envPath); err != nil {
		log.Printf("No .env file found at %s", envPath)
	}
}

func main() {
	dbOpts := postgres.NewPostgresConfig("kafka_inventory")
	db, err := postgres.NewDBConn(dbOpts)
	if err != nil {
		panic(fmt.Sprintf("unable to conn to db, err = %v\n", err))
	}
	defer db.Close()

	inboxRepo := repo.NewInboxEventRepo(db)
	invRepo := repo.NewInventoryRepo(db)
	s := NewServer(inboxRepo, invRepo, dbOpts.DBName)
	<-s.ctx.Done()
}
