// Copyright ©2026 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"math"
	"testing"
	"time"
	_ "time/tzdata"
)

// newYork returns a fixed DST-observing location for the time tick tests.
func newYork(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("could not load location: %v", err)
	}
	return loc
}

// checkWellFormedTicks asserts that ticks are strictly increasing and
// all lie within [min, max].
func checkWellFormedTicks(t *testing.T, min, max float64, ticks []Tick) {
	t.Helper()
	if len(ticks) == 0 {
		t.Fatal("expected at least one tick")
	}
	prev := math.Inf(-1)
	for _, tk := range ticks {
		if tk.Value <= prev {
			t.Errorf("ticks not strictly increasing: %v after %v", tk.Value, prev)
		}
		if tk.Value < min || tk.Value > max {
			t.Errorf("tick %v out of range [%v, %v]", tk.Value, min, max)
		}
		prev = tk.Value
	}
}

// tickTimes converts tick values to times in loc.
func tickTimes(ticks []Tick, loc *time.Location) []time.Time {
	ts := make([]time.Time, len(ticks))
	for i, tk := range ticks {
		ts[i] = time.Unix(int64(tk.Value), 0).In(loc)
	}
	return ts
}

func unix(t time.Time) float64 { return float64(t.Unix()) }

func TestTimeTicksCalendarBoundaries(t *testing.T) {
	loc := newYork(t)

	for _, test := range []struct {
		name     string
		min, max time.Time
		// want is the expected wall-clock time of each tick.
		want []time.Time
	}{
		{
			// A span of a few minutes: ticks on whole minutes.
			name: "minutes",
			min:  time.Date(2024, 3, 10, 0, 3, 7, 0, loc),
			max:  time.Date(2024, 3, 10, 0, 41, 53, 0, loc),
			want: []time.Time{
				time.Date(2024, 3, 10, 0, 10, 0, 0, loc),
				time.Date(2024, 3, 10, 0, 20, 0, 0, loc),
				time.Date(2024, 3, 10, 0, 30, 0, 0, loc),
				time.Date(2024, 3, 10, 0, 40, 0, 0, loc),
			},
		},
		{
			// Spring forward: 2024-03-10 02:00 does not exist
			// in New York. Hourly ticks must skip it.
			name: "spring-forward-hourly",
			min:  time.Date(2024, 3, 10, 0, 30, 0, 0, loc),
			max:  time.Date(2024, 3, 10, 5, 30, 0, 0, loc),
			want: []time.Time{
				time.Date(2024, 3, 10, 1, 0, 0, 0, loc),
				time.Date(2024, 3, 10, 3, 0, 0, 0, loc),
				time.Date(2024, 3, 10, 4, 0, 0, 0, loc),
				time.Date(2024, 3, 10, 5, 0, 0, 0, loc),
			},
		},
		{
			// Fall back: 2024-11-03 01:00 occurs twice in
			// New York. It must be reported only once.
			name: "fall-back-hourly",
			min:  time.Date(2024, 11, 3, 0, 30, 0, 0, loc),
			max:  time.Date(2024, 11, 3, 5, 30, 0, 0, loc),
			want: []time.Time{
				time.Date(2024, 11, 3, 1, 0, 0, 0, loc),
				time.Date(2024, 11, 3, 2, 0, 0, 0, loc),
				time.Date(2024, 11, 3, 3, 0, 0, 0, loc),
				time.Date(2024, 11, 3, 4, 0, 0, 0, loc),
				time.Date(2024, 11, 3, 5, 0, 0, 0, loc),
			},
		},
		{
			// A four-day span: ticks on local midnights,
			// including across the spring DST transition.
			name: "midnights-four-days",
			min:  time.Date(2024, 3, 8, 4, 26, 0, 0, loc),
			max:  time.Date(2024, 3, 12, 8, 13, 0, 0, loc),
			want: []time.Time{
				time.Date(2024, 3, 9, 0, 0, 0, 0, loc),
				time.Date(2024, 3, 10, 0, 0, 0, 0, loc),
				time.Date(2024, 3, 11, 0, 0, 0, 0, loc),
				time.Date(2024, 3, 12, 0, 0, 0, 0, loc),
			},
		},
		{
			// A few months: ticks on the first of each month.
			name: "first-of-month",
			min:  time.Date(2024, 1, 15, 0, 0, 0, 0, loc),
			max:  time.Date(2024, 5, 15, 0, 0, 0, 0, loc),
			want: []time.Time{
				time.Date(2024, 2, 1, 0, 0, 0, 0, loc),
				time.Date(2024, 3, 1, 0, 0, 0, 0, loc),
				time.Date(2024, 4, 1, 0, 0, 0, 0, loc),
				time.Date(2024, 5, 1, 0, 0, 0, 0, loc),
			},
		},
		{
			// A decade: ticks on the first of January.
			name: "first-of-january",
			min:  time.Date(2000, 6, 1, 0, 0, 0, 0, loc),
			max:  time.Date(2010, 6, 1, 0, 0, 0, 0, loc),
			want: []time.Time{
				time.Date(2002, 1, 1, 0, 0, 0, 0, loc),
				time.Date(2004, 1, 1, 0, 0, 0, 0, loc),
				time.Date(2006, 1, 1, 0, 0, 0, 0, loc),
				time.Date(2008, 1, 1, 0, 0, 0, 0, loc),
				time.Date(2010, 1, 1, 0, 0, 0, 0, loc),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			min, max := unix(test.min), unix(test.max)
			xt := TimeTicks{
				Format: "2006-01-02 15:04",
				Time:   UnixTimeIn(loc),
			}
			ticks := xt.Ticks(min, max)
			checkWellFormedTicks(t, min, max, ticks)

			got := tickTimes(ticks, loc)
			if len(got) != len(test.want) {
				t.Fatalf("unexpected number of ticks: got %d %v, want %d", len(got), got, len(test.want))
			}
			for i, want := range test.want {
				if !got[i].Equal(want) {
					t.Errorf("tick %d: got %v, want %v", i, got[i], want)
				}
			}
		})
	}
}

func TestTimeTicksDSTSpacing(t *testing.T) {
	loc := newYork(t)
	xt := TimeTicks{Time: UnixTimeIn(loc)}

	// The 2024-03-10 spring-forward day has 23 hours: the two
	// midnight ticks bracketing it must be 23 hours apart.
	min := unix(time.Date(2024, 3, 8, 4, 26, 0, 0, loc))
	max := unix(time.Date(2024, 3, 12, 8, 13, 0, 0, loc))
	ticks := xt.Ticks(min, max)
	checkWellFormedTicks(t, min, max, ticks)
	if d := ticks[2].Value - ticks[1].Value; d != 23*3600 {
		t.Errorf("spring day length: got %v s, want %v s", d, 23*3600)
	}

	// The 2024-11-03 fall-back day has 25 hours: the repeated
	// 01:00 is folded into a single tick two hours before 02:00.
	min = unix(time.Date(2024, 11, 3, 0, 30, 0, 0, loc))
	max = unix(time.Date(2024, 11, 3, 5, 30, 0, 0, loc))
	ticks = xt.Ticks(min, max)
	checkWellFormedTicks(t, min, max, ticks)
	ones := 0
	for _, tm := range tickTimes(ticks, loc) {
		if tm.Hour() == 1 {
			ones++
		}
	}
	if ones != 1 {
		t.Errorf("got %d ticks at 01:00, want exactly 1", ones)
	}
	if d := ticks[1].Value - ticks[0].Value; d != 2*3600 {
		t.Errorf("01:00 to 02:00 on fall-back day: got %v s, want %v s", d, 2*3600)
	}
}

func TestTimeTicksDefaultUTC(t *testing.T) {
	// The default Time conversion is UTCUnixTime: ticks fall on
	// UTC calendar boundaries.
	min := unix(time.Date(2024, 3, 8, 4, 26, 0, 0, time.UTC))
	max := unix(time.Date(2024, 3, 12, 8, 13, 0, 0, time.UTC))
	ticks := (TimeTicks{}).Ticks(min, max)
	checkWellFormedTicks(t, min, max, ticks)
	for _, tm := range tickTimes(ticks, time.UTC) {
		if tm.Hour() != 0 || tm.Minute() != 0 || tm.Second() != 0 {
			t.Errorf("default UTC tick not at midnight: %v", tm)
		}
	}
	if got, want := ticks[0].Label, "2024-03-09T00:00:00Z"; got != want {
		t.Errorf("unexpected default RFC3339 label: got %q, want %q", got, want)
	}
}

func TestTimeTicksCustomTickerAndFormat(t *testing.T) {
	// A user-provided Ticker is used as-is, with labels produced
	// by Format and Time.
	xt := TimeTicks{
		Ticker: ConstantTicks{{Value: 100, Label: "ignored"}},
		Format: "15:04",
	}
	ticks := xt.Ticks(0, 200)
	if len(ticks) != 1 || ticks[0].Value != 100 {
		t.Fatalf("custom ticker not used: %+v", ticks)
	}
	if got, want := ticks[0].Label, time.Unix(100, 0).UTC().Format("15:04"); got != want {
		t.Errorf("unexpected label: got %q, want %q", got, want)
	}
}

func TestTimeTicksNonUnixTimeFallsBack(t *testing.T) {
	// A Time function that is not a unix-seconds conversion
	// falls back to DefaultTicks instead of misplacing ticks.
	xt := TimeTicks{
		Time: func(v float64) time.Time {
			return time.Unix(int64(v)+3600, 0).UTC()
		},
	}
	ticks := xt.Ticks(0, 100)
	if len(ticks) == 0 {
		t.Fatal("expected fallback ticks")
	}
}
