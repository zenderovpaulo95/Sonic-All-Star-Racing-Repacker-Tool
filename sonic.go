package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	"./methods"
)

var Path, FileName string

func ReadHeaderFile(header []byte, table []methods.FileTable, Offset uint32, InfoOff uint32, NameOffset uint32, n uint32) []methods.FileTable {
	var fileInfo methods.FileTable

	Path += FileName + "/"

	bytesReader := bytes.NewReader(header)

	for i := 0; i < int(n); i++ {
		fileInfo.HeadOffset = int64(Offset)
		fileInfo.FileID = make([]byte, 4)
		bytesReader.ReadAt(fileInfo.FileID, int64(Offset))
		Offset += 4

		tmp := make([]byte, 4)
		bytesReader.ReadAt(tmp, int64(Offset))
		fileInfo.IsDir = binary.LittleEndian.Uint32(tmp)
		Offset += 4

		tmp = make([]byte, 4)
		bytesReader.ReadAt(tmp, int64(Offset))
		fileInfo.ArcNum = binary.LittleEndian.Uint32(tmp)
		Offset += 4

		tmp = make([]byte, 4)
		bytesReader.ReadAt(tmp, int64(Offset))
		fileInfo.Offset = binary.LittleEndian.Uint32(tmp)
		Offset += 4

		tmp = make([]byte, 4)
		bytesReader.ReadAt(tmp, int64(Offset))
		fileInfo.Size = binary.LittleEndian.Uint32(tmp)
		Offset += 4

		tmp = make([]byte, 4)
		bytesReader.ReadAt(tmp, int64(Offset))
		fileInfo.NameOff = binary.LittleEndian.Uint32(tmp)
		Offset += 4

		curOffset := Offset

		fileInfo.FileName = ""

		Offset = NameOffset + fileInfo.NameOff

		FileName = methods.GetFileName(header, Offset)

		Offset = curOffset

		if fileInfo.IsDir != 0 {
			curOffset = InfoOff + (fileInfo.Offset * (4 + 4 + 4 + 4 + 4 + 4))
			var dirPath string
			dirPath = FileName + "/"
			table = ReadHeaderFile(header, table, curOffset, InfoOff, NameOffset, fileInfo.Size)
			Path = strings.ReplaceAll(Path, dirPath, "")
		} else {
			fileInfo.FileName = Path + FileName
			table = append(table, fileInfo)
		}
	}

	return table
}

func main() {
	Args := os.Args
	if len(Args) > 1 {
		var show bool
		show = true

		for i := 1; i < len(Args); i++ {
			switch Args[i] {
			case "-extract":
				show = false
				if i+1 < len(Args) {
					if _, err := os.Stat(Args[i+1]); err == nil {

						result := methods.DecHeader(Args[i+1])

						var Offset, NameOffset uint32

						Offset = 0
						NameOffset = 0

						if result != nil {
							bytesReader := bytes.NewReader(result)

							tmp := make([]byte, 4)
							bytesReader.ReadAt(tmp, 20)
							Offset = binary.LittleEndian.Uint32(tmp)

							tmp = make([]byte, 4)
							bytesReader.ReadAt(tmp, 24)
							NameOffset = binary.LittleEndian.Uint32(tmp)

							FileName = ""
							Path = ""

							table := make([]methods.FileTable, 0)
							table = ReadHeaderFile(result, table, Offset, Offset, NameOffset, 1)

							methods.Unpack(table, Args[i+1])
						}
					} else if os.IsNotExist(err) {
						fmt.Printf("File %s doesn't exists!\n", Args[i+1])
						return
					} else {
						fmt.Printf("Unknown error: %s\n", err)
						return
					}
				}
				break

			case "-repack":
				show = false

				if i+1 < len(Args) {
					if _, err := os.Stat(Args[i+1]); err == nil {

						result := methods.DecHeader(Args[i+1])

						var Offset, NameOffset uint32

						Offset = 0
						NameOffset = 0

						if result != nil {
							bytesReader := bytes.NewReader(result)

							tmp := make([]byte, 4)
							bytesReader.ReadAt(tmp, 20)
							Offset = binary.LittleEndian.Uint32(tmp)

							tmp = make([]byte, 4)
							bytesReader.ReadAt(tmp, 24)
							NameOffset = binary.LittleEndian.Uint32(tmp)

							FileName = ""
							Path = ""

							table := make([]methods.FileTable, 0)
							table = ReadHeaderFile(result, table, Offset, Offset, NameOffset, 1)

							methods.Repack(table, Args[i+1], result)
						}
					} else if os.IsNotExist(err) {
						fmt.Printf("File %s doesn't exists!\n", Args[i+1])
						return
					} else {
						fmt.Printf("Unknown error: %s\n", err)
						return
					}
				}

				break
			}

			if show {
				fmt.Println("Please enter either -extract or -replace commands!")
			}
		}
	} else {
		fmt.Println("Use tool.exe -extract arc.file")
		fmt.Println("or")
		fmt.Println("Use tool.exe -replace arc.file")
		fmt.Println("Directory will be created nearby tool.exe if you extract files.")
		fmt.Println("If you want replace files make sure that directory with mod files")
		fmt.Println("nearby tool.exe")
	}
}
