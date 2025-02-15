package core

import (
	"bytes"
	"fmt"
)

// Edit Subtitle00.uasset to resize subtitle widget
// The original asset uses 930 x 210
// My dual subtitle mod uses 1170 x 260
func ResizeSubtitleWidget(filePath string, outPath string, width int, height int) {
	s := NewSerializer()

	// Open files
	fmt.Printf("Reading %s...\n", filePath)
	uassetFile := OpenFile(filePath)
	defer uassetFile.Close()

	s.SetReadFile(uassetFile)
	signature := s.Read(4)
	ver := VER_FF7R
	if !bytes.Equal(signature, UNREAL_SIGNATURE) {
		ver = VER_FF7R2
	}
	s.Seek(0, 0)
	bin := s.ReadAll()

	fmt.Printf("Writing %s...\n", outPath)
	newFile := CreateFile(outPath)
	defer newFile.Close()

	s.SetWriteFile(newFile)
	s.Write(bin)

	if ver == VER_FF7R2 {
		s.Seek(36459, 0)
		s.WriteFloat32(float32(width))
		s.Seek(36488, 0)
		s.WriteFloat32(float32(height))
		s.Seek(38688, 0)
		s.WriteFloat32(float32(width))
	} else {
		Throw("Sorry. This feature only supports FF7R2 for now")
	}
}
