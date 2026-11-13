// Copyright ©2026 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgimg_test

import (
	"image"
	"image/color"
	"testing"

	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

// pixelProbe reads pixels of an image in vg bottom-left coordinates.
type pixelProbe struct {
	img   *image.RGBA
	scale float64
}

func newPixelProbe(c *vgimg.Canvas) *pixelProbe {
	return &pixelProbe{
		img:   c.Image().(*image.RGBA),
		scale: c.DPI() / 72.0,
	}
}

// at returns the pixel at the given vg coordinates (in points).
func (p *pixelProbe) at(xPt, yPt float64) color.RGBA {
	col := int(xPt*p.scale + 0.5)
	row := p.img.Bounds().Max.Y - 1 - int(yPt*p.scale+0.5)
	return p.img.RGBAAt(col, row)
}

// countOutside counts pixels that differ from bg on a ring marginPt
// outside the given rectangle.
func (p *pixelProbe) countOutside(r vg.Rectangle, bg color.RGBA, marginPt float64) int {
	bad := 0
	check := func(xPt, yPt float64) {
		if px := p.at(xPt, yPt); px != bg {
			bad++
		}
	}
	for xPt := r.Min.X.Points(); xPt < r.Max.X.Points(); xPt += 1 / p.scale {
		check(xPt, r.Min.Y.Points()-marginPt)
		check(xPt, r.Max.Y.Points()+marginPt)
	}
	for yPt := r.Min.Y.Points(); yPt < r.Max.Y.Points(); yPt += 1 / p.scale {
		check(r.Min.X.Points()-marginPt, yPt)
		check(r.Max.X.Points()+marginPt, yPt)
	}
	return bad
}

// TestClipCutsInkAtRectangle verifies, pixel by pixel, that a glyph
// and a thick stroke crossing the rectangle boundary are cut off.
func TestClipCutsInkAtRectangle(t *testing.T) {
	const (
		wpt, hpt = 200, 200
		dpi      = 96
	)
	bg := color.RGBA{255, 255, 255, 255}
	c := vgimg.NewWith(vgimg.UseWH(wpt, hpt), vgimg.UseDPI(dpi))
	dc := draw.New(c)
	rect := vg.Rectangle{
		Min: vg.Point{X: 40, Y: 40},
		Max: vg.Point{X: 160, Y: 160},
	}
	sub := draw.Canvas{Canvas: dc.Canvas, Rectangle: rect}
	restore := sub.Clip()

	sub.StrokeLine2(draw.LineStyle{
		Color: color.RGBA{B: 255, A: 255}, Width: vg.Points(8),
	}, rect.Min.X-30, rect.Min.Y-30, rect.Max.X+30, rect.Max.Y+30)
	sub.DrawGlyphNoClip(draw.GlyphStyle{
		Color:  color.RGBA{R: 255, A: 255},
		Radius: vg.Points(10),
		Shape:  draw.CircleGlyph{},
	}, rect.Min)
	restore()

	if bad := newPixelProbe(c).countOutside(rect, bg, 2); bad != 0 {
		t.Errorf("found %d non-background pixels 2pt outside the clip rectangle", bad)
	}
}

// TestClipRestoreLeavesCanvasUsable checks that, after restore,
// drawing outside the former rectangle is no longer clipped.
func TestClipRestoreLeavesCanvasUsable(t *testing.T) {
	c := vgimg.NewWith(vgimg.UseWH(200, 200), vgimg.UseDPI(96))
	dc := draw.New(c)
	rect := vg.Rectangle{
		Min: vg.Point{X: 40, Y: 40},
		Max: vg.Point{X: 160, Y: 160},
	}
	sub := draw.Canvas{Canvas: dc.Canvas, Rectangle: rect}
	restore := sub.Clip()
	sub.DrawGlyphNoClip(draw.GlyphStyle{
		Color:  color.RGBA{R: 255, A: 255},
		Radius: vg.Points(10),
		Shape:  draw.CircleGlyph{},
	}, rect.Min)
	restore()

	grn := color.RGBA{G: 200, A: 255}
	center := vg.Point{X: rect.Min.X, Y: (rect.Min.Y + rect.Max.Y) / 2}
	path := vg.Path{}
	path.Move(vg.Point{X: center.X + 15, Y: center.Y})
	path.Arc(center, vg.Points(15), 0, 2*3.141592653589793)
	path.Close()
	dc.SetColor(grn)
	dc.Fill(path)

	if got := newPixelProbe(c).at(rect.Min.X.Points()-2, center.Y.Points()); got != grn {
		t.Errorf("post-restore drawing was still clipped: got %+v, want %+v", got, grn)
	}
}

// TestNestedClipIntersectsRegions checks that a second, smaller clip
// intersects the first one rather than replacing it.
func TestNestedClipIntersectsRegions(t *testing.T) {
	bg := color.RGBA{255, 255, 255, 255}
	red := color.RGBA{R: 255, A: 255}
	c := vgimg.NewWith(vgimg.UseWH(200, 200), vgimg.UseDPI(96))
	dc := draw.New(c)
	outer := draw.Canvas{
		Canvas: dc.Canvas,
		Rectangle: vg.Rectangle{
			Min: vg.Point{X: 20, Y: 20},
			Max: vg.Point{X: 180, Y: 180},
		},
	}
	innerRect := vg.Rectangle{
		Min: vg.Point{X: 60, Y: 60},
		Max: vg.Point{X: 140, Y: 140},
	}
	inner := draw.Canvas{Canvas: outer.Canvas, Rectangle: innerRect}

	restoreOuter := outer.Clip()
	restoreInner := inner.Clip()
	// Fill the whole canvas: only the inner rectangle may be red.
	dc.SetColor(red)
	dc.Fill(dc.Rectangle.Path())
	restoreInner()
	restoreOuter()

	probe := newPixelProbe(c)
	// Just inside the inner rectangle must be red.
	if got := probe.at(100, 100); got != red {
		t.Errorf("inside nested clip: got %+v, want %+v", got, red)
	}
	// Between outer and inner must still be background: that proves
	// the second clip intersected instead of widening the region.
	if got := probe.at(30, 100); got != bg {
		t.Errorf("outside inner clip but inside outer: got %+v, want %+v", got, bg)
	}
	if bad := probe.countOutside(innerRect, bg, 2); bad != 0 {
		t.Errorf("found %d non-background pixels outside the inner clip", bad)
	}
}
