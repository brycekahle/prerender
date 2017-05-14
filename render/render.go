package render

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/wirepair/gcd"
	"github.com/wirepair/gcd/gcdapi"
)

// ErrPageLoadTimeout is returned when the page did not fire the "load" event
// before the timeout expired
var ErrPageLoadTimeout = errors.New("timed out waiting for page load")

// Renderer is the interface implemented by renderers capable of
// fetching a webpage and returning the HTML after JavaScript has run
type Renderer interface {
	Render(string) (*Result, error)
	SetPageLoadTimeout(time.Duration)
	Close()
}

// Result describes the result of the rendering operation
type Result struct {
	HTML     string
	Status   int
	Etag     string
	Duration time.Duration
}

type chromeRenderer struct {
	debugger *gcd.Gcd
	timeout  time.Duration
}

// NewRenderer launches a headless Google Chrome instance
// ready to render pages
func NewRenderer() (Renderer, error) {
	chromePath := "/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary"
	if os.Getenv("CHROME_PATH") != "" {
		chromePath = os.Getenv("CHROME_PATH")
	}

	debugger := gcd.NewChromeDebugger()
	debugger.SetTerminationHandler(func(reason string) {
		log.Printf("chrome termination: %s\n", reason)
	})
	debugger.AddFlags([]string{"--headless", "--disable-gpu"})
	debugger.StartProcess(chromePath, os.TempDir(), "9222")

	return &chromeRenderer{
		debugger: debugger,
		timeout:  60 * time.Second,
	}, nil
}

func (r *chromeRenderer) SetPageLoadTimeout(t time.Duration) {
	r.timeout = t
}

func (r *chromeRenderer) Close() {
	r.debugger.ExitProcess()
}

func (r *chromeRenderer) Render(url string) (*Result, error) {
	start := time.Now()
	navigated := make(chan bool)
	res := Result{}
	var err error

	tab, err := r.debugger.NewTab()
	if err != nil {
		return nil, errors.Wrap(err, "creating new tab failed")
	}
	defer r.debugger.CloseTab(tab)
	// tab.Debug(true)

	tab.Subscribe("Page.loadEventFired", func(target *gcd.ChromeTarget, v []byte) {
		navigated <- true
	})

	tab.Subscribe("Network.responseReceived", func(target *gcd.ChromeTarget, v []byte) {
		event := &gcdapi.NetworkResponseReceivedEvent{}
		if err = json.Unmarshal(v, event); err != nil {
			err = errors.Wrap(err, "getting network response failed")
			return
		}
		r := event.Params.Response
		res.Status = int(r.Status)
		if etag, ok := r.Headers["Etag"]; ok {
			res.Etag = etag.(string)
		}
	})

	if _, err = tab.Page.Enable(); err != nil {
		return nil, errors.Wrap(err, "enabling tab page failed")
	}
	if _, err = tab.Network.Enable(-1, -1); err != nil {
		return nil, errors.Wrap(err, "enabling tab network failed")
	}
	if _, err = tab.Page.Navigate(url, ""); err != nil {
		return nil, errors.Wrap(err, "navigating to url failed: "+url)
	}

	select {
	case <-time.After(r.timeout):
		return nil, ErrPageLoadTimeout
	case <-navigated:
	}

	if res.Status == 200 {
		doc, err := tab.DOM.GetDocument(1, false)
		if err != nil {
			return nil, errors.Wrap(err, "getting tab document failed")
		}
		html, err := tab.DOM.GetOuterHTML(doc.NodeId)
		if err != nil {
			return nil, errors.Wrap(err, "get outer html for document failed")
		}
		res.HTML = html

		if res.Etag == "" {
			hash := md5.Sum([]byte(res.HTML))
			res.Etag = hex.EncodeToString(hash[:])
		}
	}

	res.Duration = time.Since(start)
	return &res, nil
}
