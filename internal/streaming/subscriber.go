package streaming

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"
	"wb-test-task/internal/db"

	"github.com/nats-io/stan.go"
)

type Subscriber struct {
	sub      stan.Subscription
	dbObject *db.DB
	sc       *stan.Conn
	name     string
}

func NewSubscriber(db *db.DB, conn *stan.Conn) *Subscriber {
	return &Subscriber{
		name:     "Subscriber",
		dbObject: db,
		sc:       conn,
	}
}

func (s *Subscriber) Subscribe() {
	ackWait, err := strconv.Atoi(os.Getenv("NATS_ACK_WAIT_SECONDS"))
	if err != nil {
		log.Printf("%s: received a message!\n", s.name)
		return
	}

	s.sub, err = (*s.sc).Subscribe(
		os.Getenv("NATS_SUBJECT"),
		func(m *stan.Msg) {
			log.Printf("%s: received a message!\n", s.name)
			if s.messageHandler(m.Data) {
				err := m.Ack()
				if err != nil {
					log.Printf("%s ack() err: %s", s.name, err)
				}
			}
		},
		stan.AckWait(time.Duration(ackWait)*time.Second),
		stan.DurableName(os.Getenv("NATS_DURABLE_NAME")),
		stan.SetManualAckMode(),
		stan.MaxInflight(10),
	)

	if err != nil {
		log.Printf("%s: error: %v\n", s.name, err)
	}

	log.Printf("%s: subscribed to subject %s\n", s.name, os.Getenv("NATS_SUBJECT"))
}

func (s *Subscriber) messageHandler(data []byte) bool {
	receivedOrder := db.Order{}
	err := json.Unmarshal(data, &receivedOrder)
	if err != nil {
		log.Printf("%s: messageHandler() error, %v\n", s.name, err)

		return true
	}
	log.Printf("%s: unmarshal Order to struct: %v\n", s.name, receivedOrder)

	_, err = s.dbObject.AddOrder(receivedOrder)
	if err != nil {
		log.Printf("%s: unable to add order: %v\n", s.name, err)
		return false
	}
	return true
}

func (s *Subscriber) Unsubscribe() {
	if s.sub != nil {
		err := s.sub.Unsubscribe()

		if err != nil {
			return
		}
	}
}
