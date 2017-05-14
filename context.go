package main

import (
	"context"

	"github.com/brycekahle/prerender/cache"
	"github.com/brycekahle/prerender/render"
)

type contextKey string

func (c contextKey) String() string {
	return "prerender context key " + string(c)
}

const (
	rendererKey = contextKey("renderer")
	cacheKey    = contextKey("cache")
)

func setRenderer(ctx context.Context, r render.Renderer) context.Context {
	return context.WithValue(ctx, rendererKey, r)
}
func getRenderer(ctx context.Context) render.Renderer {
	r, _ := ctx.Value(rendererKey).(render.Renderer)
	return r
}

func setCache(ctx context.Context, r cache.Cache) context.Context {
	return context.WithValue(ctx, cacheKey, r)
}
func getCache(ctx context.Context) cache.Cache {
	r := ctx.Value(cacheKey)
	if r != nil {
		return r.(cache.Cache)
	}
	return nil
}
