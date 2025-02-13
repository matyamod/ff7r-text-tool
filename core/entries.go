package core

import (
	"encoding/csv"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

type SubEntry struct {
	nameId int    // id for name map in uasset header
	Id     string `json:"id"`
	Text   string `json:"text"`
}

func (e *SubEntry) Read(s *Serializer) {
	e.nameId = int(s.ReadUint32())
	s.ReadNull()
	e.Text = s.ReadString()
}

func (e *SubEntry) Write(s *Serializer) {
	s.WriteUint32(uint32(e.nameId))
	s.WriteNull()
	s.WriteString(e.Text)
}

func (e *SubEntry) GetBinSize() int {
	return 8 + GetStringBinSize(e.Text)
}

func (e *SubEntry) NameIdToString(uasset *Uasset) {
	if e.nameId < 0 || e.nameId > len(uasset.Names) {
		Throw(fmt.Sprintf("unexpected name id: %d", e.nameId))
	}
	e.Id = uasset.Names[e.nameId]
}

func (e *SubEntry) UpdateNameId(uasset *Uasset) {
	for i := range len(uasset.Names) {
		if uasset.Names[i] == e.Id {
			e.nameId = i
			return
		}
	}
	Throw(fmt.Errorf("SubEntry.Name (%s) is not found in uasset name map", e.Id))
}

func (e *SubEntry) WriteAsCsv(mainId string, w *csv.Writer) {
	record := []string{mainId, e.Id, GoStrToCsvStr(e.Text)}
	if err := w.Write(record); err != nil {
		Throw(err)
	}
}

func (e *SubEntry) Print() {
	fmt.Printf("      id: %s\n", e.Id)
	fmt.Printf("        text: %s\n", e.Text)
}

type Entry struct {
	Id         string     `json:"id"`
	Text       string     `json:"text"`
	SubEntries []SubEntry `json:"sub_entries,omitempty"`
}

func (e *Entry) Read(s *Serializer) {
	e.Id = s.ReadString()
	e.Text = s.ReadString()
	subEntryCount := s.ReadUint32()
	// Note: In the actual game assets, an entry has four sub entries at most.
	if subEntryCount >= 16 {
		Throw(fmt.Errorf("unexpected sub entry count: %d", subEntryCount))
	}

	e.SubEntries = make([]SubEntry, 0, subEntryCount)
	for range subEntryCount {
		se := SubEntry{}
		se.Read(s)
		e.SubEntries = append(e.SubEntries, se)
	}
}

func (e *Entry) Write(s *Serializer) {
	s.WriteString(e.Id)
	s.WriteString(e.Text)
	subEntryCount := len(e.SubEntries)
	s.WriteUint32(uint32(subEntryCount))
	for i := range subEntryCount {
		e.SubEntries[i].Write(s)
	}
}

// Check if there are duplicated ids in sub entries
func (e *Entry) CheckDuplication() {
	for i := range len(e.SubEntries) {
		id := e.SubEntries[i].nameId
		for j := i + 1; j < len(e.SubEntries); j++ {
			if id == e.SubEntries[j].nameId {
				Throw("Duplicated sub entry id detected.")
			}
		}
	}
}

func (e *Entry) GetBinSize() int {
	var size int = 4
	size += GetStringBinSize(e.Id)
	size += GetStringBinSize(e.Text)
	for _, se := range e.SubEntries {
		size += se.GetBinSize()
	}
	return size
}

func (e *Entry) NameIdToString(uasset *Uasset) {
	for i := range len(e.SubEntries) {
		e.SubEntries[i].NameIdToString(uasset)
	}
}

func (e *Entry) UpdateNameId(uasset *Uasset) {
	for i := range len(e.SubEntries) {
		e.SubEntries[i].UpdateNameId(uasset)
	}
}

func (e *Entry) UpdateWithNewEntry(newE *Entry) {
	e.Text = newE.Text
	for _, se := range newE.SubEntries {
		var found bool = false
		for i := range len(e.SubEntries) {
			if se.Id == e.SubEntries[i].Id {
				e.SubEntries[i].Text = se.Text
				found = true
				break
			}
		}
		if !found {
			Throw(fmt.Errorf("unknown sub entry id detected (%s)", se.Id))
		}
	}
}

func (e *Entry) UpdateWithCsv(row []string) {
	sub_id := row[1]
	if row[1] == "" {
		e.Text = CsvStrToGoStr(row[2])
		return
	}
	for i := range len(e.SubEntries) {
		if sub_id == e.SubEntries[i].Id {
			e.SubEntries[i].Text = CsvStrToGoStr(row[2])
			return
		}
	}
	Throw(fmt.Errorf("unknown sub entry id detected (%s)", sub_id))
}

func (e *Entry) WriteAsCsv(w *csv.Writer) {
	record := []string{e.Id, "", GoStrToCsvStr(e.Text)}
	if err := w.Write(record); err != nil {
		Throw(err)
	}
	for i := range len(e.SubEntries) {
		e.SubEntries[i].WriteAsCsv(e.Id, w)
	}
}

var SUBTTILE_CATEGORIES = []string{
	"MAIN",
	"QST_",
	"NPC_",
	"MGV_",
	"CDV_",
}

func (e *Entry) IsSubtitle() bool {
	for i := range len(e.SubEntries) {
		if e.SubEntries[i].Id == "ACTOR" {
			return true
		}
	}
	// Note: Some voice lines don't have the "ACTOR" property in FF7R2.
	//       So, we have to check id.
	if len(e.Id) < 11 {
		return false
	}
	cat := e.Id[7:11]
	return slices.Contains(SUBTTILE_CATEGORIES, cat) &&
		!strings.HasSuffix(cat, "_sys")
}

func (e *Entry) CountLines() int {
	return strings.Count(e.Text, "\n") + 1
}

func (e *Entry) CountLunes() int {
	return utf8.RuneCountInString(e.Text)
}

// Append another text data
func (e *Entry) Merge(e2 *Entry, sep string) {
	if len(e.Text) == 0 {
		e.Text = e2.Text
		return
	} else if len(e2.Text) == 0 {
		return
	}
	e.Text += sep + e2.Text
}

func (e *Entry) Print() {
	fmt.Printf("  id: %s\n", e.Id)
	fmt.Printf("    text: %s\n", e.Text)
	subEntryCount := len(e.SubEntries)
	fmt.Printf("    sub entry count:%d\n", subEntryCount)
	if subEntryCount > 0 {
		fmt.Println("    sub entries:")
	}
	for _, e := range e.SubEntries {
		e.Print()
	}
}
