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
		v1 := s.get2DVertex(obj, line.FaceIndex, line.VertexIndex)
		if v1 == nil {
			continue
		}

		var v2 *pdo.Face2DVertex
		if line.IsConnectingFaces {
			// Connects to another face?
			v2 = s.get2DVertex(obj, line.Face2Index, line.Vertex2Index)
		} else {
			// Usually connects to next vertex in the face?
			// Ref: `pdo_common.pas`: "line type for flaps added to unjoined lines"
			// Wait, if it's an edge of the face?
			// `Faces` define polygons. `Lines` define cuts or folds between them.
			// But `Part` lines seem to be the outline.

			// If `IsConnectingFaces` is false, what does it connect?
			// Maybe it's just a flap line or something?
			// The logic in `pdo2opf` relies on `lines_extra`.
			// Since we don't have `lines_extra` populated (unless I reverse engineer how to populate it),
			// I can only draw what I have.
			// Maybe I should assume `lines` form loops?

			// Let's assume for now that if `IsConnectingFaces` is true, it's a fold line?
			// No, `ConnectsFaces` means it's an internal edge (fold).
			// If false, it's a boundary cut?
			continue
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

func (s *SVGWriter) get2DVertex(obj pdo.Object, faceIdx, vertIdx int32) *pdo.Face2DVertex {
	if int(faceIdx) >= len(obj.Faces) {
		return nil
	}
	face := obj.Faces[faceIdx]

	// vertIdx is "objects vertex list" index (3D ID).
	// We need to find which 2D vertex in the face corresponds to this 3D ID.
	for i := range face.Vertices {
		if face.Vertices[i].IDVertex == vertIdx {
			return &face.Vertices[i]
		}
	}
	return nil
}

func ExportSVG(p *pdo.PDO, w io.Writer) error {
	// Assume A4 if page settings are 0?
	// Use settings from PDO
	width := p.Settings.CustomWidth
	height := p.Settings.CustomHeight
	if p.Settings.PageType == 0 { // A4
		width = 210
		height = 297
	}

	svg := NewSVGWriter(w, width, height)
	svg.WriteHeader()
	svg.WritePDO(p)
	svg.WriteFooter()
	return nil
}
