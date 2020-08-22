package main

import (
	"fmt"
	"io"
	"encoding/binary"
	"bytes"
	"debug/elf"
	"os"
	"unsafe"
	)

// Goals
// [1] Determine if file is elf binary (abort process if it isn't)
// [2] Determine architecture of elf binary if it is 32bit or 64bit
// [3] Abstract away architectural differences from the perspective of main function
// [4] Create functionality to print elfheader
// [5] Create functionality to print imported symbols
// [6] Create functionality to print exported symbols
// [7] Create functionality to print all symbols
// [8] Create functionality to print relocation entries
// [9] Create functionality to print shared library dependencies
// [10] Create functionality to locate a specific symbol

type ElfBiner interface {
	Header(amb_elf_arch interface{})
}

type EnumIdent struct {
	Endianness binary.ByteOrder
	Arch elf.Class
	Machine elf.Machine
}

type ShdrTble struct {
	Section interface{}
	SectionName []string
}

type Symtab struct{
	Symbol interface{}
	SymbolName []string
	
}

type ElfFile struct{
	Fh *os.File
	Ident [16]byte
	FileHdr EnumIdent
	Hdr interface{}
	Err error
	ElfSections ShdrTble
	ElfSymbols Symtab
	Rels map[uint32]interface{}
	Size int64


//	XSections map[uint32]interface{}
	Symbols map[uint32]interface{}
	SymbolsName map[uint32]string
	DynSymbols map[uint32]interface{}
	DynSymbolsName map[uint32]string
	
}

const (
	SUCCESS int = 0
	ERROR int = 1
	DYN_SYM int = 0xa
	SYM int = 0xb
)

func (elfFs *ElfFile) Header(amb_elf_arch interface{}){
	switch v := amb_elf_arch.(type) {
		case *elf.Header32:
			fmt.Printf("Elf32 detected: %v\n", v)
		case *elf.Header64:
			fmt.Println("Elf64 detected: %v\n", v)
		default:
			fmt.Println("Invalid Type detected: %v\n", v)
	}
}

func (elfFs *ElfFile) SetArch() {
	switch elf.Class(elfFs.Ident[elf.EI_CLASS]) {
		case elf.ELFCLASS64:
			elfFs.Hdr = new(elf.Header64)
			elfFs.FileHdr.Arch = elf.ELFCLASS64
			
		case elf.ELFCLASS32:
			elfFs.Hdr = new(elf.Header32)
			elfFs.FileHdr.Arch = elf.ELFCLASS32
		default:
			fmt.Println("Elf Arch Class Invalid !\n")
			os.Exit(ERROR)
	}
}

func (elfFs *ElfFile) MapHeader() {

	switch elf.Data(elfFs.Ident[elf.EI_DATA]) {
	case elf.ELFDATA2LSB:
		elfFs.FileHdr.Endianness = binary.LittleEndian
	case elf.ELFDATA2MSB:
		elfFs.FileHdr.Endianness = binary.BigEndian
	default:
			fmt.Println("Possible Corruption, Endianness unknown\n")
	}

	elfFs.Fh.Seek(0, io.SeekStart)
	err := binary.Read(elfFs.Fh, elfFs.FileHdr.Endianness, elfFs.Hdr)
	checkError(err)

	switch h := elfFs.Hdr.(type) {
	case *elf.Header32:
		elfFs.FileHdr.Machine = elf.Machine(h.Machine)
	case *elf.Header64:
		elfFs.FileHdr.Machine = elf.Machine(h.Machine)
	}
}


func (elfFs *ElfFile) findSectionByName(name string) {
	return
}




//Section Header Table Offset = Shoff
//Number of Section Header Table Entries = Shnum
//Size per entry in Section Header Table = Shentsize
//Calculate the size of Section Header Table = Shnum * Shentsize


func (elfFs *ElfFile) getSections() {

	if h, ok := elfFs.Hdr.(*elf.Header64); ok {
		shdrTableSize := h.Shentsize * h.Shnum

		elfFs.ElfSections.Section = make([]elf.Section64, h.Shnum)
		elfFs.ElfSections.SectionName = make([]string, h.Shnum)
				
		sr := io.NewSectionReader(elfFs.Fh, int64(h.Shoff), int64(shdrTableSize))
		err := binary.Read(sr, elfFs.FileHdr.Endianness, elfFs.ElfSections.Section.([]elf.Section64))
		checkError(err)

		shstrtab := make([]byte, elfFs.ElfSections.Section.([]elf.Section64)[h.Shstrndx].Size)
		shstrtabOff := elfFs.ElfSections.Section.([]elf.Section64)[h.Shstrndx].Off
		shstrtabSize := elfFs.ElfSections.Section.([]elf.Section64)[h.Shstrndx].Size

		shstrtabSec := io.NewSectionReader(elfFs.Fh, int64(shstrtabOff), int64(shstrtabSize) + int64(shstrtabOff))
		err = binary.Read(shstrtabSec, elfFs.FileHdr.Endianness, shstrtab)
		checkError(err)

		for i := 0; i < int(h.Shnum); i++ {
			sIndex := elfFs.ElfSections.Section.([]elf.Section64)[i].Name
			elfFs.ElfSections.SectionName[i] = getSectionName(sIndex, shstrtab)
		}
	}

	if h, ok := elfFs.Hdr.(*elf.Header32); ok {
		shdrTableSize := h.Shentsize * h.Shnum

		elfFs.ElfSections.Section = make([]elf.Section32, h.Shnum)
		elfFs.ElfSections.SectionName = make([]string, h.Shnum)

		sr := io.NewSectionReader(elfFs.Fh, int64(h.Shoff), int64(shdrTableSize))
		err := binary.Read(sr, elfFs.FileHdr.Endianness, elfFs.ElfSections.Section.([]elf.Section32))
		checkError(err)

		shstrtab := make([]byte, elfFs.ElfSections.Section.([]elf.Section32)[h.Shstrndx].Size)
		shstrtabOff := elfFs.ElfSections.Section.([]elf.Section32)[h.Shstrndx].Off
		shstrtabSize := elfFs.ElfSections.Section.([]elf.Section32)[h.Shstrndx].Size
		shstrTableEnd := shstrtabOff + shstrtabSize

		shstrtabSec := io.NewSectionReader(elfFs.Fh, int64(shstrtabOff), int64(shstrTableEnd))
		err = binary.Read(shstrtabSec, elfFs.FileHdr.Endianness, shstrtab)
		checkError(err)

		for i := 0; i < int(h.Shnum); i++ {
			sIndex := elfFs.ElfSections.Section.([]elf.Section32)[i].Name
			elfFs.ElfSections.SectionName[i] = getSectionName(sIndex, shstrtab)
		}
	}
}



func (elfFs *ElfFile) getSymbols() {
    
	var dsymtabNdx uint32
	if dsymtabNdx = getSectionNdx(".dynsym", elfFs); dsymtabNdx != 0 {
		var  dynstrNdx uint32
		dynstrNdx  = getSectionNdx(".dynstr", elfFs)
		elfFs.loadSymbols(dsymtabNdx, dynstrNdx, DYN_SYM)

		fmt.Printf("%d entries found in .dynsym\n", len(elfFs.DynSymbols))
		printSymbols(elfFs)
	} else {
		fmt.Println("No Dynamic symbols found - .dynsym missing from target")
	}

	var symtabNdx uint32
	var symstrNdx  uint32
	if symtabNdx  = getSectionNdx(".symtab", elfFs); symtabNdx != 0 {
		symstrNdx  = getSectionNdx(".strtab", elfFs)
		elfFs.loadSymbols(symtabNdx, symstrNdx, SYM)

		fmt.Printf("%d entries found in .symtab\n", len(elfFs.Symbols))
		printSymbols(elfFs)
	} else {
		fmt.Println("Section .symtab mising -- Binary is stripped no exported symbols available !")
	}
}

func (elfFs *ElfFile) loadSymbols(sectionNdx uint32, symstrNdx uint32, symType int) {
	switch elfFs.FileHdr.Arch {
	case elf.ELFCLASS32:
		var sym32 elf.Sym32
		symSize := uint32(unsafe.Sizeof(sym32))
		symtabSize := elfFs.ElfSections.Section.([]elf.Section32)[sectionNdx].Size
		numSymbols := symtabSize / symSize
		off := elfFs.ElfSections.Section.([]elf.Section32)[sectionNdx].Off

		/* strtab can be either .dynstr or .strtab depending on the symbol table*/
		strtab := make([]byte, elfFs.ElfSections.Section.([]elf.Section32)[symstrNdx].Size)
		strtabOff := elfFs.ElfSections.Section.([]elf.Section32)[symstrNdx].Off
		strtabSize := elfFs.ElfSections.Section.([]elf.Section32)[symstrNdx].Size

		n := int64(strtabOff + strtabSize)	
		shstrtabSec := io.NewSectionReader(elfFs.Fh, int64(strtabOff), n)

		err := binary.Read(shstrtabSec, elfFs.FileHdr.Endianness, strtab)
		checkError(err)

		if symType == SYM {
			elfFs.Symbols = make(map[uint32]interface{})
			elfFs.SymbolsName = make(map[uint32]string)
			n := int64(off + symtabSize)
			sr := io.NewSectionReader(elfFs.Fh, int64(off), n)

			for symNdx := uint32(0); symNdx < numSymbols; symNdx++ {
				elfFs.Symbols[symNdx] = new(elf.Sym32)
				err := binary.Read(sr, elfFs.FileHdr.Endianness, elfFs.Symbols[symNdx])
				checkError(err)
				symEntry := elfFs.Symbols[symNdx]
				elfFs.SymbolsName[symEntry.(*elf.Sym32).Name] = getSymbolName(symEntry.(*elf.Sym32).Name, strtab)
			}
		} 

		if symType == DYN_SYM {
			elfFs.DynSymbols = make(map[uint32]interface{})
			elfFs.DynSymbolsName = make(map[uint32]string)
			n := int64(off + symtabSize)
			sr := io.NewSectionReader(elfFs.Fh, int64(off), n)

			for symNdx := uint32(0); symNdx < numSymbols; symNdx++ {
				elfFs.DynSymbols[symNdx] = new(elf.Sym32)
				err := binary.Read(sr, elfFs.FileHdr.Endianness, elfFs.DynSymbols[symNdx])
				checkError(err)
				symEntry := elfFs.DynSymbols[symNdx]
				elfFs.DynSymbolsName[symEntry.(*elf.Sym32).Name] = getSymbolName(symEntry.(*elf.Sym32).Name, strtab)
			}
		}
		
	
	case elf.ELFCLASS64:
		var sym64 elf.Sym64
		symSize := uint32(unsafe.Sizeof(sym64))
		symtabSize := elfFs.ElfSections.Section.([]elf.Section64)[sectionNdx].Size
		numSymbols := symtabSize / uint64(symSize)
		off := elfFs.ElfSections.Section.([]elf.Section64)[sectionNdx].Off

		/* strtab can be either .dynstr or .strtab depending on the symbol table*/
		strtab := make([]byte, elfFs.ElfSections.Section.([]elf.Section64)[symstrNdx].Size)
		strtabOff := elfFs.ElfSections.Section.([]elf.Section64)[symstrNdx].Off
		strtabSize := elfFs.ElfSections.Section.([]elf.Section64)[symstrNdx].Size
		n := int64(strtabOff + strtabSize)

		shstrtabSec := io.NewSectionReader(elfFs.Fh, int64(strtabOff), n)

		err := binary.Read(shstrtabSec, elfFs.FileHdr.Endianness, strtab)
		checkError(err)

		if symType == SYM {
			elfFs.Symbols = make(map[uint32]interface{})
			elfFs.SymbolsName = make(map[uint32]string)
			n := int64(off + symtabSize)
			sr := io.NewSectionReader(elfFs.Fh, int64(off), n)

			for symNdx := uint32(0); uint64(symNdx) < numSymbols; symNdx++ {
				elfFs.Symbols[symNdx] = new(elf.Sym64)
				err := binary.Read(sr, elfFs.FileHdr.Endianness, elfFs.Symbols[symNdx])
				checkError(err)
				symEntry := elfFs.Symbols[symNdx]
				elfFs.SymbolsName[symEntry.(*elf.Sym64).Name] = getSymbolName(symEntry.(*elf.Sym64).Name, strtab)
			}
		} 

		if symType == DYN_SYM {
			elfFs.DynSymbols = make(map[uint32]interface{})
			elfFs.DynSymbolsName = make(map[uint32]string)
			n  := int64(off + symtabSize)
			sr := io.NewSectionReader(elfFs.Fh, int64(off), n)

			for symNdx := uint32(0); uint64(symNdx) < numSymbols; symNdx++ {
				elfFs.DynSymbols[symNdx] = new(elf.Sym64)
				err := binary.Read(sr, elfFs.FileHdr.Endianness, elfFs.DynSymbols[symNdx])
				checkError(err)
				symEntry := elfFs.DynSymbols[symNdx]
				elfFs.DynSymbolsName[symEntry.(*elf.Sym64).Name] = getSymbolName(symEntry.(*elf.Sym64).Name, strtab)
			}
		}
	}
}


func getSymbolName(symIndex uint32, sectionStrtab []byte) string {
	return getSectionName(symIndex, sectionStrtab)
}

func getSectionNdx(name string, elfFs *ElfFile) uint32 {
	var ndx uint32
	for ndx = 0; ndx < uint32(len(elfFs.ElfSections.SectionName)); ndx++ {
		if elfFs.ElfSections.SectionName[ndx] == name {
				return ndx
		}
	}
	return uint32(0)
}

func getSectionName(sIndex uint32, sectionShstrTab []byte) string {
	end := sIndex
	for end < uint32(len(sectionShstrTab)) {
		if sectionShstrTab[end] == 0x0 {
			break;
		}
		end++
	}

	var name bytes.Buffer
	name.Write(sectionShstrTab[sIndex:end])
	return name.String()
}

func getSectionByType(t elf.SectionType, elfFs *ElfFile) []uint32 {

	var indexList []uint32

	if s, ok := elfFs.ElfSections.Section.([]elf.Section32); ok {
		for sNdx := uint32(0); sNdx < uint32(len(s)); sNdx++{
			if t == elf.SectionType(s[sNdx].Type) {
				indexList = append(indexList, sNdx)
			}
		}
	}

	if s, ok := elfFs.ElfSections.Section.([]elf.Section64); ok {
		for sNdx := uint32(0); sNdx < uint32(len(s)); sNdx++{
			if t == elf.SectionType(s[sNdx].Type) {
				indexList = append(indexList, sNdx)
			}
		} 
	}

	return indexList
}

func (elfFs *ElfFile) getRelocations() {

	elfFs.Rels = make(map[uint32]interface{})
	if s, ok := elfFs.ElfSections.Section.([]elf.Section32); ok {
		for sNdx := uint32(0); sNdx < uint32(len(s)); sNdx++ {
			switch elf.SectionType(s[sNdx].Type) {
				case elf.SHT_REL:
					var rel elf.Rel32
					sr := io.NewSectionReader(elfFs.Fh, int64(s[sNdx].Off), int64(s[sNdx].Size))
					numRels := s[sNdx].Size / uint32(unsafe.Sizeof(rel))
					elfFs.Rels[sNdx] = make([]elf.Rel32, numRels)  
					err := binary.Read(sr, elfFs.FileHdr.Endianness, elfFs.Rels[sNdx])
					checkError(err)
	
				case elf.SHT_RELA:
					var rel elf.Rela32
					sr := io.NewSectionReader(elfFs.Fh, int64(s[sNdx].Off), int64(s[sNdx].Size))
					numRels := s[sNdx].Size / uint32(unsafe.Sizeof(rel))
					elfFs.Rels[sNdx] = make([]elf.Rela32, numRels)  
					err := binary.Read(sr, elfFs.FileHdr.Endianness, elfFs.Rels[sNdx])
					checkError(err)

			}
		}
	}

	if s, ok := elfFs.ElfSections.Section.([]elf.Section64); ok {
		for sNdx := uint32(0); sNdx < uint32(len(s)); sNdx++ {
			switch elf.SectionType(s[sNdx].Type) {
				case elf.SHT_REL:
					var rel elf.Rel64
					n := int64(s[sNdx].Off) + int64(s[sNdx].Size)

					sr := io.NewSectionReader(elfFs.Fh, int64(s[sNdx].Off), n)
					numRels := s[sNdx].Size / uint64(unsafe.Sizeof(rel))
					elfFs.Rels[sNdx] = make([]elf.Rel64, numRels)  
					err := binary.Read(sr, elfFs.FileHdr.Endianness, elfFs.Rels[sNdx])
					checkError(err)
		
				case elf.SHT_RELA:
					var rel elf.Rela64
					n := int64(s[sNdx].Off) + int64(s[sNdx].Size)
					sr := io.NewSectionReader(elfFs.Fh, int64(s[sNdx].Off), n)
					numRels := s[sNdx].Size / uint64(unsafe.Sizeof(rel))
					elfFs.Rels[sNdx] = make([]elf.Rela64, numRels)  
					err := binary.Read(sr, elfFs.FileHdr.Endianness, elfFs.Rels[sNdx])
					checkError(err)
	
			}
		}
	}
}

func resolveRelocType(rType uint32, mType elf.Machine) string {
	switch mType {
	case elf.EM_X86_64:
		return fmt.Sprintf("%s", elf.R_X86_64(rType))
	case elf.EM_386:
		return fmt.Sprintf("%s", elf.R_386(rType))
	case elf.EM_ARM:
		return fmt.Sprintf("%s", elf.R_ARM(rType))
	case elf.EM_AARCH64:
		return fmt.Sprintf("%s", elf.R_AARCH64(rType))
	case elf.EM_PPC:
		return fmt.Sprintf("%s", elf.R_PPC(rType))
	case elf.EM_PPC64:
		return fmt.Sprintf("%s", elf.R_PPC64(rType))
	case elf.EM_MIPS:
		return fmt.Sprintf("%s", elf.R_MIPS(rType))
	case elf.EM_RISCV:
		return fmt.Sprintf("%s", elf.R_RISCV(rType))
	case elf.EM_S390:
		return fmt.Sprintf("%s", elf.R_390(rType))
	case elf.EM_SPARCV9:
		return fmt.Sprintf("%s", elf.R_SPARC(rType))
	default:
		return "R_UNKNOWN"
	}
}

func printRelocations(elfFs *ElfFile) {
	if _, ok := elfFs.ElfSections.Section.([]elf.Section32); ok {
		for k, v := range elfFs.Rels {
			sName := elfFs.ElfSections.SectionName[k]
			switch r := v.(type) {
				case []elf.Rel32:
					l := len(r)
					fmt.Printf("\nSection %s has %d relocation entries\n\n", sName, l)
					for rNdx := 0; rNdx < l; rNdx++ {
						fmt.Printf("Offset: 0x%x\n", r[rNdx].Off)
					}
				case []elf.Rela32:
					l := len(r)
					fmt.Printf("\nSection %s has %d relocation entries\n\n", sName, l)
					for rNdx := 0; rNdx < l; rNdx++ {
						fmt.Printf("Offset: 0x%x\t", r[rNdx].Off)
					}
			}
		}
	}

	if _, ok := elfFs.ElfSections.Section.([]elf.Section64); ok {
		for k, v := range elfFs.Rels {
			sName := elfFs.ElfSections.SectionName[k]
			switch r := v.(type) {
			case []elf.Rel64:
				l := len(r)
				fmt.Printf("\nSection %s has %d relocation entries\n\n", sName, l)
				fmt.Println("Offset\t\t\tInfo\t\t\t\tType\t\t\tSym.Value\t\t\tSym.Name\n")
				
				for rNdx := 0; rNdx < l; rNdx++ {

				}
			case []elf.Rela64:
				l := len(r)
				fmt.Printf("\nSection %s has %d relocation entries\n\n", sName, l)
				fmt.Println("Offset\t\t\tInfo\t\t\t\tType\t\t\tSym.Value\t\t\tSym.Name + Addend\n")

				for rNdx := 0; rNdx < l; rNdx++ {
					o := r[rNdx].Off
					a := r[rNdx].Addend
					t := elf.R_TYPE64(r[rNdx].Info)
					s := elf.R_SYM64(r[rNdx].Info)
					i := elf.R_INFO(s, t)

					relName := resolveRelocType(t, elfFs.FileHdr.Machine)
	
					var symName string
					var symValue uint64
					var symbol interface{}

					secNdx := elfFs.ElfSections.Section.([]elf.Section64)[k].Link
					switch elfFs.ElfSections.SectionName[secNdx] {
					case ".dynsym":
						symbol = elfFs.DynSymbols[s]
						symName = elfFs.DynSymbolsName[symbol.(*elf.Sym64).Name]
					case ".symtab":
						symbol = elfFs.Symbols[s]
						symName = elfFs.SymbolsName[symbol.(*elf.Sym64).Name]
					default:
						fmt.Printf("Error when locating symbol tables in printRelocations()")
						os.Exit(ERROR)
					}
					symValue = symbol.(*elf.Sym64).Value
					if s != uint32(elf.SHN_UNDEF) {
						symName += " + "
					}
					fmt.Printf("%016x\t%016x\t%s\t%016x\t\t%s\t%d\n", o, i, relName, symValue, symName, a)
				}
			}
		}
	}
}

func printSymbols(elfFs *ElfFile) {

	//ndx := getSectionNdx(".dynsym", elfFs)
	nsym := len(elfFs.ElfSymbols.SymbolName)
	//fmt.Printf("%d Symbol entries found in .dynsym\n\n", nsym)

	switch elfFs.FileHdr.Arch{
		case elf.ELFCLASS32:
			for i := 0; i < nsym; i++ {
				fmt.Printf("Entry: #%d\n", i)
				fmt.Printf("Value: 0x%x\n",elfFs.ElfSymbols.Symbol.([]elf.Sym32)[i].Value)
				fmt.Printf("Size: %d\n", elfFs.ElfSymbols.Symbol.([]elf.Sym32)[i].Size)
				fmt.Printf("Type: %s\n", elf.ST_TYPE(elfFs.ElfSymbols.Symbol.([]elf.Sym32)[i].Info))
				fmt.Printf("Bind: %s\n", elf.ST_BIND(elfFs.ElfSymbols.Symbol.([]elf.Sym32)[i].Info))
			//	fmt.Printf("Info: %s\n", elf.ST_INFO(elfFs.ElfSymbols.Symbol.([]elf.Sym32)[i].Info))
				fmt.Printf("Visibility: %s\n", elf.ST_VISIBILITY(elfFs.ElfSymbols.Symbol.([]elf.Sym32)[i].Other))
				fmt.Printf("Section: %d\n", elfFs.ElfSymbols.Symbol.([]elf.Sym32)[i].Shndx)
				fmt.Printf("Name: %s\n", elfFs.ElfSymbols.SymbolName[i])
				fmt.Printf("\n\n\n\n")
			}

		case elf.ELFCLASS64:
			for i := 0; i < nsym; i++ {
				fmt.Printf("Entry: #%d\n", i)
				fmt.Printf("Value: 0x%x\n",elfFs.ElfSymbols.Symbol.([]elf.Sym64)[i].Value)
				fmt.Printf("Size: %d\n", elfFs.ElfSymbols.Symbol.([]elf.Sym64)[i].Size)
				fmt.Printf("Type: %s\n", elf.ST_TYPE(elfFs.ElfSymbols.Symbol.([]elf.Sym64)[i].Info))
				fmt.Printf("Bind: %s\n", elf.ST_BIND(elfFs.ElfSymbols.Symbol.([]elf.Sym64)[i].Info))
			//	fmt.Printf("Info: %s\n", elf.ST_INFO(elfFs.ElfSymbols.Symbol.([]elf.Sym32)[i].Info))
				fmt.Printf("Visibility: %s\n", elf.ST_VISIBILITY(elfFs.ElfSymbols.Symbol.([]elf.Sym64)[i].Other))
				fmt.Printf("Section: %d\n", elfFs.ElfSymbols.Symbol.([]elf.Sym64)[i].Shndx)
				fmt.Printf("Name: %s\n", elfFs.ElfSymbols.SymbolName[i])
				fmt.Printf("\n\n\n\n")
			}
	}

}

func printSections(ElfSections ShdrTble, numSec uint16, secOff interface{}) {
	
	fmt.Printf("------------------------------------------\n\n\n")
	switch v := secOff.(type) {
		case uint32:
		fmt.Printf("%d Sections @ Offset 0x%x\n", numSec, v)

		case uint64:
		fmt.Printf("%d Sections @ Offset 0x%x\n", numSec, v)
	}

	if section, ok := ElfSections.Section.([]elf.Section32); ok {
		for i := uint16(0); i < numSec; i++ {
			fmt.Printf("Section Number: %d\n", i)
			fmt.Printf("Name: %s\n", ElfSections.SectionName[i])
			fmt.Printf("Type: %s\n", elf.SectionType(section[i].Type))
			fmt.Printf("Flags: %s\n", elf.SectionFlag(section[i].Flags))
			fmt.Printf("Address: 0x%x\n", section[i].Addr)
			fmt.Printf("Offset: 0x%x\n", section[i].Off)
			fmt.Printf("Size: 0x%x\n", section[i].Size)
			fmt.Printf("Link: 0x%x\n", section[i].Link)
			fmt.Printf("Info: 0x%x\n", section[i].Info)
			fmt.Printf("Alignment: 0x%x\n", section[i].Addralign)
			fmt.Printf("Entry Size: 0x%x\n", section[i].Entsize)
		}
	}

	if section, ok := ElfSections.Section.([]elf.Section64); ok {
		for i := uint16(0); i < numSec; i++ {
			fmt.Printf("------------------------------------------\n\n\n")
			fmt.Printf("Section Number: %d\n", i)
			fmt.Printf("Name: %s\n", ElfSections.SectionName[i])
			fmt.Printf("Type: %s\n", elf.SectionType(section[i].Type))
			fmt.Printf("Flags: %s\n", elf.SectionFlag(section[i].Flags))
			fmt.Printf("Address: 0x%x\n", section[i].Addr)
			fmt.Printf("Offset: 0x%x\n", section[i].Off)
			fmt.Printf("Size: 0x%x\n", section[i].Size)
			fmt.Printf("Link: 0x%x\n", section[i].Link)
			fmt.Printf("Info: 0x%x\n", section[i].Info)
			fmt.Printf("Alignment: 0x%x\n", section[i].Addralign)
			fmt.Printf("Entry Size: 0x%x\n", section[i].Entsize)
		}
	}
	return
}

func printHeader(hdr interface{}) {
        if h, ok := hdr.(*elf.Header64); ok{
                fmt.Printf("-------------------------- Elf Header ------------------------\n")
                fmt.Printf("Magic: % x\n", h.Ident)
                fmt.Printf("Elf Type: %s\n", elf.Type(h.Type))
                fmt.Printf("Machine: %s\n", elf.Machine(h.Machine))
                fmt.Printf("Version: %s\n", elf.Version(h.Version))
                fmt.Printf("Entry: 0x%x\n", h.Entry)
                fmt.Printf("Program Header Offset: 0x%x\n", h.Phoff)
                fmt.Printf("Section Header Offset: 0x%x\n", h.Shoff)
                fmt.Printf("Flags: 0x%x\n", h.Flags)
                fmt.Printf("Elf Header Size (bytes): %d\n", h.Ehsize)
                fmt.Printf("Program Header Entry Size (bytes): %d\n", h.Phentsize)
                fmt.Printf("Number of Program Header Entries: %d\n", h.Phnum)
                fmt.Printf("Size of Section Header Entry: %d\n", h.Shentsize)
                fmt.Printf("Number of Section Header Entries: %d\n", h.Shnum)
                fmt.Printf("Index In Section Header Table For String Section: %d\n", h.Shstrndx)
	}

        if h, ok := hdr.(*elf.Header32); ok{
                fmt.Printf("-------------------------- Elf Header ------------------------\n")
                fmt.Printf("Magic: % x\n", h.Ident)
                fmt.Printf("Elf Type: %s\n", elf.Type(h.Type))
                fmt.Printf("Machine: %s\n", elf.Machine(h.Machine))
                fmt.Printf("Version: %s\n", elf.Version(h.Version))
                fmt.Printf("Entry: 0x%x\n", h.Entry)
                fmt.Printf("Program Header Offset: 0x%x\n", h.Phoff)
                fmt.Printf("Section Header Offset: 0x%x\n", h.Shoff)
                fmt.Printf("Flags: 0x%x\n", h.Flags)
                fmt.Printf("Elf Header Size (bytes): %d\n", h.Ehsize)
                fmt.Printf("Program Header Entry Size (bytes): %d\n", h.Phentsize)
                fmt.Printf("Number of Program Header Entries: %d\n", h.Phnum)
                fmt.Printf("Size of Section Header Entry: %d\n", h.Shentsize)
                fmt.Printf("Number of Section Header Entries: %d\n", h.Shnum)
                fmt.Printf("Index In Section Header Table For String Section: %d\n", h.Shstrndx)
	}
	return
}

func main() {

	if len(os.Args) < 3{
		usage()
		os.Exit(ERROR)

	}

	var target ElfFile;

	bin := os.Args[2]
	target.Fh, target.Err = os.Open(bin)
	checkError(target.Err)

	target.Fh.Read(target.Ident[:16])

	if isElf(target.Ident[:4]) == false {
		fmt.Println("This is not an Elf binary\n")
		os.Exit(1)
	}
	target.SetArch()
	target.MapHeader()


	options := os.Args[1]
	if options[0] != '-' {
		usage()
		os.Exit(ERROR)
	}

	var optHeader, optSections, optSymbols, optRelocations, optTest bool
	for i := 1; i < len(options) ; i++ {
		switch {
			case options[i] == 'h':
				optHeader = true
			case options[i] == 'S':
				optSections = true
			case options[i] == 's':
				optSymbols = true
			case options[i] == 'r':
				optRelocations = true
			case options[i] == 't':
				optTest = true
			default:
				fmt.Println("Unrecognizable parameters");
				os.Exit(ERROR)
		}
	}

	if optHeader {
		printHeader(target.Hdr)
	}

	if optSections {
		target.getSections()
		switch target.FileHdr.Arch {
			case elf.ELFCLASS32:
				printSections(target.ElfSections, target.Hdr.(*elf.Header32).Shnum, target.Hdr.(*elf.Header32).Shoff)
			case elf.ELFCLASS64:
				printSections(target.ElfSections, target.Hdr.(*elf.Header64).Shnum, target.Hdr.(*elf.Header64).Shoff)
		}
	}

	if optSymbols {
		if optSections == false {
			target.getSections()
		}
		target.getSymbols()
	}

	if optRelocations {
		if optSections == false {
			target.getSections()
		}

		if optSymbols == false {
			target.getSymbols()
		}
		target.getRelocations()
		printRelocations(&target)

	}

	if optTest{
		fmt.Println(resolveRelocType(7, elf.EM_X86_64))
	}
}

func usage() {
	fmt.Println("Usage information")
}

func checkError(checkError error){
	if checkError != nil{
		panic(checkError)
	}
}

func isElf(magic []byte) bool {
	return !(magic[0] != '\x7f' || magic[1] != 'E' || magic [2] != 'L' || magic[3] != 'F')
}
