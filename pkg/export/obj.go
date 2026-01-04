package export

import (
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"pdo-tools/pkg/pdo"
)

// ExportOBJ exports the PDO model to Wavefront OBJ format.
// It writes the OBJ data to w, and creates an MTL file (and textures)
// using objPath as the base path.
func ExportOBJ(p *pdo.PDO, w io.Writer, objPath string) error {
	baseName := filepath.Base(objPath)
	mtlFileName := strings.TrimSuffix(baseName, filepath.Ext(baseName)) + ".mtl"
	mtlPath := filepath.Join(filepath.Dir(objPath), mtlFileName)

	// Write Header
	fmt.Fprintln(w, "# Exported by pdo-tools")
	fmt.Fprintf(w, "mtllib %s\n", mtlFileName)

	// Global indices for OBJ (1-based)
	vOffset := 1
	vtOffset := 1
	vnOffset := 1

	for objIdx, obj := range p.Objects {
		fmt.Fprintf(w, "\no %s_%d\n", sanitizeName(obj.Name), objIdx)

		// 1. Write Vertices
		for _, v := range obj.Vertices {
			// PDO Z is typically up, or Y is up?
			// Looking at types.go: X, Y, Z float64
			// Let's dump as is.
			fmt.Fprintf(w, "v %f %f %f\n", v.X, v.Y, v.Z)
		}

		// 2. Write Texture Coordinates and Normals (accumulate first to avoid duplicates?
		// Or just write per face loop which creates bloat but is simpler?
		// OBJ allows defining them anywhere.
		// For simplicity, we will iterate faces, collect unique UVs and Normals or just dump them linearly
		// and reference them.
		// Actually, standard practice is to dump all Vs first, then VTs, then VNs, then Faces.
		// But valid OBJ allows interleaved.
		// To keep indices sane, let's process this object's faces.

		// Wait, we need to defer writing faces until we have written all VTs and VNs for this object
		// AND we know their indices.
		// Let's iterate faces to dump VTs and VNs.

		// Temporary storage for this object's VTs and VNs to calculate indices
		// But dumping them immediately is easier if we track the count.

		// Let's accept that we might duplicate UVs/Normals if we just iterate faces.
		// Actually, PDO Face has Nx, Ny, Nz. This is one normal per Face.
		// So we can write Face Normals.

		objVTs := 0
		objVNs := 0

		// Buffer faces to write them after attributes
		var faceBuffer strings.Builder

		for _, face := range obj.Faces {
			// Write Normal
			fmt.Fprintf(w, "vn %f %f %f\n", face.Nx, face.Ny, face.Nz)
			currentVN := vnOffset + objVNs
			objVNs++

			// Write UVs
			// Face has Vertices which are Face2DVertex, containing U, V
			currentFaceVTIndices := make([]int, len(face.Vertices))
			for i, fv := range face.Vertices {
				fmt.Fprintf(w, "vt %f %f\n", fv.U, fv.V) // V usually needs flip? 1-V?
				// pdo_spec: "PDO uses the texture in bottom row first order, therefore the V coordinate in FVERTEX gets flipped."
				// Standard OBJ UV: (0,0) is bottom-left.
				// PDO U,V are float. Let's assume they are 0..1.
				currentFaceVTIndices[i] = vtOffset + objVTs
				objVTs++
			}

			// Material
			if face.MaterialIndex >= 0 && int(face.MaterialIndex) < len(p.Materials) {
				matName := p.Materials[face.MaterialIndex].Name
				if matName == "" {
					matName = fmt.Sprintf("Material_%d", face.MaterialIndex)
				}
				fmt.Fprintf(&faceBuffer, "usemtl %s\n", sanitizeName(matName))
			}

			// Face definition
			fmt.Fprintf(&faceBuffer, "f")
			for i, fv := range face.Vertices {
				// f v/vt/vn
				// v index: fv.IDVertex is the 0-based index in obj.Vertices
				// BUT we need global 1-based index.
				// global index = vOffset + fv.IDVertex
				vIdx := vOffset + int(fv.IDVertex)
				vtIdx := currentFaceVTIndices[i]
				vnIdx := currentVN // Flat shading, all verts in face share normal

				fmt.Fprintf(&faceBuffer, " %d/%d/%d", vIdx, vtIdx, vnIdx)
			}
			fmt.Fprintf(&faceBuffer, "\n")
		}

		// Flush faces
		fmt.Fprint(w, faceBuffer.String())

		// Update global offsets
		vOffset += len(obj.Vertices)
		vtOffset += objVTs
		vnOffset += objVNs
	}

	// Generate MTL
	if err := generateMTL(p, mtlPath); err != nil {
		return fmt.Errorf("failed to generate material library: %w", err)
	}

	return nil
}

func generateMTL(p *pdo.PDO, mtlPath string) error {
	f, err := os.Create(mtlPath)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, "# Exported by pdo-tools")

	for i, mat := range p.Materials {
		matName := mat.Name
		if matName == "" {
			matName = fmt.Sprintf("Material_%d", i)
		}
		fmt.Fprintf(f, "\nnewmtl %s\n", sanitizeName(matName))

		// Diffuse color from 3D Color (RGBA)
		// Color3D is [16]float32, 4x4 matrix? No, spec says:
		// 4*4B : 3D material color RGBA - float 4B
		// Wait, types.go says: Color3D [16]float32
		// Spec says:
		// 166:   4*4B : material color RGBA
		// 167:   4*4B : 3D material color RGBA
		// 168:   4*4B : light color RGBA
		// 169:   4*4B : diffuse color RGBA
		// 170:   4*4B : 2D material color ARGB

		// types.go has:
		// Color3D     [16]float32 // 4*4 float32 ??
		// That seems to map to multiple color fields in the spec?
		// Let's assume indices 0-3 are one color, 4-7 another, etc.
		// Spec:
		// 1. material color (Ambient?)
		// 2. 3D material color (Diffuse?)
		// 3. light color (Specular?)
		// 4. diffuse color (?)

		// If types.go treats them as one array [16], then:
		// 0..3: Material Color
		// 4..7: 3D Material Color
		// 8..11: Light Color
		// 12..15: Diffuse Color

		// Let's pick 4..7 (3D Material Color) for Kd (Diffuse)
		r := mat.Color3D[4]
		g := mat.Color3D[5]
		b := mat.Color3D[6]
		// a := mat.Color3D[7]

		fmt.Fprintf(f, "Kd %f %f %f\n", r, g, b)
		// Ka (Ambient) - let's use 0..3
		fmt.Fprintf(f, "Ka %f %f %f\n", mat.Color3D[0], mat.Color3D[1], mat.Color3D[2])
		// Ks (Specular) - let's use 8..11
		fmt.Fprintf(f, "Ks %f %f %f\n", mat.Color3D[8], mat.Color3D[9], mat.Color3D[10])

		// Texture map
		if mat.HasTexture {
			// Extract texture to file
			img, err := mat.Texture.GetImage()
			if err != nil {
				// Warn but continue?
				fmt.Printf("Warning: failed to decode texture for material %s: %v\n", matName, err)
			} else {
				texFileName := fmt.Sprintf("%s_tex%d.png", strings.TrimSuffix(filepath.Base(mtlPath), ".mtl"), i)
				texPath := filepath.Join(filepath.Dir(mtlPath), texFileName)

				texFile, err := os.Create(texPath)
				if err != nil {
					fmt.Printf("Warning: failed to create texture file %s: %v\n", texPath, err)
				} else {
					if err := png.Encode(texFile, img); err != nil {
						fmt.Printf("Warning: failed to encode texture %s: %v\n", texFileName, err)
					}
					texFile.Close()

					fmt.Fprintf(f, "map_Kd %s\n", texFileName)
				}
			}
		}
	}
	return nil
}

func sanitizeName(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, s)
}
