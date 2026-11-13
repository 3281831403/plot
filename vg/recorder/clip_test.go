// Copyright ©2026 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package recorder

import (
	"testing"

	"gonum.org/v1/plot/vg"
)

// TestClipReplay records a Clip ... PopClip pair and checks that
// ReplayOn opens and closes the clip on a supporting destination.
func TestClipReplay(t *testing.T) {
	var rec Canvas
	rect := vg.Rectangle{
		Min: vg.Point{X: 1, Y: 2},
		Max: vg.Point{X: 3, Y: 4},
	}
	restore := rec.Clip(rect)
	rec.SetColor(nil)
	restore()

	if len(rec.Actions) != 3 {
		t.Fatalf("unexpected action count: got %d, want 3", len(rec.Actions))
	}
	clip, ok := rec.Actions[0].(*Clip)
	if !ok {
		t.Fatalf("first action is %T, want *Clip", rec.Actions[0])
	}
	if clip.Rectangle != rect {
		t.Errorf("clip rectangle: got %+v, want %+v", clip.Rectangle, rect)
	}
	if _, ok := rec.Actions[2].(*PopClip); !ok {
		t.Fatalf("last action is %T, want *PopClip", rec.Actions[2])
	}

	var dst Canvas
	if err := rec.ReplayOn(&dst); err != nil {
		t.Fatalf("ReplayOn returned error: %v", err)
	}
	if len(dst.Actions) != len(rec.Actions) {
		t.Fatalf("replayed action count: got %d, want %d", len(dst.Actions), len(rec.Actions))
	}
	if _, ok := dst.Actions[0].(*Clip); !ok {
		t.Errorf("replayed first action is %T, want *Clip", dst.Actions[0])
	}
	if _, ok := dst.Actions[2].(*PopClip); !ok {
		t.Errorf("replayed last action is %T, want *PopClip", dst.Actions[2])
	}
}

// TestClipReplayMismatch checks that unbalanced Clip or PopClip
// actions are reported instead of silently accepted.
func TestClipReplayMismatch(t *testing.T) {
	var rec Canvas
	rec.PopClip()
	var dst Canvas
	if err := rec.ReplayOn(&dst); err == nil {
		t.Error("expected error for PopClip without a matching Clip")
	}

	rec.Reset()
	rec.Clip(vg.Rectangle{})
	if err := rec.ReplayOn(&Canvas{}); err == nil {
		t.Error("expected error for Clip without a matching PopClip")
	}
}
