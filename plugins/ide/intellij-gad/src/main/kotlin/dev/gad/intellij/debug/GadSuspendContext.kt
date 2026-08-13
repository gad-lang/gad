package dev.gad.intellij.debug

import com.intellij.icons.AllIcons
import com.intellij.xdebugger.frame.XExecutionStack
import com.intellij.xdebugger.frame.XStackFrame
import com.intellij.xdebugger.frame.XSuspendContext

/** The single Gad thread's suspended state: its call stack. */
class GadSuspendContext(frames: List<GadStackFrame>) : XSuspendContext() {
    private val stack = GadExecutionStack(frames)
    override fun getActiveExecutionStack(): XExecutionStack = stack
    override fun getExecutionStacks(): Array<XExecutionStack> = arrayOf(stack)
}

class GadExecutionStack(private val frames: List<GadStackFrame>) :
    XExecutionStack("main", AllIcons.Debugger.ThreadCurrent) {

    override fun getTopFrame(): XStackFrame? = frames.firstOrNull()

    override fun computeStackFrames(firstFrameIndex: Int, container: XStackFrameContainer) {
        if (firstFrameIndex < frames.size) {
            container.addStackFrames(frames.subList(firstFrameIndex, frames.size), true)
        } else {
            container.addStackFrames(emptyList(), true)
        }
    }
}
