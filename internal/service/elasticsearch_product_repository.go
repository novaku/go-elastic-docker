package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	esinternal "github.com/novaku/go-elastic-search/internal/elasticsearch"
	"go.uber.org/zap"
)

type elasticsearchProductRepository struct {
	es      *esinternal.Client
	logger  *zap.Logger
	builder SearchQueryBuilder
}

func NewElasticsearchProductRepository(es *esinternal.Client, logger *zap.Logger, builder SearchQueryBuilder) ProductRepository {
	return &elasticsearchProductRepository{es: es, logger: logger, builder: builder}
}

func (r *elasticsearchProductRepository) EnsureIndex(ctx context.Context) error {
	res, err := r.es.ES.Indices.Exists([]string{ProductIndexName})
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		r.logger.Info("index already exists", zap.String("index", ProductIndexName))
		return nil
	}

	res2, err := r.es.ES.Indices.Create(
		ProductIndexName,
		r.es.ES.Indices.Create.WithBody(strings.NewReader(productMapping)),
		r.es.ES.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res2.Body.Close()

	if res2.IsError() {
		body, _ := io.ReadAll(res2.Body)
		return fmt.Errorf("creating index: %s", string(body))
	}

	r.logger.Info("index created", zap.String("index", ProductIndexName))
	return nil
}

func (r *elasticsearchProductRepository) Create(ctx context.Context, product Product) error {
	body, err := json.Marshal(product)
	if err != nil {
		return err
	}

	res, err := r.es.ES.Index(
		ProductIndexName,
		bytes.NewReader(body),
		r.es.ES.Index.WithDocumentID(product.ID),
		r.es.ES.Index.WithRefresh("wait_for"),
		r.es.ES.Index.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("indexing document: %s", string(b))
	}

	return nil
}

func (r *elasticsearchProductRepository) Get(ctx context.Context, id string) (*Product, error) {
	res, err := r.es.ES.Get(ProductIndexName, id,
		r.es.ES.Get.WithContext(ctx),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, ErrNotFound
	}
	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("getting document: %s", string(b))
	}

	var hit struct {
		Source Product `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&hit); err != nil {
		return nil, err
	}
	return &hit.Source, nil
}

func (r *elasticsearchProductRepository) Update(ctx context.Context, id string, product Product) error {
	doc := map[string]interface{}{"doc": product}
	body, _ := json.Marshal(doc)

	res, err := r.es.ES.Update(
		ProductIndexName, id,
		bytes.NewReader(body),
		r.es.ES.Update.WithRefresh("wait_for"),
		r.es.ES.Update.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("updating document: %s", string(b))
	}

	return nil
}

func (r *elasticsearchProductRepository) Delete(ctx context.Context, id string) error {
	res, err := r.es.ES.Delete(
		ProductIndexName, id,
		r.es.ES.Delete.WithRefresh("wait_for"),
		r.es.ES.Delete.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return ErrNotFound
	}
	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("deleting document: %s", string(b))
	}

	return nil
}

func (r *elasticsearchProductRepository) BulkIndex(ctx context.Context, products []Product) (int, error) {
	var buf bytes.Buffer

	for _, product := range products {
		meta := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": ProductIndexName,
				"_id":    product.ID,
			},
		}
		metaLine, _ := json.Marshal(meta)
		docLine, _ := json.Marshal(product)

		buf.Write(metaLine)
		buf.WriteByte('\n')
		buf.Write(docLine)
		buf.WriteByte('\n')
	}

	res, err := r.es.ES.Bulk(
		bytes.NewReader(buf.Bytes()),
		r.es.ES.Bulk.WithRefresh("wait_for"),
		r.es.ES.Bulk.WithContext(ctx),
	)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return 0, fmt.Errorf("bulk index: %s", string(b))
	}

	var bulkResp struct {
		Errors bool `json:"errors"`
		Items  []map[string]struct {
			Status int `json:"status"`
			Error  *struct {
				Reason string `json:"reason"`
			} `json:"error,omitempty"`
		} `json:"items"`
	}
	if err := json.NewDecoder(res.Body).Decode(&bulkResp); err != nil {
		return 0, err
	}

	indexed := 0
	for _, item := range bulkResp.Items {
		if v, ok := item["index"]; ok && v.Error == nil {
			indexed++
		}
	}

	return indexed, nil
}

func (r *elasticsearchProductRepository) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	req = applySearchDefaults(req)
	query := r.builder.Build(req)
	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	from := (req.Page - 1) * req.PageSize

	res, err := r.es.ES.Search(
		r.es.ES.Search.WithIndex(ProductIndexName),
		r.es.ES.Search.WithBody(bytes.NewReader(body)),
		r.es.ES.Search.WithFrom(from),
		r.es.ES.Search.WithSize(req.PageSize),
		r.es.ES.Search.WithTrackTotalHits(true),
		r.es.ES.Search.WithContext(ctx),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("search: %s", string(b))
	}

	var esResp esSearchResponse
	if err := json.NewDecoder(res.Body).Decode(&esResp); err != nil {
		return nil, err
	}

	products := make([]Product, 0, len(esResp.Hits.Hits))
	for _, hit := range esResp.Hits.Hits {
		products = append(products, hit.Source)
	}

	return &SearchResponse{
		Total:    esResp.Hits.Total.Value,
		Page:     req.Page,
		PageSize: req.PageSize,
		Took:     esResp.Took,
		Products: products,
	}, nil
}

type esSearchResponse struct {
	Took int `json:"took"`
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source Product `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}
