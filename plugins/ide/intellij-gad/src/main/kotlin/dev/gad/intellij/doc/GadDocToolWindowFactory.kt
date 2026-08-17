package dev.gad.intellij.doc

import com.intellij.openapi.project.DumbAware
import com.intellij.openapi.project.Project
import com.intellij.openapi.wm.ToolWindow
import com.intellij.openapi.wm.ToolWindowFactory

/**
 * Registers the "Gad Doc" side panel: a live-rendered documentation preview of
 * the active Gad file (see [GadDocPanel]).
 */
class GadDocToolWindowFactory : ToolWindowFactory, DumbAware {
    override fun createToolWindowContent(project: Project, toolWindow: ToolWindow) {
        val panel = GadDocPanel(project, toolWindow.disposable)
        val content = toolWindow.contentManager.factory.createContent(panel.component, "", false)
        content.isCloseable = false
        toolWindow.contentManager.addContent(content)
    }
}
