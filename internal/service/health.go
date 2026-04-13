package service

import "context"

type pinger interface {
	Ping() error
}

type ElasticsearchHealthChecker struct {
	pinger pinger
}

func NewElasticsearchHealthChecker(pinger pinger) *ElasticsearchHealthChecker {
	return &ElasticsearchHealthChecker{pinger: pinger}
}

func (h *ElasticsearchHealthChecker) Check(_ context.Context) error {
	return h.pinger.Ping()
}
