package core

import (
	"encoding/binary"
	"fmt"
	"os"
)

type VersionEnum int

const (
	VER_FF7R VersionEnum = iota
	VER_FF7R2
)

type Serializer struct {
	Ver       VersionEnum
	order     binary.ByteOrder
	file      *os.File
	endOffset int
}

func (s *Serializer) SetVersion(ver VersionEnum) {
	s.Ver = ver
}

func (s *Serializer) SetOrder(order binary.ByteOrder) {
	s.order = order
}

func (s *Serializer) SetReadFile(file *os.File) {
	s.file = file
	s.endOffset = s.GetFileSize()
}

func (s *Serializer) SetWriteFile(file *os.File) {
	s.file = file
}

func NewSerializer() *Serializer {
	s := &Serializer{}
	s.SetOrder(binary.LittleEndian)
	return s
}

func (s *Serializer) Seek(offset int, whence int) {
	_, err := s.file.Seek(int64(offset), whence)
	if err != nil {
		Throw(err)
	}
}

func (s *Serializer) GetOffset() int {
	offset, err := s.file.Seek(0, 1)
	if err != nil {
		Throw(err)
	}
	return int(offset)
}

func (s *Serializer) GetFileSize() int {
	offset := s.GetOffset()
	s.Seek(0, 2)
	size := s.GetOffset()
	s.Seek(offset, 0)
	return size
}

func (s *Serializer) Read(size int) []byte {
	if s.GetOffset()+size > s.endOffset {
		Throw("EOF")
	}
	buf := make([]byte, size)
	_, err := s.file.Read(buf)
	if err != nil {
		Throw(err)
	}
	return buf
}

func (s *Serializer) Write(buf []byte) {
	_, err := s.file.Write(buf)
	if err != nil {
		Throw(err)
	}
}

func (s *Serializer) ReadStruct(any interface{}) {
	err := binary.Read(s.file, s.order, any)
	if err != nil {
		Throw(err)
	}
}

func (s *Serializer) ReadInt32() int32 {
	var num int32 = 0
	err := binary.Read(s.file, s.order, &num)
	if err != nil {
		Throw(err)
	}
	return num
}

func (s *Serializer) WriteInt32(num int32) {
	err := binary.Write(s.file, s.order, &num)
	if err != nil {
		Throw(err)
	}
}

func (s *Serializer) ReadUint32() uint32 {
	var num uint32 = 0
	err := binary.Read(s.file, s.order, &num)
	if err != nil {
		Throw(err)
	}
	return num
}

func (s *Serializer) WriteUint32(num uint32) {
	err := binary.Write(s.file, s.order, &num)
	if err != nil {
		Throw(err)
	}
}

func (s *Serializer) ReadNull() {
	num := s.ReadInt32()
	if num != 0 {
		Throw(fmt.Errorf("not null: %d", num))
	}
}

func (s *Serializer) WriteNull() {
	s.WriteInt32(0)
}

func (s *Serializer) ReadStringBase(strlen int32, isUTF16 bool) string {
	var str string
	if strlen == 0 {
	} else if isUTF16 {
		buf := s.Read(int(strlen * 2))
		str = UTF16BytesToStr(buf)
	} else {
		buf := s.Read(int(strlen))
		str = string(buf)
	}
	return str
}

func (s *Serializer) ReadString() string {
	strlen := s.ReadInt32()
	if strlen == 0 {
		return ""
	} else if strlen > 0 {
		// UTF8
		str := s.ReadStringBase(strlen-1, false)
		s.Seek(1, 1)
		return str
	}
	// UTF16
	str := s.ReadStringBase(-strlen-1, true)
	s.Seek(2, 1)
	return str
}

func (s *Serializer) ReadZenString() string {
	bin := s.Read(2)
	strlen := int32(bin[1]) + (int32(bin[0]&0x7F) << 8)
	isUTF16 := (bin[0] & 0x80) > 0
	return s.ReadStringBase(strlen, isUTF16)
}

func GetStringBinSize(str string) int {
	// Get length of string for uasset
	var size int = 0
	if len(str) == 0 {
	} else if isASCII(str) {
		size = len(str) + 1
	} else {
		size = (GetUTF16Len(str) + 1) * 2
	}
	return size + 4 // buffer + buffer size (int32)
}

func (s *Serializer) WriteString(str string) {
	if len(str) == 0 {
		s.WriteNull()
	} else if isASCII(str) {
		s.WriteInt32(int32(len(str) + 1))
		s.Write([]byte(str))
		s.Write([]byte{0})
	} else {
		buf, size := StrToUTF16Bytes(str)
		size = -(size + 1)
		s.WriteInt32(int32(size))
		s.Write(buf)
		s.Write([]byte{0, 0})
	}
}

func ZenLengthBin(strlen int, isUTF16 bool) []byte {
	if isUTF16 {
		return []byte{byte(strlen >> 8), byte(strlen & 0xFF)}
	}
	return []byte{byte(strlen>>8 + 0x80), byte(strlen & 0xFF)}
}

func (s *Serializer) WriteZenString(str string) {
	isUTF16 := !isASCII(str)
	if isUTF16 {
		buf, size := StrToUTF16Bytes(str)
		s.Write(ZenLengthBin(size, isUTF16))
		s.Write(buf)
	} else {
		buf := []byte(str)
		s.Write(ZenLengthBin(len(buf), isUTF16))
		s.Write(buf)
	}
}
