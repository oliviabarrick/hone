package utils

import (
	"hash/crc32"
)

func Crc(identifier string) int64 {
	crcTable := crc32.MakeTable(0xD5828281)
	result := int64(crc32.Checksum([]byte(identifier), crcTable))
	return result
}
