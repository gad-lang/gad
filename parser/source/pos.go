// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// Copyright (c) 2019 Daniel Kang.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE.tengo file.

// Copyright 2009 The ToInterface Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.golang file.

package source

import (
	"fmt"
	"io"
)

// Pos represents a position in the file set.
type Pos int

// NoPos represents an invalid position.
const NoPos Pos = 0

// IsValid returns true if the position is valid.
func (p Pos) IsValid() bool {
	return p != NoPos
}

// PosStackTrace is the stack of source positions
type PosStackTrace []Pos

// Format formats the FilePosStackTrace to the fmt.Formatter
func (t PosStackTrace) Format(fs *FileSet, s fmt.State, verb rune) {
	if len(t) == 0 {
		io.WriteString(s, "no stack trace")
		return
	}

	switch verb {
	case 'v', 's':
		if s.Flag('+') {
			st := make(FilePosStackTrace, len(t))
			for i, pos := range t {
				st[i] = fs.Position(pos)
			}
			st.Format(s, verb)
		} else {
			fmt.Fprintf(s, "%v", []Pos(t))
		}
	}
}
