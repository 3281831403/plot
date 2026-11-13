// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"image/color"
	"math"
	"strconv"
	"time"

	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/text"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// Ticker creates Ticks in a specified range
type Ticker interface {
	// Ticks returns Ticks in a specified range
	Ticks(min, max float64) []Tick
}

// Normalizer rescales values from the data coordinate system to the
// normalized coordinate system.
type Normalizer interface {
	// Normalize transforms a value x in the data coordinate system to
	// the normalized coordinate system.
	Normalize(min, max, x float64) float64
}

// An Axis represents either a horizontal or vertical
// axis of a plot.
type Axis struct {
	// Min and Max are the minimum and maximum data
	// values represented by the axis.
	Min, Max float64

	Label struct {
		// Text is the axis label string.
		Text string

		// Padding is the distance between the label and the axis.
		Padding vg.Length

		// TextStyle is the style of the axis label text.
		// For the vertical axis, one quarter turn
		// counterclockwise will be added to the label
		// text before drawing.
		TextStyle text.Style

		// Position is where the axis label string should be drawn.
		// The default value is draw.PosCenter, displaying the label
		// at the center of the axis.
		// Valid values are [-1,+1], with +1 being the far right/top
		// of the axis, and -1 the far left/bottom of the axis.
		Position float64
	}

	// LineStyle is the style of the axis line.
	draw.LineStyle

	// Padding between the axis line and the data.  Having
	// non-zero padding ensures that the data is never drawn
	// on the axis, thus making it easier to see.
	Padding vg.Length

	Tick struct {
		// Label is the TextStyle on the tick labels.
		Label text.Style

		// LineStyle is the LineStyle of the tick lines.
		draw.LineStyle

		// Length is the length of a major tick mark.
		// Minor tick marks are half of the length of major
		// tick marks.
		Length vg.Length

		// Marker returns the tick marks.  Any tick marks
		// returned by the Marker function that are not in
		// range of the axis are not drawn.
		Marker Ticker
	}

	// Scale transforms a value given in the data coordinate system
	// to the normalized coordinate system of the axis—its distance
	// along the axis as a fraction of the axis range.
	Scale Normalizer

	// AutoRescale enables an axis to automatically adapt its minimum
	// and maximum boundaries, according to its underlying Ticker.
	AutoRescale bool
}

// makeAxis returns a default Axis.
//
// The default range is (∞, ­∞), and thus any finite
// value is less than Min and greater than Max.
func makeAxis(o orientation) Axis {

	a := Axis{
		Min: math.Inf(+1),
		Max: math.Inf(-1),
		LineStyle: draw.LineStyle{
			Color: color.Black,
			Width: vg.Points(0.5),
		},
		Padding: vg.Points(5),
		Scale:   LinearScale{},
	}
	a.Label.TextStyle = text.Style{
		Color:   color.Black,
		Font:    font.From(DefaultFont, 12),
		XAlign:  draw.XCenter,
		YAlign:  draw.YBottom,
		Handler: DefaultTextHandler,
	}
	a.Label.Position = draw.PosCenter

	var (
		xalign draw.XAlignment
		yalign draw.YAlignment
	)
	switch o {
	case vertical:
		xalign = draw.XRight
		yalign = draw.YCenter
	case horizontal:
		xalign = draw.XCenter
		yalign = draw.YTop
	}

	a.Tick.Label = text.Style{
		Color:   color.Black,
		Font:    font.From(DefaultFont, 10),
		XAlign:  xalign,
		YAlign:  yalign,
		Handler: DefaultTextHandler,
	}
	a.Tick.LineStyle = draw.LineStyle{
		Color: color.Black,
		Width: vg.Points(0.5),
	}
	a.Tick.Length = vg.Points(8)
	a.Tick.Marker = DefaultTicks{}

	return a
}

// sanitizeRange ensures that the range of the
// axis makes sense.
func (a *Axis) sanitizeRange() {
	if math.IsInf(a.Min, 0) {
		a.Min = 0
	}
	if math.IsInf(a.Max, 0) {
		a.Max = 0
	}
	if a.Min > a.Max {
		a.Min, a.Max = a.Max, a.Min
	}
	if a.Min == a.Max {
		a.Min--
		a.Max++
	}

	if a.AutoRescale {
		marks := a.Tick.Marker.Ticks(a.Min, a.Max)
		for _, t := range marks {
			a.Min = math.Min(a.Min, t.Value)
			a.Max = math.Max(a.Max, t.Value)
		}
	}
}

// LinearScale an be used as the value of an Axis.Scale function to
// set the axis to a standard linear scale.
type LinearScale struct{}

var _ Normalizer = LinearScale{}

// Normalize returns the fractional distance of x between min and max.
func (LinearScale) Normalize(min, max, x float64) float64 {
	return (x - min) / (max - min)
}

// LogScale can be used as the value of an Axis.Scale function to
// set the axis to a log scale.
type LogScale struct{}

var _ Normalizer = LogScale{}

// Normalize returns the fractional logarithmic distance of
// x between min and max.
func (LogScale) Normalize(min, max, x float64) float64 {
	if min <= 0 || max <= 0 || x <= 0 {
		panic("Values must be greater than 0 for a log scale.")
	}
	logMin := math.Log(min)
	return (math.Log(x) - logMin) / (math.Log(max) - logMin)
}

// InvertedScale can be used as the value of an Axis.Scale function to
// invert the axis using any Normalizer.
type InvertedScale struct{ Normalizer }

var _ Normalizer = InvertedScale{}

// Normalize returns a normalized [0, 1] value for the position of x.
func (is InvertedScale) Normalize(min, max, x float64) float64 {
	return is.Normalizer.Normalize(max, min, x)
}

// Norm returns the value of x, given in the data coordinate
// system, normalized to its distance as a fraction of the
// range of this axis.  For example, if x is a.Min then the return
// value is 0, and if x is a.Max then the return value is 1.
func (a Axis) Norm(x float64) float64 {
	return a.Scale.Normalize(a.Min, a.Max, x)
}

// drawTicks returns true if the tick marks should be drawn.
func (a Axis) drawTicks() bool {
	return a.Tick.Width > 0 && a.Tick.Length > 0
}

// A horizontalAxis draws horizontally across the bottom
// of a plot.
type horizontalAxis struct {
	Axis
}

// size returns the height of the axis.
func (a horizontalAxis) size() (h vg.Length) {
	if a.Label.Text != "" { // We assume that the label isn't rotated.
		h += a.Label.TextStyle.Height(a.Label.Text)
		h += a.Label.Padding
	}

	marks := a.Tick.Marker.Ticks(a.Min, a.Max)
	if len(marks) > 0 {
		if a.drawTicks() {
			h += a.Tick.Length
		}
		h += tickLabelHeight(a.Tick.Label, marks)
	}
	h += a.Width / 2
	h += a.Padding

	return h
}

// draw draws the axis along the lower edge of a draw.Canvas.
func (a horizontalAxis) draw(c draw.Canvas) {
	var (
		x vg.Length
		y = c.Min.Y
	)
	switch a.Label.Position {
	case draw.PosCenter:
		x = c.Center().X
	case draw.PosRight:
		x = c.Max.X
		x -= a.Label.TextStyle.Width(a.Label.Text) / 2
	}
	if a.Label.Text != "" {
		descent := a.Label.TextStyle.FontExtents().Descent
		c.FillText(a.Label.TextStyle, vg.Point{X: x, Y: y + descent}, a.Label.Text)
		y += a.Label.TextStyle.Height(a.Label.Text)
		y += a.Label.Padding
	}

	marks := a.Tick.Marker.Ticks(a.Min, a.Max)
	ticklabelheight := tickLabelHeight(a.Tick.Label, marks)
	descent := a.Tick.Label.FontExtents().Descent
	for _, t := range marks {
		x := c.X(a.Norm(t.Value))
		if !c.ContainsX(x) || t.IsMinor() {
			continue
		}
		c.FillText(a.Tick.Label, vg.Point{X: x, Y: y + ticklabelheight + descent}, t.Label)
	}

	if len(marks) > 0 {
		y += ticklabelheight
	} else {
		y += a.Width / 2
	}

	if len(marks) > 0 && a.drawTicks() {
		len := a.Tick.Length
		for _, t := range marks {
			x := c.X(a.Norm(t.Value))
			if !c.ContainsX(x) {
				continue
			}
			start := t.lengthOffset(len)
			c.StrokeLine2(a.Tick.LineStyle, x, y+start, x, y+len)
		}
		y += len
	}

	c.StrokeLine2(a.LineStyle, c.Min.X, y, c.Max.X, y)
}

// GlyphBoxes returns the GlyphBoxes for the tick labels.
func (a horizontalAxis) GlyphBoxes(p *Plot) []GlyphBox {
	var (
		boxes []GlyphBox
		yoff  font.Length
	)

	if a.Label.Text != "" {
		x := a.Norm(p.X.Max)
		switch a.Label.Position {
		case draw.PosCenter:
			x = a.Norm(0.5 * (p.X.Max + p.X.Min))
		case draw.PosRight:
			x -= a.Norm(0.5 * a.Label.TextStyle.Width(a.Label.Text).Points()) // FIXME(sbinet): want data coordinates
		}
		descent := a.Label.TextStyle.FontExtents().Descent
		boxes = append(boxes, GlyphBox{
			X:         x,
			Rectangle: a.Label.TextStyle.Rectangle(a.Label.Text).Add(vg.Point{Y: yoff + descent}),
		})
		yoff += a.Label.TextStyle.Height(a.Label.Text)
		yoff += a.Label.Padding
	}

	var (
		marks   = a.Tick.Marker.Ticks(a.Min, a.Max)
		height  = tickLabelHeight(a.Tick.Label, marks)
		descent = a.Tick.Label.FontExtents().Descent
	)
	for _, t := range marks {
		if t.IsMinor() {
			continue
		}
		box := GlyphBox{
			X:         a.Norm(t.Value),
			Rectangle: a.Tick.Label.Rectangle(t.Label).Add(vg.Point{Y: yoff + height + descent}),
		}
		boxes = append(boxes, box)
	}
	return boxes
}

// A verticalAxis is drawn vertically up the left side of a plot.
type verticalAxis struct {
	Axis
}

// size returns the width of the axis.
func (a verticalAxis) size() (w vg.Length) {
	if a.Label.Text != "" { // We assume that the label isn't rotated.
		w += a.Label.TextStyle.FontExtents().Descent
		w += a.Label.TextStyle.Height(a.Label.Text)
		w += a.Label.Padding
	}

	marks := a.Tick.Marker.Ticks(a.Min, a.Max)
	if len(marks) > 0 {
		if lwidth := tickLabelWidth(a.Tick.Label, marks); lwidth > 0 {
			w += lwidth
			w += a.Label.TextStyle.Width(" ")
		}
		if a.drawTicks() {
			w += a.Tick.Length
		}
	}
	w += a.Width / 2
	w += a.Padding

	return w
}

// draw draws the axis along the left side of a draw.Canvas.
func (a verticalAxis) draw(c draw.Canvas) {
	var (
		x = c.Min.X
		y vg.Length
	)
	if a.Label.Text != "" {
		sty := a.Label.TextStyle
		sty.Rotation += math.Pi / 2
		x += a.Label.TextStyle.Height(a.Label.Text)
		switch a.Label.Position {
		case draw.PosCenter:
			y = c.Center().Y
		case draw.PosTop:
			y = c.Max.Y
			y -= a.Label.TextStyle.Width(a.Label.Text) / 2
		}
		descent := a.Label.TextStyle.FontExtents().Descent
		c.FillText(sty, vg.Point{X: x - descent, Y: y}, a.Label.Text)
		x += descent
		x += a.Label.Padding
	}
	marks := a.Tick.Marker.Ticks(a.Min, a.Max)
	if w := tickLabelWidth(a.Tick.Label, marks); len(marks) > 0 && w > 0 {
		x += w
	}

	major := false
	descent := a.Tick.Label.FontExtents().Descent
	for _, t := range marks {
		y := c.Y(a.Norm(t.Value))
		if !c.ContainsY(y) || t.IsMinor() {
			continue
		}
		c.FillText(a.Tick.Label, vg.Point{X: x, Y: y + descent}, t.Label)
		major = true
	}
	if major {
		x += a.Tick.Label.Width(" ")
	}
	if a.drawTicks() && len(marks) > 0 {
		len := a.Tick.Length
		for _, t := range marks {
			y := c.Y(a.Norm(t.Value))
			if !c.ContainsY(y) {
				continue
			}
			start := t.lengthOffset(len)
			c.StrokeLine2(a.Tick.LineStyle, x+start, y, x+len, y)
		}
		x += len
	}

	c.StrokeLine2(a.LineStyle, x, c.Min.Y, x, c.Max.Y)
}

// GlyphBoxes returns the GlyphBoxes for the tick labels
func (a verticalAxis) GlyphBoxes(p *Plot) []GlyphBox {
	var (
		boxes []GlyphBox
		xoff  font.Length
	)

	if a.Label.Text != "" {
		yoff := a.Norm(p.Y.Max)
		switch a.Label.Position {
		case draw.PosCenter:
			yoff = a.Norm(0.5 * (p.Y.Max + p.Y.Min))
		case draw.PosTop:
			yoff -= a.Norm(0.5 * a.Label.TextStyle.Width(a.Label.Text).Points()) // FIXME(sbinet): want data coordinates
		}

		sty := a.Label.TextStyle
		sty.Rotation += math.Pi / 2

		xoff += a.Label.TextStyle.Height(a.Label.Text)
		descent := a.Label.TextStyle.FontExtents().Descent
		boxes = append(boxes, GlyphBox{
			Y:         yoff,
			Rectangle: sty.Rectangle(a.Label.Text).Add(vg.Point{X: xoff - descent}),
		})
		xoff += descent
		xoff += a.Label.Padding
	}

	marks := a.Tick.Marker.Ticks(a.Min, a.Max)
	if w := tickLabelWidth(a.Tick.Label, marks); len(marks) != 0 && w > 0 {
		xoff += w
	}

	var (
		ext  = a.Tick.Label.FontExtents()
		desc = ext.Height - ext.Ascent // descent + linegap
	)
	for _, t := range marks {
		if t.IsMinor() {
			continue
		}
		box := GlyphBox{
			Y:         a.Norm(t.Value),
			Rectangle: a.Tick.Label.Rectangle(t.Label).Add(vg.Point{X: xoff, Y: desc}),
		}
		boxes = append(boxes, box)
	}
	return boxes
}

// DefaultTicks is suitable for the Tick.Marker field of an Axis,
// it returns a reasonable default set of tick marks.
type DefaultTicks struct{}

var _ Ticker = DefaultTicks{}

// Ticks returns Ticks in the specified range.
func (DefaultTicks) Ticks(min, max float64) []Tick {
	if max <= min {
		panic("illegal range")
	}

	const suggestedTicks = 3

	labels, step, q, mag := talbotLinHanrahan(min, max, suggestedTicks, withinData, nil, nil, nil)
	majorDelta := step * math.Pow10(mag)
	if q == 0 {
		// Simple fall back was chosen, so
		// majorDelta is the label distance.
		majorDelta = labels[1] - labels[0]
	}

	// Choose a reasonable, but ad
	// hoc formatting for labels.
	fc := byte('f')
	var off int
	if mag < -1 || 6 < mag {
		off = 1
		fc = 'g'
	}
	if math.Trunc(q) != q {
		off += 2
	}
	prec := minInt(6, maxInt(off, -mag))
	ticks := make([]Tick, len(labels))
	for i, v := range labels {
		ticks[i] = Tick{Value: v, Label: strconv.FormatFloat(v, fc, prec, 64)}
	}

	var minorDelta float64
	// See talbotLinHanrahan for the values used here.
	switch step {
	case 1, 2.5:
		minorDelta = majorDelta / 5
	case 2, 3, 4, 5:
		minorDelta = majorDelta / step
	default:
		if majorDelta/2 < dlamchP {
			return ticks
		}
		minorDelta = majorDelta / 2
	}

	// Find the first minor tick not greater
	// than the lowest data value.
	var i float64
	for labels[0]+(i-1)*minorDelta > min {
		i--
	}
	// Add ticks at minorDelta intervals when
	// they are not within minorDelta/2 of a
	// labelled tick.
	for {
		val := labels[0] + i*minorDelta
		if val > max {
			break
		}
		found := false
		for _, t := range ticks {
			if math.Abs(t.Value-val) < minorDelta/2 {
				found = true
			}
		}
		if !found {
			ticks = append(ticks, Tick{Value: val})
		}
		i++
	}

	return ticks
}

func minInt(a, b int) int {
	return min(a, b)
}

func maxInt(a, b int) int {
	return max(a, b)
}

// LogTicks is suitable for the Tick.Marker field of an Axis,
// it returns tick marks suitable for a log-scale axis.
type LogTicks struct {
	// Prec specifies the precision of tick rendering
	// according to the documentation for strconv.FormatFloat.
	Prec int
}

var _ Ticker = LogTicks{}

// Ticks returns Ticks in a specified range
func (t LogTicks) Ticks(min, max float64) []Tick {
	if min <= 0 || max <= 0 {
		panic("Values must be greater than 0 for a log scale.")
	}

	val := math.Pow10(int(math.Log10(min)))
	max = math.Pow10(int(math.Ceil(math.Log10(max))))
	var ticks []Tick
	for val < max {
		for i := 1; i < 10; i++ {
			if i == 1 {
				ticks = append(ticks, Tick{Value: val, Label: formatFloatTick(val, t.Prec)})
			}
			ticks = append(ticks, Tick{Value: val * float64(i)})
		}
		val *= 10
	}
	ticks = append(ticks, Tick{Value: val, Label: formatFloatTick(val, t.Prec)})

	return ticks
}

// ConstantTicks is suitable for the Tick.Marker field of an Axis.
// This function returns the given set of ticks.
type ConstantTicks []Tick

var _ Ticker = ConstantTicks{}

// Ticks returns Ticks in a specified range
func (ts ConstantTicks) Ticks(float64, float64) []Tick {
	return ts
}

// UnixTimeIn returns a time conversion function for the given location.
func UnixTimeIn(loc *time.Location) func(t float64) time.Time {
	return func(t float64) time.Time {
		return time.Unix(int64(t), 0).In(loc)
	}
}

// UTCUnixTime is the default time conversion for TimeTicks.
var UTCUnixTime = UnixTimeIn(time.UTC)

// TimeTicks is suitable for axes representing time values.
type TimeTicks struct {
	// Ticker is used to generate a set of ticks.
	// If nil, ticks are placed on calendar boundaries (whole
	// minutes, hours, local midnights, first of month or first
	// of January) in the location of the Time function, chosen
	// according to the span of the axis. Calendar ticks are
	// computed in local wall-clock time, so they never fall on
	// times that do not exist (such as during a spring DST
	// transition) and are not duplicated when a wall-clock time
	// occurs twice (such as during an autumn DST transition).
	// If the Time function is not a unix-seconds conversion,
	// DefaultTicks will be used instead.
	Ticker Ticker

	// Format is the textual representation of the time value.
	// If empty, time.RFC3339 will be used
	Format string

	// Time takes a float64 value and converts it into a time.Time.
	// If nil, UTCUnixTime is used.
	Time func(t float64) time.Time
}

var _ Ticker = TimeTicks{}

// Ticks implements plot.Ticker.
func (t TimeTicks) Ticks(min, max float64) []Tick {
	if t.Format == "" {
		t.Format = time.RFC3339
	}
	if t.Time == nil {
		t.Time = UTCUnixTime
	}

	if t.Ticker == nil {
		if ticks, ok := t.calendarTicks(min, max); ok {
			return ticks
		}
		t.Ticker = DefaultTicks{}
	}

	ticks := t.Ticker.Ticks(min, max)
	for i := range ticks {
		tick := &ticks[i]
		if tick.Label == "" {
			continue
		}
		tick.Label = t.Time(tick.Value).Format(t.Format)
	}
	return ticks
}

// calendarTicks returns ticks placed on calendar boundaries in the
// location of the Time function. It reports whether calendar ticks
// apply: they require a unix-seconds time conversion such as the one
// returned by UnixTimeIn.
func (t TimeTicks) calendarTicks(min, max float64) ([]Tick, bool) {
	if !(min < max) {
		return nil, false
	}
	if t.Time(min).Unix() != int64(min) || t.Time(max).Unix() != int64(max) {
		// The time conversion is not based on unix seconds,
		// so calendar boundaries cannot be inverted back to
		// axis values.
		return nil, false
	}
	loc := t.Time(min).Location()

	vals, ok := calendarTickValues(min, max, loc)
	if !ok {
		return nil, false
	}
	ticks := make([]Tick, len(vals))
	for i, v := range vals {
		ticks[i] = Tick{Value: v, Label: t.Time(v).Format(t.Format)}
	}
	return ticks, true
}

// Units for calendarStep.
const (
	calendarSecond = iota
	calendarMinute
	calendarHour
	calendarDay
	calendarMonth
	calendarYear
)

// calendarStep is a step between successive calendar-aligned ticks,
// expressed in wall-clock (civil) time.
type calendarStep struct {
	unit int
	n    int
}

// calendarSteps is the ordered list of candidate steps between
// calendar-aligned ticks, from the shortest to the longest.
var calendarSteps = []calendarStep{
	{calendarSecond, 1},
	{calendarSecond, 2},
	{calendarSecond, 5},
	{calendarSecond, 10},
	{calendarSecond, 15},
	{calendarSecond, 30},
	{calendarMinute, 1},
	{calendarMinute, 2},
	{calendarMinute, 5},
	{calendarMinute, 10},
	{calendarMinute, 15},
	{calendarMinute, 30},
	{calendarHour, 1},
	{calendarHour, 2},
	{calendarHour, 3},
	{calendarHour, 6},
	{calendarHour, 12},
	{calendarDay, 1},
	{calendarDay, 2},
	{calendarDay, 7},
	{calendarDay, 14},
	{calendarMonth, 1},
	{calendarMonth, 3},
	{calendarMonth, 6},
	{calendarYear, 1},
	{calendarYear, 2},
	{calendarYear, 5},
	{calendarYear, 10},
	{calendarYear, 20},
	{calendarYear, 50},
	{calendarYear, 100},
	{calendarYear, 200},
	{calendarYear, 500},
	{calendarYear, 1000},
	{calendarYear, 2000},
	{calendarYear, 5000},
}

// suggestedCalendarTicks is the target number of ticks on a time axis.
const suggestedCalendarTicks = 3

// calendarTickValues returns unix-seconds values of calendar-aligned
// ticks within [min, max] in the given location. The step between
// ticks is the longest calendar step that yields at least
// suggestedCalendarTicks ticks. The returned values are strictly
// increasing and all lie within [min, max].
func calendarTickValues(min, max float64, loc *time.Location) ([]float64, bool) {
	var best calendarStep
	found := false
	for _, s := range calendarSteps {
		// Count the ticks for this step, stopping as soon as
		// we know there are at least suggestedCalendarTicks+1.
		if vals := s.values(min, max, loc, suggestedCalendarTicks+1); len(vals) >= suggestedCalendarTicks {
			best = s
			found = true
		}
	}
	if !found {
		// No step yields enough ticks; use the longest step
		// that yields at least one.
		for i := len(calendarSteps) - 1; i >= 0; i-- {
			if vals := calendarSteps[i].values(min, max, loc, 2); len(vals) != 0 {
				best = calendarSteps[i]
				found = true
				break
			}
		}
	}
	if !found {
		return nil, false
	}
	return best.values(min, max, loc, 0), true
}

// values returns the unix-seconds values of ticks for s within
// [min, max] in loc. If cap is positive, at most cap values are
// returned. Ticks are placed on wall-clock calendar boundaries:
// wall-clock times that do not exist in loc (such as during a
// spring DST transition) are skipped, and wall-clock times that
// occur twice (such as during an autumn DST transition) are
// reported only once.
func (s calendarStep) values(min, max float64, loc *time.Location, cap int) []float64 {
	start := time.Unix(int64(min), 0).In(loc)
	y, mo, d := start.Date()
	h, mi, sec := start.Clock()

	// Align the first candidate to the step.
	switch s.unit {
	case calendarSecond:
		sec -= sec % s.n
	case calendarMinute:
		sec = 0
		mi -= mi % s.n
	case calendarHour:
		sec, mi = 0, 0
		h -= h % s.n
	case calendarDay:
		sec, mi, h = 0, 0, 0
		d = 1 + (d-1)/s.n*s.n
	case calendarMonth:
		sec, mi, h, d = 0, 0, 0, 1
		mo = 1 + (mo-1)/time.Month(s.n)*time.Month(s.n)
	case calendarYear:
		sec, mi, h, d, mo = 0, 0, 0, 1, 1
		y -= y % s.n
	}

	var vals []float64
	last := int64(math.MinInt64)
	for {
		cand := time.Date(y, mo, d, h, mi, sec, 0, loc)
		if u := cand.Unix(); float64(u) > max {
			break
		} else if u > last && float64(u) >= min &&
			cand.Year() == y && cand.Month() == mo && cand.Day() == d &&
			cand.Hour() == h && cand.Minute() == mi && cand.Second() == sec {
			// The wall-clock time exists in loc (it is not
			// skipped by a DST transition) and is new.
			vals = append(vals, float64(u))
			last = u
			if cap > 0 && len(vals) >= cap {
				return vals
			}
		}

		// Advance to the next candidate in wall-clock time,
		// normalizing field overflows with civil arithmetic.
		switch s.unit {
		case calendarSecond:
			sec += s.n
		case calendarMinute:
			mi += s.n
		case calendarHour:
			h += s.n
		case calendarDay:
			d += s.n
		case calendarMonth:
			mo += time.Month(s.n)
		case calendarYear:
			y += s.n
		}
		t := time.Date(y, mo, d, h, mi, sec, 0, time.UTC)
		y, mo, d = t.Date()
		h, mi, sec = t.Clock()
	}
	return vals
}

// A Tick is a single tick mark on an axis.
type Tick struct {
	// Value is the data value marked by this Tick.
	Value float64

	// Label is the text to display at the tick mark.
	// If Label is an empty string then this is a minor
	// tick mark.
	Label string
}

// IsMinor returns true if this is a minor tick mark.
func (t Tick) IsMinor() bool {
	return t.Label == ""
}

// lengthOffset returns an offset that should be added to the
// tick mark's line to accout for its length.  I.e., the start of
// the line for a minor tick mark must be shifted by half of
// the length.
func (t Tick) lengthOffset(len vg.Length) vg.Length {
	if t.IsMinor() {
		return len / 2
	}
	return 0
}

// tickLabelHeight returns height of the tick mark labels.
func tickLabelHeight(sty text.Style, ticks []Tick) vg.Length {
	maxHeight := vg.Length(0)
	for _, t := range ticks {
		if t.IsMinor() {
			continue
		}
		r := sty.Rectangle(t.Label)
		h := r.Max.Y - r.Min.Y
		if h > maxHeight {
			maxHeight = h
		}
	}
	return maxHeight
}

// tickLabelWidth returns the width of the widest tick mark label.
func tickLabelWidth(sty text.Style, ticks []Tick) vg.Length {
	maxWidth := vg.Length(0)
	for _, t := range ticks {
		if t.IsMinor() {
			continue
		}
		r := sty.Rectangle(t.Label)
		w := r.Max.X - r.Min.X
		if w > maxWidth {
			maxWidth = w
		}
	}
	return maxWidth
}

// formatFloatTick returns a g-formated string representation of v
// to the specified precision.
func formatFloatTick(v float64, prec int) string {
	return strconv.FormatFloat(v, 'g', prec, 64)
}

// TickerFunc is suitable for the Tick.Marker field of an Axis.
// It is an adapter which allows to quickly setup a Ticker using a function with an appropriate signature.
type TickerFunc func(min, max float64) []Tick

var _ Ticker = TickerFunc(nil)

// Ticks implements plot.Ticker.
func (f TickerFunc) Ticks(min, max float64) []Tick {
	return f(min, max)
}
