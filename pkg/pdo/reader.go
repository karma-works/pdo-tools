package pdo

import (
	"encoding/binary"
	"io"
	"unicode/utf16"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
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

		// Read count items. Spec implies null termination.
		// However, reading exact buffer is safer.
		buf := make([]uint16, count)
		if err := binary.Read(r.r, binary.LittleEndian, buf); err != nil {
			return "", err
		}

		// Apply shift and collect valid chars
		runes := make([]uint16, 0, count)
		for _, w := range buf {
			val := w - uint16(shift)
			if val == 0 {
				break // Null terminator
			}
			runes = append(runes, val)
		}

		// Decode UTF-16 (Little Endian)
		return string(utf16.Decode(runes)), nil

	} else {
		// Single byte
		count := wrappedLen
		if count <= 0 {
			return "", nil
		}

		buf := make([]byte, count)
		if err := binary.Read(r.r, binary.LittleEndian, buf); err != nil {
			return "", err
		}

		// Apply shift
		validBytes := make([]byte, 0, count)
		for _, b := range buf {
			val := b - shift
			if val == 0 {
				break
			}
			validBytes = append(validBytes, val)
		}

		// Decode Shift-JIS
		decoder := japanese.ShiftJIS.NewDecoder()
		utf8Bytes, _, err := transform.Bytes(decoder, validBytes)
		if err != nil {
			return string(validBytes), nil
		}

		return string(utf8Bytes), nil
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
