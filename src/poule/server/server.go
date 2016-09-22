package server

import (
	"encoding/json"
	"net/http"
	"poule/gh"
	"poule/operations"
	"poule/operations/catalog"

	"github.com/Sirupsen/logrus"
	"github.com/bitly/go-nsq"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

type Server struct {
	config *ServerConfig
}

func NewServer(cfg *ServerConfig) (*Server, error) {
	return &Server{
		config: cfg,
	}, nil
}

func (s *Server) Run() error {
	config := nsq.NewConfig()
	q, err := nsq.NewConsumer(s.config.Topic, s.config.Channel, config)
	if err != nil {
		return err
	}
	logrus.Debugf("connecting to nsq: %s", s.config.NSQLookupdAddr)
	q.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
		logrus.Debugf("nsq message: id=%s timestamp=%d", message.ID, message.Timestamp)
		for _, evt := range s.config.Events {
			logrus.Debugf("event operations: %v", evt.Operations)
			for _, operationConfig := range evt.Operations {
				logrus.Debugf("running operation: type=%s", operationConfig.Type)
				descriptor, ok := catalog.ByNameIndex[operationConfig.Type]
				if !ok {
					return errors.Errorf("unknown operation %q", operationConfig.Type)
				}
				op, err := descriptor.OperationFromConfig(operationConfig.Settings)
				if err != nil {
					return err
				}

				var pr *github.PullRequest
				if err := json.Unmarshal(message.Body, &pr); err != nil {
					return err
				}

				item := gh.MakePullRequestItem(pr)
				if err := operations.RunSingle(s.config.Config, op, item); err != nil {
					return err
				}
			}
		}
		return nil
	}))
	if err := q.ConnectToNSQLookupd(s.config.NSQLookupdAddr); err != nil {
		return err
	}

	logrus.Infof("listening on %s", s.config.ListenAddr)
	if err := http.ListenAndServe(s.config.ListenAddr, nil); err != nil {
		return err
	}

	return nil
}
