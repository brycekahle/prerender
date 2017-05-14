package main

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/brycekahle/prerender/render"
)

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
	r.URL.Path = reqURL

	res, err := getData(r)
	writeResult(res, err, w)
}

func getData(r *http.Request) (*render.Result, error) {
	cache := getCache(r.Context())
	if cache != nil {
		res, err := cache.Check(r)
		if err != nil || res != nil {
			return res, err
		}
	}

	renderer := getRenderer(r.Context())
	res, err := renderer.Render(r.URL.Path)
	if err == nil && res.Status == http.StatusOK && cache != nil {
		err = cache.Save(res, 24*time.Hour)
	}
	return res, err
}

func writeResult(res *render.Result, err error, w http.ResponseWriter) {
	if err != nil {
		if err == render.ErrPageLoadTimeout {
			w.WriteHeader(http.StatusGatewayTimeout)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			log.WithError(err).Errorf("error rendering")
		}
		return
	}

	if res.Status != http.StatusOK {
		w.WriteHeader(res.Status)
		return
	}
	if res.Etag != "" {
		w.Header().Add("Etag", res.Etag)
	}
	if res.HTML != "" {
		fmt.Fprint(w, res.HTML)
	}
}
