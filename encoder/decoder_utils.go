package encoder

import (
	"encoding/binary"
	"io"
)

func chunkReader(r io.Reader) (_ io.Reader, err error) {
	buf := make([]byte, 4)

	if _, err = io.ReadFull(r, buf); err != nil {
		return
	}

	size := binary.BigEndian.Uint32(buf[:4])

	return io.LimitReader(r, int64(size)), nil
}

func readChunk(r io.Reader) (data []byte, err error) {
	if r, err = chunkReader(r); err != nil {
		return
	}
	return io.ReadAll(r)
}

func readBool(r Reader) (v bool, err error) {
	var b byte
	if b, err = r.ReadByte(); err != nil {
		return
	}
	return b == 1, nil
}

func readString(r io.Reader) (data string, err error) {
	var d []byte
	if d, err = readChunk(r); err != nil {
		return
	}
	return string(d), nil
}

func readUint16(r io.Reader) (v uint16, err error) {
	d := make([]byte, 2)
	if _, err = io.ReadFull(r, d); err != nil {
		return
	}
	return binary.BigEndian.Uint16(d), nil
}

func readUint32(r io.Reader) (v uint32, err error) {
	d := make([]byte, 4)
	if _, err = io.ReadFull(r, d); err != nil {
		return
	}
	return binary.BigEndian.Uint32(d), nil
}

func readUint64(r io.Reader) (v uint64, err error) {
	d := make([]byte, 8)
	if _, err = io.ReadFull(r, d); err != nil {
		return
	}
	return binary.BigEndian.Uint64(d), nil
}

func readInt64(r io.Reader) (v int64, err error) {
	var u uint64
	if u, err = readUint64(r); err != nil {
		return
	}
	return int64(u), nil
}

func readInt(r io.Reader) (v int, err error) {
	var u uint64
	if u, err = readUint64(r); err != nil {
		return
	}
	return int(u), nil
}
