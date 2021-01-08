package methods

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io/ioutil"
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

type stzFile struct {
	FileName       string
	Offset         uint32
	Size           int32
	CompressedSize int32
}

var stzFormats [2]string = [2]string{".dat", ".rel"}

//Repack - repack archive by extracted files
func Repack(table []FileTable, FilePath string, header []byte, stz bool) {
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

			file, _ = os.Create(strings.ReplaceAll(FilePath, ".toc", ".M"+fmt.Sprintf("%02d", ArcNum)))
		}

		var fileInfo os.FileInfo
		fileInfo, _ = os.Stat(strings.ReplaceAll(FilePath, ".toc", "") + "/" + table[i].FileName)

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

		read, _ := ioutil.ReadFile(strings.ReplaceAll(FilePath, ".toc", "") + "/" + table[i].FileName)

		tmp = make([]byte, Pad(table[i].Size, 4))
		copy(tmp[0:], read)

		if strings.Contains(table[i].FileName, ".stz") && stz == true {

			/*TODO: think about repack stz file with dat and rel files...

			_, err1 := os.Stat(strings.ReplaceAll(FilePath, ".toc", "") + "/" + strings.TrimSuffix(table[i].FileName, ".stz") + stzFormats[0])
			_, err2 := os.Stat(strings.ReplaceAll(FilePath, ".toc", "") + "/" + strings.TrimSuffix(table[i].FileName, ".stz") + stzFormats[1])

			if os.IsExist(err1) && os.IsExist(err2) {
				for k := 0; k < 2; k++ {
					datfile, err := ioutil.(strings.ReplaceAll(FilePath, ".toc", "") + "/" + strings.TrimSuffix(table[i].FileName, ".stz") + stzFormats[k])

					z, err := zlib.NewWriterLevel(b, zlib.DefaultCompression)
					if err != nil {
						fmt.Println(err)
					}
					defer z.Close()
				}
			}*/
		}

		file.Write(tmp)

		tmp = nil
		read = nil

		fmt.Printf("Arc num:%d\tOff: %d\tSize: %d\tFileName: %s\n", ArcNum, table[i].Offset, table[i].Size, table[i].FileName)
	}

	tmpArcNum := make([]byte, 4)
	binary.LittleEndian.PutUint32(tmpArcNum, uint32(ArcNum+1))
	copy(header[16:], tmpArcNum)

	header = EncHeader(header)

	file, _ = os.Create(FilePath)
	file.Write(header)

	file.Close()
}

//Unpack - extract files from Mxx archives where xx - number of archive
func Unpack(table []FileTable, FilePath string, stz bool) {
	ArcFilePath := strings.ReplaceAll(FilePath, ".toc", ".M")

	err := os.MkdirAll(strings.ReplaceAll(ArcFilePath, ".M", ""), 0666)
	if err != nil {
		panic(err)
	}

	for i := 0; i < len(table); i++ {
		table[i].FileName = strings.ReplaceAll(table[i].FileName, "//", "/")

		file, _ := os.Open(ArcFilePath + fmt.Sprintf("%02d", table[i].ArcNum))

		block := make([]byte, table[i].Size)

		off, err := file.Seek(int64(table[i].Offset), 0)
		_, _ = file.ReadAt(block, off)

		Dir := filepath.Dir(strings.ReplaceAll(ArcFilePath, ".M", "") + table[i].FileName)
		_, err = os.Stat(Dir)

		if os.IsNotExist(err) {
			os.MkdirAll(Dir, 0666)
		}

		file, _ = os.Create(strings.ReplaceAll(ArcFilePath, ".M", "") + table[i].FileName)

		_, _ = file.Write(block)

		fmt.Printf("Arc num: %d\tOff: %d\tSize: %d\tFileName: %s\n", table[i].ArcNum, table[i].Offset, table[i].Size, table[i].FileName)

		if strings.Contains(table[i].FileName, ".stz") && stz == true {
			var files [2]stzFile
			var headOffset uint32 = 36
			var tmp []byte

			for k := 0; k < 2; k++ {
				tmp = block[headOffset : headOffset+4]
				headOffset += 4
				files[k].Offset = binary.LittleEndian.Uint32(tmp)

				tmp = block[headOffset : headOffset+4]
				headOffset += 4
				files[k].Size = int32(binary.LittleEndian.Uint32(tmp))

				tmp = block[headOffset : headOffset+4]
				headOffset += 4
				files[k].CompressedSize = int32(binary.LittleEndian.Uint32(tmp))

				b := bytes.NewReader(block[files[k].Offset : files[k].Offset+uint32(files[k].CompressedSize)])
				z, err := zlib.NewReader(b)
				if err != nil {
					fmt.Println(err)
				}
				defer z.Close()

				tmp, err = ioutil.ReadAll(z)
				fmt.Printf("size %d -> tmp size %d\n", files[k].Size, len(tmp))
				if err != nil {
					fmt.Println(err)
				}

				file, _ = os.Create(strings.ReplaceAll(ArcFilePath, ".M", "") + strings.TrimSuffix(table[i].FileName, ".stz") + stzFormats[k])

				file.Write(tmp)
				file.Close()
			}

			tmp = nil
		}

		block = nil
		file.Close()
	}
}
