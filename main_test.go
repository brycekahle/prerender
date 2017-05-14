package main

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/brycekahle/prerender/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {
	log.SetLevel(log.PanicLevel)
	os.Exit(m.Run())
}

func TestEmptyURL(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	w := httptest.NewRecorder()
	handle(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestInvalidURL(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/asdfasdf", nil)
	w := httptest.NewRecorder()
	handle(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRelativeURL(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/netlify.com/", nil)
	w := httptest.NewRecorder()
	handle(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

type MockRenderer struct {
	mock.Mock
}

func (r *MockRenderer) Render(url string) (*render.Result, error) {
	args := r.Called(url)
	err := args.Error(0)
	if err != nil {
		return nil, err
	}

	return &render.Result{
		URL:      url,
		Status:   args.Int(1),
		HTML:     args.String(2),
		Etag:     args.String(3),
		Duration: time.Duration(args.Int(4)),
	}, nil
}
func (r *MockRenderer) Close()                             {}
func (r *MockRenderer) SetPageLoadTimeout(t time.Duration) {}

func TestETag(t *testing.T) {
	r := new(MockRenderer)
	req := httptest.NewRequest("GET", "http://example.com/https://netlify.com/", nil)
	ctx := setRenderer(req.Context(), r)
	w := httptest.NewRecorder()

	r.On("Render", "https://netlify.com/").Return(nil, http.StatusOK, "<html></html>", "etagetag", 1).Once()
	handle(w, req.WithContext(ctx))

	resp := w.Result()
	r.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "etagetag", resp.Header.Get("Etag"))
}

func TestNon200(t *testing.T) {
	r := new(MockRenderer)
	req := httptest.NewRequest("GET", "http://example.com/https://netlify.com/", nil)
	ctx := setRenderer(req.Context(), r)
	w := httptest.NewRecorder()

	r.On("Render", "https://netlify.com/").Return(nil, http.StatusNotFound, "", "", 1).Once()
	handle(w, req.WithContext(ctx))

	resp := w.Result()
	r.AssertExpectations(t)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestPageLoadTimeout(t *testing.T) {
	r := new(MockRenderer)
	req := httptest.NewRequest("GET", "http://example.com/https://netlify.com/", nil)
	ctx := setRenderer(req.Context(), r)
	w := httptest.NewRecorder()

	r.On("Render", "https://netlify.com/").Return(render.ErrPageLoadTimeout).Once()
	handle(w, req.WithContext(ctx))

	resp := w.Result()
	r.AssertExpectations(t)
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
}

func TestRenderError(t *testing.T) {
	r := new(MockRenderer)
	req := httptest.NewRequest("GET", "http://example.com/https://netlify.com/", nil)
	ctx := setRenderer(req.Context(), r)
	w := httptest.NewRecorder()

	r.On("Render", "https://netlify.com/").Return(errors.New("random error")).Once()
	handle(w, req.WithContext(ctx))

	resp := w.Result()
	r.AssertExpectations(t)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

type MockCache struct {
	mock.Mock
}

func (c *MockCache) Check(r *http.Request) (*render.Result, error) {
	args := c.Called(r)
	err := args.Error(0)
	if err != nil {
		return nil, err
	}

	if args.Int(1) == 0 {
		return nil, nil
	}

	return &render.Result{
		URL:      r.URL.Path,
		Status:   args.Int(1),
		HTML:     args.String(2),
		Etag:     args.String(3),
		Duration: time.Duration(args.Int(4)),
	}, nil
}

func (c *MockCache) Save(res *render.Result, ttl time.Duration) error {
	args := c.Called(res, ttl)
	return args.Error(0)
}

func TestCacheHit(t *testing.T) {
	c := new(MockCache)
	req := httptest.NewRequest("GET", "http://example.com/https://netlify.com/", nil)
	ctx := setCache(req.Context(), c)
	w := httptest.NewRecorder()

	c.On("Check", mock.MatchedBy(func(r *http.Request) bool {
		return r.Host == "example.com"
	})).Return(nil, http.StatusOK, "<html></html>", "etagetag", 0).Once()
	handle(w, req.WithContext(ctx))

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	c.AssertExpectations(t)
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	assert.Equal(t, resp.Header.Get("Etag"), "etagetag")
	assert.Equal(t, string(body), "<html></html>")
}

func TestCacheMiss(t *testing.T) {
	r := new(MockRenderer)
	c := new(MockCache)
	req := httptest.NewRequest("GET", "http://example.com/https://netlify.com/", nil)
	ctx := setCache(req.Context(), c)
	ctx = setRenderer(ctx, r)
	w := httptest.NewRecorder()

	c.On("Check", mock.MatchedBy(func(r *http.Request) bool {
		return r.Host == "example.com"
	})).Return(nil, 0).Once()
	c.On("Save", mock.MatchedBy(func(r *render.Result) bool {
		return r.HTML == "<html></html>"
	}), 24*time.Hour).Return(nil)
	r.On("Render", "https://netlify.com/").Return(nil, http.StatusOK, "<html></html>", "etagetag", 1).Once()

	handle(w, req.WithContext(ctx))

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	c.AssertExpectations(t)
	r.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "etagetag", resp.Header.Get("Etag"))
	assert.Equal(t, "<html></html>", string(body))
}

func TestCacheCheckError(t *testing.T) {
	c := new(MockCache)
	req := httptest.NewRequest("GET", "http://example.com/https://netlify.com/", nil)
	ctx := setCache(req.Context(), c)
	w := httptest.NewRecorder()

	c.On("Check", mock.MatchedBy(func(r *http.Request) bool {
		return r.Host == "example.com"
	})).Return(errors.New("cache error")).Once()
	handle(w, req.WithContext(ctx))

	resp := w.Result()
	c.AssertExpectations(t)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestCacheSaveError(t *testing.T) {
	r := new(MockRenderer)
	c := new(MockCache)
	req := httptest.NewRequest("GET", "http://example.com/https://netlify.com/", nil)
	ctx := setCache(req.Context(), c)
	ctx = setRenderer(ctx, r)
	w := httptest.NewRecorder()

	c.On("Check", mock.MatchedBy(func(r *http.Request) bool {
		return r.Host == "example.com"
	})).Return(nil, 0).Once()
	c.On("Save", mock.MatchedBy(func(r *render.Result) bool {
		return r.HTML == "<html></html>"
	}), 24*time.Hour).Return(errors.New("save error"))
	r.On("Render", "https://netlify.com/").Return(nil, http.StatusOK, "<html></html>", "etagetag", 1).Once()

	handle(w, req.WithContext(ctx))

	resp := w.Result()
	c.AssertExpectations(t)
	r.AssertExpectations(t)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
