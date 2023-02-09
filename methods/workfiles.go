package methods

import (
	"bufio"
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
	Block          []byte
}

var stzFormats [2]string = [2]string{".dat", ".rel"}

//Repack - repack archive by extracted files
func Repack(table []FileTable, FilePath string, header []byte, stz bool, BE bool) {
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
	var success bool = true

	for i := 0; i < len(table); i++ {
		success = true
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

		tmp = make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, table[i].ArcNum)
		if BE == true {
			tmp = make([]byte, 4)
			binary.BigEndian.PutUint32(tmp, table[i].ArcNum)
		}
		copy(header[table[i].HeadOffset+8:], tmp)

		tmp = make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, table[i].Offset)
		if BE == true {
			tmp = make([]byte, 4)
			binary.BigEndian.PutUint32(tmp, table[i].Offset)
		}
		copy(header[table[i].HeadOffset+12:], tmp)

		read, _ := ioutil.ReadFile(strings.ReplaceAll(FilePath, ".toc", "") + "/" + table[i].FileName)

		tmp = make([]byte, Pad(table[i].Size, 4))
		copy(tmp[0:], read)

		if strings.Contains(table[i].FileName, ".stz") && stz == true {
			_, err1 := os.Stat(strings.ReplaceAll(FilePath, ".toc", "") + "/" + strings.TrimSuffix(table[i].FileName, ".stz") + stzFormats[0])
			_, err2 := os.Stat(strings.ReplaceAll(FilePath, ".toc", "") + "/" + strings.TrimSuffix(table[i].FileName, ".stz") + stzFormats[1])

			if err1 == nil && err2 == nil {
				head := make([]byte, 0x48)
				var commonSize uint32 = 0x48
				copy(head, tmp[:0x48])
				var files [2]stzFile

				for k := 0; k < 2; k++ {
					files[k].Offset = commonSize
					tmpfile, err := os.Open(strings.ReplaceAll(FilePath, ".toc", "") + "/" + strings.TrimSuffix(table[i].FileName, ".stz") + stzFormats[k])
					if err != nil {
						success = false
					}
					defer tmpfile.Close()

					info, _ := tmpfile.Stat()

					rawbytes := make([]byte, info.Size())
					files[k].Size = int32(info.Size())

					b := bufio.NewReader(tmpfile)
					_, err = b.Read(rawbytes)

					if err != nil {
						success = false
					}

					var buf bytes.Buffer

					writer := zlib.NewWriter(&buf)
					writer.Write(rawbytes)
					writer.Close()

					rawbytes = buf.Bytes()
					files[k].CompressedSize = int32(len(rawbytes))
					commonSize += Pad(uint32(files[k].CompressedSize), 8)

					files[k].Block = make([]byte, Pad(uint32(files[k].CompressedSize), 8))
					copy(files[k].Block, rawbytes)

					rawbytes = nil
				}

				if success {
					tmp = make([]byte, Pad(commonSize, 32))
					table[i].Size = uint32(len(tmp))
					copy(tmp, head)
					var offset uint32 = 36
					var sOff []byte

					for k := 0; k < 2; k++ {
						sOff = make([]byte, 4)
						binary.LittleEndian.PutUint32(sOff, files[k].Offset)
						if BE == true {
							tmp = make([]byte, 4)
							binary.BigEndian.PutUint32(sOff, files[k].Offset)
						}
						copy(tmp[offset:], sOff)
						offset += 4

						sOff = make([]byte, 4)
						binary.LittleEndian.PutUint32(sOff, uint32(files[k].Size))
						if BE == true {
							tmp = make([]byte, 4)
							binary.BigEndian.PutUint32(sOff, uint32(files[k].Size))
						}
						copy(tmp[offset:], sOff)
						offset += 4

						sOff = make([]byte, 4)
						binary.LittleEndian.PutUint32(sOff, uint32(files[k].CompressedSize))
						if BE == true {
							tmp = make([]byte, 4)
							binary.BigEndian.PutUint32(sOff, uint32(files[k].CompressedSize))
						}
						copy(tmp[offset:], sOff)
						offset += 4

						copy(tmp[files[k].Offset:], files[k].Block)
					}

					sOff = make([]byte, 4)
					binary.LittleEndian.PutUint32(sOff, commonSize)
					if BE == true {
						tmp = make([]byte, 4)
						binary.BigEndian.PutUint32(sOff, commonSize)
					}
					copy(tmp[offset:], sOff)
				}

				head = nil
			}
		}

		file.Write(tmp)

		tmp = nil
		read = nil

		tmp = make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, table[i].Size)
		if BE == true {
			tmp = make([]byte, 4)
			binary.BigEndian.PutUint32(tmp, table[i].Size)
		}
		copy(header[table[i].HeadOffset+16:], tmp)

		Size += uint64(Pad(table[i].Size, 4))
		FileOffset += Pad(table[i].Size, 4)

		fmt.Printf("Arc num:%d\tOff: %d\tSize: %d\tFileName: %s\n", ArcNum, table[i].Offset, table[i].Size, table[i].FileName)
	}

	tmpArcNum := make([]byte, 4)
	binary.LittleEndian.PutUint32(tmpArcNum, uint32(ArcNum+1))
	if BE == true {
		tmpArcNum = make([]byte, 4)
		binary.BigEndian.PutUint32(tmpArcNum, uint32(ArcNum+1))
	}
	copy(header[16:], tmpArcNum)

	header = EncHeader(header)

	file, _ = os.Create(FilePath)
	file.Write(header)

	file.Close()
}

//Unpack - extract files from Mxx archives where xx - number of archive
func Unpack(table []FileTable, FilePath string, stz bool, BE bool) {
	ArcFilePath := strings.ReplaceAll(FilePath, ".toc", ".M")

	err := os.MkdirAll(strings.ReplaceAll(ArcFilePath, ".M", ""), os.ModePerm)
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
			os.MkdirAll(Dir, os.ModePerm)
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
				if BE == true {
					files[k].Offset = binary.BigEndian.Uint32(tmp)
				}

				tmp = block[headOffset : headOffset+4]
				headOffset += 4
				files[k].Size = int32(binary.LittleEndian.Uint32(tmp))
				if BE == true {
					files[k].Size = int32(binary.BigEndian.Uint32(tmp))
				}

				tmp = block[headOffset : headOffset+4]
				headOffset += 4
				files[k].CompressedSize = int32(binary.LittleEndian.Uint32(tmp))
				if BE == true {
					files[k].CompressedSize = int32(binary.BigEndian.Uint32(tmp))
				}

				b := bytes.NewReader(block[files[k].Offset : files[k].Offset+uint32(files[k].CompressedSize)])
				z, err := zlib.NewReader(b)
				if err != nil {
					fmt.Println(err)
				}
				defer z.Close()

				files[k].Block, err = ioutil.ReadAll(z)

				if err != nil {
					fmt.Println(err)
				}

				file, _ = os.Create(strings.ReplaceAll(ArcFilePath, ".M", "") + strings.TrimSuffix(table[i].FileName, ".stz") + stzFormats[k])

				file.Write(files[k].Block)
				file.Close()
			}

			tmp = nil
		}

		block = nil
		file.Close()
	}
}
