package elasticsearch

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/novaku/go-elastic-search/config"
	"go.uber.org/zap"
)

// Client wraps the official Elasticsearch client with app-level helpers
type Client struct {
	ES     *elasticsearch.Client
	logger *zap.Logger
}

// New creates a new Elasticsearch client from app config
func New(cfg *config.ESConfig, logger *zap.Logger) (*Client, error) {
	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		APIKey:    cfg.APIKey,
		CloudID:   cfg.CloudID,

		// Retry configuration — production-safe defaults
		MaxRetries:            3,
		DiscoverNodesOnStart:  false, // Set to true for multi-node clusters in prod
		DiscoverNodesInterval: 10 * time.Minute,

		Transport: buildTransport(cfg),
	}

	es, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("creating elasticsearch client: %w", err)
	}

	client := &Client{ES: es, logger: logger}

	if err := client.Ping(); err != nil {
		return nil, fmt.Errorf("elasticsearch ping failed: %w", err)
	}

	logger.Info("connected to elasticsearch", zap.Strings("addresses", cfg.Addresses))
	return client, nil
}

// Ping verifies the cluster is reachable
func (c *Client) Ping() error {
	res, err := c.ES.Ping()
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ping response: %s", res.Status())
	}
	return nil
}

func buildTransport(cfg *config.ESConfig) *http.Transport {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Load custom CA cert if provided (common in private ES clusters / prod)
	if cfg.CACert != "" {
		cert, err := os.ReadFile(cfg.CACert)
		if err == nil {
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(cert)
			tlsCfg.RootCAs = pool
		}
	}

	return &http.Transport{
		TLSClientConfig: tlsCfg,
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
	}
}
