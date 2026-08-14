package dev.gad.intellij.lang

import com.intellij.ide.FileIconProvider
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VirtualFile
import javax.swing.Icon

/**
 * Gives `*_test.gad` files the Gad test icon (the Gad logo with a test badge),
 * like GoLand does for `*_test.go`. Other Gad files keep their file-type icon.
 */
class GadFileIconProvider : FileIconProvider {
    override fun getIcon(file: VirtualFile, flags: Int, project: Project?): Icon? =
        if (!file.isDirectory && file.name.endsWith("_test.${GadFile.EXT_GAD}")) {
            GadIcons.GAD_TEST_FILE
        } else {
            null
        }
}
