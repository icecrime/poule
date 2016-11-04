package server

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"poule/configuration"

	"github.com/Sirupsen/logrus"
	"github.com/bitly/go-nsq"
)

type Server struct {
	config *configuration.Server
}

func NewServer(cfg *configuration.Server) (*Server, error) {
	return &Server{
		config: cfg,
	}, nil
}

func (s *Server) Run() error {
	// Create and start monitoring queues.
	queues := createQueues(&s.config.NSQConfig, s)
	stopChan := monitorQueues(queues)

	// Graceful stop on SIGTERM and SIGINT.
	sigChan := make(chan os.Signal, 64)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case <-stopChan:
			logrus.Debug("All queues exited")
			return nil
		case sig := <-sigChan:
			logrus.WithField("signal", sig).Debug("received signal")
			for _, q := range queues {
				q.Consumer.Stop()
			}
		}
	}

	return nil
}

type Queue struct {
	Consumer *nsq.Consumer
}

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

func createQueues(c *configuration.NSQConfig, handler nsq.Handler) []*Queue {
	// Subscribe to the message queues for each repository.
	queues := make([]*Queue, 0, len(c.Topics))
	for _, topic := range c.Topics {
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
	}()
	return stopChan
}
