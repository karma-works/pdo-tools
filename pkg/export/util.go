package export

import "pdo-tools/pkg/pdo"

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
