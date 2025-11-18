package source

func MustFileSetPos(f *File, offset int) Pos {
	p, err := f.FileSetPos(offset)
	if err != nil {
		panic(err)
	}
	return p
}

func MustFilePosition(f *File, pos Pos) FilePos {
	p, err := f.Position(pos)
	if err != nil {
		panic(err)
	}
	return p
}

func MustFilePositionFromOffset(f *File, offset int) FilePos {
	return MustFilePosition(f, MustFileSetPos(f, offset))
}

func MustFileLine(f *File, pos Pos) int {
	return MustFilePosition(f, pos).Line
}

func MustFileLineStartPos(f *File, line int) Pos {
	offset, err := f.Data.LineOffset(line)
	if err != nil {
		panic(err)
	}
	return Pos(f.Base + offset)
}

func MustFileOffset(f *File, pos Pos) int {
	offset, err := f.Offset(pos)
	if err != nil {
		panic(err)
	}
	return offset
}
