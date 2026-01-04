package pdo

import (
	"encoding/binary"
	"io"
)

// Reader handles PDO specific binary reading
type Reader struct {
	r           io.Reader
	StringShift byte
	MultiByteC  bool
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

func (r *Reader) ReadBytes(data interface{}) error {
	return binary.Read(r.r, binary.LittleEndian, data)
}

func (r *Reader) ReadInt32() (int32, error) {
	var v int32
	err := binary.Read(r.r, binary.LittleEndian, &v)
	return v, err
}

func (r *Reader) ReadUInt32() (uint32, error) {
	var v uint32
	err := binary.Read(r.r, binary.LittleEndian, &v)
	return v, err
}

func (r *Reader) ReadUInt8() (uint8, error) {
	var v uint8
	err := binary.Read(r.r, binary.LittleEndian, &v)
	return v, err
}

func (r *Reader) ReadFloat64() (float64, error) {
	var v float64
	err := binary.Read(r.r, binary.LittleEndian, &v)
	return v, err
}

// ReadString reads a length-prefixed string.
// The length is 4 bytes.
// If MultiByteC is true, characters are 2 bytes (UTF-16ish).
// The string is expected to be null-terminated, so the last character is read but discarded.
// The 'shift' is applied to each character.
func (r *Reader) ReadString(shift byte) (string, error) {
	var wrappedLen int32
	if err := binary.Read(r.r, binary.LittleEndian, &wrappedLen); err != nil {
		return "", err
	}

	if wrappedLen == 0 {
		return "", nil
	}

	if r.MultiByteC {
		// Length is in bytes, convert to number of wchars
		count := wrappedLen / 2
		if count <= 0 {
			return "", nil
		}

		// Read count-1 characters
		buf := make([]uint16, count-1)
		for i := 0; i < int(count-1); i++ {
			var w uint16
			if err := binary.Read(r.r, binary.LittleEndian, &w); err != nil {
				return "", err
			}
			// Apply shift. Note: The reference code does (w - shift) & 0xFF.
			// This suggests it's converting to "byte" string even if it was u16?
			// But for now let's reproduce the reference logic.
			// val := (w - uint16(shift)) & 0xFF
			// But we return string (utf8/ansi).
			// We'll store it as is, but if we follow reference logic, we might need to be careful.
			// Reference: ucs += widechar((w - shift) and $ff);
			// This implies it only keeps the lower 8 bits after shift.

			buf[i] = (w - uint16(shift)) & 0xFF
		}

		// Consume the null terminator
		var term uint16
		if err := binary.Read(r.r, binary.LittleEndian, &term); err != nil {
			return "", err
		}

		// Convert []uint16 (which are effectively bytes) to string
		b := make([]byte, len(buf))
		for i, v := range buf {
			b[i] = byte(v)
		}
		return string(b), nil

	} else {
		// Single byte
		count := wrappedLen
		if count <= 0 {
			return "", nil
		}

		buf := make([]byte, count-1)
		for i := 0; i < int(count-1); i++ {
			var b byte
			if err := binary.Read(r.r, binary.LittleEndian, &b); err != nil {
				return "", err
			}
			buf[i] = b - shift
		}

		// Consume null terminator
		var term byte
		if err := binary.Read(r.r, binary.LittleEndian, &term); err != nil {
			return "", err
		}

		return string(buf), nil
	}
}

func (r *Reader) ReadShiftedString() (string, error) {
	return r.ReadString(r.StringShift)
}

func (r *Reader) ReadRect() (Rect, error) {
	var rect Rect
	if err := binary.Read(r.r, binary.LittleEndian, &rect); err != nil {
		return rect, err
	}
	return rect, nil
}
