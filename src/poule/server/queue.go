package server

import (
	"log"
	"os"
	"sync"

	"poule/configuration"

	"github.com/Sirupsen/logrus"
	nsq "github.com/bitly/go-nsq"
)

// Queue represents one NSQ queue.
type Queue struct {
	Consumer *nsq.Consumer
}

// NewQueue returns a new queue instance.
func NewQueue(topic, channel, lookupd string, handler nsq.Handler) (*Queue, error) {
	logger := log.New(os.Stderr, "", log.Flags())
	consumer, err := nsq.NewConsumer(topic, channel, nsq.NewConfig())
	if err != nil {
		return nil, err
	}

	consumer.AddHandler(handler)
	consumer.SetLogger(logger, nsq.LogLevelWarning)
	if err := consumer.ConnectToNSQLookupd(lookupd); err != nil {
		return nil, err
	}

	return &Queue{Consumer: consumer}, nil
}

func createQueues(c *configuration.Server, handler nsq.Handler) []*Queue {
	// Subscribe to the message queues for each repository.
	queues := make([]*Queue, 0, len(c.Repositories))
	for _, topic := range c.Repositories {
		queue, err := NewQueue(topic, c.Channel, c.LookupdAddr, handler)
		if err != nil {
			logrus.Fatal(err)
		}
		queues = append(queues, queue)
	}
	return queues
}

func monitorQueues(queues []*Queue) <-chan struct{} {
	// Start one goroutine per queue and monitor the StopChan event.
	wg := sync.WaitGroup{}
	for _, q := range queues {
		wg.Add(1)
		go func(queue *Queue) {
			<-queue.Consumer.StopChan
			logrus.Debug("Queue stop channel signaled")
			wg.Done()
		}(q)
	}

	// Multiplex all queues exit into a single channel we can select on.
	stopChan := make(chan struct{})
	go func() {
		wg.Wait()
		stopChan <- struct{}{}
		close(stopChan)
	}()
	return stopChan
}
