package dev.gad.intellij.lang

import com.intellij.openapi.vfs.VirtualFile

/**
 * The Gad language family recognized by the plugin. The three dialects have real
 * file types ([GadFileType], [GadTemplateFileType], [GadxFileType]); this helper
 * classifies a file by extension for run/debug and breakpoints.
 */
object GadFile {
    /** Plain Gad scripts and modules. */
    const val EXT_GAD = "gad"

    /** Mixed-mode templates (`{% … %}` / `{%= … %}`). */
    const val EXT_GADT = "gadt"

    /** Indentation/pug-style templates lowered to Gad. */
    const val EXT_GADX = "gadx"

    val EXTENSIONS = setOf(EXT_GAD, EXT_GADT, EXT_GADX)

    fun isGadFile(file: VirtualFile?): Boolean =
        file != null && !file.isDirectory && file.extension?.lowercase() in EXTENSIONS
}
