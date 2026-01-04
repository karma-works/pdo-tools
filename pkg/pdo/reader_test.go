package pdo

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestReadString_ShiftJIS(t *testing.T) {
	// "日本語" in Shift-JIS: 93 FA 96 7B 8C EA
	sjisBytes := []byte{0x93, 0xFA, 0x96, 0x7B, 0x8C, 0xEA, 0x00}

	// Create a buffer resembling a PDO string: 4 bytes length, then bytes
	// Length = 7 (6 bytes + null)
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, int32(7))
	buf.Write(sjisBytes)

	reader := NewReader(buf)
	reader.MultiByteC = false // Single byte encoding
	reader.StringShift = 0    // No shift for this test

	got, err := reader.ReadString(0)
	if err != nil {
		t.Fatalf("ReadString failed: %v", err)
	}

	want := "日本語"
	if got != want {
		t.Errorf("ReadString got %q, want %q", got, want)
	}
}

func TestReadString_ShiftJIS_WithShift(t *testing.T) {
	// "ABC" -> 41 42 43 00
	// Shifted by 1 -> 42 43 44 01

	shiftedBytes := []byte{0x42, 0x43, 0x44, 0x01}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, int32(4))
	buf.Write(shiftedBytes)

	reader := NewReader(buf)
	reader.MultiByteC = false
	reader.StringShift = 1

	got, err := reader.ReadShiftedString()
	if err != nil {
		t.Fatalf("ReadShiftedString failed: %v", err)
	}

	want := "ABC"
	if got != want {
		t.Errorf("ReadShiftedString got %q, want %q", got, want)
	}
}

func TestReadString_MultiByte_UTF16(t *testing.T) {
	// "A" in UTF-16LE: 41 00
	// Shift by 1: 42 01 (applied to uint16 value? or bytes?)
	// Our code: val = w - uint16(shift).
	// So if verify A (0x0041), and shift is 1, stored is 0x0042.

	// "あ" (Hiragana A) in UTF-16LE: 0x3042
	// Stored: 0x3043 (if shift 1)

	term := uint16(0 + 1)
	raw := []uint16{0x3043, term}

	buf := new(bytes.Buffer)
	// Length in BYTES: 2 * 2 = 4
	binary.Write(buf, binary.LittleEndian, int32(4))
	binary.Write(buf, binary.LittleEndian, raw)

	reader := NewReader(buf)
	reader.MultiByteC = true
	reader.StringShift = 1

	got, err := reader.ReadShiftedString()
	if err != nil {
		t.Fatalf("ReadShiftedString failed: %v", err)
	}

	want := "あ"
	if got != want {
		t.Errorf("ReadShiftedString got %q, want %q", got, want)
	}
}
