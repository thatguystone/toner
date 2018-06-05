package crawl

import (
	"net/http"
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestTransformHTMLInlineStyles(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body:     `<div style="background: url(img.gif);"></div>`,
			},
			"/img.gif": stringHandler{
				contType: gifType,
				body:     string(gifBin),
			},
		}),
		Entries: []string{"/"},
		Output:  fs.Path("."),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	fs.FileExists("img.gif")
}

func TestTransformHTMLCoverage(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body: `` +
					`<style>body { background: url(/r/); }</style>` +
					`<a href="/r/"></a>` +
					`<img src="/r/" srcset="/r/, /r/ 2x">` +
					`<div style="background: url(/r/)"></div>`,
			},
			"/r/": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					http.Redirect(w, r, "/f/", http.StatusFound)
				}),
			"/f/": stringHandler{
				contType: htmlType,
				body:     `redirected`,
			},
		}),
		Entries: []string{"/"},
		Output:  fs.Path("."),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	index := fs.SReadFile("index.html")
	c.NotContains(index, "/r/")
	c.Contains(index, "/f/")
}