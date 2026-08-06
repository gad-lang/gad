// Package gadconfig centralizes where Gad tooling keeps its per-workspace
// configuration and templates. Everything is resolved around a WORK_DIR (the
// project root, default the current directory or $GAD_WORK_DIR) and a config
// directory GAD_CONFIG_DIR (default WORK_DIR/.gad, or $GAD_CONFIG_DIR):
//
//	WORK_DIR/
//	  .gad/                 (GAD_CONFIG_DIR)
//	    gad.yaml            project config (fmt/doc/template/env sections)
//	    ide.yaml            IDE layout/editor state
//	    doc-templates/
//	      html.gadx         `gad doc` HTML template
//	      md.gadx           `gad doc` Markdown template
package gadconfig

import (
	"os"
	"path/filepath"
)

// Environment variables and fixed names.
const (
	// EnvWorkDir overrides the workspace root (default: current directory).
	EnvWorkDir = "GAD_WORK_DIR"
	// EnvConfigDir overrides the config directory (default: WORK_DIR/.gad).
	EnvConfigDir = "GAD_CONFIG_DIR"

	// DirName is the config directory placed under the workspace root.
	DirName = ".gad"
	// ConfigFileName is the project config inside the config directory.
	ConfigFileName = "gad.yaml"
	// IDEConfigFileName holds the IDE layout/editor state.
	IDEConfigFileName = "ide.yaml"
	// DocTemplatesDirName holds the `gad doc` output templates.
	DocTemplatesDirName = "doc-templates"
	// DocHTMLTemplateName renders `gad doc` output as HTML.
	DocHTMLTemplateName = "html.gadx"
	// DocMDTemplateName renders `gad doc` output as Markdown.
	DocMDTemplateName = "md.gadx"
	// DocMDIndexTemplateName renders a per-directory Markdown index (README.md).
	DocMDIndexTemplateName = "md-index.gadx"
	// DocHTMLIndexTemplateName renders a per-directory HTML index (index.html).
	DocHTMLIndexTemplateName = "html-index.gadx"
)

// WorkDir resolves the workspace root. An explicit non-empty dir wins; otherwise
// $GAD_WORK_DIR, then the process working directory (".") as a last resort.
func WorkDir(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if d := os.Getenv(EnvWorkDir); d != "" {
		return d
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

// Dir returns the config directory for a workspace: $GAD_CONFIG_DIR when set,
// else workDir/.gad.
func Dir(workDir string) string {
	if d := os.Getenv(EnvConfigDir); d != "" {
		return d
	}
	return filepath.Join(workDir, DirName)
}

// File returns the project config file path (config-dir/gad.yaml).
func File(workDir string) string { return filepath.Join(Dir(workDir), ConfigFileName) }

// IDEFile returns the IDE config file path (config-dir/ide.yaml).
func IDEFile(workDir string) string { return filepath.Join(Dir(workDir), IDEConfigFileName) }

// DocTemplatesDir returns the doc-templates directory (config-dir/doc-templates).
func DocTemplatesDir(workDir string) string {
	return filepath.Join(Dir(workDir), DocTemplatesDirName)
}

// DocHTMLTemplate returns the HTML doc-template path (doc-templates/html.gadx).
func DocHTMLTemplate(workDir string) string {
	return filepath.Join(DocTemplatesDir(workDir), DocHTMLTemplateName)
}

// DocMDTemplate returns the Markdown doc-template path (doc-templates/md.gadx).
func DocMDTemplate(workDir string) string {
	return filepath.Join(DocTemplatesDir(workDir), DocMDTemplateName)
}

// DocMDIndexTemplate returns the Markdown index template path
// (doc-templates/md-index.gadx).
func DocMDIndexTemplate(workDir string) string {
	return filepath.Join(DocTemplatesDir(workDir), DocMDIndexTemplateName)
}

// DocHTMLIndexTemplate returns the HTML index template path
// (doc-templates/html-index.gadx).
func DocHTMLIndexTemplate(workDir string) string {
	return filepath.Join(DocTemplatesDir(workDir), DocHTMLIndexTemplateName)
}
