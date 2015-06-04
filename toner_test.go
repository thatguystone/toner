package toner

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/thatguystone/assert"
)

type testToner struct {
	*Toner
	a assert.A
}

type testFile struct {
	dir bool
	p   string
	sc  string
	bc  []byte
}

var (
	gifBin = []byte{
		0x47, 0x49, 0x46, 0x38, 0x37, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x2c, 0x00, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01,
		0x00, 0x3b,
	}
	pngBin = []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00,
		0x0d, 0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x01, 0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde,
		0x00, 0x00, 0x00, 0x09, 0x70, 0x48, 0x59, 0x73, 0x00, 0x00, 0x0b,
		0x13, 0x00, 0x00, 0x0b, 0x13, 0x01, 0x00, 0x9a, 0x9c, 0x18, 0x00,
		0x00, 0x00, 0x07, 0x74, 0x49, 0x4d, 0x45, 0x07, 0xdf, 0x06, 0x03,
		0x16, 0x11, 0x34, 0xd8, 0x8f, 0x56, 0x73, 0x00, 0x00, 0x00, 0x19,
		0x74, 0x45, 0x58, 0x74, 0x43, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x74,
		0x00, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x20, 0x77, 0x69,
		0x74, 0x68, 0x20, 0x47, 0x49, 0x4d, 0x50, 0x57, 0x81, 0x0e, 0x17,
		0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63,
		0xf8, 0xff, 0xff, 0x3f, 0x00, 0x05, 0xfe, 0x02, 0xfe, 0xdc, 0xcc,
		0x59, 0xe7, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}

	basicSite = []testFile{
		testFile{
			p: "content/blog/post1.md",
			sc: "---\ntitle: test\n---\n" +
				"# post 1\n" +
				"{{ \"post 1\" }}\n" +
				"{% js \"post1.js\" %}\n" +
				"{% css \"post1.css\" %}\n",
		},
		testFile{
			p:  "content/blog/post1.js",
			sc: "post 1 js stuff!",
		},
		testFile{
			p:  "content/blog/post1.css",
			sc: "post 1 css stuff!",
		},
		testFile{
			p: "content/blog/post2.md",
			sc: "---\ntitle: post2\n---\n" +
				"# post 2\n" +
				"{{ \"post 2\" }}\n" +
				"{% js \"post2.js\" %}\n" +
				"{% css \"post2.css\" %}\n",
		},
		testFile{
			p:  "content/blog/post2.js",
			sc: "post 2 js stuff!",
		},
		testFile{
			p:  "content/blog/post2.css",
			sc: "post 2 css stuff!",
		},
		testFile{
			p: "layouts/blog/_single.html",
			sc: "<body>" +
				"Blog layout: {{ Content }}" +
				`{% img "img.png" %}` +
				"</body>" +
				"{% js \"layout.js\" %}\n" +
				"{% css \"layout.css\" %}\n" +
				"{% js_all %}\n" +
				"{% css_all %}\n" +
				"{% js \"layout2.js\" %}\n" +
				"{% css \"layout2.css\" %}\n",
		},
		testFile{
			p:  "layouts/blog/layout.js",
			sc: "layout js!",
		},
		testFile{
			p:  "layouts/blog/layout.css",
			sc: "layout css!",
		},
		testFile{
			p:  "layouts/blog/layout2.js",
			sc: "---\nrender: true\nval: layout 2\n---\n{{ Page.Meta.val }} js!",
		},
		testFile{
			p:  "layouts/blog/layout2.css",
			sc: "{{ Page.Meta.val }} css!",
		},
		testFile{
			p:  "layouts/blog/layout2.meta",
			sc: "render: true\nval: layout 2",
		},
		testFile{
			p:  "layouts/blog/img.png",
			bc: pngBin,
		},
		testFile{
			dir: true,
			p:   "content/blog/empty",
		},
	}
)

func testNew(t *testing.T, build bool, files []testFile) *testToner {
	cfg := Config{
		Root:       filepath.Join("test_data", assert.GetTestName()),
		MinifyHTML: true,
	}

	tt := &testToner{
		Toner: New(cfg),
		a:     assert.From(t),
	}

	tt.createFiles(files)

	if build {
		_, errs := tt.Build()
		tt.a.MustEqual(0, len(errs), "failed to build site; errs=%v", errs)
	}

	return tt
}

func (tt *testToner) createFiles(files []testFile) {
	for _, file := range files {
		p := filepath.Join(tt.cfg.Root, file.p)

		if file.dir {
			err := os.MkdirAll(p, 0700)
			tt.a.MustNotError(err, "failed to create dir %s", p)
		} else {
			f, err := fCreate(p, createFlags, 0600)
			tt.a.MustNotError(err, "failed to create file %s", p)

			if len(file.sc) > 0 {
				_, err = f.Write([]byte(file.sc))
			} else {
				_, err = f.Write(file.bc)
			}

			tt.a.MustNotError(err, "failed to write file %s", p)

			err = f.Close()
			tt.a.MustNotError(err, "failed to write file %s", p)
		}
	}
}

func (tt *testToner) exists(path string) {
	p := filepath.Join(tt.cfg.Root, path)
	_, err := os.Stat(p)
	tt.a.True(err == nil, "file %s does not exist", p)
}

func (tt *testToner) checkFile(path, contents string) {
	f, err := os.Open(filepath.Join(tt.cfg.Root, path))
	tt.a.MustNotError(err, "failed to open %s", path)
	defer f.Close()

	fc, err := ioutil.ReadAll(f)
	tt.a.MustNotError(err, "failed to read %s", path)

	tt.a.Equal(contents, string(fc), "content mismatch for %s", path)
}

func (tt *testToner) checkBinFile(path string, contents []byte) {

}

func (tt *testToner) cleanup() {
	os.RemoveAll(tt.cfg.Root)
}

func TestEmptySite(t *testing.T) {
	t.Parallel()
	tt := testNew(t, true, nil)
	defer tt.cleanup()
}

func TestBasicSite(t *testing.T) {
	t.Parallel()
	tt := testNew(t, true, basicSite)
	defer tt.cleanup()

	tt.exists("public/static/js/blog/post1.js")
	tt.exists("public/static/js/blog/post2.js")
	tt.exists("public/static/css/blog/post1.css")
	tt.exists("public/static/css/blog/post2.css")

	tt.checkFile("public/blog/post1.html",
		`Blog layout:<h1>post 1</h1><p>post 1<script src=../static/js/blog/post1.js></script><link rel=stylesheet href=../static/css/blog/post1.css></p><img src=../static/img/layout/blog/img.png style=width:1px;height:1px;><script src=../../static/js/layout/blog/layout.js></script><link rel=stylesheet href=../../static/css/layout/blog/layout.css><script src=../../static/js/layout/blog/layout2.js></script><link rel=stylesheet href=../../static/css/layout/blog/layout2.css>`)
	tt.checkFile("public/blog/post2.html",
		`Blog layout:<h1>post 2</h1><p>post 2<script src=../static/js/blog/post2.js></script><link rel=stylesheet href=../static/css/blog/post2.css></p><img src=../static/img/layout/blog/img.png style=width:1px;height:1px;><script src=../../static/js/layout/blog/layout.js></script><link rel=stylesheet href=../../static/css/layout/blog/layout.css><script src=../../static/js/layout/blog/layout2.js></script><link rel=stylesheet href=../../static/css/layout/blog/layout2.css>`)

	tt.checkFile("public/static/js/layout/blog/layout2.js",
		`layout 2 js!`)
	tt.checkFile("public/static/css/layout/blog/layout2.css",
		`layout 2 css!`)
}
