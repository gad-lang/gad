package importers

import (
	"os"
	"path"
	"path/filepath"
)

func ResolvePath(dirs []string, allowAbs bool, name string) (relPath, absPath string, err error) {
	if !allowAbs {
		relPath = path.Clean(name)
		for _, d := range dirs {
			absPath, err = filepath.Abs(filepath.Join(d, relPath))
		}
	} else if filepath.IsAbs(name) {
		absPath = name
		relPath = name
	} else {
		for _, d := range dirs {
			if absPath, err = filepath.Abs(filepath.Join(d, name)); err != nil {
				return
			}
			if _, err = os.Stat(absPath); err == nil {
				relPath, err = filepath.Rel(d, absPath)
				return
			}
		}
		absPath = ""
		err = os.ErrNotExist
	}
	return
}
