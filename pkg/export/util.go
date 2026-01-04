package export

import "pdo-tools/pkg/pdo"

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
