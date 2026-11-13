// Copyright ©2018 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot_test

import (
	"fmt"
	"image/color"
	"testing"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/cmpimg"
	"gonum.org/v1/plot/text"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

func TestLegend_standalone(t *testing.T) {
	cmpimg.CheckPlot(ExampleLegend_standalone, t, "legend_standalone.png")
}

// recordHandler is a text.Handler that records every string it is
// asked to draw, together with the style and position of the draw
// call.
type recordHandler struct {
	text.Handler
	txts []string
	stys []text.Style
	pts  []vg.Point
}

// Draw implements text.Handler, recording the call before
// delegating to the wrapped handler.
func (h *recordHandler) Draw(c vg.Canvas, txt string, sty text.Style, pt vg.Point) {
	h.txts = append(h.txts, txt)
	h.stys = append(h.stys, sty)
	h.pts = append(h.pts, pt)
	h.Handler.Draw(c, txt, sty, pt)
}

// rect returns the rectangle occupied by the i-th recorded text,
// in canvas coordinates.
func (h *recordHandler) rect(i int) vg.Rectangle {
	return h.stys[i].Rectangle(h.txts[i]).Add(h.pts[i])
}

// newRecordingLegend returns a Legend whose text handler records
// every drawn string, and the handler itself.
func newRecordingLegend() (plot.Legend, *recordHandler) {
	hdlr := &recordHandler{Handler: plot.DefaultTextHandler}
	l := plot.NewLegend()
	l.TextStyle.Handler = hdlr
	return l, hdlr
}

// entryHeight returns the height of the tallest entry of the
// legend, mirroring the legend's own layout computation.
func entryHeight(l plot.Legend, names []string) (height vg.Length) {
	for _, name := range names {
		if h := l.TextStyle.Rectangle(name).Max.Y; h > height {
			height = h
		}
	}
	return height
}

// TestLegend_wrapping checks that a legend holding more entries
// than fit in a single column wraps its entries into as many
// columns as needed for the whole legend to remain inside the
// canvas, keeping the entries in the order they were added.
func TestLegend_wrapping(t *testing.T) {
	const (
		n    = 40 // number of legend entries.
		rows = 8  // number of rows made available to the legend.
	)
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("entry-%02d", i)
	}

	for _, test := range []struct {
		name string
		top  bool
		left bool
	}{
		{name: "bottom-right", top: false, left: false},
		{name: "bottom-left", top: false, left: true},
		{name: "top-right", top: true, left: false},
		{name: "top-left", top: true, left: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			l, hdlr := newRecordingLegend()
			l.Top, l.Left = test.top, test.left
			l.Padding = vg.Points(2)
			for _, name := range names {
				l.Add(name, exampleThumbnailer{Color: color.NRGBA{R: 200, A: 255}})
			}

			var (
				sty     = l.TextStyle
				em      = sty.Rectangle(" ")
				enth    = entryHeight(l, names)
				ext     = sty.FontExtents()
				descent = ext.Descent
				linegap = ext.Height - ext.Ascent - ext.Descent
				pad     = l.Padding
			)

			// Size the canvas so that exactly rows entries fit
			// in a single column, with a little slack.  The slack
			// must stay below enth+pad so that no additional row
			// fits.
			slack := descent/2 + linegap + 1
			height := vg.Length(rows)*enth + vg.Length(rows-1)*pad + descent + slack
			c := draw.New(vgimg.New(vg.Points(640), height))

			l.Draw(c)

			cols := (n + rows - 1) / rows
			if got, want := len(hdlr.txts), n; got != want {
				t.Fatalf("unexpected number of drawn entries: got %d, want %d", got, want)
			}

			// Every recorded entry must lie within the canvas —
			// up to the same descent/2+linegap margin the
			// historical single-column layout allows below the
			// anchor edge — and entries must be laid out
			// column-wise in the order they were added: the first
			// rows entries in the first column, the next rows
			// entries in the second column, and so on.
			var (
				entryW = sty.Rectangle(" " + names[0]).Max.X
				colW   = l.ThumbnailWidth + entryW
				gap    = em.Max.X + pad
			)
			const tol = vg.Length(1e-6)
			ytol := descent/2 + linegap + tol
			for i, name := range names {
				if hdlr.txts[i] != name {
					t.Errorf("entry %d drawn out of order: got %q, want %q", i, hdlr.txts[i], name)
				}

				r := hdlr.rect(i)
				if r.Min.X < c.Min.X-tol || c.Max.X+tol < r.Max.X ||
					r.Min.Y < c.Min.Y-ytol || c.Max.Y+ytol < r.Max.Y {
					t.Errorf("entry %d (%q) lies outside the canvas: rect=%+v canvas=%+v",
						i, name, r, c.Rectangle)
				}

				var (
					j = i / rows // column of the i-th entry.
					k = i % rows // row of the i-th entry within its column.
				)
				xoff := vg.Length(j) * (colW + gap)
				var wantX vg.Length
				if test.left {
					wantX = c.Min.X + xoff + l.ThumbnailWidth + em.Max.X
				} else {
					wantX = c.Max.X - xoff - l.ThumbnailWidth - em.Max.X
				}
				var ytop vg.Length
				if test.top {
					ytop = c.Max.Y - enth - descent
				} else {
					ytop = c.Min.Y + (enth+pad)*vg.Length(rows-1)
				}
				wantY := ytop - vg.Length(k)*(enth+pad) +
					(enth-descent-sty.Rectangle(name).Max.Y)/2 + descent

				if pt := hdlr.pts[i]; !approxEq(pt.X, wantX) || !approxEq(pt.Y, wantY) {
					t.Errorf("entry %d (%q) at unexpected position: got %+v, want {%v, %v}",
						i, name, pt, wantX, wantY)
				}
			}

			// The legend must report an extent that fits inside
			// the canvas, and must not claim the whole of it.
			r := l.Rectangle(c)
			if r.Min.X < c.Min.X-tol || c.Max.X+tol < r.Max.X ||
				r.Min.Y < c.Min.Y-tol || c.Max.Y+tol < r.Max.Y {
				t.Errorf("legend extent lies outside the canvas: rect=%+v canvas=%+v", r, c.Rectangle)
			}
			if got, want := r.Size().X, vg.Length(cols)*colW+vg.Length(cols-1)*gap; !approxEq(got, want) {
				t.Errorf("unexpected legend width: got %v, want %v", got, want)
			}
			if got := r.Size().X; got >= c.Max.X-c.Min.X {
				t.Errorf("legend covers the whole canvas width: %v", got)
			}
		})
	}
}

// TestLegend_fewEntriesUnchanged checks that legends with three
// entries or fewer keep the exact single-column layout they had
// before automatic column wrapping was introduced.
func TestLegend_fewEntriesUnchanged(t *testing.T) {
	names := []string{"red", "green", "blue"}

	for _, test := range []struct {
		name string
		top  bool
		left bool
	}{
		{name: "bottom-right", top: false, left: false},
		{name: "bottom-left", top: false, left: true},
		{name: "top-right", top: true, left: false},
		{name: "top-left", top: true, left: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			l, hdlr := newRecordingLegend()
			l.Top, l.Left = test.top, test.left
			l.Padding = vg.Millimeter
			for _, name := range names {
				l.Add(name, exampleThumbnailer{Color: color.NRGBA{R: 200, A: 255}})
			}

			c := draw.New(vgimg.New(vg.Points(120), vg.Points(100)))
			l.Draw(c)

			if got, want := len(hdlr.txts), len(names); got != want {
				t.Fatalf("unexpected number of drawn entries: got %d, want %d", got, want)
			}

			// Reference positions, derived from the historical
			// single-column layout.
			var (
				sty     = l.TextStyle
				em      = sty.Rectangle(" ")
				enth    = entryHeight(l, names)
				descent = sty.FontExtents().Descent
			)
			var textx vg.Length
			if test.left {
				textx = c.Min.X + l.ThumbnailWidth + em.Max.X
			} else {
				textx = c.Max.X - l.ThumbnailWidth - em.Max.X
			}
			y := c.Max.Y - enth - descent
			if !test.top {
				y = c.Min.Y + (enth+l.Padding)*vg.Length(len(names)-1)
			}
			for i, name := range names {
				if hdlr.txts[i] != name {
					t.Errorf("entry %d drawn out of order: got %q, want %q", i, hdlr.txts[i], name)
				}
				want := vg.Point{
					X: textx,
					Y: y - vg.Length(i)*(enth+l.Padding) +
						(enth-descent-sty.Rectangle(name).Max.Y)/2 + descent,
				}
				if pt := hdlr.pts[i]; !approxEq(pt.X, want.X) || !approxEq(pt.Y, want.Y) {
					t.Errorf("entry %d (%q) moved: got %+v, want %+v", i, name, pt, want)
				}
			}
		})
	}
}

// approxEq reports whether a and b are equal to within a small
// tolerance, to allow for floating-point rounding differences.
func approxEq(a, b vg.Length) bool {
	const tol = vg.Length(1e-6)
	d := a - b
	return -tol < d && d < tol
}
