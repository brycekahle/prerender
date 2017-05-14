package main

import (
	"context"

	"github.com/brycekahle/prerender/render"
)

type contextKey string

func (c contextKey) String() string {
	return "prerender context key " + string(c)
}

const (
	rendererKey = contextKey("renderer")
)

func setRenderer(ctx context.Context, r render.Renderer) context.Context {
	return context.WithValue(ctx, rendererKey, r)
}
func getRenderer(ctx context.Context) render.Renderer {
	r, _ := ctx.Value(rendererKey).(render.Renderer)
	return r
}
