package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/wirepair/gcd"
	"github.com/wirepair/gcd/gcdapi"
)

type Renderer struct {
	path     string
	debugger *gcd.Gcd
	Timeout  time.Duration
}

func NewRenderer() (*Renderer, error) {
	chromePath := "/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary"
	if os.Getenv("CHROME_PATH") != "" {
		chromePath = os.Getenv("CHROME_PATH")
	}

	debugger := gcd.NewChromeDebugger()
	debugger.AddFlags([]string{"--headless", "--disable-gpu"})
	debugger.StartProcess(chromePath, os.TempDir(), "9222")

	return &Renderer{
		path:     chromePath,
		debugger: debugger,
		Timeout:  60 * time.Second,
	}, nil
}

func (r *Renderer) Destroy() {
	r.debugger.ExitProcess()
}

type RenderResult struct {
	HTML     string
	Duration time.Duration
}

func (r *Renderer) Render(url string) (*RenderResult, error) {
	start := time.Now()
	done := make(chan bool)
	var html string
	var err error

	tab, err := r.debugger.NewTab()
	if err != nil {
		return nil, errors.Wrap(err, "creating new tab failed")
	}
	defer r.debugger.CloseTab(tab)

	tab.Subscribe("Page.loadEventFired", func(target *gcd.ChromeTarget, v []byte) {
		defer func() { done <- true }()
		var doc *gcdapi.DOMNode

		dom := tab.DOM
		if doc, err = dom.GetDocument(1, false); err != nil {
			err = errors.Wrap(err, "getting tab document failed")
			return
		}
		if html, err = dom.GetOuterHTML(doc.NodeId); err != nil {
			err = errors.Wrap(err, "get outer html for document failed")
			return
		}
	})

	if _, err = tab.Page.Enable(); err != nil {
		return nil, errors.Wrap(err, "enabling tab page failed")
	}
	if _, err = tab.Page.Navigate(url, ""); err != nil {
		return nil, errors.Wrap(err, "navigating to url failed: "+url)
	}

	select {
	case <-time.After(60 * time.Second):
		return nil, fmt.Errorf("timed out waiting for page load")
	case <-done:
	}

	if err != nil {
		return nil, err
	}
	return &RenderResult{
		HTML:     html,
		Duration: time.Since(start),
	}, nil
}
