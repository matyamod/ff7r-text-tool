package core

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

var LANG_LIST = []string{
	"BR", "CN", "DE", "ES",
	"FR", "IT", "JP", "KR",
	"MX", "TW", "US",
}

type Uexp struct {
	head    []byte
	Lang    string  `json:"language"`
	noneId  []byte  // name map id for "None"
	Entries []Entry `json:"entries,omitempty"`
}

type ZenPackageSummary struct {
	NameId                    uint32
	NameNumber                uint32
	SourceNameId              uint32
	SourceNameNumber          uint32
	PkgFlags                  uint32
	CookedHeaderSize          uint32
	NameMapOffset             int32
	NameMapSize               int32
	NameHashesOffset          int32
	NameHashesSize            int32
	ImportOffset              int32
	ExportOffset              int32
	ExportBundleEntriesOffset int32
	GraphDataOffset           int32
	GraphDataSize             int32
}

func (s *ZenPackageSummary) GetNameMapEndOffset() int {
	return int(s.NameMapOffset + s.NameMapSize)
}

func (s *ZenPackageSummary) GetUassetEndOffset() int {
	return int(s.GraphDataOffset + s.GraphDataSize)
}

type Uasset struct {
	Names   []string
	rawBin  []byte
	Uexp    *Uexp
	Ver     VersionEnum
	Summary *ZenPackageSummary
}

var HEAD_MAGIC = []byte{0x00, 0x03}
var UNREAL_SIGNATURE = []byte{0xC1, 0x83, 0x2A, 0x9E}

func (uexp *Uexp) Read(s *Serializer) {
	if s.Ver == VER_FF7R {
		uexp.head = s.Read(2)
	} else if s.Ver >= VER_FF7R2 {
		uexp.head = s.Read(25)
	}

	uexp.Lang = s.ReadString()

	if !slices.Contains(LANG_LIST, uexp.Lang) {
		Throw(fmt.Errorf("unknown language detected. (%s)", uexp.Lang))
	}

	if s.Ver != VER_FF7R {
		uexp.noneId = s.Read(8)
	}
	s.ReadNull()
	entryCount := s.ReadUint32()
	if entryCount >= 65536 {
		Throw(fmt.Errorf("unexpected entry count: %d", entryCount))
	}
	uexp.Entries = make([]Entry, 0, entryCount)
	for range entryCount {
		e := Entry{}
		e.Read(s)
		uexp.Entries = append(uexp.Entries, e)
	}

	if s.Ver != VER_FF7R {
		return
	}

	signature := s.Read(4)
	if !bytes.Equal(signature, UNREAL_SIGNATURE) {
		Throw(fmt.Errorf("unexpected signature: %v", signature))
	}
}

func (uexp *Uexp) Write(s *Serializer) {
	s.Write(uexp.head)
	s.WriteString(uexp.Lang)
	if s.Ver != VER_FF7R {
		s.Write(uexp.noneId)
	}
	s.WriteNull()
	entryCount := len(uexp.Entries)
	s.WriteUint32(uint32(entryCount))
	for i := range entryCount {
		uexp.Entries[i].Write(s)
	}

	if s.Ver == VER_FF7R {
		s.Write(UNREAL_SIGNATURE)
	}
}

func (uexp *Uexp) GetBinSize() int {
	size := 8 + len(uexp.head) + len(uexp.noneId)
	size += GetStringBinSize(uexp.Lang)
	for i := range len(uexp.Entries) {
		size += uexp.Entries[i].GetBinSize()
	}
	return size
}

func (uexp *Uexp) NameIdToString(uasset *Uasset) {
	for i := range len(uexp.Entries) {
		uexp.Entries[i].NameIdToString(uasset)
	}
}

func (uexp *Uexp) UpdateNameId(uasset *Uasset) {
	for i := range len(uexp.Entries) {
		uexp.Entries[i].UpdateNameId(uasset)
	}
}

func (uexp *Uexp) FindEntry(key string, firstId int) int {
	id := firstId
	maxId := len(uexp.Entries)
	minId := 0
	for id >= minId && id < maxId {
		res := strings.Compare(key, uexp.Entries[id].Id)
		if res == 0 {
			return id // Found
		} else if res < 0 {
			if id <= minId {
				return -1
			}
			maxId = id
			id -= max((id-minId)/2, 1)
		} else {
			if id >= maxId-1 {
				return -1
			}
			minId = id
			id += max((maxId-id)/2, 1)
		}
	}
	return -1
}

func (uexp *Uexp) UpdateWithNewUexp(newUexp *Uexp) {
	if !slices.Contains(LANG_LIST, newUexp.Lang) {
		Throw(fmt.Errorf("unknown language detected. (%s)", newUexp.Lang))
	}
	uexp.Lang = newUexp.Lang
	for i := range len(newUexp.Entries) {
		e := newUexp.Entries[i]
		id := uexp.FindEntry(e.Id, min(i, len(newUexp.Entries)))
		if id < 0 {
			Throw(fmt.Errorf("unknown entry detected. (%s)", e.Id))
		}
		uexp.Entries[id].UpdateWithNewEntry(&e)
	}
}

func (uexp *Uexp) Print(verbose ...bool) {
	fmt.Printf("lang: %s\n", uexp.Lang)
	entryCount := len(uexp.Entries)
	fmt.Printf("entry count: %d\n", entryCount)
	if len(verbose) == 0 || !verbose[0] {
		return
	}
	if entryCount > 0 {
		fmt.Println("entries:")
	}
	for i := range len(uexp.Entries) {
		uexp.Entries[i].Print()
	}
}

func (uexp *Uexp) ReadFromCsv(r *csv.Reader) {
	last_id := 0
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			Throw(err)
		} else if len(row) != 3 {
			Throw("each row should has 3 items in csv")
		}
		id := row[0]
		if id == "id" {
			continue // first row
		} else if id == "language" {
			lang := row[2]
			if !slices.Contains(LANG_LIST, lang) {
				Throw(fmt.Errorf("unknown language detected. (%s)", lang))
			}
			uexp.Lang = lang
			continue
		}
		i := uexp.FindEntry(id, last_id)
		if i < 0 {
			Throw(fmt.Errorf("unknown entry detected. (%s)", id))
		}
		last_id = i
		uexp.Entries[i].UpdateWithCsv(row)
	}
}

func (uexp *Uexp) WriteAsCsv(w *csv.Writer) {
	record := []string{"id", "sub_id", "text"}
	if err := w.Write(record); err != nil {
		Throw(err)
	}
	record = []string{"language", "", uexp.Lang}
	if err := w.Write(record); err != nil {
		Throw(err)
	}
	for i := range len(uexp.Entries) {
		uexp.Entries[i].WriteAsCsv(w)
	}
}

func (uasset *Uasset) Read(s *Serializer) {
	// Make sure it has fourCC for uasset
	signature := s.Read(4)
	if bytes.Equal(signature, UNREAL_SIGNATURE) {
		uasset.Ver = VER_FF7R
		s.SetVersion(uasset.Ver)

		// We just read a name map, don't parse the whole binary.
		s.Seek(41, 0)
		nameCount := s.ReadUint32()
		if nameCount >= 2048 {
			Throw(fmt.Errorf("unexpected name count: %d", nameCount))
		}
		s.Seek(193, 0)
		uasset.Names = make([]string, 0, nameCount)
		for range nameCount {
			name := s.ReadString()
			uasset.Names = append(uasset.Names, name)
			s.Seek(4, 1) // skip hash
		}

		wholeSize := s.GetFileSize()
		s.Seek(0, 0)
		uasset.rawBin = s.Read(wholeSize)
	} else if bytes.Equal(signature, []byte{0, 0, 0, 0}) {
		uasset.Ver = VER_FF7R2
		s.SetVersion(uasset.Ver)

		s.Seek(0, 0)
		uasset.Summary = &ZenPackageSummary{}
		s.ReadStruct(uasset.Summary)
		s.Seek(int(uasset.Summary.NameMapOffset), 0)
		uasset.Names = make([]string, 0, 16)
		namesEndOffset := uasset.Summary.GetNameMapEndOffset()
		for s.GetOffset() < namesEndOffset {
			name := s.ReadZenString()
			uasset.Names = append(uasset.Names, name)
		}
		uassetEndOffset := uasset.Summary.GetUassetEndOffset()
		s.Seek(0, 0)
		uasset.rawBin = s.Read(uassetEndOffset)

	} else {
		Throw(fmt.Errorf("unexpected fourCC: %v", signature))
	}
}

func (uasset *Uasset) Write(s *Serializer) {
	// Make sure it has fourCC for uasset
	s.Write(uasset.rawBin)

	if uasset.Ver == VER_FF7R {
		s.Seek(-92, 2)
		uexpSize := int32(uasset.Uexp.GetBinSize())
		s.WriteInt32(uexpSize)
	} else {
		s.Seek(int(uasset.Summary.ExportOffset+8), 0)
		uexpSize := int32(uasset.Uexp.GetBinSize())
		s.WriteInt32(uexpSize)
		s.Seek(uasset.Summary.GetUassetEndOffset(), 0)
	}
}

func (uasset *Uasset) Update() {
	uasset.Uexp.UpdateNameId(uasset)
}

func (uasset *Uasset) Print(verbose ...bool) {
	nameCount := len(uasset.Names)
	fmt.Printf("name count: %d\n", nameCount)
	if len(verbose) != 0 && verbose[0] {
		for i := range len(uasset.Names) {
			fmt.Printf("  %s\n", uasset.Names[i])
		}
	}
	uasset.Uexp.Print(verbose[0])
}

func (uasset *Uasset) ReadFromFile(filePath string) {
	uexp := &Uexp{}

	serializer := NewSerializer()

	// Open a read only file
	fmt.Printf("Reading %s...\n", filePath)
	uassetFile, err := os.Open(filePath)
	if err != nil {
		Throw(err)
	}
	defer uassetFile.Close()

	serializer.SetReadFile(uassetFile)
	uasset.Read(serializer)

	if uasset.Ver == VER_FF7R {
		// Read .uexp
		uexpPath := RemoveExtension(filePath) + ".uexp"
		fmt.Printf("Reading %s...\n", uexpPath)
		uexpFile, err := os.Open(uexpPath)
		if err != nil {
			Throw(err)
		}
		defer uassetFile.Close()
		serializer.SetReadFile(uexpFile)
	}

	uexp.Read(serializer)
	uexp.NameIdToString(uasset)
	uasset.Uexp = uexp
}

func (uasset *Uasset) WriteToFile(filePath string) {
	uasset.Update()

	serializer := NewSerializer()

	// Open or create a file
	fmt.Printf("Writing %s...\n", filePath)
	uassetFile, err := os.Create(filePath)
	if err != nil {
		Throw(err)
	}
	defer uassetFile.Close()

	serializer.SetWriteFile(uassetFile)
	serializer.SetVersion(uasset.Ver)
	uasset.Write(serializer)

	if uasset.Ver == VER_FF7R {
		// Read .uexp
		uexpPath := RemoveExtension(filePath) + ".uexp"
		fmt.Printf("Writing %s...\n", uexpPath)
		uexpFile, err := os.Create(uexpPath)
		if err != nil {
			Throw(err)
		}
		defer uassetFile.Close()
		serializer.SetWriteFile(uexpFile)
	}

	uasset.Uexp.Write(serializer)
}
