package encoder

import (
	"encoding/binary"
)

const FieldEOF byte = 255

func writeChunk(w Writer, data []byte) (err error) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(len(data)))
	if _, err = w.Write(buf); err != nil {
		return
	}
	_, err = w.Write(data)
	return
}

func writeString(w Writer, data string) (err error) {
	return writeChunk(w, []byte(data))
}

func writeUint16(w Writer, v uint16) (err error) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, v)
	_, err = w.Write(buf)
	return
}

func writeUint32(w Writer, v uint32) (err error) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, v)
	_, err = w.Write(buf)
	return
}

func writeUint64(w Writer, v uint64) (err error) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, v)
	_, err = w.Write(buf)
	return
}

func writeInt(w Writer, v int) (err error) {
	return writeUint64(w, uint64(v))
}

func writeInt64(w Writer, v int64) (err error) {
	return writeUint64(w, uint64(v))
}

func writeBool(w Writer, v bool) (err error) {
	buf := make([]byte, 1)
	if v {
		buf[0] = 1
	}
	_, err = w.Write(buf)
	return
}

func WriteArray[T any](w Writer, arr []T, do func(w Writer, v T) error) (err error) {
	if err = writeInt(w, len(arr)); err != nil {
		return
	}

	for _, v := range arr {
		if err = do(w, v); err != nil {
			return
		}
	}
	return
}
