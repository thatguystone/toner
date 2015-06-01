package toner

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	p2 "github.com/flosch/pongo2"
)

type contents struct {
	s    *site
	mtx  sync.Mutex
	srcs map[string]*content // All available content: srcPath -> content
	dsts map[string]*content // All rendered content: dstPath -> content
}

type content struct {
	cs         *contents
	f          file
	cpath      string
	metaEnd    int
	meta       meta
	tplContext p2.Context
	gen        interface{}
}

const (
	layoutPubDir = "layout"
	staticPubDir = "static"
	themePubDir  = "theme"
)

var (
	metaDelim = []byte("---\n")

	bannedContentTags = []string{
		"css_all",
		"extends",
		"js_all",
	}
)

func (cs *contents) init(s *site) {
	cs.s = s
	cs.srcs = map[string]*content{}
	cs.dsts = map[string]*content{}
}

func (cs *contents) add(f file) error {
	ext := filepath.Ext(f.srcPath)
	cpath := fChangeExt(f.dstPath, "")
	f.dstPath = filepath.Join(cs.s.cfg.Root, cs.s.cfg.PublicDir, f.dstPath)

	c := &content{
		cs:    cs,
		f:     f,
		cpath: cpath,
		tplContext: p2.Context{
			siteKey: cs.s,
		},
	}

	c.tplContext[contentKey] = c
	c.gen = cs.getGenerator(c, ext)

	cs.mtx.Lock()
	cs.srcs[f.srcPath] = c
	cs.mtx.Unlock()

	return c.load()
}

func (cs *contents) find(currFile, rel string) (*content, error) {
	// No lock needed: find should only be called after ALL content has been
	// added
	src := filepath.Join(currFile, "../", rel)
	c := cs.srcs[src]

	if c == nil {
		return nil, fmt.Errorf("content `%s` from rel path `%s` not found", src, rel)
	}

	return c, nil
}

func (cs *contents) getGenerator(c *content, ext string) interface{} {
	for _, g := range generators {
		cg := g.getGenerator(c, ext)
		if cg != nil {
			return cg
		}
	}

	panic(fmt.Errorf("no content generator found for %s", ext))
}

func (cs *contents) claimDest(dst string, c *content) (alreadyClaimed bool, err error) {
	cs.mtx.Lock()

	if co, ok := cs.dsts[dst]; ok {
		if co == c {
			alreadyClaimed = true
		} else {
			err = fmt.Errorf("content conflict: destination file `%s` already generated by `%s`",
				dst,
				co.f.srcPath)
		}
	} else {
		cs.dsts[dst] = c
	}

	cs.mtx.Unlock()

	return
}

func (c *content) load() error {
	mp := fChangeExt(c.f.srcPath, ".meta")
	if fExists(mp) {
		meta, err := ioutil.ReadFile(mp)
		if err != nil {
			return err
		}

		err = c.processMeta(meta, true)
		if err != nil {
			return err
		}
	}

	f, err := os.Open(c.f.srcPath)
	if err != nil {
		return err
	}

	defer f.Close()
	r := bufio.NewReader(f)

	del := make([]byte, len(metaDelim))

	i, err := r.Read(del)
	if err != nil {
		if err != io.EOF {
			return err
		}

		return nil
	}

	if !bytes.Equal(metaDelim, del[:i]) {
		return nil
	}

	mb := &bytes.Buffer{}
	mb.Write(metaDelim)

	for err == nil {
		var l []byte
		l, err = r.ReadBytes('\n')
		mb.Write(l)

		if bytes.Equal(metaDelim, l) {
			break
		}
	}

	if err != nil {
		if err == io.EOF {
			err = fmt.Errorf("metadata missing closing `%s`", metaDelim)
		}

		return fmt.Errorf("failed to read content metadata: %v", err)
	}

	c.metaEnd = mb.Len()
	return c.processMeta(mb.Bytes(), false)
}

func (c *content) processMeta(m []byte, isMetaFile bool) error {
	if !bytes.HasPrefix(m, metaDelim) && !isMetaFile {
		return nil
	}

	end := bytes.Index(m[3:], metaDelim)
	if end == -1 && !isMetaFile {
		return nil
	}

	m = bytes.TrimSpace(m[3 : end+3])
	return c.meta.merge(m)
}

func (c *content) relDestTo(o *content) string {
	od := filepath.Dir(o.f.dstPath)
	d, f := filepath.Split(c.f.dstPath)

	rel, err := filepath.Rel(od, d)
	if err != nil {
		panic(err)
	}

	return filepath.Join(rel, f)
}

func (c *content) claimDest(ext string) (string, bool, error) {
	dst := c.f.dstPath
	if ext != "" {
		dst = fChangeExt(dst, ext)
	}

	alreadyClaimed, err := c.cs.claimDest(dst, c)
	return dst, alreadyClaimed, err
}

func (c *content) templatize(w io.Writer) error {
	b, err := ioutil.ReadFile(c.f.srcPath)
	if err != nil {
		return err
	}

	// Run in total isolation from everything
	set := p2.NewSet("temp")

	for _, t := range bannedContentTags {
		set.BanTag(t)
	}

	tpl, err := set.FromString(string(b[c.metaEnd:]))
	if err != nil {
		return err
	}

	return tpl.ExecuteWriter(c.tplContext, w)
}

func (c *content) readAll(w io.Writer) error {
	f, err := os.Open(c.f.srcPath)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.Seek(int64(c.metaEnd), 0)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, f)
	return err
}
