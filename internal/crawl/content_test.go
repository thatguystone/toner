package crawl

import (
	"net/http"
	"testing"
)

func TestContentOpaqueURL(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			str: `<!DOCTYPE html>
				<a href="mailto:a@stoney.io">a@stoney.io</a>`,
		})

	ct.NotPanics(func() {
		ct.run(mux)
	})

	css := ct.fs.SReadFile("output/index.html")
	ct.Contains(css, `href="mailto:a@stoney.io"`)
}

func TestContentInvalidEntryURL(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			str:  `<!DOCTYPE html>`,
		})

	ct.Panics(func() {
		ct.run(mux, "://drunk-url")
	})
}

func TestContentExternalEntryURL(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			str:  `<!DOCTYPE html>`,
		})

	ct.Panics(func() {
		ct.run(mux, "http://example.com")
	})
}

func TestContentInvalidRelURL(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			str: `<!DOCTYPE html>
				<a href="://drunk-url">Test</a>`,
		})

	ct.Panics(func() {
		ct.run(mux)
	})
}

func TestContentRedirectLoop(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			str: `<!DOCTYPE html>
				<a href="redirect">Redirect</a>`,
		},
		testHandler{
			path: "/redirect",
			handler: http.RedirectHandler("/redirect",
				http.StatusMovedPermanently),
		})

	ct.Panics(func() {
		ct.run(mux)
	})
}

func TestContentInvalidRedirect(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			str: `<!DOCTYPE html>
				<a href="redirect">Redirect</a>`,
		},
		testHandler{
			path: "/redirect",
			fn: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "://herp-derp")
				w.WriteHeader(http.StatusFound)
			},
		})

	ct.Panics(func() {
		ct.run(mux)
	})
}

func TestContentInvalidLastModified(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			fn: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Last-Modified", "What time is it?!")
				w.WriteHeader(http.StatusOK)
			},
		})

	ct.NotPanics(func() {
		ct.run(mux)
	})
}

func TestContentInvalidContentType(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			fn: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "cookies and cake")
				w.WriteHeader(http.StatusOK)
			},
		})

	ct.Panics(func() {
		ct.run(mux)
	})
}

func TestContent500(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			fn: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
		})

	ct.Panics(func() {
		ct.run(mux)
	})
}
