package export

import (
	"io"
	"math"

	"pdo-tools/pkg/pdo"

	"github.com/go-pdf/fpdf"
)

// ExportPDF exports the PDO data to a PDF file.
// It uses "github.com/go-pdf/fpdf".
func ExportPDF(p *pdo.PDO, w io.Writer) error {
	// Initialize PDF
	// Default A4 portrait
	// If PDO has custom size, we might need to adjust.
	// PDO uses mm. FPDF uses mm by default.

	orientation := "P"
	// Check Settings
	width := p.Settings.CustomWidth
	height := p.Settings.CustomHeight
	format := "A4"

	if p.Settings.PageType == 0 { // A4
		// width=210, height=297
	} else if p.Settings.PageType == 1 { // A3?
		// ...
	}
	// For simplicity, we stick to A4 or custom.
	// If custom size is set and non-zero
	var size fpdf.SizeType
	if width > 0 && height > 0 {
		format = "Custom"
		size = fpdf.SizeType{Wd: width, Ht: height}
	}

	// Calculate Page Grid
	dims := getPageDims(p)
	maxPX, maxPY := calculatePageGrid(p, dims)

	pdf := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: orientation,
		UnitStr:        "mm",
		SizeStr:        format,
		Size:           size,
	})

	pdf.SetFont("Arial", "", 10)

	// Loop Pages
	for py := 0; py <= maxPY; py++ {
		for px := 0; px <= maxPX; px++ {
			// Check if page has content
			partsOnPage := getPartsOnPage(p, px, py, dims)
			if len(partsOnPage) == 0 {
				continue
			}

			pdf.AddPage()

			// Calculate Offset
			// Logic: Global (px*CW, py*CH) -> Local (MarginL, MarginT)
			// DrawX = GlobalX - OffsetX
			// LocalX = GlobalX - OffsetX
			// We want GlobalX=px*CW to map to MarginL.
			// MarginL = px*CW - OffsetX => OffsetX = px*CW - MarginL

			offX := float64(px)*dims.ClippedWidth - dims.MarginLeft
			offY := float64(py)*dims.ClippedHeight - dims.MarginTop

			for _, part := range partsOnPage {
				writePartPDF(pdf, p, part, offX, offY)
			}

			// Text? (Skipping per-page text filtering for brevity, just dumping all? No, should filter)
			// For now, skip text filtering or implement it similarly.
		}
	}

	return pdf.Output(w)
}

func getPartsOnPage(p *pdo.PDO, px, py int, dims PageDims) []*pdo.Part {
	var parts []*pdo.Part
	for i := range p.Parts {
		part := &p.Parts[i]
		// Determine part page
		// Note: Parts can span? pdo2opf assigns owner page based on anchor?
		ppx := int(math.Floor(part.BoundingBox.Left / dims.ClippedWidth))
		ppy := int(math.Floor(part.BoundingBox.Top / dims.ClippedHeight))

		if ppx == px && ppy == py {
			parts = append(parts, part)
		}
	}
	return parts
}

func writePartPDF(pdf *fpdf.Fpdf, p *pdo.PDO, part *pdo.Part, offX, offY float64) {
	obj := p.Objects[part.ObjectIndex]

	for _, line := range part.Lines {
		if line.Hidden {
			continue
		}

		v1 := get2DVertex(obj, line.FaceIndex, line.VertexIndex)
		if v1 == nil {
			continue
		}

		var v2 *pdo.Face2DVertex
		if line.IsConnectingFaces {
			v2 = get2DVertex(obj, line.Face2Index, line.Vertex2Index)
		} else {
			v2 = getNext2DVertex(obj, line.FaceIndex, line.VertexIndex)
		}

		if v2 == nil {
			continue
		}

		// Apply Offset
		// Vertex coordinates are Local. Add Part BoundingBox to get Global.
		// Then subtract Page Offset.
		x1 := (v1.X + part.BoundingBox.Left) - offX
		y1 := (v1.Y + part.BoundingBox.Top) - offY
		x2 := (v2.X + part.BoundingBox.Left) - offX
		y2 := (v2.Y + part.BoundingBox.Top) - offY

		// Set Style
		pdf.SetLineWidth(0.1)
		if line.Type == 1 { // Mountain
			pdf.SetDrawColor(0, 0, 255) // Blue
			pdf.SetDashPattern([]float64{1, 1}, 0)
		} else if line.Type == 2 { // Valley
			pdf.SetDrawColor(255, 0, 0) // Red
			pdf.SetDashPattern([]float64{1, 1}, 0)
		} else { // Cut
			pdf.SetDrawColor(0, 0, 0) // Black
			pdf.SetDashPattern([]float64{}, 0)
		}

		pdf.Line(x1, y1, x2, y2)
	}
}

// Reuse get2DVertex from svg.go?
// I'll copy it for now to keep packages independent or move to common.
// Given they are in the same package 'export', I can access it if I remove the receiver?
// No, svg.go func uses 's *SVGWriter'.
// I'll make a helper function in a new file `common.go` or just duplicate it here lightly.

// get2DVertex is shared with svg.go (same package)
