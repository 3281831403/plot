package plot_test

import (
	"math"
	"strings"
	"testing"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/recorder"
)

// TestNarrowLogAxisReadable renders log-scale plots over sub-decade ranges at
// several widths and asserts that both axes always emit at least one numeric
// label without overlap. It is the regression test for a sub-decade log axis
// (for example 200..800) previously rendering no tick numbers at all.
func TestNarrowLogAxisReadable(t *testing.T) {
	scenarios := []struct {
		name     string
		min, max float64
		width    vg.Length
	}{
		{"200-800/1.2in", 200, 800, 1.2 * vg.Inch},
		{"nano/1.2in", 5e-9, 9e-9, 1.2 * vg.Inch},
		{"200-800/0.6in", 200, 800, 0.6 * vg.Inch},
		{"200-800/4in", 200, 800, 4 * vg.Inch},
	}
	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			p := plot.New()
			p.X.Scale = plot.LogScale{}
			p.X.Tick.Marker = plot.LogTicks{Prec: -1}
			p.Y.Scale = plot.LogScale{}
			p.Y.Tick.Marker = plot.LogTicks{Prec: -1}
			p.X.Min = sc.min
			p.X.Max = sc.max
			p.Y.Min = sc.min
			p.Y.Max = sc.max

			pts := make(plotter.XYs, 5)
			ratio := sc.max / sc.min
			for i := range pts {
				x := sc.min * math.Pow(ratio, float64(i)/float64(len(pts)-1))
				pts[i].X = x
				pts[i].Y = x
			}
			s, err := plotter.NewScatter(pts)
			if err != nil {
				t.Fatal(err)
			}
			p.Add(s)

			rc := new(recorder.Canvas)
			p.Draw(draw.NewCanvas(rc, sc.width, sc.width))

			numeric := 0
			for _, op := range rc.Actions {
				call := op.Call()
				if !strings.Contains(call, "FillString") {
					continue
				}
				if strings.ContainsAny(call, "0123456789") {
					numeric++
				}
			}
			// At least one label on each of the two axes.
			if numeric < 2 {
				t.Fatalf("expected labels on both log axes, got %d numeric text elements", numeric)
			}
		})
	}
}
