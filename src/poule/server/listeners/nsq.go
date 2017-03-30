package listeners

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"poule/configuration"

	"github.com/Sirupsen/logrus"
	nsq "github.com/bitly/go-nsq"
)

// NSQListener listens for GitHub events from an NSQ message queue.
type NSQListener struct {
	config *configuration.Server
}

// NewNSQListener returns a new NSQListener instance.
func NewNSQListener(config *configuration.Server) *NSQListener {
	return &NSQListener{
		config: config,
	}
}

// Start starts an HTTP server to receive GitHub WebHooks.
func (l *NSQListener) Start(handler Handler) error {
	// Create and start monitoring queues.
	queues := createQueues(l.config, newNSQHandler(handler))
	stopChan := monitorQueues(queues)

	// Graceful stop on SIGTERM and SIGINT.
	sigChan := make(chan os.Signal, 64)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case _, ok := <-stopChan:
			if !ok {
				return nil
			}
			logrus.Debug("All queues exited")
			break
		case sig := <-sigChan:
			logrus.WithField("signal", sig).Debug("received signal")
			for _, q := range queues {
				q.Consumer.Stop()
			}
			break
		}
	}
}

type nsqHandler struct {
	handler Handler
}

func newNSQHandler(handler Handler) *nsqHandler {
	return &nsqHandler{
		handler: handler,
	}
}

func (h *nsqHandler) HandleMessage(message *nsq.Message) error {
	// Unserialize the GitHub webhook payload into a partial message in order to inspect the type
	// of event and handle accordingly.
	var m partialMessage
	if err := json.Unmarshal(message.Body, &m); err != nil {
		return err
	}
	return h.handler.HandleMessage(m.GitHubEvent, message.Body)
}

type partialMessage struct {
	GitHubEvent    string `json:"X-GitHub-Event"`
	GitHubDelivery string `json:"X-GitHub-Delivery"`
	HubSignature   string `json:"X-Hub-Signature"`
	Action         string `json:"action"`
}

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
