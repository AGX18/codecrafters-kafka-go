package main

import "bufio"

func writeUvarint(buf *bufio.Writer, x uint64) error {
	for x >= 0x80 {
		err := buf.WriteByte(byte(x) | 0x80)
		if err != nil {
			return err
		}
		x >>= 7
	}
	return buf.WriteByte(byte(x))
}

func calcUvarintSize(x uint64) int {
	size := 0

	for x >= 0x80 {
		size++
		x >>= 7
	}
	size++

	return size
}
