package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	esinternal "github.com/novaku/go-elastic-search/internal/elasticsearch"
	"go.uber.org/zap"
)

func newRepoForServer(t *testing.T, h http.HandlerFunc) ProductRepository {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		h(w, r)
	}))
	t.Cleanup(ts.Close)

	es, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{ts.URL}})
	if err != nil {
		t.Fatalf("create es client: %v", err)
	}

	client := &esinternal.Client{ES: es}
	return NewElasticsearchProductRepository(client, zap.NewNop(), NewElasticsearchQueryBuilder())
}

func TestRepository_EnsureIndex(t *testing.T) {
	t.Run("already exists", func(t *testing.T) {
		repo := newRepoForServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodHead && r.URL.Path == "/"+ProductIndexName {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
		})

		if err := repo.EnsureIndex(context.Background()); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("create index", func(t *testing.T) {
		calls := 0
		repo := newRepoForServer(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodHead && r.URL.Path == "/"+ProductIndexName:
				calls++
				w.WriteHeader(http.StatusNotFound)
			case r.Method == http.MethodPut && r.URL.Path == "/"+ProductIndexName:
				calls++
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, `{"acknowledged":true}`)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}
		})

		if err := repo.EnsureIndex(context.Background()); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if calls != 2 {
			t.Fatalf("expected 2 calls, got %d", calls)
		}
	})
}

func TestRepository_CRUD_Bulk_Search(t *testing.T) {
	repo := newRepoForServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case (r.Method == http.MethodPost || r.Method == http.MethodPut) && strings.HasPrefix(r.URL.Path, "/"+ProductIndexName+"/_doc/"):
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"result":"created"}`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/"+ProductIndexName+"/_doc/"):
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"_source":{"id":"p-1","name":"Phone","category":"electronics","price":10}}`)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/"+ProductIndexName+"/_update/"):
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"result":"updated"}`)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/"+ProductIndexName+"/_doc/"):
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"result":"deleted"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/_bulk":
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"errors":false,"items":[{"index":{"status":201}},{"index":{"status":201}}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/"+ProductIndexName+"/_search":
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"took":2,"hits":{"total":{"value":1},"hits":[{"_source":{"id":"p-1","name":"Phone","category":"electronics","price":10}}]}}`)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	ctx := context.Background()
	if err := repo.Create(ctx, Product{ID: "p-1", Name: "Phone", Category: "electronics", Price: 10}); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.Get(ctx, "p-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != "p-1" {
		t.Fatalf("expected id p-1, got %q", got.ID)
	}
	if err := repo.Update(ctx, "p-1", Product{ID: "p-1", Name: "New"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := repo.Delete(ctx, "p-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	indexed, err := repo.BulkIndex(ctx, []Product{{ID: "a"}, {ID: "b"}})
	if err != nil {
		t.Fatalf("bulk: %v", err)
	}
	if indexed != 2 {
		t.Fatalf("expected 2 indexed, got %d", indexed)
	}
	resp, err := repo.Search(ctx, SearchRequest{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if resp.Total != 1 || len(resp.Products) != 1 {
		t.Fatalf("unexpected search response: %+v", resp)
	}
}

func TestRepository_NotFoundAndErrors(t *testing.T) {
	repo := newRepoForServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"found":false}`)
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"result":"not_found"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/"+ProductIndexName+"/_search":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `search failed`)
		default:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `error`)
		}
	})

	ctx := context.Background()
	if _, err := repo.Get(ctx, "missing"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for get, got %v", err)
	}
	if err := repo.Delete(ctx, "missing"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for delete, got %v", err)
	}
	if _, err := repo.Search(ctx, SearchRequest{}); err == nil {
		t.Fatalf("expected search error")
	}
}

func TestRepository_Search_BuildsJSON(t *testing.T) {
	repo := newRepoForServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/"+ProductIndexName+"/_search" {
			b, _ := io.ReadAll(r.Body)
			var payload map[string]interface{}
			if err := json.Unmarshal(b, &payload); err != nil {
				t.Fatalf("expected valid json payload: %v", err)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"took":1,"hits":{"total":{"value":0},"hits":[]}}`)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := repo.Search(context.Background(), SearchRequest{Query: "phone"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
}
