package cache

import (
	"net/http"
	"time"

	"github.com/brycekahle/prerender/render"
	"github.com/go-redis/redis"
	"github.com/pkg/errors"
)

type redisCache struct {
	client *redis.Client
}

// Cache caches prerendering results for quick retrieval later
type Cache interface {
	Check(*http.Request) (*render.Result, error)
	Save(*render.Result, time.Duration) error
}

// NewCache creates a new caching layer using Redis as backend
func NewCache(client *redis.Client) Cache {
	return &redisCache{client}
}

func (c *redisCache) checkEtag(r *http.Request) (bool, error) {
	if etag := r.Header.Get("If-None-Match"); etag != "" {
		redisEtag, err := c.client.HGet(r.URL.Path, "Etag").Result()
		if err != nil && err != redis.Nil {
			return false, errors.Wrap(err, "getting cached etag failed")
		}
		return etag == redisEtag, nil
	}
	return false, nil
}

func (c *redisCache) Check(r *http.Request) (*render.Result, error) {
	matches, err := c.checkEtag(r)
	if err != nil {
		return nil, err
	}
	if matches {
		return &render.Result{Status: http.StatusNotModified}, nil
	}

	data, err := c.client.HGetAll(r.URL.Path).Result()
	if err != nil {
		return nil, errors.Wrap(err, "getting cached data failed")
	}
	html, ok := data["html"]
	if !ok {
		return nil, nil
	}

	res := render.Result{
		Status: http.StatusOK,
		HTML:   html,
		Etag:   data["Etag"],
	}
	return &res, nil
}

func (c *redisCache) Save(res *render.Result, ttl time.Duration) error {
	tx := c.client.TxPipeline()
	tx.HSet(res.URL, "Etag", res.Etag)
	tx.HSet(res.URL, "html", res.HTML)
	tx.PExpire(res.URL, ttl)

	_, err := tx.Exec()
	return err
}
