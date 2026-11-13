// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/text"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// A Legend gives a description of the meaning of different
// data elements of the plot.  Each legend entry has a name
// and a thumbnail, where the thumbnail shows a small
// sample of the display style of the corresponding data.
//
// The legend entries are laid out in a single column as long
// as they fit within the vertical extent of the canvas they
// are drawn on.  When more entries are present than fit in a
// single column, the entries are wrapped into as many columns
// as needed for the whole legend to remain inside the canvas.
// Entries fill each column from top to bottom, in the order
// they were added, before moving on to the next column.
type Legend struct {
	// TextStyle is the style given to the legend
	// entry texts.
	TextStyle text.Style

	// Padding is the amount of padding to add
	// between each entry in the legend.  If Padding
	// is zero then entries are spaced based on the
	// font size.
	Padding vg.Length

	// Top and Left specify the location of the legend.
	// If Top is true the legend is located along the top
	// edge of the plot, otherwise it is located along
	// the bottom edge.  If Left is true then the legend
	// is located along the left edge of the plot, and the
	// text is positioned after the icons, otherwise it is
	// located along the right edge and the text is
	// positioned before the icons.
	Top, Left bool

	// XOffs and YOffs are added to the legend's
	// final position.
	XOffs, YOffs vg.Length

	// YPosition specifies the vertical position of a legend entry.
	// Valid values are [-1,+1], with +1 being the top of the
	// entry vertical space, and -1 the bottom.
	YPosition float64

	// ThumbnailWidth is the width of legend thumbnails.
	ThumbnailWidth vg.Length

	// entries are all of the legendEntries described
	// by this legend.
	entries []legendEntry
}

// A legendEntry represents a single line of a legend, it
// has a name and an icon.
type legendEntry struct {
	// text is the text associated with this entry.
	text string

	// thumbs is a slice of all of the thumbnails styles
	thumbs []Thumbnailer
}

// Thumbnailer wraps the Thumbnail method, which
// draws the small image in a legend representing the
// style of data.
type Thumbnailer interface {
	// Thumbnail draws an thumbnail representing
	// a legend entry.  The thumbnail will usually show
	// a smaller representation of the style used
	// to plot the corresponding data.
	Thumbnail(c *draw.Canvas)
}

// NewLegend returns a legend with the default parameter settings.
func NewLegend() Legend {
	return newLegend(DefaultTextHandler)
}

func newLegend(hdlr text.Handler) Legend {
	return Legend{
		YPosition:      draw.PosBottom,
		ThumbnailWidth: vg.Points(20),
		TextStyle: text.Style{
			Font:    font.From(DefaultFont, 12),
			Handler: hdlr,
		},
	}
}

// Draw draws the legend to the given draw.Canvas.
func (l *Legend) Draw(c draw.Canvas) {
	sty := l.TextStyle
	em := sty.Rectangle(" ")
	enth := l.entryHeight()
	rows, cols, widths := l.layout(c)

	descent := sty.FontExtents().Descent

	if l.YPosition < draw.PosBottom || draw.PosTop < l.YPosition {
		panic("plot: invalid vertical offset for the legend's entries")
	}
	yoff := vg.Length(l.YPosition-draw.PosBottom) / 2
	yoff += descent

	if !l.Left {
		sty.XAlign--
	}

	// xoffs[j] is the horizontal offset of the j-th column of
	// entries from the edge of the canvas the legend is drawn
	// along.  Columns grow towards the inside of the canvas,
	// keeping the first entries where a single-column legend
	// would put them.
	xoffs := make([]vg.Length, cols)
	for j := 1; j < cols; j++ {
		xoffs[j] = xoffs[j-1] + widths[j-1] + l.columnGap()
	}

	// y is the y-coordinate of the bottom of the icon of the
	// first entry of each column.
	y := c.Max.Y - enth - descent
	if !l.Top {
		y = c.Min.Y + (enth+l.Padding)*vg.Length(rows-1)
	}
	y += l.YOffs

	icon := &draw.Canvas{Canvas: c.Canvas}
	for j := 0; j < cols; j++ {
		iconx := c.Min.X + xoffs[j]
		textx := iconx + l.ThumbnailWidth + em.Max.X
		if !l.Left {
			iconx = c.Max.X - l.ThumbnailWidth - xoffs[j]
			textx = iconx - em.Max.X
		}
		textx += l.XOffs
		iconx += l.XOffs

		icon.Rectangle = vg.Rectangle{
			Min: vg.Point{X: iconx, Y: y},
			Max: vg.Point{X: iconx + l.ThumbnailWidth, Y: y + enth},
		}

		end := (j + 1) * rows
		if end > len(l.entries) {
			end = len(l.entries)
		}
		for _, e := range l.entries[j*rows : end] {
			for _, t := range e.thumbs {
				t.Thumbnail(icon)
			}
			yoffs := (enth - descent - sty.Rectangle(e.text).Max.Y) / 2
			yoffs += yoff
			c.FillText(sty, vg.Point{X: textx, Y: icon.Min.Y + yoffs}, e.text)
			icon.Min.Y -= enth + l.Padding
			icon.Max.Y -= enth + l.Padding
		}
	}
}

// Rectangle returns the extent of the Legend.
func (l *Legend) Rectangle(c draw.Canvas) vg.Rectangle {
	rows, cols, widths := l.layout(c)
	var width, height vg.Length
	for _, w := range widths {
		width += w
	}
	if cols > 1 {
		width += vg.Length(cols-1) * l.columnGap()
	}
	if rows > 0 {
		height = vg.Length(rows)*l.entryHeight() + vg.Length(rows-1)*l.Padding
	}
	var r vg.Rectangle
	if l.Left {
		r.Max.X = c.Max.X
		r.Min.X = c.Max.X - width
	} else {
		r.Max.X = c.Min.X + width
		r.Min.X = c.Min.X
	}
	if l.Top {
		r.Max.Y = c.Max.Y
		r.Min.Y = c.Max.Y - height
	} else {
		r.Max.Y = c.Min.Y + height
		r.Min.Y = c.Min.Y
	}
	return r
}

// layout returns the number of rows and columns used to lay out the
// legend's entries on the given canvas, and the width of each column.
// Entries are laid out column-wise, in the order they were added to
// the legend: the first rows entries fill the first column, the next
// rows entries the second column, and so on.  A single column is used
// whenever all the entries fit within the vertical extent of the
// canvas.
func (l *Legend) layout(c draw.Canvas) (rows, cols int, widths []vg.Length) {
	n := len(l.entries)
	if n == 0 {
		return 0, 0, nil
	}
	enth := l.entryHeight()
	// maxRows is the maximum number of entries that fit in a
	// single column within the vertical extent of the canvas.
	maxRows := int((c.Max.Y - c.Min.Y + l.Padding) / (enth + l.Padding))
	if maxRows < 1 {
		maxRows = 1
	}
	rows = n
	if rows > maxRows {
		rows = maxRows
	}
	cols = (n + rows - 1) / rows
	rows = (n + cols - 1) / cols

	sty := l.TextStyle
	widths = make([]vg.Length, cols)
	for i, e := range l.entries {
		w := l.ThumbnailWidth + sty.Rectangle(" "+e.text).Max.X
		if j := i / rows; w > widths[j] {
			widths[j] = w
		}
	}
	return rows, cols, widths
}

// columnGap returns the horizontal space inserted between two
// adjacent columns of legend entries.
func (l *Legend) columnGap() vg.Length {
	return l.TextStyle.Rectangle(" ").Max.X + l.Padding
}

// entryHeight returns the height of the tallest legend
// entry text.
func (l *Legend) entryHeight() (height vg.Length) {
	for _, e := range l.entries {
		if h := l.TextStyle.Rectangle(e.text).Max.Y; h > height {
			height = h
		}
	}
	return
}

// Add adds an entry to the legend with the given name.
// The entry's thumbnail is drawn as the composite of all of the
// thumbnails.
func (l *Legend) Add(name string, thumbs ...Thumbnailer) {
	l.entries = append(l.entries, legendEntry{text: name, thumbs: thumbs})
}
