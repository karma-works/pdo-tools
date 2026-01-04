package export

import (
	"io"

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

	pdf := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: orientation,
		UnitStr:        "mm",
		SizeStr:        format,
		Size:           size,
	})

	pdf.SetFont("Arial", "", 10)

	// Add Page
	pdf.AddPage()

	// Draw Parts (Cut lines, mountain, valley)
	// Similar logic to SVG export
	// For PDF we draw lines directly.

	// Colors
	// Cut: Black
	// Mountain: Blue dashed
	// Valley: Red dashed

	for _, part := range p.Parts {
		writePartPDF(pdf, p, &part)
	}

	// Text
	for _, tb := range p.TextBlocks {
		pdf.SetXY(tb.BoundingBox.Left, tb.BoundingBox.Top)
		pdf.SetTextColor(0, 0, 0)
		for _, line := range tb.Lines {
			pdf.Cell(0, float64(tb.FontSize), line) // FontSize might need scaling?
			pdf.Ln(tb.LineSpacing)
		}
	}

	return pdf.Output(w)
}

func writePartPDF(pdf *fpdf.Fpdf, p *pdo.PDO, part *pdo.Part) {
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

		pdf.Line(v1.X, v1.Y, v2.X, v2.Y)
	}
}

// Reuse get2DVertex from svg.go?
// I'll copy it for now to keep packages independent or move to common.
// Given they are in the same package 'export', I can access it if I remove the receiver?
// No, svg.go func uses 's *SVGWriter'.
// I'll make a helper function in a new file `common.go` or just duplicate it here lightly.

// get2DVertex is shared with svg.go (same package)
