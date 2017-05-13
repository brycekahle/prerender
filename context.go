package main

import "context"

type contextKey string

func (c contextKey) String() string {
	return "prerender context key " + string(c)
}

const (
	rendererKey = contextKey("renderer")
)

func setRenderer(ctx context.Context, r *Renderer) context.Context {
	return context.WithValue(ctx, rendererKey, r)
}
func getRenderer(ctx context.Context) *Renderer {
	r, _ := ctx.Value(rendererKey).(*Renderer)
	return r
}
