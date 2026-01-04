package pdo

// Basic types mapping to PDO structure

type Rect struct {
	Left, Top, Width, Height float64
}

type Header struct {
	Version          int32
	MultiByteChars   int32
	DesignerID       string
	StringShift      int32
	TexLock          int32
	Locale           string
	Codepage         string
	Key              string
	V6Lock           int32
	ShowStartupNotes uint8
	PasswordFlag     uint8
	AssembledHeight  float64
	OriginOffset     [3]float64
}

type Face2DVertex struct {
	IDVertex     int32
	X, Y         float64
	U, V         float64
	Flap         uint8
	FlapHeight   float64
	FlapAAngle   float64
	FlapBAngle   float64
	FlapFoldInfo [24]byte // 3*4 + 3*4
}

type Face struct {
	MaterialIndex int32
	PartIndex     int32
	Nx, Ny, Nz    float64
	Coord         float64
	Vertices      []Face2DVertex
}

type Edge struct {
	Face1Index      int32
	Face2Index      int32
	Vertex1Index    int32
	Vertex2Index    int32
	ConnectsFaces   int16 // Using int16 to match 2 bytes
	NoConnectedFace int32
	// LineType is added in reference logic but physically read as 22 bytes?
	// Reference: f.ReadBytes(Result, 22);
	// TPdoEdge in pdo_common.pas is packed.
	// 4+4+4+4+2+4 = 22 bytes.
	// TPdoEdge struct has LineType at end, but ReadBytes reads 22 bytes.
	// So LineType is NOT in the file at this point. It must be computed or ignored.
}

type Vertex3D struct {
	X, Y, Z float64
}

type Object struct {
	Name     string
	Visible  uint8
	Vertices []Vertex3D
	Faces    []Face
	Edges    []Edge
}

type Texture struct {
	Width      int32
	Height     int32
	DataSize   uint32
	DataHeader uint16
	DataHash   uint32
	TextureID  int32
	RawData    []byte
}

type Material struct {
	Name        string
	Color3D     [16]float32 // 4*4 float32
	Color2DRGBA [4]float32
	HasTexture  bool
	Texture     Texture
}

type TextBlock struct {
	BoundingBox Rect
	LineSpacing float64
	Color       int32
	FontSize    int32
	FontName    string
	Lines       []string
}

type Image struct {
	BoundingBox Rect
	Texture     Texture
}

type Line struct {
	Hidden            bool
	Type              int32
	FaceIndex         int32
	VertexIndex       int32
	IsConnectingFaces bool
	Face2Index        int32
	Vertex2Index      int32
}

type Part struct {
	ObjectIndex int32
	BoundingBox Rect
	Name        string
	Lines       []Line
}

type Settings struct {
	ShowFlaps                 uint8
	ShowEdgeID                uint8
	EdgeIDPlacement           uint8
	FaceMaterial              uint8
	HideAlmostFlatFoldLines   uint8
	FoldLinesHidingAngle      int32
	DrawWhiteLineUnderDotLine uint8
	MountainFoldLineStyle     int32
	ValleyFoldLineStyle       int32
	CutLineStyle              int32
	EdgeIDFontSize            int32

	PageType     int32
	CustomWidth  float64
	CustomHeight float64
	Orientation  int32
	MarginTop    int32
	MarginSide   int32

	MountainFoldLinePattern [6]float64
	ValleyFoldLinePattern   [6]float64

	AddOutlinePadding uint8
	ScaleFactor       float64
	AuthorName        string
	Comment           string
}

type Unfold struct {
	Scale       float64
	BoundingBox Rect
}

type PDO struct {
	Header     Header
	Objects    []Object
	Materials  []Material
	TextBlocks []TextBlock
	Parts      []Part
	Images     []Image
	Settings   Settings
	Unfold     Unfold
}
