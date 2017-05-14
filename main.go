package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"context"

	log "github.com/Sirupsen/logrus"
	"github.com/brycekahle/prerender/render"
	"github.com/felixge/httpsnoop"
)

func main() {
	var renderer render.Renderer
	var err error
	renderer, err = render.NewRenderer()
	if err != nil {
		log.Fatal(err)
	}
	defer renderer.Close()

	// a custom handler is necessary because ServeMux redirects // to /
	// in all urls, regardless of escaping
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := setRenderer(r.Context(), renderer)
		handle(w, r.WithContext(ctx))
	})
	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := httpsnoop.CaptureMetrics(handler, w, r)
		log.WithFields(log.Fields{
			"method":   r.Method,
			"path":     r.URL.Path,
			"status":   m.Code,
			"duration": m.Duration.Nanoseconds(),
			"size":     m.Written,
		}).Infof("Completed request")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	l := fmt.Sprintf(":%s", port)
	log.Printf("listening on %s", l)
	server := http.Server{Addr: l, Handler: wrappedHandler}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		log.Info("signal caught, shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()
	err = server.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Error(err)
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	reqURL := r.URL.Path[1:]
	if reqURL == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "url is required")
		return
	}

	u, err := url.Parse(reqURL)
	if err != nil || !u.IsAbs() {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Invalid URL")
		return
	}

	renderer := getRenderer(r.Context())
	res, err := renderer.Render(reqURL)
	if err != nil {
		if err == render.ErrPageLoadTimeout {
			w.WriteHeader(http.StatusGatewayTimeout)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
		}
		return
	}
	if res.Status != 200 {
		w.WriteHeader(res.Status)
		return
	}
	if res.Etag != "" {
		w.Header().Add("ETag", res.Etag)
	}
	fmt.Fprint(w, res.HTML)
}
