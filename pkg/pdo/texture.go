package pdo

import (
	"bytes"
	"compress/flate"
	"fmt"
	"image"
	"image/color"
	"io"
)

// DecompressTexture decodes the texture data into an image.Image
// The data structure seems to be:
// - wrapped_size (4 bytes) [Read by Parser]
// - header (2 bytes) [Read by Parser]
// - Deflate Stream (wrapped_size - 6 bytes) [Read into RawData by Parser]
// - Hash/Adler (4 bytes) [Read by Parser]
// So RawData contains the raw deflate stream.
func (t *Texture) GetImage() (image.Image, error) {
	if len(t.RawData) == 0 {
		return nil, fmt.Errorf("no texture data")
	}

	// Raw deflate stream
	r := flate.NewReader(bytes.NewReader(t.RawData))
	defer r.Close()

	// Decompressed size should be Width * Height * 3 (RGB)
	// Or maybe RGBA? Pascal code says "size := tex.width * tex.height * 3;"
	// So it's RGB.
	expectedSize := int(t.Width) * int(t.Height) * 3
	out := make([]byte, expectedSize)

	if _, err := io.ReadFull(r, out); err != nil {
		return nil, fmt.Errorf("deflate read failed: %w", err)
	}

	// Create image
	img := image.NewRGBA(image.Rect(0, 0, int(t.Width), int(t.Height)))

	// Convert RGB to RGBA
	// Pascal "pnm" format usually is RGB.
	// Structure: Top-Left to Bottom-Right?
	// Reference: `pdo_spec.txt` line 182: "texture format: RGB (interleaved color channel values), top left row first"
	// But line 186: "PDO uses the texture in bottom row first order, therefore the V coordinate in FVERTEX gets flipped."
	// Wait, "PDO uses the texture in bottom row first order".
	// "Top left row first order is in reference to original data... PDO uses... bottom row first".
	// So the data inside PDO might be bottom-up?
	// But `ReadTexture` logic doesn't flip data.
	// `tex_storage.pas` dump uses `PnmSave`.
	// `PnmSave` usually saves top-down.
	// So data might be standard top-down RGB? And UVs are flipped?
	// Or data is bottom-up?
	// If the spec says "PDO uses texture in bottom row first order", it implies the stored data is bottom-up?
	// Let's assume standard order first, then verify if UVs are flipped in `parser.go`?
	// `Read2DVertex` reads U, V.
	// `pdo_format.pas` doesn't flip V.
	// `pdo_common.pas` uses U,V.
	// `pdo2opf.pas` `ToOpfVert`: `result.v := v.v;`.
	// If V is 0..1, usually 0 is bottom in OpenGL, top in SVG/DirectX.
	// We'll stick to treating data as RGB pixels.

	k := 0
	for y := 0; y < int(t.Height); y++ {
		for x := 0; x < int(t.Width); x++ {
			if k+2 >= len(out) {
				break
			}
			r := out[k]
			g := out[k+1]
			b := out[k+2]
			img.SetRGBA(x, y, color.RGBA{r, g, b, 255})
			k += 3
		}
	}

	return img, nil
}
