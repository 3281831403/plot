// Copyright ©2026 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpdf_test

import (
	"bytes"
	"compress/zlib"
	"io"
	"regexp"
	"testing"

	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgpdf"
)

// TestClipEmitsPdfClip checks that the PDF content stream contains a
// clipping rectangle wrapped in q/Q save/restore operators.
func TestClipEmitsPdfClip(t *testing.T) {
	c := vgpdf.New(200, 200)
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

	var content []byte
	raw := buf.Bytes()
	for _, m := range regexp.MustCompile(`stream\r?\n`).FindAllStringIndex(string(raw), -1) {
		start := m[1]
		end := bytes.Index(raw[start:], []byte("\nendstream"))
		if end < 0 {
			continue
		}
		z, err := zlib.NewReader(bytes.NewReader(raw[start : start+end]))
		if err != nil {
			continue
		}
		decoded, err := io.ReadAll(z)
		_ = z.Close()
		if err != nil {
			continue
		}
		content = append(content, decoded...)
	}
	if len(content) == 0 {
		t.Fatal("could not locate a compressed content stream")
	}
	if !regexp.MustCompile(`re W n`).Match(content) {
		t.Errorf("content stream has no clipping operator:\n%s", content)
	}
}
