package dev.gad.intellij.debug

import com.google.gson.JsonObject
import com.intellij.openapi.vfs.LocalFileSystem
import com.intellij.ui.ColoredTextContainer
import com.intellij.ui.SimpleTextAttributes
import com.intellij.xdebugger.XDebuggerUtil
import com.intellij.xdebugger.XSourcePosition
import com.intellij.xdebugger.evaluation.XDebuggerEvaluator
import com.intellij.xdebugger.frame.XCompositeNode
import com.intellij.xdebugger.frame.XStackFrame
import com.intellij.xdebugger.frame.XValueChildrenList

/**
 * A single call-stack frame. Its source position is built from the DAP frame's
 * own file, so selecting a frame from an imported module opens that module's
 * file (cross-file debug navigation).
 */
class GadStackFrame(
    private val process: GadDebugProcess,
    private val frameId: Int,
    private val name: String,
    filePath: String?,
    line: Int,
) : XStackFrame() {

    private val position: XSourcePosition? = run {
        val path = filePath ?: return@run null
        val vFile = LocalFileSystem.getInstance().findFileByPath(path) ?: return@run null
        // DAP lines are 1-based; XSourcePosition lines are 0-based.
        XDebuggerUtil.getInstance().createPosition(vFile, (line - 1).coerceAtLeast(0))
    }

    override fun getSourcePosition(): XSourcePosition? = position

    override fun getEvaluator(): XDebuggerEvaluator = GadEvaluator(process, frameId)

    override fun customizePresentation(component: ColoredTextContainer) {
        component.append(name.ifEmpty { "<anonymous>" }, SimpleTextAttributes.REGULAR_ATTRIBUTES)
        position?.let {
            component.append("  ${it.file.name}:${it.line + 1}", SimpleTextAttributes.GRAYED_ATTRIBUTES)
        }
    }

    override fun computeChildren(node: XCompositeNode) {
        val client = process.client()
        if (client == null) {
            node.addChildren(XValueChildrenList.EMPTY, true)
            return
        }
        val scopesArgs = JsonObject().apply { addProperty("frameId", frameId) }
        client.request("scopes", scopesArgs).whenComplete { scopesBody, err ->
            val ref = scopesBody?.getAsJsonArray("scopes")
                ?.firstOrNull()?.asJsonObject?.get("variablesReference")?.asInt
            if (err != null || ref == null) {
                node.addChildren(XValueChildrenList.EMPTY, true)
                return@whenComplete
            }
            val varsArgs = JsonObject().apply { addProperty("variablesReference", ref) }
            client.request("variables", varsArgs).whenComplete { varsBody, _ ->
                val list = XValueChildrenList()
                varsBody?.getAsJsonArray("variables")?.forEach { v ->
                    val o = v.asJsonObject
                    val nm = o.get("name")?.asString ?: return@forEach
                    val value = o.get("value")?.asString ?: ""
                    val type = o.get("type")?.asString
                    list.add(nm, GadValue(nm, value, type))
                }
                node.addChildren(list, true)
            }
        }
    }
}
