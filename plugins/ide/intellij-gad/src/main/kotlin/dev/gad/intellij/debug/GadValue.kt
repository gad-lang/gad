package dev.gad.intellij.debug

import com.intellij.icons.AllIcons
import com.intellij.xdebugger.frame.XValue
import com.intellij.xdebugger.frame.XValueNode
import com.intellij.xdebugger.frame.XValuePlace

/**
 * One entry in the Variables tree — a Gad local from the adapter's `variables`
 * response. Gad values are rendered flat (the adapter does not expose nested
 * variable references), so this is always a leaf.
 */
class GadValue(
    private val name: String,
    private val value: String,
    private val type: String?,
) : XValue() {
    override fun computePresentation(node: XValueNode, place: XValuePlace) {
        node.setPresentation(AllIcons.Debugger.Value, type, value, false)
    }

    /** Used as the watch/tree label. */
    fun displayName(): String = name
}
