package dev.gad.intellij.lang

import com.intellij.ide.FileIconProvider
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VirtualFile
import javax.swing.Icon

/**
 * Gives `.gad` / `.gadt` / `.gadx` files the Gad icon without registering a
 * custom file type (which would replace the TextMate highlighting).
 */
class GadFileIconProvider : FileIconProvider {
    override fun getIcon(file: VirtualFile, flags: Int, project: Project?): Icon? =
        if (GadFile.isGadFile(file)) GadIcons.FILE else null
}
