// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This extends the official golang tabwriter by adding support for
// - "lineindent"
// - any line with a last cell that doesn't end with \t is treated as
//   a desire to have that text appear without column constraints and will not
//   terminate the section (e.g. will not call Flush() to reset the columns)
// - remove trailing empty cells
// - remove trailing paddchars from last cell in a line

// Package tabwriter implements a write filter (tabwriter.Writer) that
// translates tabbed columns in input into properly aligned text.
//
// The package is using the Elastic Tabstops algorithm described at
// http://nickgravgaard.com/elastictabstops/index.html.
//
// The text/tabwriter package is frozen and is not accepting new features.
package common

import (
	"fmt"
	"io"
	"unicode/utf8"
)

// ----------------------------------------------------------------------------
// Filter implementation

// A cell represents a segment of text terminated by tabs or line breaks.
// The text itself is stored in a separate buffer; cell only describes the
// segment's size in bytes, its width in runes, and whether it's an htab
// ('\t') terminated cell.
type cell struct {
	size  int  // cell size in bytes
	width int  // cell width in runes
	htab  bool // true if the cell is terminated by an htab ('\t')
}

// A TabWriter is a filter that inserts padding around tab-delimited
// columns in its input to align them in the output.
//
// The TabWriter treats incoming bytes as UTF-8-encoded text consisting
// of cells terminated by horizontal ('\t') or vertical ('\v') tabs,
// and newline ('\n') or formfeed ('\f') characters; both newline and
// formfeed act as line breaks.
//
// Tab-terminated cells in contiguous lines constitute a column. The
// TabWriter inserts padding as needed to make all cells in a column have
// the same width, effectively aligning the columns. It assumes that
// all characters have the same width, except for tabs for which a
// tabwidth must be specified. Column cells must be tab-terminated, not
// tab-separated: non-tab terminated trailing text at the end of a line
// forms a cell but that cell is not part of an aligned column.
// For instance, in this example (where | stands for a horizontal tab):
//
//  aaaa|bbb|d
//  aa  |b  |dd
//  a   |
//  aa  |cccc|eee
//
// the b and c are in distinct columns (the b column is not contiguous
// all the way). The d and e are not in a column at all (there's no
// terminating tab, nor would the column be contiguous).
//
// The TabWriter assumes that all Unicode code points have the same width;
// this may not be true in some fonts or if the string contains combining
// characters.
//
// If [DiscardEmptyColumns] is set, empty columns that are terminated
// entirely by vertical (or "soft") tabs are discarded. Columns
// terminated by horizontal (or "hard") tabs are not affected by
// this flag.
//
// If a TabWriter is configured to filter HTML, HTML tags and entities
// are passed through. The widths of tags and entities are
// assumed to be zero (tags) and one (entities) for formatting purposes.
//
// A segment of text may be escaped by bracketing it with [Escape]
// characters. The tabwriter passes escaped text segments through
// unchanged. In particular, it does not interpret any tabs or line
// breaks within the segment. If the [StripEscape] flag is set, the
// Escape characters are stripped from the output; otherwise they
// are passed through as well. For the purpose of formatting, the
// width of the escaped text is always computed excluding the Escape
// characters.
//
// The formfeed character acts like a newline but it also terminates
// all columns in the current line (effectively calling [TabWriter.Flush]). Tab-
// terminated cells in the next line start new columns. Unless found
// inside an HTML tag or inside an escaped text segment, formfeed
// characters appear as newlines in the output.
//
// The TabWriter must buffer input internally, because proper spacing
// of one line may depend on the cells in future lines. Clients must
// call Flush when done calling [TabWriter.Write].
type TabWriter struct {
	// configuration
	output     io.Writer
	minwidth   int
	tabwidth   int
	padding    int
	padbytes   [8]byte
	lineindent []byte // custom/new
	flags      uint

	// current state
	buf     []byte   // collected text excluding tabs or line breaks
	pos     int      // buffer position up to which cell.width of incomplete cell has been computed
	cell    cell     // current incomplete cell; cell.width is up to buf[pos] excluding ignored sections
	endChar byte     // terminating char of escaped sequence (Escape for escapes, '>', ';' for HTML tags/entities, or 0)
	lines   [][]cell // list of lines; each line is a list of cells
	widths  []int    // list of column widths in runes - re-used during formatting
}

func Show(msg string, args ...any) {
	// fmt.Printf(msg, args...)
}

// addLine adds a new line.
// flushed is a hint indicating whether the underlying writer was just flushed.
// If so, the previous line is not likely to be a good indicator of the new line's cells.
func (b *TabWriter) addLine(flushed bool) {
	// Grow slice instead of appending,
	// as that gives us an opportunity
	// to re-use an existing []cell.
	if n := len(b.lines) + 1; n <= cap(b.lines) {
		b.lines = b.lines[:n]
		b.lines[n-1] = b.lines[n-1][:0]
	} else {
		b.lines = append(b.lines, nil)
	}

	if !flushed {
		// The previous line is probably a good indicator
		// of how many cells the current line will have.
		// If the current line's capacity is smaller than that,
		// abandon it and make a new one.
		if n := len(b.lines); n >= 2 {
			if prev := len(b.lines[n-2]); prev > cap(b.lines[n-1]) {
				b.lines[n-1] = make([]cell, 0, prev)
			}
		}
	}
}

// Reset the current state.
func (b *TabWriter) reset() {
	b.buf = b.buf[:0]
	b.pos = 0
	b.cell = cell{}
	b.endChar = 0
	b.lines = b.lines[0:0]
	b.widths = b.widths[0:0]
	b.addLine(true)
}

// Internal representation (current state):
//
// - all text written is appended to buf; tabs and line breaks are stripped away
// - at any given time there is a (possibly empty) incomplete cell at the end
//   (the cell starts after a tab or line break)
// - cell.size is the number of bytes belonging to the cell so far
// - cell.width is text width in runes of that cell from the start of the cell to
//   position pos; html tags and entities are excluded from this width if html
//   filtering is enabled
// - the sizes and widths of processed text are kept in the lines list
//   which contains a list of cells for each line
// - the widths list is a temporary list with current widths used during
//   formatting; it is kept in TabWriter because it's re-used
//
//                    |<---------- size ---------->|
//                    |                            |
//                    |<- width ->|<- ignored ->|  |
//                    |           |             |  |
// [---processed---tab------------<tag>...</tag>...]
// ^                  ^                         ^
// |                  |                         |
// buf                start of incomplete cell  pos

// Formatting can be controlled with these flags.
const (
	// Ignore html tags and treat entities (starting with '&'
	// and ending in ';') as single characters (width = 1).
	FilterHTML uint = 1 << iota

	// Strip Escape characters bracketing escaped text segments
	// instead of passing them through unchanged with the text.
	StripEscape

	// Force right-alignment of cell content.
	// Default is left-alignment.
	AlignRight

	// Handle empty columns as if they were not present in
	// the input in the first place.
	DiscardEmptyColumns

	// Always use tabs for indentation columns (i.e., padding of
	// leading empty cells on the left) independent of padchar.
	TabIndent

	// Print a vertical bar ('|') between columns (after formatting).
	// Discarded columns appear as zero-width columns ("||").
	DDebug
)

// A [TabWriter] must be initialized with a call to Init. The first parameter (output)
// specifies the filter output. The remaining parameters control the formatting:
//
//  lineindent  chars to add to the front of each line  (custom/new)
//  minwidth    minimal cell width including any padding
//  tabwidth    width of tab characters (equivalent number of spaces)
//  padding     padding added to a cell before computing its width
//  padchar     ASCII char used for padding
//          if padchar == '\t', the TabWriter will assume that the
//          width of a '\t' in the formatted output is tabwidth,
//          and cells are left-aligned independent of align_left
//          (for correct-looking results, tabwidth must correspond
//          to the tab width in the viewer displaying the result)
//  flags       formatting control
func (b *TabWriter) Init(output io.Writer, lineindent []byte, minwidth, tabwidth, padding int, padchar byte, flags uint) *TabWriter {
	if minwidth < 0 || tabwidth < 0 || padding < 0 {
		panic("negative minwidth, tabwidth, or padding")
	}
	b.output = output
	b.minwidth = minwidth
	b.tabwidth = tabwidth
	b.padding = padding
	for i := range b.padbytes {
		b.padbytes[i] = padchar
	}
	if padchar == '\t' {
		// tab padding enforces left-alignment
		flags &^= AlignRight
	}
	b.lineindent = lineindent // custom/new
	b.flags = flags

	b.reset()

	return b
}

// debugging support (keep code around)
func (b *TabWriter) dump() {
	pos := 0
	for i, line := range b.lines {
		print("(", i, ") ")
		for _, c := range line {
			print("[", string(b.buf[pos:pos+c.size]), "]")
			pos += c.size
		}
		print("\n")
	}
	print("\n")
}

// local error wrapper so we can distinguish errors we want to return
// as errors from genuine panics (which we don't want to return as errors)
type osError struct {
	err error
}

func (b *TabWriter) write0(buf []byte) {
	n, err := b.output.Write(buf)
	if n != len(buf) && err == nil {
		err = io.ErrShortWrite
	}
	if err != nil {
		panic(osError{err})
	}
}

func (b *TabWriter) writeN(src []byte, n int) {
	for n > len(src) {
		b.write0(src)
		n -= len(src)
	}
	b.write0(src[0:n])
}

var (
	newline = []byte{'\n'}
	tabs    = []byte("\t\t\t\t\t\t\t\t")
)

func (b *TabWriter) writePadding(textw, cellw int, useTabs bool) {
	if b.padbytes[0] == '\t' || useTabs {
		// padding is done with tabs
		if b.tabwidth == 0 {
			return // tabs have no width - can't do any padding
		}
		// make cellw the smallest multiple of b.tabwidth
		cellw = (cellw + b.tabwidth - 1) / b.tabwidth * b.tabwidth
		n := cellw - textw // amount of padding
		if n < 0 {
			panic("internal error")
		}
		b.writeN(tabs, (n+b.tabwidth-1)/b.tabwidth)
		return
	}

	// padding is done with non-tab characters
	b.writeN(b.padbytes[0:], cellw-textw)
}

var vbar = []byte{'|'}

func (b *TabWriter) writeLines(pos0 int, line0, line1 int) (pos int) {
	pos = pos0
	// Show("  Enter: Writing lines %d-%d\n", line0, line1)
	for i := line0; i < line1; i++ {
		line := b.lines[i]
		Show("  Line: %d (cols: %d, widths: %d)\n", i, len(line), b.widths)

		// if TabIndent is set, use tabs to pad leading empty cells
		useTabs := b.flags&TabIndent != 0

		// Remove extra blank cells at the end - for consistency/to be clean
		for len(line) > 0 && line[len(line)-1].size == 0 {
			line = line[:len(line)-1]
		}

		for j, c := range line {
			if j == 0 { // custom/new
				// Add any indent before each line
				b.write0(b.lineindent)
			}
			if j > 0 && b.flags&DDebug != 0 {
				// indicate column break
				b.write0(vbar)
			}

			if c.size == 0 {
				// empty cell
				if j < len(b.widths) {
					b.writePadding(c.width, b.widths[j], useTabs)
				}
			} else {
				// non-empty cell
				useTabs = false
				if b.flags&AlignRight == 0 { // align left
					b.write0(b.buf[pos : pos+c.size])
					pos += c.size
					// if cell didn't end with \t then just dump this non-column
					// cell out w/o regard for column widths/padding.
					// htab check is custom/new
					if j < len(b.widths) && c.htab {
						if j+1 != len(line) { // custom/new, old:always writePad
							b.writePadding(c.width, b.widths[j], false)
						}
					}
					// Show("    cell: %d htab:%v w:%d\n", j, c.htab, len(b.widths))
				} else { // align right
					if j < len(b.widths) {
						if c.htab { // custom/new - old: always writepadding
							b.writePadding(c.width, b.widths[j], false)
						} else if j != 0 {
							// custom/new
							// If the cell didn't end with \t then make sure
							// any previous cell has some padding, and then
							// just dump this non-column cell out w/o regard
							// for column widths
							// Make sure we add 'padding' # of spaces
							// before we dump rest of the non-column cell
							b.write0(b.padbytes[:b.padding])
						}
					}
					b.write0(b.buf[pos : pos+c.size])
					pos += c.size
				}
			}
		}

		if i+1 == len(b.lines) {
			// last buffered line - we don't have a newline, so just write
			// any outstanding buffered data
			b.write0(b.buf[pos : pos+b.cell.size])
			pos += b.cell.size
		} else {
			// not the last line - write newline
			b.write0(newline)
		}
	}
	// Show("  Exit: Writing lines\n")
	return
}

// Format the text between line0 and line1 (excluding line1); pos
// is the buffer position corresponding to the beginning of line0.
// Returns the buffer position corresponding to the beginning of
// line1 and an error, if any.
func (b *TabWriter) format(pos0 int, line0, line1 int) (pos int) {
	pos = pos0
	column := len(b.widths)
	Show("In format %d-%d b.lines:%v b.widths:%v\n", line0, line1, b.lines, b.widths)
	for this := line0; this < line1; this++ {
		line := b.lines[this]

		// Show("Formatting line #%d\n", this)

		if column >= len(line)-1 {
			// Show("  skipping, col(%d) >= len(line)-1(%d)\n", column, len(line)-1)
			continue
		}
		// cell exists in this column => this line
		// has more cells than the previous line
		// (the last cell per line is ignored because cells are
		// tab-terminated; the last cell per line describes the
		// text before the newline/formfeed and does not belong
		// to a column)

		// print unprinted lines until beginning of block
		pos = b.writeLines(pos, line0, this)
		line0 = this

		// column block begin
		width := b.minwidth // minimal column width
		discardable := true // true if all cells in this column are empty and "soft"

		for ; this < line1; this++ {
			line = b.lines[this]

			if column >= len(line)-1 {
				// custom/new
				if len(line) != 0 {
					Show("htab=%v\n", line[len(line)-1].htab)
				}
				if len(line) == 0 || line[len(line)-1].htab == false {
					continue
				}
				break
			}
			// cell exists in this column
			c := line[column]
			// update width
			if w := c.width + b.padding; w > width {
				width = w
			}
			// update discardable
			if c.width > 0 || c.htab {
				discardable = false
			}

		}
		Show("Ending block: line=%d\n", this)
		// column block end

		// discard empty columns if necessary
		if discardable && b.flags&DiscardEmptyColumns != 0 {
			width = 0
		}

		// format and print all columns to the right of this column
		// (we know the widths of this column and all columns to the left)
		b.widths = append(b.widths, width) // push width
		pos = b.format(pos, line0, this)
		b.widths = b.widths[0 : len(b.widths)-1] // pop width
		line0 = this
	}

	// print unprinted lines until end
	p := b.writeLines(pos, line0, line1)
	Show("Exit format\n")
	return p
}

// Append text to current cell.
func (b *TabWriter) append(text []byte) {
	b.buf = append(b.buf, text...)
	b.cell.size += len(text)
}

// Update the cell width.
func (b *TabWriter) updateWidth() {
	b.cell.width += utf8.RuneCount(b.buf[b.pos:])
	b.pos = len(b.buf)
}

// To escape a text segment, bracket it with Escape characters.
// For instance, the tab in this string "Ignore this tab: \xff\t\xff"
// does not terminate a cell and constitutes a single character of
// width one for formatting purposes.
//
// The value 0xff was chosen because it cannot appear in a valid UTF-8 sequence.
const Escape = '\xff'

// Start escaped mode.
func (b *TabWriter) startEscape(ch byte) {
	switch ch {
	case Escape:
		b.endChar = Escape
	case '<':
		b.endChar = '>'
	case '&':
		b.endChar = ';'
	}
}

// Terminate escaped mode. If the escaped text was an HTML tag, its width
// is assumed to be zero for formatting purposes; if it was an HTML entity,
// its width is assumed to be one. In all other cases, the width is the
// unicode width of the text.
func (b *TabWriter) endEscape() {
	switch b.endChar {
	case Escape:
		b.updateWidth()
		if b.flags&StripEscape == 0 {
			b.cell.width -= 2 // don't count the Escape chars
		}
	case '>': // tag of zero width
	case ';':
		b.cell.width++ // entity, count as one rune
	}
	b.pos = len(b.buf)
	b.endChar = 0
}

// Terminate the current cell by adding it to the list of cells of the
// current line. Returns the number of cells in that line.
func (b *TabWriter) terminateCell(htab bool) int {
	b.cell.htab = htab
	line := &b.lines[len(b.lines)-1]
	*line = append(*line, b.cell)
	b.cell = cell{}
	return len(*line)
}

func (b *TabWriter) handlePanic(err *error, op string) {
	if e := recover(); e != nil {
		if op == "Flush" {
			// If Flush ran into a panic, we still need to reset.
			b.reset()
		}
		if nerr, ok := e.(osError); ok {
			*err = nerr.err
			return
		}
		panic(fmt.Sprintf("tabwriter: panic during %s (%v)", op, e))
	}
}

// Flush should be called after the last call to [TabWriter.Write] to ensure
// that any data buffered in the [TabWriter] is written to output. Any
// incomplete escape sequence at the end is considered
// complete for formatting purposes.
func (b *TabWriter) Flush() error {
	return b.flush()
}

// flush is the internal version of Flush, with a named return value which we
// don't want to expose.
func (b *TabWriter) flush() (err error) {
	defer b.handlePanic(&err, "Flush")
	b.flushNoDefers()
	return nil
}

// flushNoDefers is like flush, but without a deferred handlePanic call. This
// can be called from other methods which already have their own deferred
// handlePanic calls, such as Write, and avoid the extra defer work.
func (b *TabWriter) flushNoDefers() {
	// add current cell if not empty
	if b.cell.size > 0 {
		if b.endChar != 0 {
			// inside escape - terminate it even if incomplete
			b.endEscape()
		}
		b.terminateCell(false)
	}

	// format contents of buffer
	b.format(0, 0, len(b.lines))
	b.reset()
}

var hbar = []byte("---\n")

// Write writes buf to the writer b.
// The only errors returned are ones encountered
// while writing to the underlying output stream.
func (b *TabWriter) Write(buf []byte) (n int, err error) {
	defer b.handlePanic(&err, "Write")

	// split text into cells
	n = 0
	for i, ch := range buf {
		if b.endChar == 0 {
			// outside escape
			switch ch {
			case '\t', '\v', '\n', '\f':
				// end of cell
				b.append(buf[n:i])
				b.updateWidth()
				n = i + 1 // ch consumed
				// ncells := b.terminateCell(ch == '\t') // old
				b.terminateCell(ch == '\t') // custom/new
				if ch == '\n' || ch == '\f' {
					// terminate line
					b.addLine(ch == '\f')
					if ch == '\f' { // || ncells == 1 {  // OR part is old
						// A '\f' always forces a flush. Otherwise, if the previous
						// line has only one cell which does not have an impact on
						// the formatting of the following lines (the last cell per
						// line is ignored by format()), thus we can flush the
						// TabWriter contents.
						b.flushNoDefers()
						if ch == '\f' && b.flags&DDebug != 0 {
							// indicate section break
							b.write0(hbar)
						}
					}
				}

			case Escape:
				// start of escaped sequence
				b.append(buf[n:i])
				b.updateWidth()
				n = i
				if b.flags&StripEscape != 0 {
					n++ // strip Escape
				}
				b.startEscape(Escape)

			case '<', '&':
				// possibly an html tag/entity
				if b.flags&FilterHTML != 0 {
					// begin of tag/entity
					b.append(buf[n:i])
					b.updateWidth()
					n = i
					b.startEscape(ch)
				}
			}

		} else {
			// inside escape
			if ch == b.endChar {
				// end of tag/entity
				j := i + 1
				if ch == Escape && b.flags&StripEscape != 0 {
					j = i // strip Escape
				}
				b.append(buf[n:j])
				n = i + 1 // ch consumed
				b.endEscape()
			}
		}
	}

	// append leftover text
	b.append(buf[n:])
	n = len(buf)
	return
}

// NewTabWriter allocates and initializes a new [TabWriter].
// The parameters are the same as for the Init function.
func NewTabWriter(output io.Writer, lineindent []byte, minwidth, tabwidth, padding int, padchar byte, flags uint) *TabWriter {
	return new(TabWriter).Init(output, lineindent, minwidth, tabwidth, padding, padchar, flags)
}
