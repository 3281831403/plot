// Copyright ©2026 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgsvg_test

import (
	"bytes"
	"image/color"
	"strings"
	"testing"

	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgsvg"
)

// TestClipEmitsClipPath checks that the SVG output wraps clipped
// drawing in a group referencing a clipPath, and that the group is
// closed again after the restore.
func TestClipEmitsClipPath(t *testing.T) {
	c := vgsvg.New(200, 200)
	dc := draw.New(c)
	rect := vg.Rectangle{
		Min: vg.Point{X: 40, Y: 40},
		Max: vg.Point{X: 160, Y: 160},
	}
	sub := draw.Canvas{Canvas: dc.Canvas, Rectangle: rect}
	restore := sub.Clip()
	sub.StrokeLine2(draw.LineStyle{
		Color: color.Black, Width: vg.Points(2),
	}, rect.Min.X-30, rect.Min.Y-30, rect.Max.X+30, rect.Max.Y+30)
	restore()

	var buf bytes.Buffer
	if _, err := c.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "<clipPath") {
		t.Error("output has no <clipPath> element")
	}
	if !strings.Contains(out, `clip-path="url(#gonum-clip-0)"`) {
		t.Error("output has no group referencing the clip path")
	}
	// One group opened for the clip and one closed on restore.
	open := strings.Count(out, "<g ")
	closed := strings.Count(out, "</g>")
	if open != closed {
		t.Errorf("unbalanced clip group: %d opens vs %d closes", open, closed)
	}
}

// TestClipWriteToWithActiveClip reports unbalanced clips instead of
// emitting malformed SVG.
func TestClipWriteToWithActiveClip(t *testing.T) {
	c := vgsvg.New(100, 100)
	dc := draw.New(c)
	sub := draw.Crop(dc, 10, -10, 10, -10)
	_ = sub.Clip()

	var buf bytes.Buffer
	if _, err := c.WriteTo(&buf); err == nil {
		t.Error("expected error when writing with an unrestored clip")
	}
}
