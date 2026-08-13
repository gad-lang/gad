package dev.gad.intellij.debug

import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VirtualFile
import com.intellij.xdebugger.breakpoints.XBreakpointProperties
import com.intellij.xdebugger.breakpoints.XLineBreakpointType
import dev.gad.intellij.lang.GadFile

/**
 * Line breakpoints for Gad files. Conditions are supported by the platform
 * (`getSupportedSubtypes`/condition editor) and forwarded to the adapter as the
 * DAP breakpoint `condition`.
 */
class GadBreakpointType :
    XLineBreakpointType<XBreakpointProperties<*>>(ID, "Gad Breakpoints") {

    override fun createBreakpointProperties(file: VirtualFile, line: Int): XBreakpointProperties<*>? = null

    override fun canPutAt(file: VirtualFile, line: Int, project: Project): Boolean =
        GadFile.isGadFile(file)

    override fun isSuspendThreadSupported(): Boolean = false

    companion object {
        const val ID = "gad-line-breakpoint"
    }
}
