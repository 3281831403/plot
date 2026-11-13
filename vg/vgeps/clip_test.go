// Copyright ©2026 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgeps_test

import (
	"bytes"
	"strings"
	"testing"

	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgeps"
)

// TestClipEmitsPostScriptClip checks that clipping emits a balanced
// gsave/rectpath clip ... grestore pair in the EPS output.
func TestClipEmitsPostScriptClip(t *testing.T) {
	c := vgeps.New(200, 200)
	dc := draw.New(c)
	rect := vg.Rectangle{
		Min: vg.Point{X: 40, Y: 40},
		Max: vg.Point{X: 160, Y: 160},
	}
	sub := draw.Canvas{Canvas: dc.Canvas, Rectangle: rect}
	restore := sub.Clip()
	sub.StrokeLine2(draw.LineStyle{Width: vg.Points(2)}, rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y)
	restore()

	var buf bytes.Buffer
	if _, err := c.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "rectpath clip") {
		t.Error("output has no rectpath clip operator")
	}
	if got := strings.Count(out, "gsave"); got != strings.Count(out, "grestore") {
		t.Errorf("unbalanced gsave/grestore: %d vs %d", got, strings.Count(out, "grestore"))
	}
}
