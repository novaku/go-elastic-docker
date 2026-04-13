package service

// ProductIndexName is the Elasticsearch index for products.
const ProductIndexName = "products"

// productMapping defines the index mapping and settings.
// This is applied once when the index is first created.
const productMapping = `{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 1,
    "analysis": {
      "analyzer": {
        "product_analyzer": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase", "asciifolding", "stop", "snowball"]
        },
        "autocomplete_analyzer": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase", "asciifolding", "edge_ngram_filter"]
        },
        "autocomplete_search_analyzer": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase", "asciifolding"]
        }
      },
      "filter": {
        "edge_ngram_filter": {
          "type": "edge_ngram",
          "min_gram": 2,
          "max_gram": 20
        }
      }
    }
  },
  "mappings": {
    "dynamic": "strict",
    "properties": {
      "id":          { "type": "keyword" },
      "name": {
        "type": "text",
        "analyzer": "product_analyzer",
        "fields": {
          "keyword":      { "type": "keyword" },
          "autocomplete": {
            "type":            "text",
            "analyzer":        "autocomplete_analyzer",
            "search_analyzer": "autocomplete_search_analyzer"
          }
        }
      },
      "description": { "type": "text", "analyzer": "product_analyzer" },
      "category":    { "type": "keyword" },
      "brand":       { "type": "keyword" },
      "price":       { "type": "double" },
      "stock":       { "type": "integer" },
      "tags":        { "type": "keyword" },
      "is_active":   { "type": "boolean" },
      "created_at":  { "type": "date" },
      "updated_at":  { "type": "date" }
    }
  }
}`
