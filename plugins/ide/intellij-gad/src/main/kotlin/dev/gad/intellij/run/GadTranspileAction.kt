package dev.gad.intellij.run

import com.intellij.execution.RunContentExecutor
import com.intellij.execution.configurations.GeneralCommandLine
import com.intellij.execution.process.OSProcessHandler
import com.intellij.execution.process.ProcessAdapter
import com.intellij.execution.process.ProcessEvent
import com.intellij.openapi.actionSystem.ActionUpdateThread
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.actionSystem.CommonDataKeys
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.fileEditor.FileDocumentManager
import com.intellij.openapi.vfs.VfsUtil
import com.intellij.openapi.vfs.VirtualFile
import dev.gad.intellij.lang.GadFile
import dev.gad.intellij.settings.GadExecutable
import dev.gad.intellij.settings.GadSettings
import java.io.File

/**
 * Context-menu "Gad Transpile" for `.gadt` / `.gadx` files and directories: runs
 * `gad transpile <path>`, lowering templates to `.gad` files of the same name (a
 * directory is transpiled recursively). Output is shown in a console and the VFS
 * is refreshed so the generated files appear.
 */
class GadTranspileAction : AnAction("Gad Transpile", "Transpile Gad templates (.gadt / .gadx) to .gad", null) {

    override fun getActionUpdateThread(): ActionUpdateThread = ActionUpdateThread.BGT

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabledAndVisible = target(e) != null
    }

    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val file = target(e) ?: return
        FileDocumentManager.getInstance().saveAllDocuments()

        val exe = GadExecutable.resolveForActionOrNotify(project, GadSettings.getInstance()) ?: return
        val cmd = GeneralCommandLine(exe, "transpile", file.path)
        file.parent?.path?.let { cmd.setWorkDirectory(it) }
        cmd.charset = Charsets.UTF_8

        val handler = OSProcessHandler(cmd)
        handler.addProcessListener(object : ProcessAdapter() {
            override fun processTerminated(event: ProcessEvent) {
                // Make the freshly written .gad files visible in the IDE.
                ApplicationManager.getApplication().invokeLater {
                    VfsUtil.markDirtyAndRefresh(true, true, true, file.parent ?: file)
                }
            }
        })

        RunContentExecutor(project, handler)
            .withTitle("Gad Transpile")
            .withActivateToolWindow(true)
            .run()
    }

    /**
     * The action target: a `.gadt` / `.gadx` file, or a directory that contains
     * at least one template (checked shallowly to keep `update` cheap).
     */
    private fun target(e: AnActionEvent): VirtualFile? {
        val file = e.getData(CommonDataKeys.VIRTUAL_FILE) ?: return null
        return when {
            file.isDirectory -> if (dirHasTemplate(File(file.path))) file else null
            isTemplate(file) -> file
            else -> null
        }
    }

    private fun isTemplate(file: VirtualFile): Boolean {
        val ext = file.extension?.lowercase()
        return ext == GadFile.EXT_GADT || ext == GadFile.EXT_GADX
    }

    private fun dirHasTemplate(dir: File): Boolean {
        val children = dir.listFiles() ?: return true // unknown: offer the action
        return children.any {
            if (it.isDirectory) {
                !it.name.startsWith(".")
            } else {
                val n = it.name
                n.endsWith(".${GadFile.EXT_GADT}") || n.endsWith(".${GadFile.EXT_GADX}")
            }
        }
    }
}
