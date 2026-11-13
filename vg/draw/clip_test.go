// Copyright ©2026 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package draw_test

import (
	"image"
	"image/color"
	"testing"

	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/recorder"
)

// TestClipUnwrapsNestedCanvases checks that Clip reaches the real
// backend even when several draw.Canvas values are nested through
// Crop, and that the recorded rectangle is the innermost one.
func TestClipUnwrapsNestedCanvases(t *testing.T) {
	var rec recorder.Canvas
	root := draw.NewCanvas(&rec, 100, 100)
	inner := draw.Crop(root, 10, -10, 10, -10)
	innermost := draw.Crop(inner, 5, 0, 5, 0)

	restore := innermost.Clip()
	restore()

	var (
		clips    []*recorder.Clip
		popClips int
	)
	for _, a := range rec.Actions {
		switch a := a.(type) {
		case *recorder.Clip:
			clips = append(clips, a)
		case *recorder.PopClip:
			popClips++
		}
	}
	if len(clips) != 1 {
		t.Fatalf("expected exactly one recorded Clip, got %d", len(clips))
	}
	if popClips != 1 {
		t.Fatalf("expected exactly one recorded PopClip, got %d", popClips)
	}
	want := innermost.Rectangle
	if got := clips[0].Rectangle; got != want {
		t.Errorf("clip rectangle mismatch:\ngot:  %+v\nwant: %+v", got, want)
	}
}

// TestClipUnsupportedBackend checks that a backend without clipping
// still gets a working restore function and never panics.
func TestClipUnsupportedBackend(t *testing.T) {
	plain := nonClippingCanvas{}
	c := draw.NewCanvas(plain, 50, 50)
	restore := c.Clip()
	restore()
}

// nonClippingCanvas is a vg.Canvas that deliberately does not
// implement vg.ClipCanvas.
type nonClippingCanvas struct{}

func (nonClippingCanvas) SetLineWidth(vg.Length)                 {}
func (nonClippingCanvas) SetLineDash([]vg.Length, vg.Length)     {}
func (nonClippingCanvas) SetColor(color.Color)                   {}
func (nonClippingCanvas) Rotate(float64)                         {}
func (nonClippingCanvas) Translate(vg.Point)                     {}
func (nonClippingCanvas) Scale(float64, float64)                 {}
func (nonClippingCanvas) Push()                                  {}
func (nonClippingCanvas) Pop()                                   {}
func (nonClippingCanvas) Stroke(vg.Path)                         {}
func (nonClippingCanvas) Fill(vg.Path)                           {}
func (nonClippingCanvas) FillString(font.Face, vg.Point, string) {}
func (nonClippingCanvas) DrawImage(vg.Rectangle, image.Image)    {}

var _ vg.Canvas = nonClippingCanvas{}
