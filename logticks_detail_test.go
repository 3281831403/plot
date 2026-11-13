package plot

import (
	"math"
	"testing"
)

func TestLogTicksSubDecadeDetailed(t *testing.T) {
	const eps = dlamchP * 1000
	cases := []struct {
		name       string
		min, max   float64
		prec       int
		wantLabels []string // exact label strings that must appear
	}{
		{"200-800", 200, 800, -1, []string{"200", "400", "800"}},
		{"no-grid-5n-9n", 5e-9, 9e-9, -1, []string{"5e-09", "7e-09", "9e-09"}},
		{"fraction-0.2-0.8", 0.2, 0.8, -1, []string{"0.2", "0.4", "0.8"}},
		{"millions", 2e6, 8e6, -1, []string{"2e+06", "4e+06", "8e+06"}},
		{"single-grid-210-240", 210, 240, -1, []string{"210", "240"}},
		{"endpoint-grid-200-300", 200, 300, -1, []string{"200", "300"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ticks := LogTicks{Prec: tc.prec}.Ticks(tc.min, tc.max)

			// Every tick must be in non-decreasing order.
			prev := -1.0
			gotLabels := map[string]bool{}
			var inRangeValues []float64
			for _, tk := range ticks {
				if tk.Value < prev {
					t.Fatalf("tick %g out of order (prev %g)", tk.Value, prev)
				}
				prev = tk.Value
				if tk.IsMinor() {
					continue
				}
				// Labels must render cleanly without float noise.
				if noisyLabel(tk.Label) {
					t.Errorf("label %q contains floating-point noise", tk.Label)
				}
				if tk.Value >= tc.min-math.Abs(tc.min)*eps && tk.Value <= tc.max+math.Abs(tc.max)*eps {
					gotLabels[tk.Label] = true
					inRangeValues = append(inRangeValues, tk.Value)
				}
			}

			if len(inRangeValues) == 0 {
				t.Fatal("no in-range labels")
			}
			t.Logf("in-range labels: %v", mapKeys(gotLabels))

			for _, want := range tc.wantLabels {
				if !gotLabels[want] {
					t.Errorf("expected label %q, not present; got %v", want, mapKeys(gotLabels))
				}
			}
		})
	}
}

// noisyLabel reports whether a rendered tick label exposes long floating-point
// artifacts such as "7.000000000000001e-09" rather than the intended "7e-09".
func noisyLabel(s string) bool {
	// A clean label has at most a few significant digits; noise shows up as
	// many digits after the decimal point or in the mantissa.
	digits := 0
	seenDot := false
	for _, r := range s {
		switch {
		case r >= '1' && r <= '9':
			digits++
		case r == '.':
			seenDot = true
		}
		if seenDot && digits > 6 {
			return true
		}
	}
	return false
}

func mapKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
