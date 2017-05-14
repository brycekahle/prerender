package cache

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/brycekahle/prerender/render"
	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var s *miniredis.Miniredis
var client Cache

func TestMain(m *testing.M) {
	var err error
	s, err = miniredis.Run()
	if err != nil {
		log.Fatal(err)
	}
	client = NewCache(redis.NewClient(&redis.Options{
		Addr: s.Addr(),
		DB:   0,
	}))
	code := m.Run()
	s.Close()
	os.Exit(code)
}

func TestEtagMatch(t *testing.T) {
	s.FlushAll()
	s.HSet("https://netlify.com/", "Etag", "etagetag")
	req := httptest.NewRequest("GET", "http://example.com/https://netlify.com/", nil)
	// TODO having to do this kinda sucks
	req.URL.Path = req.URL.Path[1:]
	req.Header.Add("If-None-Match", "etagetag")
	res, err := client.Check(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotModified, res.Status)
}

func TestEtagMismatch(t *testing.T) {
	s.FlushAll()
	s.HSet("https://netlify.com/", "Etag", "etagetag")
	s.HSet("https://netlify.com/", "html", "<html></html>")
	req := httptest.NewRequest("GET", "http://example.com/https://netlify.com/", nil)
	// TODO having to do this kinda sucks
	req.URL.Path = req.URL.Path[1:]
	req.Header.Add("If-None-Match", "nottag")
	res, err := client.Check(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, "<html></html>", res.HTML)
	assert.Equal(t, "etagetag", res.Etag)
}

func TestEtagNoData(t *testing.T) {
	s.FlushAll()
	req := httptest.NewRequest("GET", "http://example.com/https://netlify.com/", nil)
	// TODO having to do this kinda sucks
	req.URL.Path = req.URL.Path[1:]
	req.Header.Add("If-None-Match", "etagetag")
	res, err := client.Check(req)
	require.NoError(t, err)
	assert.Nil(t, res)
}

func TestSave(t *testing.T) {
	s.FlushAll()
	err := client.Save(&render.Result{
		URL:  "https://netlify.com/",
		HTML: "<html></html>",
		Etag: "etagetag",
	}, 24*time.Hour)
	require.NoError(t, err)
	etag := s.HGet("https://netlify.com/", "Etag")
	assert.Equal(t, "etagetag", etag)
}

func TestSaveExpire(t *testing.T) {
	s.FlushAll()
	err := client.Save(&render.Result{
		URL:  "https://netlify.com/",
		HTML: "<html></html>",
		Etag: "etagetag",
	}, 24*time.Hour)
	require.NoError(t, err)
	etag := s.HGet("https://netlify.com/", "Etag")
	assert.Equal(t, "etagetag", etag)
	s.FastForward(24 * time.Hour)
	etag = s.HGet("https://netlify.com/", "Etag")
	assert.Empty(t, etag)
}

func TestCheckError(t *testing.T) {
	s.Close()
	req := httptest.NewRequest("GET", "http://example.com/https://netlify.com/", nil)
	// TODO having to do this kinda sucks
	req.URL.Path = req.URL.Path[1:]
	_, err := client.Check(req)
	assert.NotNil(t, err)

	req.Header.Add("If-None-Match", "etagetag")
	_, err = client.Check(req)
	assert.NotNil(t, err)
}
