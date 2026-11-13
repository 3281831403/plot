// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package vgimg implements the vg.Canvas interface using
// git.sr.ht/~sbinet/gg as a backend to output raster images.
package vgimg // import "gonum.org/v1/plot/vg/vgimg"

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math"

	"git.sr.ht/~sbinet/gg"
	"golang.org/x/image/tiff"
	"golang.org/x/image/vector"

	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/vg"
	vgdraw "gonum.org/v1/plot/vg/draw"
)

func init() {
	vgdraw.RegisterFormat("png", func(w, h vg.Length) vg.CanvasWriterTo {
		return PngCanvas{Canvas: New(w, h)}
	})

	vgdraw.RegisterFormat("jpg", func(w, h vg.Length) vg.CanvasWriterTo {
		return JpegCanvas{Canvas: New(w, h)}
	})

	vgdraw.RegisterFormat("jpeg", func(w, h vg.Length) vg.CanvasWriterTo {
		return JpegCanvas{Canvas: New(w, h)}
	})

	vgdraw.RegisterFormat("tif", func(w, h vg.Length) vg.CanvasWriterTo {
		return TiffCanvas{Canvas: New(w, h)}
	})

	vgdraw.RegisterFormat("tiff", func(w, h vg.Length) vg.CanvasWriterTo {
		return TiffCanvas{Canvas: New(w, h)}
	})
}

// Canvas implements the vg.Canvas interface,
// drawing to an image.Image using draw2d.
type Canvas struct {
	ctx   *gg.Context
	img   draw.Image
	w, h  vg.Length
	color []color.Color

	// dpi is the number of dots per inch for this canvas.
	dpi int

	// width is the current line width.
	width vg.Length

	// backgroundColor is the background color, set by
	// UseBackgroundColor.
	backgroundColor color.Color

	// matrix mirrors the affine transform applied to the gg context
	// (including its Y inversion), mapping user-space dots to device
	// pixels. gg does not expose its matrix, so it is tracked here.
	matrix      affine
	matrixStack []affine

	// clips holds the mask in place before each still-active Clip
	// call; a nil entry means no clip was active at that point.
	clips      []*image.Alpha
	activeMask *image.Alpha
}

// affine is a 2D affine transform:
//
//	x' = xx*x + xy*y + x0
//	y' = yx*x + yy*y + y0
type affine struct {
	xx, yx, xy, yy, x0, y0 float64
}

func identityAffine() affine {
	return affine{xx: 1, yy: 1}
}

// transform maps a point through the affine.
func (a affine) transform(x, y float64) (float64, float64) {
	return a.xx*x + a.xy*y + a.x0, a.yx*x + a.yy*y + a.y0
}

// translate post-multiplies a translation, matching gg's semantics.
func (a affine) translate(tx, ty float64) affine {
	return affine{
		xx: a.xx, yx: a.yx,
		xy: a.xy, yy: a.yy,
		x0: tx*a.xx + ty*a.xy + a.x0,
		y0: tx*a.yx + ty*a.yy + a.y0,
	}
}

// scale post-multiplies a scaling, matching gg's semantics.
func (a affine) scale(sx, sy float64) affine {
	return affine{
		xx: sx * a.xx, yx: sx * a.yx,
		xy: sy * a.xy, yy: sy * a.yy,
		x0: a.x0, y0: a.y0,
	}
}

// rotate post-multiplies an anticlockwise rotation, matching gg's
// semantics.
func (a affine) rotate(rad float64) affine {
	c, s := math.Cos(rad), math.Sin(rad)
	return affine{
		xx: c*a.xx + s*a.xy, yx: c*a.yx + s*a.yy,
		xy: -s*a.xx + c*a.xy, yy: -s*a.yx + c*a.yy,
		x0: a.x0, y0: a.y0,
	}
}

const (
	// DefaultDPI is the default dot resolution for image
	// drawing in dots per inch.
	DefaultDPI = 96

	// DefaultWidth and DefaultHeight are the default canvas
	// dimensions.
	DefaultWidth  = 4 * vg.Inch
	DefaultHeight = 4 * vg.Inch
)

// New returns a new image canvas.
func New(w, h vg.Length) *Canvas {
	return NewWith(UseWH(w, h), UseBackgroundColor(color.White))
}

// NewWith returns a new image canvas created according to the specified
// options. The currently accepted options are UseWH,
// UseDPI, UseImage, and UseImageWithContext.
// Each of the options specifies the size of the canvas (UseWH, UseImage),
// the resolution of the canvas (UseDPI), or both (useImageWithContext).
// If size or resolution are not specified, defaults are used.
// It panics if size and resolution are overspecified (i.e., too many options are
// passed).
func NewWith(o ...option) *Canvas {
	c := new(Canvas)
	c.backgroundColor = color.White
	var g uint32
	for _, opt := range o {
		f := opt(c)
		if g&f != 0 {
			panic("incompatible options")
		}
		g |= f
	}
	if c.dpi == 0 {
		c.dpi = DefaultDPI
	}
	if c.w == 0 { // h should also == 0.
		if c.img == nil {
			c.w = DefaultWidth
			c.h = DefaultHeight
		} else {
			w := float64(c.img.Bounds().Max.X - c.img.Bounds().Min.X)
			h := float64(c.img.Bounds().Max.Y - c.img.Bounds().Min.Y)
			c.w = vg.Length(w/float64(c.dpi)) * vg.Inch
			c.h = vg.Length(h/float64(c.dpi)) * vg.Inch
		}
	}
	if c.img == nil {
		w := c.w / vg.Inch * vg.Length(c.dpi)
		h := c.h / vg.Inch * vg.Length(c.dpi)
		c.img = draw.Image(image.NewRGBA(image.Rect(0, 0, int(w+0.5), int(h+0.5))))
	}
	if c.ctx == nil {
		c.ctx = gg.NewContextForImage(c.img)
		c.ctx.SetLineCapButt()
		c.img = c.ctx.Image().(draw.Image)
		c.ctx.InvertY()
		// Mirror the context's InvertY: translate by the image
		// height then flip the Y axis, in device pixels. A context
		// supplied through UseImageWithContext is not inverted and
		// keeps the identity matrix.
		c.matrix = identityAffine().
			translate(0, float64(c.img.Bounds().Dy())).
			scale(1, -1)
	} else {
		c.matrix = identityAffine()
	}
	draw.Draw(c.img, c.img.Bounds(), &image.Uniform{c.backgroundColor}, image.Point{}, draw.Src)
	c.color = []color.Color{color.Black}
	vg.Initialize(c)
	return c
}

// These constants are used to ensure that the options
// used when initializing a canvas are compatible with
// each other.
const (
	setsDPI uint32 = 1 << iota
	setsSize
	setsBackground
)

type option func(*Canvas) uint32

// UseWH specifies the width and height of the canvas.
// The size is rounded up to the nearest pixel.
func UseWH(w, h vg.Length) option {
	return func(c *Canvas) uint32 {
		if w <= 0 || h <= 0 {
			panic("w and h must both be > 0.")
		}
		c.w, c.h = w, h
		return setsSize
	}
}

// UseDPI sets the dots per inch of a canvas. It should only be
// used as an option argument when initializing a new canvas.
func UseDPI(dpi int) option {
	if dpi <= 0 {
		panic("DPI must be > 0.")
	}
	return func(c *Canvas) uint32 {
		c.dpi = dpi
		return setsDPI
	}
}

// UseImage specifies an image to create
// the canvas from. The
// minimum point of the given image
// should probably be 0,0.
//
// Note that a copy of the input image is performed.
// This means that modifications applied to the canvas are not reflected
// on the original image.
func UseImage(img draw.Image) option {
	return func(c *Canvas) uint32 {
		c.img = img
		return setsSize | setsBackground
	}
}

// UseImageWithContext specifies both an image
// and a graphic context to create the canvas from.
// The minimum point of the given image
// should probably be 0,0.
func UseImageWithContext(img draw.Image, ctx *gg.Context) option {
	return func(c *Canvas) uint32 {
		c.img = img
		c.ctx = ctx
		return setsSize | setsBackground
	}
}

// UseBackgroundColor specifies the image background color.
// Without UseBackgroundColor, the default color is white.
func UseBackgroundColor(c color.Color) option {
	return func(canvas *Canvas) uint32 {
		canvas.backgroundColor = c
		return setsBackground
	}
}

// Image returns the image the canvas is drawing to.
//
// The dimensions of the returned image must not be modified.
func (c *Canvas) Image() draw.Image {
	return c.img
}

func (c *Canvas) Size() (w, h vg.Length) {
	return c.w, c.h
}

func (c *Canvas) SetLineWidth(w vg.Length) {
	c.width = w
	c.ctx.SetLineWidth(w.Dots(c.DPI()))
}

func (c *Canvas) SetLineDash(ds []vg.Length, offs vg.Length) {
	dashes := make([]float64, len(ds))
	for i, d := range ds {
		dashes[i] = d.Dots(c.DPI())
	}
	c.ctx.SetDashOffset(offs.Dots(c.DPI()))
	c.ctx.SetDash(dashes...)
}

func (c *Canvas) SetColor(clr color.Color) {
	if clr == nil {
		clr = color.Black
	}
	c.ctx.SetColor(clr)
	c.color[len(c.color)-1] = clr
}

func (c *Canvas) Rotate(t float64) {
	c.ctx.Rotate(t)
	c.matrix = c.matrix.rotate(t)
}

func (c *Canvas) Translate(pt vg.Point) {
	c.ctx.Translate(pt.X.Dots(c.DPI()), pt.Y.Dots(c.DPI()))
	c.matrix = c.matrix.translate(pt.X.Dots(c.DPI()), pt.Y.Dots(c.DPI()))
}

func (c *Canvas) Scale(x, y float64) {
	c.ctx.Scale(x, y)
	c.matrix = c.matrix.scale(x, y)
}

func (c *Canvas) Push() {
	c.color = append(c.color, c.color[len(c.color)-1])
	c.ctx.Push()
	c.matrixStack = append(c.matrixStack, c.matrix)
}

func (c *Canvas) Pop() {
	c.color = c.color[:len(c.color)-1]
	c.ctx.Pop()
	n := len(c.matrixStack)
	c.matrix = c.matrixStack[n-1]
	c.matrixStack = c.matrixStack[:n-1]
}

func (c *Canvas) Stroke(p vg.Path) {
	if c.width <= 0 {
		return
	}
	c.outline(p)
	c.ctx.Stroke()
}

func (c *Canvas) Fill(p vg.Path) {
	c.outline(p)
	c.ctx.Fill()
}

func (c *Canvas) outline(p vg.Path) {
	for _, comp := range p {
		switch comp.Type {
		case vg.MoveComp:
			c.ctx.MoveTo(comp.Pos.X.Dots(c.DPI()), comp.Pos.Y.Dots(c.DPI()))

		case vg.LineComp:
			c.ctx.LineTo(comp.Pos.X.Dots(c.DPI()), comp.Pos.Y.Dots(c.DPI()))

		case vg.ArcComp:
			c.ctx.DrawArc(comp.Pos.X.Dots(c.DPI()), comp.Pos.Y.Dots(c.DPI()),
				comp.Radius.Dots(c.DPI()),
				comp.Start, comp.Start+comp.Angle,
			)

		case vg.CurveComp:
			switch len(comp.Control) {
			case 1:
				c.ctx.QuadraticTo(
					comp.Control[0].X.Dots(c.DPI()), comp.Control[0].Y.Dots(c.DPI()),
					comp.Pos.X.Dots(c.DPI()), comp.Pos.Y.Dots(c.DPI()),
				)
			case 2:
				c.ctx.CubicTo(
					comp.Control[0].X.Dots(c.DPI()), comp.Control[0].Y.Dots(c.DPI()),
					comp.Control[1].X.Dots(c.DPI()), comp.Control[1].Y.Dots(c.DPI()),
					comp.Pos.X.Dots(c.DPI()), comp.Pos.Y.Dots(c.DPI()),
				)
			default:
				panic("vgimg: invalid number of control points")
			}

		case vg.CloseComp:
			c.ctx.ClosePath()

		default:
			panic(fmt.Sprintf("Unknown path component: %d", comp.Type))
		}
	}
}

// DPI returns the resolution of the receiver in pixels per inch.
func (c *Canvas) DPI() float64 {
	return float64(c.dpi)
}

// Clip restricts drawing to the intersection of the given rectangle
// with the current clipping region. The rectangle is expressed in the
// canvas's current user coordinate system, so any active transform is
// applied to it. The returned function restores the clipping region
// that was in place when Clip was called.
func (c *Canvas) Clip(r vg.Rectangle) (restore func()) {
	// gg's own Clip rasterizes its mask through the current context
	// transform, but consults the mask in untransformed device pixels;
	// with InvertY active that places the clip region on the wrong
	// side of the image. Rasterize the transformed rectangle into a
	// device-space mask instead, mapping the corners through the
	// tracked transform.
	dpi := c.DPI()
	pts := [4][2]float64{
		{r.Min.X.Dots(dpi), r.Min.Y.Dots(dpi)},
		{r.Max.X.Dots(dpi), r.Min.Y.Dots(dpi)},
		{r.Max.X.Dots(dpi), r.Max.Y.Dots(dpi)},
		{r.Min.X.Dots(dpi), r.Max.Y.Dots(dpi)},
	}
	var dev [4][2]float64
	for i, p := range pts {
		dev[i][0], dev[i][1] = c.matrix.transform(p[0], p[1])
	}

	b := c.img.Bounds()
	next := image.NewAlpha(b)
	rasterizer := vector.NewRasterizer(b.Dx(), b.Dy())
	rasterizer.MoveTo(float32(dev[0][0]), float32(dev[0][1]))
	for _, p := range dev[1:] {
		rasterizer.LineTo(float32(p[0]), float32(p[1]))
	}
	rasterizer.ClosePath()
	rasterizer.Draw(next, image.Rect(0, 0, b.Dx(), b.Dy()), image.NewUniform(color.Alpha{A: 255}), image.Point{})

	// gg keeps its current mask private, so the canvas tracks the
	// active mask itself. Intersect the new rectangle with whatever
	// clip region was in place before this call.
	prev := c.activeMask
	n := len(c.clips)
	if prev != nil {
		inter := image.NewAlpha(b)
		draw.DrawMask(inter, b, next, image.Point{}, prev, image.Point{}, draw.Over)
		next = inter
	}
	c.clips = append(c.clips, prev)
	c.activeMask = next
	_ = c.ctx.SetMask(next)

	return func() {
		c.clips = c.clips[:n]
		c.activeMask = prev
		if prev == nil {
			c.ctx.ResetClip()
			return
		}
		_ = c.ctx.SetMask(prev)
	}
}

var _ vg.ClipCanvas = (*Canvas)(nil)

func (c *Canvas) FillString(font font.Face, pt vg.Point, str string) {
	if font.Font.Size == 0 {
		return
	}

	c.ctx.Push()
	defer c.ctx.Pop()

	face := font.FontFace(c.DPI())
	defer face.Close()

	c.ctx.SetFontFace(face)

	x := pt.X.Dots(c.DPI())
	y := pt.Y.Dots(c.DPI())
	h := c.h.Dots(c.DPI())

	c.ctx.InvertY()
	c.ctx.DrawString(str, x, h-y)
}

// DrawImage implements the vg.Canvas.DrawImage method.
func (c *Canvas) DrawImage(rect vg.Rectangle, img image.Image) {
	var (
		dpi    = c.DPI()
		min    = rect.Min
		xmin   = min.X.Dots(dpi)
		ymin   = min.Y.Dots(dpi)
		rsz    = rect.Size()
		width  = rsz.X.Dots(dpi)
		height = rsz.Y.Dots(dpi)
		dx     = float64(img.Bounds().Dx())
		dy     = float64(img.Bounds().Dy())
	)
	c.ctx.Push()
	c.ctx.Scale(1, -1)
	c.ctx.Translate(xmin, -ymin-height)
	c.ctx.Scale(width/dx, height/dy)
	c.ctx.DrawImage(img, 0, 0)
	c.ctx.Pop()
}

// WriterCounter implements the io.Writer interface, and counts
// the total number of bytes written.
type writerCounter struct {
	io.Writer
	n int64
}

func (w *writerCounter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	w.n += int64(n)
	return n, err
}

// A JpegCanvas is an image canvas with a WriteTo method
// that writes a jpeg image.
type JpegCanvas struct {
	*Canvas
}

// WriteTo implements the io.WriterTo interface, writing a jpeg image.
func (c JpegCanvas) WriteTo(w io.Writer) (int64, error) {
	wc := writerCounter{Writer: w}
	b := bufio.NewWriter(&wc)
	if err := jpeg.Encode(b, c.img, nil); err != nil {
		return wc.n, err
	}
	err := b.Flush()
	return wc.n, err
}

// A PngCanvas is an image canvas with a WriteTo method that
// writes a png image.
type PngCanvas struct {
	*Canvas
}

// WriteTo implements the io.WriterTo interface, writing a png image.
func (c PngCanvas) WriteTo(w io.Writer) (int64, error) {
	wc := writerCounter{Writer: w}
	b := bufio.NewWriter(&wc)
	if err := png.Encode(b, c.img); err != nil {
		return wc.n, err
	}
	err := b.Flush()
	return wc.n, err
}

// A TiffCanvas is an image canvas with a WriteTo method that
// writes a tiff image.
type TiffCanvas struct {
	*Canvas
}

// WriteTo implements the io.WriterTo interface, writing a tiff image.
func (c TiffCanvas) WriteTo(w io.Writer) (int64, error) {
	wc := writerCounter{Writer: w}
	b := bufio.NewWriter(&wc)
	if err := tiff.Encode(b, c.img, nil); err != nil {
		return wc.n, err
	}
	err := b.Flush()
	return wc.n, err
}
