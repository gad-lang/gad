package os

import (
	"os"
	"os/user"
	"reflect"

	"github.com/gad-lang/gad"
	cmdu "github.com/unapu-go/cmd-utils"
)

const ModuleName = "os"

var (
	ModuleInit gad.ModuleInitFunc = func(module *gad.Module, c gad.Call) (err error) {
		spec := module.Spec
		module.Data = gad.Dict{
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
				Module:   spec,
				FuncName: "exec",
				Value:    Exec,
			},
			"exists": &gad.Function{
				Module:   spec,
				FuncName: "exists",
				Value:    Exists,
			},
			"createFile": &gad.Function{
				Module:   spec,
				FuncName: "createFile",
				Value:    CreateFile,
			},
			"openFile": &gad.Function{
				Module:   spec,
				FuncName: "openFile",
				Value:    OpenFile,
			},
			"readFile": &gad.Function{
				Module:   spec,
				FuncName: "readFile",
				Value:    ReadFile,
			},
		}

		return
	}
)
