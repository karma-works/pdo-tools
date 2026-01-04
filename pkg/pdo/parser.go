package pdo

import (
	"fmt"
	"io"
	"os"
)

const (
	FileMagic = "version 3\n"
	PDO_V4    = 4
	PDO_V5    = 5
	PDO_V6    = 6

	TextureDataWrapperSize = 6
)

type Parser struct {
	reader *Reader
	PDO    *PDO
}

func NewParser(r io.Reader) *Parser {
	return &Parser{
		reader: NewReader(r),
		PDO:    &PDO{},
	}
}

func ParseFile(filename string) (*PDO, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	parser := NewParser(f)
	if err := parser.Load(); err != nil {
		return nil, err
	}
	return parser.PDO, nil
}

func (p *Parser) Load() error {
	if err := p.ReadHeader(); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}
	if err := p.ReadObjects(); err != nil {
		return fmt.Errorf("failed to read objects: %w", err)
	}
	if err := p.ReadMaterials(); err != nil {
		return fmt.Errorf("failed to read materials: %w", err)
	}
	if err := p.ReadUnfoldData(); err != nil {
		return fmt.Errorf("failed to read unfold data: %w", err)
	}
	if err := p.ReadSettings(); err != nil {
		return fmt.Errorf("failed to read settings: %w", err)
	}
	return nil
}

func (p *Parser) ReadHeader() error {
	// Read Magic
	magicBuf := make([]byte, len(FileMagic))
	if err := p.reader.ReadBytes(magicBuf); err != nil {
		return fmt.Errorf("read magic failed: %w", err)
	}
	if string(magicBuf) != FileMagic {
		return fmt.Errorf("invalid file magic: %q", string(magicBuf))
	}

	h := &p.PDO.Header

	if err := p.reader.ReadBytes(&h.Version); err != nil {
		return fmt.Errorf("read version failed: %w", err)
	}
	if err := p.reader.ReadBytes(&h.MultiByteChars); err != nil {
		return err
	}
	p.reader.MultiByteC = h.MultiByteChars == 1

	var unknownInt int32
	if err := p.reader.ReadBytes(&unknownInt); err != nil { // Unknown int
		return fmt.Errorf("read unknown int failed: %w", err)
	}

	// Need to sync exactly with Pascal ReadHeader
	// fpdo.ReadBytes(header.version, 4);
	// fpdo.ReadBytes(header.multi_byte_chars, 4);
	// fpdo.ReadBytes(unknown_int, 4);

	// My previous ReadUInt32 would read it. But I should store it or discard it.
	// I used ReadBytes directly above.
	// Let's use p.reader.ReadInt32() for better readability.

	if h.Version > PDO_V4 {
		var err error
		h.DesignerID, err = p.reader.ReadString(0)
		if err != nil {
			return err
		}
		if err := p.reader.ReadBytes(&h.StringShift); err != nil {
			return err
		}
		p.reader.StringShift = byte(h.StringShift)
	}

	var err error
	h.Locale, err = p.reader.ReadShiftedString()
	if err != nil {
		return err
	}

	h.Codepage, err = p.reader.ReadShiftedString()
	if err != nil {
		return err
	}

	if err := p.reader.ReadBytes(&h.TexLock); err != nil {
		return err
	}

	if h.Version == PDO_V6 {
		if err := p.reader.ReadBytes(&h.ShowStartupNotes); err != nil {
			return err
		}
		if err := p.reader.ReadBytes(&h.PasswordFlag); err != nil {
			return err
		}
	}

	// Basic check from Pascal
	// peeksize := pbyte(fpdo.Memory)[fpdo.Position];
	// We are streaming, so we can't peek easily without ensuring buffer.
	// Skip the check or implement peek if critical. It seems to be for "dodgy files".
	// We'll trust standard files for now.

	h.Key, err = p.reader.ReadShiftedString()
	if err != nil {
		return err
	}

	if h.Version == PDO_V6 {
		if err := p.reader.ReadBytes(&h.V6Lock); err != nil {
			return err
		}
		if h.V6Lock > 0 {
			junk := make([]byte, 8)
			for i := 0; i < int(h.V6Lock); i++ {
				p.reader.ReadBytes(junk)
			}
		}
	} else {
		if h.Version > PDO_V4 {
			if err := p.reader.ReadBytes(&h.ShowStartupNotes); err != nil {
				return err
			}
			if err := p.reader.ReadBytes(&h.PasswordFlag); err != nil {
				return err
			}
		}
	}

	if err := p.reader.ReadBytes(&h.AssembledHeight); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&h.OriginOffset); err != nil {
		return err
	}

	return nil
}

func (p *Parser) ReadObjects() error {
	var count int32
	if err := p.reader.ReadBytes(&count); err != nil {
		return err
	}

	p.PDO.Objects = make([]Object, count)
	for i := 0; i < int(count); i++ {
		if err := p.ReadObject(&p.PDO.Objects[i]); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) ReadObject(obj *Object) error {
	var err error
	obj.Name, err = p.reader.ReadShiftedString()
	if err != nil {
		return err
	}

	if err := p.reader.ReadBytes(&obj.Visible); err != nil {
		return err
	}

	var numVertices int32
	if err := p.reader.ReadBytes(&numVertices); err != nil {
		return err
	}

	obj.Vertices = make([]Vertex3D, numVertices)
	if err := p.reader.ReadBytes(obj.Vertices); err != nil {
		return err
	}

	var numFaces int32
	if err := p.reader.ReadBytes(&numFaces); err != nil {
		return err
	}

	obj.Faces = make([]Face, numFaces)
	for i := 0; i < int(numFaces); i++ {
		if err := p.ReadFace(&obj.Faces[i]); err != nil {
			return err
		}
	}

	var numEdges int32
	if err := p.reader.ReadBytes(&numEdges); err != nil {
		return err
	}

	obj.Edges = make([]Edge, numEdges)
	for i := 0; i < int(numEdges); i++ {
		// Read 22 bytes for each edge
		// Pascal: f.ReadBytes(Result, 22);
		// Edge struct matches 22 bytes if we exclude implicit padding?
		// Face1Index(4) + Face2Index(4) + Vertex1Index(4) + Vertex2Index(4) + ConnectsFaces(2) + NoConnectedFace(4) = 22.
		// Go struct alignment might be different.
		// But binary.Read uses serialized size of types.
		// int32=4, int16=2. 4*5 + 2 = 22. Correct.
		if err := p.reader.ReadBytes(&obj.Edges[i]); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) ReadFace(face *Face) error {
	if err := p.reader.ReadBytes(&face.MaterialIndex); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&face.PartIndex); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&face.Nx); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&face.Ny); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&face.Nz); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&face.Coord); err != nil {
		return err
	}

	var count int32
	if err := p.reader.ReadBytes(&count); err != nil {
		return err
	}

	face.Vertices = make([]Face2DVertex, count)
	for i := 0; i < int(count); i++ {
		if err := p.ReadFace2DVertex(&face.Vertices[i]); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) ReadFace2DVertex(v *Face2DVertex) error {
	// 4 + 8*4 + 1 + 8*3 + 24
	if err := p.reader.ReadBytes(&v.IDVertex); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&v.X); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&v.Y); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&v.U); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&v.V); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&v.Flap); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&v.FlapHeight); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&v.FlapAAngle); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&v.FlapBAngle); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&v.FlapFoldInfo); err != nil {
		return err
	}
	return nil
}

func (p *Parser) ReadMaterials() error {
	var count int32
	if err := p.reader.ReadBytes(&count); err != nil {
		return err
	}

	p.PDO.Materials = make([]Material, count)
	for i := 0; i < int(count); i++ {
		if err := p.ReadMaterial(&p.PDO.Materials[i]); err != nil {
			return err
		}
		if p.PDO.Materials[i].Name == "" {
			p.PDO.Materials[i].Name = fmt.Sprintf("named_material%d", i)
		}
	}
	return nil
}

func (p *Parser) ReadMaterial(mat *Material) error {
	var err error
	mat.Name, err = p.reader.ReadShiftedString()
	if err != nil {
		return err
	}

	if err := p.reader.ReadBytes(&mat.Color3D); err != nil {
		return err
	}

	// Read 2D color (RGBA, but weird order in Pascal?)
	// f.ReadBytes(result.color2d_rgba[3], 4);
	// f.ReadBytes(result.color2d_rgba[0], 4);
	// f.ReadBytes(result.color2d_rgba[1], 4);
	// f.ReadBytes(result.color2d_rgba[2], 4);
	// It reads Alpha, Red, Green, Blue.
	// Go encoding/binary reads into array linearly.
	// We need to read manually if we want to preserve order or map to RGBA.
	// Mat struct has [4]float32.

	var a, r, g, b float32
	if err := p.reader.ReadBytes(&a); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&r); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&g); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&b); err != nil {
		return err
	}

	mat.Color2DRGBA = [4]float32{r, g, b, a}

	var texFlag uint8
	if err := p.reader.ReadBytes(&texFlag); err != nil {
		return err
	}
	mat.HasTexture = texFlag == 1

	if mat.HasTexture {
		if err := p.ReadTexture(&mat.Texture); err != nil {
			return err
		}
	} else {
		mat.Texture.DataSize = 0
		mat.Texture.TextureID = -1
	}

	return nil
}

func (p *Parser) ReadTexture(tex *Texture) error {
	if err := p.reader.ReadBytes(&tex.Width); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&tex.Height); err != nil {
		return err
	}

	var wrappedSize int32
	if err := p.reader.ReadBytes(&wrappedSize); err != nil {
		return err
	}

	tex.DataSize = uint32(wrappedSize - TextureDataWrapperSize)

	if err := p.reader.ReadBytes(&tex.DataHeader); err != nil {
		return err
	}

	tex.RawData = make([]byte, tex.DataSize)
	if err := p.reader.ReadBytes(tex.RawData); err != nil {
		return err
	}

	if err := p.reader.ReadBytes(&tex.DataHash); err != nil {
		return err
	}

	// ID management is done in TexStorage in Pascal. Here we just assign?
	// The file doesn't store TextureID, the runtime calculates it?
	// Reference: `result.texture_id := tex_storage.Insert(result.data_hash);`
	// The file implicitly stores duplicates and the storage deduplicates.
	// We can leave TextureID 0 for now or implement deduplication.
	// Let's implement simple deduplication later if needed.

	return nil
}

func (p *Parser) ReadUnfoldData() error {
	var hasUnfold uint8
	if err := p.reader.ReadBytes(&hasUnfold); err != nil {
		return err
	}

	if hasUnfold == 0 {
		return nil
	}

	if err := p.reader.ReadBytes(&p.PDO.Unfold.Scale); err != nil {
		return err
	}

	var padding uint8
	if err := p.reader.ReadBytes(&padding); err != nil {
		return err
	}

	if err := p.reader.ReadBytes(&p.PDO.Unfold.BoundingBox); err != nil {
		return err
	}

	if err := p.ReadParts(); err != nil {
		return err
	}
	if err := p.ReadTextBlocks(); err != nil {
		return err
	}
	if err := p.ReadImages(); err != nil {
		return err
	}

	return nil
}

func (p *Parser) ReadParts() error {
	var count int32
	if err := p.reader.ReadBytes(&count); err != nil {
		return err
	}

	p.PDO.Parts = make([]Part, count)
	for i := 0; i < int(count); i++ {
		if err := p.ReadPart(&p.PDO.Parts[i]); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) ReadPart(part *Part) error {
	if err := p.reader.ReadBytes(&part.ObjectIndex); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&part.BoundingBox); err != nil {
		return err
	}

	if p.PDO.Header.Version > PDO_V4 {
		var err error
		part.Name, err = p.reader.ReadShiftedString()
		if err != nil {
			return err
		}
	}

	var count int32
	if err := p.reader.ReadBytes(&count); err != nil {
		return err
	}

	part.Lines = make([]Line, count)
	for i := 0; i < int(count); i++ {
		if err := p.ReadLine(&part.Lines[i]); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) ReadLine(l *Line) error {
	var isHidden uint8
	if err := p.reader.ReadBytes(&isHidden); err != nil {
		return err
	}
	l.Hidden = isHidden == 1

	if err := p.reader.ReadBytes(&l.Type); err != nil {
		return err
	}

	var unknownByte uint8
	if err := p.reader.ReadBytes(&unknownByte); err != nil {
		return err
	}

	if err := p.reader.ReadBytes(&l.FaceIndex); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&l.VertexIndex); err != nil {
		return err
	}

	var secondIndex uint8
	if err := p.reader.ReadBytes(&secondIndex); err != nil {
		return err
	}
	l.IsConnectingFaces = secondIndex == 1

	if l.IsConnectingFaces {
		if err := p.reader.ReadBytes(&l.Face2Index); err != nil {
			return err
		}
		if err := p.reader.ReadBytes(&l.Vertex2Index); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) ReadTextBlocks() error {
	var count int32
	if err := p.reader.ReadBytes(&count); err != nil {
		return err
	}

	p.PDO.TextBlocks = make([]TextBlock, count)
	for i := 0; i < int(count); i++ {
		if err := p.ReadTextBlock(&p.PDO.TextBlocks[i]); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) ReadTextBlock(tb *TextBlock) error {
	if err := p.reader.ReadBytes(&tb.BoundingBox); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&tb.LineSpacing); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&tb.Color); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&tb.FontSize); err != nil {
		return err
	}

	var err error
	tb.FontName, err = p.reader.ReadShiftedString()
	if err != nil {
		return err
	}

	var count int32
	if err := p.reader.ReadBytes(&count); err != nil {
		return err
	}

	tb.Lines = make([]string, count)
	for i := 0; i < int(count); i++ {
		tb.Lines[i], err = p.reader.ReadShiftedString()
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) ReadImages() error {
	// First block
	var count int32
	if err := p.reader.ReadBytes(&count); err != nil {
		return err
	}

	p.PDO.Images = make([]Image, count)
	for i := 0; i < int(count); i++ {
		if err := p.ReadImage(&p.PDO.Images[i]); err != nil {
			return err
		}
	}

	// Second block (additional images)
	var addCount int32
	if err := p.reader.ReadBytes(&addCount); err != nil {
		return err
	}

	if addCount > 0 {
		oldLen := len(p.PDO.Images)
		newLen := oldLen + int(addCount)
		// extend slice
		newImages := make([]Image, newLen)
		copy(newImages, p.PDO.Images)
		p.PDO.Images = newImages

		for i := 0; i < int(addCount); i++ {
			if err := p.ReadImage(&p.PDO.Images[oldLen+i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Parser) ReadImage(img *Image) error {
	if err := p.reader.ReadBytes(&img.BoundingBox); err != nil {
		return err
	}
	if err := p.ReadTexture(&img.Texture); err != nil {
		return err
	}
	return nil
}

func (p *Parser) ReadSettings() error {
	// Unknown settings (v6)
	if p.PDO.Header.Version == PDO_V6 && len(p.PDO.Parts) > 0 {
		var count int32
		if err := p.reader.ReadBytes(&count); err != nil {
			return err
		}

		for i := 0; i < int(count); i++ {
			var parts int32
			if err := p.reader.ReadBytes(&parts); err != nil {
				return err
			}

			// Skip data
			skip := make([]byte, 4*parts)
			if err := p.reader.ReadBytes(skip); err != nil {
				return err
			}
		}
	}

	s := &p.PDO.Settings
	if err := p.reader.ReadBytes(&s.ShowFlaps); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&s.ShowEdgeID); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&s.EdgeIDPlacement); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&s.FaceMaterial); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&s.HideAlmostFlatFoldLines); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&s.FoldLinesHidingAngle); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&s.DrawWhiteLineUnderDotLine); err != nil {
		return err
	}

	// Mountain/Valley/Cut line styles (4 bytes each? Reference says 4*4 for mountain? No.)
	// Reference: `fpdo.ReadBytes((@settings.mountain_fold_line_style)^, 4*4);`
	// Wait, `mountain_fold_line_style` is `integer` (4 bytes).
	// But it reads 16 bytes?
	// `mountain`, `valley`, `cut` are 3 integers.
	// Oh, `Settings` Pascal struct:
	// mountain_fold_line_style, valley_, cut_ : integer;
	// edge_id_font_size: integer;
	// That's 4 integers -> 16 bytes.
	// My struct has them as individual fields.

	if err := p.reader.ReadBytes(&s.MountainFoldLineStyle); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&s.ValleyFoldLineStyle); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&s.CutLineStyle); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&s.EdgeIDFontSize); err != nil {
		return err
	}

	if err := p.reader.ReadBytes(&s.PageType); err != nil {
		return err
	}

	// PdoPageTypeOther = 11
	if s.PageType == 11 {
		if err := p.reader.ReadBytes(&s.CustomWidth); err != nil {
			return err
		}
		if err := p.reader.ReadBytes(&s.CustomHeight); err != nil {
			return err
		}
	}

	if err := p.reader.ReadBytes(&s.Orientation); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&s.MarginSide); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&s.MarginTop); err != nil {
		return err
	}

	// Fold line patterns
	if err := p.reader.ReadBytes(&s.MountainFoldLinePattern); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&s.ValleyFoldLinePattern); err != nil {
		return err
	}

	if err := p.reader.ReadBytes(&s.AddOutlinePadding); err != nil {
		return err
	}
	if err := p.reader.ReadBytes(&s.ScaleFactor); err != nil {
		return err
	}

	if p.PDO.Header.Version > PDO_V4 {
		var err error
		s.AuthorName, err = p.reader.ReadShiftedString()
		if err != nil {
			return err
		}
		s.Comment, err = p.reader.ReadShiftedString()
		if err != nil {
			return err
		}
	}

	return nil
}
