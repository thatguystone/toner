package acryliclib

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

type contentGenImg struct {
	c      *content
	mtx    sync.Mutex
	scaled map[string]image.Point
}

type img struct {
	src  string
	ext  string
	w    int
	h    int
	crop imgCrop
}

type imgCrop int

const (
	cropNone imgCrop = iota
	cropLeft
	cropCentered
	cropLen
)

var imgExts = []string{
	".bmp",
	".gif",
	".jpeg",
	".jpg",
	".png",
	".tiff",
}

func (gi contentGenImg) getGenerator(c *content, ext string) interface{} {
	for _, e := range imgExts {
		if ext == e {
			return contentGenImg{
				c:      c,
				scaled: map[string]image.Point{},
			}
		}
	}

	return nil
}

func (gi contentGenImg) generatePage() (string, error) {
	// c := gi.c
	// s := c.cs.s

	// TODO(astone): generate image pages

	return "", nil
}

func (contentGenImg) humanName() string {
	return "image"
}

func (gi contentGenImg) scale(img img) (w, h int, dstPath string, err error) {
	c := gi.c

	ext := gi.getNewExt(img)
	dstPath, alreadyClaimed, err := c.claimStaticDest("img", ext)
	if err != nil {
		return
	}

	if alreadyClaimed {
		for {
			gi.mtx.Lock()
			p, ok := gi.scaled[dstPath]
			gi.mtx.Unlock()

			if ok {
				w, h = p.X, p.Y
				return
			}

			time.Sleep(time.Millisecond)
		}
	}

	w, h = img.w, img.h
	defer func() {
		gi.mtx.Lock()
		gi.scaled[dstPath] = image.Point{X: w, Y: h}
		gi.mtx.Unlock()
	}()

	if !fDestChanged(c.f.srcPath, dstPath) {
		var f *os.File
		f, err = os.Open(dstPath)
		if err != nil {
			return
		}

		defer f.Close()

		var ig image.Image
		ig, err = imaging.Decode(f)
		if err != nil {
			return
		}

		bounds := ig.Bounds()
		w = bounds.Dx()
		h = bounds.Dy()
		return
	}

	c.cs.s.stats.addImg()

	f, err := os.Open(c.f.srcPath)
	if err != nil {
		return
	}

	defer f.Close()

	ig, err := imaging.Decode(f)
	if err != nil {
		return
	}

	switch {
	case img.w == 0 && img.h == 0:
		// No resizing

	case img.w != 0 && img.h != 0 && img.crop != cropNone:
		// It doesn't make sense to crop if full dimensions aren't given since
		// it's just scaling if a dimension is missing.
		ig = gi.thumbnailImage(ig, img)

	default:
		ig = gi.resizeImage(ig, img)
	}

	bounds := ig.Bounds()
	w = bounds.Dx()
	h = bounds.Dy()

	err = gi.saveImage(ig, dstPath)
	return
}

func (gi contentGenImg) thumbnailImage(ig image.Image, img img) image.Image {
	igb := ig.Bounds()
	srcW, srcH := igb.Dx(), igb.Dy()

	scaleW, scaleH := srcW, srcH

	if scaleW < img.w {
		scaleH = (scaleH * img.w) / scaleW
		scaleW = img.w
	}

	if scaleH < img.h {
		scaleW = (scaleW * img.h) / scaleH
		scaleH = img.h
	}

	if scaleW == 0 {
		scaleW = 1
	}

	if scaleH == 0 {
		scaleH = 1
	}

	ig = imaging.Resize(ig, scaleW, scaleH, imaging.Lanczos)

	crop := image.Rectangle{}
	switch img.crop {
	case cropLeft:
		crop = image.Rectangle{
			Min: image.Point{X: 0, Y: 0},
			Max: image.Point{X: img.w, Y: img.h},
		}

	case cropCentered:
		centerX := scaleW / 2
		centerY := scaleH / 2

		x0 := centerX - img.w/2
		y0 := centerY - img.h/2
		x1 := x0 + img.w
		y1 := y0 + img.h

		crop = image.Rectangle{
			Min: image.Point{X: x0, Y: y0},
			Max: image.Point{X: x1, Y: y1},
		}

	default:
		panic(fmt.Errorf("unsupported crop option: %d", img.crop))
	}

	ig = imaging.Crop(ig, crop)

	return ig
}

func (contentGenImg) resizeImage(ig image.Image, img img) image.Image {
	igb := ig.Bounds()
	srcW, srcH := igb.Dx(), igb.Dy()

	scaleW, scaleH := srcW, srcH

	if img.h == 0 || (scaleW > img.w && img.w != 0) || (scaleW < img.w && scaleH < img.h) {
		scaleH = (scaleH * img.w) / scaleW
		scaleW = img.w
	}

	if img.w == 0 || (scaleH > img.h && img.h != 0) || (scaleW < img.w && scaleH < img.h) {
		scaleW = (scaleW * img.h) / scaleH
		scaleH = img.h
	}

	if scaleW == 0 {
		scaleW = 1
	}

	if scaleH == 0 {
		scaleH = 1
	}

	return imaging.Resize(ig, scaleW, scaleH, imaging.Lanczos)
}

func (gi contentGenImg) saveImage(ig image.Image, dst string) error {
	f, err := gi.c.cs.s.fCreate(dst)
	if err != nil {
		return err
	}

	defer f.Close()

	ext := filepath.Ext(dst)
	switch ext {
	case ".bmp":
		return bmp.Encode(f, ig)

	case ".gif":
		opts := &gif.Options{
			NumColors: 256,
		}

		return gif.Encode(f, ig, opts)

	case ".jpeg", ".jpg":
		opts := &jpeg.Options{
			Quality: 95,
		}

		if nrgba, ok := ig.(*image.NRGBA); ok && nrgba.Opaque() {
			rgba := &image.RGBA{
				Pix:    nrgba.Pix,
				Stride: nrgba.Stride,
				Rect:   nrgba.Rect,
			}

			return jpeg.Encode(f, rgba, opts)
		}

		return jpeg.Encode(f, ig, opts)

	case ".png":
		enc := png.Encoder{
			CompressionLevel: png.BestCompression,
		}

		return enc.Encode(f, ig)

	case ".tiff":
		opts := &tiff.Options{
			Compression: tiff.Deflate,
			Predictor:   true,
		}

		return tiff.Encode(f, ig, opts)

	default:
		panic(fmt.Errorf("image type %s is not supported and slipped through", ext))
	}
}

func (contentGenImg) getNewExt(img img) string {
	ext := ""

	if img.w != 0 || img.h != 0 {
		ext += fmt.Sprintf("%dx%d", img.w, img.h)
	}

	if img.crop != cropNone {
		ext += fmt.Sprintf(".c%c", img.crop.String()[0])
	}

	ext += img.ext

	return ext
}

func (c imgCrop) String() string {
	switch c {
	case cropLeft:
		return "left"
	case cropCentered:
		return "centered"
	default:
		return "none"
	}
}