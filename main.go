package main

import (
	"fmt"
	"net/http"
	"net/url"

	log "github.com/Sirupsen/logrus"
	"github.com/felixge/httpsnoop"
)

func main() {
	var renderer *Renderer
	var err error
	renderer, err = NewRenderer()
	if err != nil {
		log.Fatal(err)
	}
	defer renderer.Destroy()

	// a custom handler is necessary to avoid stripping of double slashes
	// ServeMux redirects // to / in all urls, regardless of escaping
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := setRenderer(r.Context(), renderer)
		render(w, r.WithContext(ctx))
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

	log.Fatal(http.ListenAndServe(":8000", wrappedHandler))
}

func render(w http.ResponseWriter, r *http.Request) {
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
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err)
		return
	}
	fmt.Fprint(w, res.HTML)
}
