package server

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/bitly/go-nsq"
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
	q.AddHandler(nsq.HandlerFunc(s.handler))
	if err := q.ConnectToNSQLookupd(s.config.NSQLookupdAddr); err != nil {
		return err
	}

	logrus.Infof("listening on %s", s.config.ListenAddr)
	if err := http.ListenAndServe(s.config.ListenAddr, nil); err != nil {
		return err
	}

	return nil
}
