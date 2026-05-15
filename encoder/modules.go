package encoder

import "github.com/gad-lang/gad"

type ModulesSpec []*gad.ModuleSpec

type GoModules map[string]gad.ModuleInitFunc

func GoModulesFromModulesMap(mm *gad.ModuleMap) GoModules {
	goModules := make(GoModules)

	for name, imp := range mm.Importers() {
		switch v := imp.(type) {
		case *gad.BuiltinInitModule:
			goModules[name] = v.Init
		case *gad.BuiltinModule:
			goModules[name] = v.InitFunc()
		}
	}

	return goModules
}
