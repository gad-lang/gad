package dev.gad.intellij.debug

import com.google.gson.JsonObject
import com.intellij.xdebugger.XSourcePosition
import com.intellij.xdebugger.evaluation.XDebuggerEvaluator
import com.intellij.xdebugger.frame.XValue

/** Evaluates watch / Debug Console / hover expressions in a paused frame. */
class GadEvaluator(
    private val process: GadDebugProcess,
    private val frameId: Int,
) : XDebuggerEvaluator() {

    override fun evaluate(
        expression: String,
        callback: XEvaluationCallback,
        expressionPosition: XSourcePosition?,
    ) {
        val args = JsonObject().apply {
            addProperty("expression", expression)
            addProperty("frameId", frameId)
            addProperty("context", "watch")
        }
        process.client()?.request("evaluate", args)?.whenComplete { body: JsonObject?, err ->
            if (err != null) {
                callback.errorOccurred(err.message ?: "evaluation failed")
                return@whenComplete
            }
            val result = body?.get("result")?.asString ?: "nil"
            val type = body?.get("type")?.asString
            callback.evaluated(GadValue(expression, result, type))
        } ?: callback.errorOccurred("not connected")
    }
}
