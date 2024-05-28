package os

import (
	"os"
	"os/user"
	"reflect"

	"github.com/gad-lang/gad"
	cmdu "github.com/unapu-go/cmd-utils"
)

var (
	Module = gad.Dict{
		"FileFlag":     TFileFlag,
		"pwd":          gad.MustNewReflectValue(os.Getwd),
		"uid":          gad.MustNewReflectValue(os.Getuid),
		"gid":          gad.MustNewReflectValue(os.Getgid),
		"homeDir":      gad.MustNewReflectValue(os.UserHomeDir),
		"user":         gad.MustNewReflectValue(user.Current),
		"getUser":      gad.MustNewReflectValue(user.Lookup),
		"getUserByID":  gad.MustNewReflectValue(user.LookupId),
		"getGroup":     gad.MustNewReflectValue(user.LookupGroup),
		"getGroupByID": gad.MustNewReflectValue(user.LookupGroupId),
		"Cmd":          gad.NewReflectType(reflect.TypeOf(cmdu.CmdBuilder{})),
		"env":          gad.MustNewReflectValue(cmdu.OsEnv),
		"mkdir":        gad.MustNewReflectValue(os.Mkdir),
		"mkdirAll":     gad.MustNewReflectValue(os.MkdirAll),
		"rm":           gad.MustNewReflectValue(os.Remove),
		"rmAll":        gad.MustNewReflectValue(os.RemoveAll),
		"stat":         gad.MustNewReflectValue(os.Stat),
		"exec": &gad.Function{
			Name:  "exec",
			Value: Exec,
		},
		"exists": &gad.Function{
			Name:  "exists",
			Value: Exists,
		},
		"createFile": &gad.Function{
			Name:  "createFile",
			Value: CreateFile,
		},
		"openFile": &gad.Function{
			Name:  "openFile",
			Value: OpenFile,
		},
		"readFile": &gad.Function{
			Name:  "readFile",
			Value: ReadFile,
		},
	}
)
