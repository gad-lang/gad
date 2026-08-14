package dev.gad.intellij.run

import com.intellij.execution.ProgramRunnerUtil
import com.intellij.execution.RunManager
import com.intellij.execution.executors.DefaultRunExecutor
import com.intellij.openapi.actionSystem.ActionUpdateThread
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.actionSystem.CommonDataKeys
import com.intellij.openapi.fileEditor.FileDocumentManager
import com.intellij.openapi.vfs.VirtualFile
import dev.gad.intellij.lang.GadFile
import dev.gad.intellij.lang.GadIcons

/**
 * Context-menu "Run" for a Gad file (.gad / .gadt / .gadx): creates a temporary
 * Gad run configuration for the file and runs it directly.
 */
class GadRunFileAction : AnAction("Run", "Run this Gad file", GadIcons.LOGO) {

    override fun getActionUpdateThread(): ActionUpdateThread = ActionUpdateThread.BGT

    override fun update(e: AnActionEvent) {
        val file = targetFile(e)
        e.presentation.isEnabledAndVisible = file != null
        if (file != null) {
            e.presentation.text = "Run '${file.name}'"
        }
    }

    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val file = targetFile(e) ?: return
        FileDocumentManager.getInstance().saveAllDocuments()

        val runManager = RunManager.getInstance(project)
        val factory = GadRunConfigurationType().configurationFactories[0]
        val settings = runManager.createConfiguration(file.name, factory)
        (settings.configuration as GadRunConfiguration).scriptPath = file.path
        settings.isTemporary = true
        runManager.setTemporaryConfiguration(settings)

        ProgramRunnerUtil.executeConfiguration(settings, DefaultRunExecutor.getRunExecutorInstance())
    }

    private fun targetFile(e: AnActionEvent): VirtualFile? {
        val file = e.getData(CommonDataKeys.VIRTUAL_FILE) ?: return null
        return if (GadFile.isGadFile(file)) file else null
    }
}
