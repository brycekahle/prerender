package main

import (
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brycekahle/prerender/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEmptyURL(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	w := httptest.NewRecorder()
	handle(w, req)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, 400)
}

func TestInvalidURL(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/asdfasdf", nil)
	w := httptest.NewRecorder()
	handle(w, req)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, 400)
}

func TestRelativeURL(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/netlify.com/", nil)
	w := httptest.NewRecorder()
	handle(w, req)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, 400)
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

	r.On("Render", "https://netlify.com/").Return(nil, 200, "<html></html>", "etagetag", 1).Once()
	handle(w, req.WithContext(ctx))

	resp := w.Result()
	r.AssertExpectations(t)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Etag"), "etagetag")
}

func TestNon200(t *testing.T) {
	r := new(MockRenderer)
	req := httptest.NewRequest("GET", "http://example.com/https://netlify.com/", nil)
	ctx := setRenderer(req.Context(), r)
	w := httptest.NewRecorder()

	r.On("Render", "https://netlify.com/").Return(nil, 404, "", "", 1).Once()
	handle(w, req.WithContext(ctx))

	resp := w.Result()
	r.AssertExpectations(t)
	assert.Equal(t, resp.StatusCode, 404)
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
	assert.Equal(t, resp.StatusCode, 504)
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
	assert.Equal(t, resp.StatusCode, 500)
}
