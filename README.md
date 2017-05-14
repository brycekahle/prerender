# Prerender

Prerender exposes a simple API that will pre-render webpages and return them. This is primarily useful for SEO purposes as search engines prefer full pages instead of the shells used by many single page apps (SPAs).

## Installation

prerender can be installed from source:
```
go get github.com/brycekahle/prerender
```

## Prerequisites

- Google Chrome 59+ (you may use [Chrome Canary](https://www.google.com/chrome/browser/canary.html) while headless Chrome is in beta)

## Usage

By default, prerender will look for Chrome Canary at `/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary`.
You can override this by specifying the `CHROME_PATH` environment variable.

The `PORT` environment variable controls what port `prerender` listens on. The default value is `8000`.

```
$ prerender
```

