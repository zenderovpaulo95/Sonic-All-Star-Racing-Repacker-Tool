package methods

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

//FileTable - struct of header files
type FileTable struct {
	FileName   string
	IsDir      uint32
	Offset     uint32
	HeadOffset int64
	Size       uint32
	ArcNum     uint32
	NameOff    uint32
	FileID     []byte
}

//Repack - repack archive by extracted files
func Repack(table []FileTable, FilePath string, header []byte) {
	var Size uint64
	var FileOffset uint32
	var ArcNum int
	ArcNum = 0
	FileOffset = 0
	Size = 0

	file, err := os.Create(strings.ReplaceAll(FilePath, ".toc", ".M"+fmt.Sprintf("%02d", ArcNum)))

	if err != nil {
		panic(err)
	}

	defer file.Close()

	for i := 0; i < len(table); i++ {
		table[i].FileName = strings.ReplaceAll(table[i].FileName, "//", "/")

		if Size+uint64(Pad(table[i].Size, 4)) > 4294967296 {
			ArcNum++
			FileOffset = 0
			Size = 0

			file.Close()

			file, err = os.Create(strings.ReplaceAll(FilePath, ".toc", ".M"+fmt.Sprintf("%02d", ArcNum)))
			if err != nil {
				panic(err)
			}

			defer file.Close()
		}

		var fileInfo os.FileInfo
		fileInfo, err = os.Stat(strings.ReplaceAll(FilePath, ".toc", "") + "/" + table[i].FileName)
		if err != nil {
			panic(err)
		}

		tmp := make([]byte, 8)
		binary.LittleEndian.PutUint64(tmp, uint64(fileInfo.Size()))

		table[i].Size = binary.LittleEndian.Uint32(tmp[:4])

		table[i].Offset = FileOffset
		table[i].ArcNum = uint32(ArcNum)
		Size += uint64(Pad(table[i].Size, 4))
		FileOffset += Pad(table[i].Size, 4)

		tmp = make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, table[i].ArcNum)
		copy(header[table[i].HeadOffset+8:], tmp)

		tmp = make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, table[i].Size)
		copy(header[table[i].HeadOffset+16:], tmp)

		tmp = make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, table[i].Offset)
		copy(header[table[i].HeadOffset+12:], tmp)

		read, err := ioutil.ReadFile(strings.ReplaceAll(FilePath, ".toc", "") + "/" + table[i].FileName)

		if err != nil {
			panic(err)
		}

		tmp = make([]byte, Pad(table[i].Size, 4))
		copy(tmp[0:], read)
		file.Write(tmp)

		tmp = nil
		read = nil

		fmt.Printf("Arc num:%d\tOff: %d\tSize: %d\tFileName: %s\n", ArcNum, table[i].Offset, table[i].Size, table[i].FileName)
	}

	tmpArcNum := make([]byte, 4)
	binary.LittleEndian.PutUint32(tmpArcNum, uint32(ArcNum+1))
	copy(header[16:], tmpArcNum)

	header = EncHeader(header)

	file, err = os.Create(FilePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	file.Write(header)
}

//Unpack - extract files from Mxx archives where xx - number of archive
func Unpack(table []FileTable, FilePath string) {
	ArcFilePath := strings.ReplaceAll(FilePath, ".toc", ".M")

	err := os.MkdirAll(strings.ReplaceAll(ArcFilePath, ".M", ""), 0666)
	if err != nil {
		panic(err)
	}

	for i := 0; i < len(table); i++ {
		file, err := os.Open(ArcFilePath + fmt.Sprintf("%02d", table[i].ArcNum))
		if err != nil {
			log.Fatal(err)
		}

		defer file.Close()

		block := make([]byte, table[i].Size)

		off, err := file.Seek(int64(table[i].Offset), 0)
		_, err = file.ReadAt(block, off)

		if err != nil {
			log.Fatal(err)
		}

		defer file.Close()

		Dir := filepath.Dir(strings.ReplaceAll(ArcFilePath, ".M", "") + "/" + table[i].FileName)
		_, err = os.Stat(Dir)

		if os.IsNotExist(err) {
			os.MkdirAll(Dir, 0666)
		}

		file, err = os.Create(strings.ReplaceAll(ArcFilePath, ".M", "") + "/" + table[i].FileName)

		if err != nil {
			panic(err)
		}

		defer file.Close()

		_, err = file.Write(block)

		if err != nil {
			panic(err)
		}

		defer file.Close()

		table[i].FileName = strings.ReplaceAll(table[i].FileName, "//", "/")
		fmt.Printf("Arc num: %d\tOff: %d\tSize: %d\tFileName: %s\n", table[i].ArcNum, table[i].Offset, table[i].Size, table[i].FileName)

		block = nil
		file.Close()
	}
}
