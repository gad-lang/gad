package dev.gad.intellij.doc

import com.intellij.openapi.actionSystem.ActionUpdateThread
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.actionSystem.CommonDataKeys
import com.intellij.openapi.wm.ToolWindowManager
import dev.gad.intellij.lang.GadFile

/**
 * Context-menu "Gad Doc": opens (and focuses) the Gad Doc side panel, which shows
 * the live-rendered documentation of the active Gad file. Enabled for
 * `.gad`/`.gadt`/`.gadx` files.
 */
class GadDocAction : AnAction("Gad Doc", "Show the documentation of this Gad file in the Gad Doc panel", null) {

    override fun getActionUpdateThread(): ActionUpdateThread = ActionUpdateThread.BGT

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabledAndVisible = GadFile.isGadFile(e.getData(CommonDataKeys.VIRTUAL_FILE))
    }

    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        ToolWindowManager.getInstance(project).getToolWindow("Gad Doc")?.activate(null, true)
    }
}
