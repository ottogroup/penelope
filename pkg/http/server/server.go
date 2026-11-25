package server

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/config"
)

type Server interface {
	Run() error
}

func CreateServer(handler http.Handler) Server {
	return &defaultServer{handler}
}

type defaultServer struct {
	handler http.Handler
}

func (s *defaultServer) Run() error {
	handler, err := newDomainAllowlist(s.handler, config.AllowedDomains.GetOrDefault(""), config.TasksAllowedDomains.GetOrDefault(""))
	if err != nil {
		return fmt.Errorf("could not configure hostname validation: %w", err)
	}
	port := config.LocalPort.GetOrDefault("8080")
	addr := fmt.Sprintf(":%s", port)
	glog.Infoln("Starting app server on port", port)

	if err := http.ListenAndServe(addr, handler); err != nil {
		return fmt.Errorf("could not start http server. err: %s", err)
	}

	return nil
}
