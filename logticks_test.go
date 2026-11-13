package plot

import (
	"testing"

	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/recorder"
)

func TestLogTicksSubDecade(t *testing.T) {
	cases := []struct {
		name     string
		min, max float64
		prec     int
	}{
		{"200-800", 200, 800, -1},
		{"5e-9-9e-9", 5e-9, 9e-9, -1},
		{"0.2-0.8", 0.2, 0.8, -1},
		{"0.002-0.008", 0.002, 0.008, 3},
		{"2e6-8e6", 2e6, 8e6, -1},
		{"boundary-100-1000", 100, 1000, -1},
		{"single-grid-inside", 210, 240, -1},
		{"equal-grid-endpoint", 200, 300, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ticks := LogTicks{Prec: tc.prec}.Ticks(tc.min, tc.max)
			var nLabeled int
			var inRange []string
			for _, tk := range ticks {
				if tk.IsMinor() {
					continue
				}
				nLabeled++
				if tk.Value >= tc.min && tk.Value <= tc.max {
					inRange = append(inRange, tk.Label)
				}
			}
			if len(inRange) == 0 {
				t.Fatalf("no in-range labels among %d labeled ticks", nLabeled)
			}
			t.Logf("labels in range: %v", inRange)

			// Ticks are produced in non-decreasing order. Note that the
			// logarithmic grid has always emitted each power-of-ten value
			// twice (once as the labelled decade tick and again as the
			// decade's first minor tick), so duplicates are not an error.
			prev := -1.0
			for _, tk := range ticks {
				if tk.Value < prev {
					t.Errorf("ticks not sorted at %g", tk.Value)
				}
				prev = tk.Value
			}
		})
	}
}

func TestLogTicksPanics(t *testing.T) {
	for _, r := range [][2]float64{{0, 10}, {-1, 10}, {10, 10}, {10, 5}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("expected panic for range %v", r)
				}
			}()
			LogTicks{}.Ticks(r[0], r[1])
		}()
	}
}

func TestAxisIsLogScale(t *testing.T) {
	cases := []struct {
		name string
		s    Normalizer
		want bool
	}{
		{"linear", LinearScale{}, false},
		{"log", LogScale{}, true},
		{"inverted-log", InvertedScale{Normalizer: LogScale{}}, true},
		{"inverted-linear", InvertedScale{Normalizer: LinearScale{}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := Axis{Scale: tc.s}
			if got := axisIsLogScale(a); got != tc.want {
				t.Errorf("axisIsLogScale(%T) = %v, want %v", tc.s, got, tc.want)
			}
		})
	}
}

func TestSnapToDecimalGrid(t *testing.T) {
	cases := []struct {
		in, want float64
	}{
		{7.000000000000001e-9, 7e-9},
		{399.99999999999994, 400},
		{0.40000000000000013, 0.4},
		{2e6, 2e6},
	}
	for _, tc := range cases {
		if got := snapToDecimalGrid(tc.in); got != tc.want {
			t.Errorf("snapToDecimalGrid(%g) = %g, want %g", tc.in, got, tc.want)
		}
	}
}

func TestPruneOnlyAffectsLogScale(t *testing.T) {
	// A linear axis must never have labels pruned, even when they would be
	// treated as overlapping, so linear plots keep their exact appearance.
	p := New()
	p.X.Min = 0
	p.X.Max = 10
	marks := []Tick{
		{Value: 0, Label: "0"},
		{Value: 5, Label: "5"},
		{Value: 10, Label: "10"},
	}
	c := draw.NewCanvas(new(recorder.Canvas), vg.Inch, vg.Inch)
	hidden := pruneOverlappingLabels(p.X, c, marks)
	for i, h := range hidden {
		if h {
			t.Errorf("linear axis label %q was pruned", marks[i].Label)
		}
	}
}
