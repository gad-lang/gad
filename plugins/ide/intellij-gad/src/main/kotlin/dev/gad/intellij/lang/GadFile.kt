package dev.gad.intellij.lang

import com.intellij.openapi.vfs.VirtualFile

/**
 * The Gad language family recognized by the plugin. Highlighting is provided by
 * a TextMate bundle (see the highlight package); file identity for run/debug is
 * keyed off the extension, so no custom [com.intellij.openapi.fileTypes.FileType]
 * is registered (which would shadow the TextMate grammar).
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
