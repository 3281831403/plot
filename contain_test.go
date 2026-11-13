package plot

import (
	"testing"

	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/recorder"
)

// TestLogLabelsContained renders log-axis plots over a range of sizes and
// verifies that every visible tick label lies wholly inside the draw canvas
// and that no two visible labels overlap.
func TestLogLabelsContained(t *testing.T) {
	ranges := [][2]float64{
		{200, 800},
		{5e-9, 9e-9},
		{1, 10000},
		{0.2, 0.8},
	}
	widths := []vg.Length{
		0.6 * vg.Inch,
		1.2 * vg.Inch,
		3 * vg.Inch,
		6 * vg.Inch,
	}
	for _, r := range ranges {
		for _, w := range widths {
			rc := new(recorder.Canvas)
			p := New()
			p.X.Scale = LogScale{}
			p.X.Tick.Marker = LogTicks{Prec: -1}
			p.X.Min = r[0]
			p.X.Max = r[1]
			p.Y.Min = 0
			p.Y.Max = 1
			c := draw.NewCanvas(rc, w, w)
			p.Draw(c)

			p.X.sanitizeRange()
			xa := horizontalAxis{p.X}
			ywidth := verticalAxis{p.Y}.size()
			// The axis draws on a horizontally padded canvas, but labels are
			// centred on tick anchors, so containment is judged against the
			// full plot canvas (including the margin area), not the cropped
			// data region.
			full := c
			xc := padX(p, draw.Crop(c, ywidth, 0, 0, 0))

			marks := xa.Tick.Marker.Ticks(xa.Min, xa.Max)
			hidden := pruneOverlappingLabels(xa.Axis, xc, marks)
			gap := xa.Tick.Label.Width(" ")

			var vis []struct {
				label string
				pos   vg.Length
				w     vg.Length
			}
			for i, m := range marks {
				if m.IsMinor() || hidden[i] {
					continue
				}
				pos := xc.X(xa.Norm(m.Value))
				if !xc.ContainsX(pos) {
					t.Fatalf("range %v width %v: label %q anchor outside axis canvas", r, w, m.Label)
				}
				lw := xa.Tick.Label.Width(m.Label)
				vis = append(vis, struct {
					label string
					pos   vg.Length
					w     vg.Length
				}{m.Label, pos, lw})

				// The label is centred on the anchor; both halves must fit
				// inside the full plot canvas.
				if pos-lw/2 < full.Min.X || pos+lw/2 > full.Max.X {
					t.Errorf("range %v width %v: label %q box [%v,%v] exceeds canvas [%v,%v]",
						r, w, m.Label, pos-lw/2, pos+lw/2, full.Min.X, full.Max.X)
				}
			}
			if len(vis) == 0 {
				t.Fatalf("range %v width %v: no visible labels", r, w)
			}
			for i := 1; i < len(vis); i++ {
				prevEnd := vis[i-1].pos + vis[i-1].w/2
				start := vis[i].pos - vis[i].w/2
				if start < prevEnd+gap {
					t.Errorf("range %v width %v: labels %q and %q overlap", r, w, vis[i-1].label, vis[i].label)
				}
			}
		}
	}
}
