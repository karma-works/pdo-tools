package export

import (
	"fmt"
	"io"

	"pdo-tools/pkg/pdo"
)

// SVGWriter exports to SVG
type SVGWriter struct {
	w      io.Writer
	width  float64
	height float64
	scale  float64
}

func NewSVGWriter(w io.Writer, width, height float64) *SVGWriter {
	return &SVGWriter{
		w:      w,
		width:  width,
		height: height,
		scale:  1.0, // Default scale
	}
}

func (s *SVGWriter) WriteHeader() {
	// Standard A4: 210 x 297 mm
	// We use mm as user units directly or scale?
	// SVG allows "width=210mm".
	// viewBox="0 0 210 297"

	fmt.Fprintf(s.w, `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
	<svg xmlns="http://www.w3.org/2000/svg" version="1.1"
	width="%.2fmm" height="%.2fmm" viewBox="0 0 %.2f %.2f">
	<style>
		.cut { fill:none; stroke:black; stroke-width:0.1; }
		.mountain { fill:none; stroke:blue; stroke-width:0.1; stroke-dasharray:1,1; }
		.valley { fill:none; stroke:red; stroke-width:0.1; stroke-dasharray:1,1; }
		.text { font-size: 5px; font-family: sans-serif; fill: black; }
	</style>
`, s.width, s.height, s.width, s.height)
}

func (s *SVGWriter) WriteFooter() {
	fmt.Fprintln(s.w, "</svg>")
}

func (s *SVGWriter) WritePDO(p *pdo.PDO) {
	// Group for parts
	fmt.Fprintln(s.w, `<g id="parts">`)
	for _, part := range p.Parts {
		s.WritePart(p, &part)
	}
	fmt.Fprintln(s.w, `</g>`)

	// Text blocks
	fmt.Fprintln(s.w, `<g id="text">`)
	for _, tb := range p.TextBlocks {
		// Just dump text at position
		x := tb.BoundingBox.Left
		y := tb.BoundingBox.Top // SVG coords are usually top-down, PDO is mm, might be consistent
		// Wait, PDO Y grows down?
		// Ref: `pdo2opf` -> `part2d.page_h`.
		// Usually coordinates are in mm relative to margins.
		// We can just plot them.

		for _, line := range tb.Lines {
			fmt.Fprintf(s.w, `<text x="%.3f" y="%.3f" class="text">%s</text>`+"\n",
				x, y+float64(tb.FontSize), line)
			y += tb.LineSpacing
		}
	}
	fmt.Fprintln(s.w, `</g>`)
}

func (s *SVGWriter) WritePart(p *pdo.PDO, part *pdo.Part) {
	// We need to resolve lines to vertices
	// part.Lines refers to face/vertex indices

	obj := p.Objects[part.ObjectIndex]

	for _, line := range part.Lines {
		if line.Hidden {
			continue
		}

		// line.FaceIndex, line.VertexIndex
		// Find start vertex
		v1 := get2DVertex(obj, line.FaceIndex, line.VertexIndex)
		if v1 == nil {
			continue
		}

		var v2 *pdo.Face2DVertex
		if line.IsConnectingFaces {
			// Connects to another face
			v2 = get2DVertex(obj, line.Face2Index, line.Vertex2Index)
		} else {
			// Boundary line: connects to next vertex in the face
			v2 = getNext2DVertex(obj, line.FaceIndex, line.VertexIndex)
		}

		if v2 == nil {
			continue
		}

		class := "cut"
		if line.Type == 1 {
			class = "mountain"
		}
		if line.Type == 2 {
			class = "valley"
		}

		fmt.Fprintf(s.w, `<line x1="%.3f" y1="%.3f" x2="%.3f" y2="%.3f" class="%s" />`+"\n",
			v1.X, v1.Y, v2.X, v2.Y, class)
	}
}

// get2DVertex is in util.go

func ExportSVG(p *pdo.PDO, w io.Writer) error {
	dims := getPageDims(p)
	maxPX, maxPY := calculatePageGrid(p, dims)

	// Total SVG size
	// +1 because indices are 0-based
	totalWidth := float64(maxPX+1) * dims.Width
	totalHeight := float64(maxPY+1) * dims.Height

	// If only 1 page, use default width/height from settings to correspond to exactly one page
	if maxPX == 0 && maxPY == 0 {
		totalWidth = dims.Width
		totalHeight = dims.Height
	} else {
		// If multi-page, we might want to put them side-by-side or vertical?
		// calculatePageGrid assumes global coordinates are already spread out.
		// If they occupy (210, 0) range, that's Page 1 (index 1).
		// So MaxPX=1 implies Width needs to be at least 2*210.
		// But wait, getPageDims returns Width=210.
		// So totalWidth should be enough to cover MaxPX.
		// Yes, (maxPX+1) * dims.Width is correct if pages are laid out horizontally/vertically in grid.
		// However, margins might complicate things if we want to "view" it as a continuous sheet.
		// But since coordinates are global, we just need a ViewBox big enough.

		// Note regarding margins: calculatePageGrid divides by ClippedWidth.
		// Global coordinate X corresponds to PageX = X / ClippedWidth.
		// Real Page Width is 'Width'.
		// If we set SVG viewBox to (MaxPX+1)*Width, we cover the area.
		// BUT the parts are positioned in "Global Content Coordinates".
		// To map them to "Physical Page Sheets" laid out in a grid implies transforming them?
		// The original tool likely treats the coordinate system as continuous.
		// So we just need to extend the ViewBox.

		// Actually, if PageX=1, the part is at X ~ ClippedWidth.
		// If we want to show it on the second A4 page placed to the right of the first one:
		// Page 2 starts at X=Width (210mm).
		// But the Part is at X=ClippedWidth (190mm if margin=10).
		// So Part is at 190mm. Page 2 starts at 210mm.
		// 190mm is still on Page 1??
		// No, ClippedWidth is the content width.
		// If PageX = floor(X / ClippedWidth) = 1. Then X >= 190.
		// If it is on Page 2, it should be visually starting at 210mm from left?
		// We are NOT changing part coordinates here.
		// We are just changing the VIEWBOX.
		// If parts are at 500mm, we need viewBox to 500mm.
		// Logic:
		// Find Max X/Y of actual parts?
		// calculatePageGrid finds Max Page Index.
		// Let's just find the max/min bounding box of all parts and use that?
		// That's safer.
	}

	// Find strict bounding box of all parts to determine necessary viewbox
	minX, minY := 99999.9, 99999.9 // sufficiently large?
	maxX, maxY := -99999.9, -99999.9

	foundParts := false
	for _, part := range p.Parts {
		foundParts = true
		// Using Part BoundingBox
		if part.BoundingBox.Left < minX {
			minX = part.BoundingBox.Left
		}
		if part.BoundingBox.Top < minY {
			minY = part.BoundingBox.Top
		}
		r := part.BoundingBox.Left + part.BoundingBox.Width
		b := part.BoundingBox.Top + part.BoundingBox.Height
		if r > maxX {
			maxX = r
		}
		if b > maxY {
			maxY = b
		}
	}

	if !foundParts {
		// Empty
		return nil
	}

	// Add some padding? Or just use Page Size multiples?
	// Using Page Size multiples looks cleaner if printing is expected.
	// But simple fitting is also fine.
	// Let's stick to strict bounding box + padding, OR Page Multiples.
	// User complained about "all on first page", presumably because content was cut off.
	// Let's use max(PageSize, ContentSize).

	if maxX > totalWidth {
		totalWidth = maxX
	}
	if maxY > totalHeight {
		totalHeight = maxY
	}

	// Also if minX < 0, we might need adjustments?
	// Usually papercraft starts at >0.

	svg := NewSVGWriter(w, totalWidth, totalHeight) // Width/Height are doubles
	svg.WriteHeader()
	svg.WritePDO(p)
	svg.WriteFooter()
	return nil
}
