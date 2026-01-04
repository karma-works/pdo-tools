package export

import (
	"math"

	"pdo-tools/pkg/pdo"
)

// get2DVertex returns the 2D vertex using the 3D vertex ID.
func get2DVertex(obj pdo.Object, faceIdx, vertIdx int32) *pdo.Face2DVertex {
	if int(faceIdx) >= len(obj.Faces) {
		return nil
	}
	face := obj.Faces[faceIdx]

	for i := range face.Vertices {
		if face.Vertices[i].IDVertex == vertIdx {
			return &face.Vertices[i]
		}
	}
	return nil
}

// getNext2DVertex returns the next vertex in the face loop starting from the given 3D vertex ID.
// This assumes the line represents an edge starting at vertIdx.
func getNext2DVertex(obj pdo.Object, faceIdx, vertIdx int32) *pdo.Face2DVertex {
	if int(faceIdx) >= len(obj.Faces) {
		return nil
	}
	face := obj.Faces[faceIdx]

	for i := range face.Vertices {
		if face.Vertices[i].IDVertex == vertIdx {
			// Found the start vertex. The next one is (i+1) % len
			nextIdx := (i + 1) % len(face.Vertices)
			return &face.Vertices[nextIdx]
		}
	}
	return nil
}

type PageDims struct {
	Width         float64
	Height        float64
	MarginLeft    float64
	MarginTop     float64
	ClippedWidth  float64
	ClippedHeight float64
}

func getPageDims(p *pdo.PDO) PageDims {
	// Defaults/Calculations based on pdo.Settings
	// PageType: 0=A4, etc.
	// For now, assume A4 or Custom.
	w := 210.0
	h := 297.0

	if p.Settings.PageType == 0 { // A4
		w = 210.0
		h = 297.0
	} else if p.Settings.PageType == 11 { // Other
		if p.Settings.CustomWidth > 0 {
			w = p.Settings.CustomWidth
		}
		if p.Settings.CustomHeight > 0 {
			h = p.Settings.CustomHeight
		}
	}
	// TODO: Handle other page types A3, Letter etc.

	mt := float64(p.Settings.MarginTop)
	ms := float64(p.Settings.MarginSide)

	// Orientation: 1 = Landscape?
	// Logic from pdo2opf.pas:
	// if _pdo.settings.page.orientation = 1 then Swap2f(width, height)
	if p.Settings.Orientation == 1 {
		w, h = h, w
		// Swap margins? pdo2opf says Swap(margin_side, margin_top)
		// but margins are usually relative to paper edges?
		// "Swap2f(_page.margin_side, _page.margin_top)" -> Yes.
		mt, ms = ms, mt
	}

	return PageDims{
		Width:         w,
		Height:        h,
		MarginLeft:    ms,
		MarginTop:     mt,
		ClippedWidth:  w - 2*ms,
		ClippedHeight: h - 2*mt,
	}
}

func calculatePageGrid(p *pdo.PDO, dims PageDims) (int, int) {
	maxX := 0
	maxY := 0

	for _, part := range p.Parts {
		// Calculate global BB (including vertices)
		// pdo2opf calculates BB from vertices + part bounding box.
		// part.BoundingBox seems to be the "placed" bounding box.
		// We trust part.BoundingBox for now.
		// Note: pdo2opf says "Stored BB can be crappy". But for positioning we use what we have.

		// PageW = floor( (Left + BBoxVert.Left) / CW ) -- pdo2opf logic uses vert offset?
		// We will use part.BoundingBox.Left/Top as the origin of the part on canvas.
		// The Pascal code adds `part.bounding_box_vert` which seems to conform to local vertex coords.
		// But `part.bounding_box` in `pdo_common.pas` is `TPdoRect`.
		// Let's assume part.BoundingBox.Left is the global X coordinate of the part's anchor.

		px := int(math.Floor(part.BoundingBox.Left / dims.ClippedWidth))
		py := int(math.Floor(part.BoundingBox.Top / dims.ClippedHeight))

		if px > maxX {
			maxX = px
		}
		if py > maxY {
			maxY = py
		}
	}
	return maxX, maxY
}
