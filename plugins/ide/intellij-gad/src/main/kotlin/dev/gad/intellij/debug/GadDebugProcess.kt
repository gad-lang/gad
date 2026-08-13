package dev.gad.intellij.debug

import com.google.gson.JsonArray
import com.google.gson.JsonObject
import com.intellij.execution.ExecutionException
import com.intellij.execution.configurations.GeneralCommandLine
import com.intellij.execution.process.ProcessHandler
import com.intellij.openapi.diagnostic.thisLogger
import com.intellij.xdebugger.XDebugProcess
import com.intellij.xdebugger.XDebugSession
import com.intellij.xdebugger.breakpoints.XBreakpointHandler
import com.intellij.xdebugger.breakpoints.XBreakpointProperties
import com.intellij.xdebugger.breakpoints.XLineBreakpoint
import com.intellij.xdebugger.evaluation.XDebuggerEditorsProvider
import com.intellij.xdebugger.frame.XSuspendContext
import dev.gad.intellij.debug.dap.DapClient
import dev.gad.intellij.run.GadProfile
import java.io.File
import java.util.concurrent.ConcurrentHashMap

/**
 * Bridges the Gad Debug Adapter (`gad debug --dap`) to the IntelliJ XDebugger:
 * launch, breakpoints (with conditions), stepping, call stack, variables and
 * expression evaluation, plus navigation into imported files via per-frame
 * source paths reported by the adapter.
 */
class GadDebugProcess(
    session: XDebugSession,
    private val profile: GadProfile,
) : XDebugProcess(session), DapClient.Listener {

    private val editorsProvider = GadDebuggerEditorsProvider()
    private val process: Process
    private val processHandler: GadDapProcessHandler
    private val client: DapClient

    // Registered breakpoints, grouped by source file path (DAP sets breakpoints
    // per source).
    private val breakpoints = ConcurrentHashMap<String, MutableList<XLineBreakpoint<*>>>()

    @Volatile private var launched = false

    init {
        val cmd = GeneralCommandLine(profile.executable, "debug", "--dap")
        if (profile.workingDirectory.isNotEmpty()) cmd.workDirectory = File(profile.workingDirectory)
        cmd.withEnvironment(profile.environment)
        process = try {
            cmd.createProcess()
        } catch (e: Exception) {
            throw ExecutionException("Failed to start the Gad debug adapter (${profile.executable})", e)
        }
        processHandler = GadDapProcessHandler(process)
        client = DapClient(process.inputStream, process.outputStream, this)
    }

    override fun getEditorsProvider(): XDebuggerEditorsProvider = editorsProvider

    override fun doGetProcessHandler(): ProcessHandler = processHandler

    override fun getBreakpointHandlers(): Array<XBreakpointHandler<*>> = arrayOf(breakpointHandler)

    fun client(): DapClient = client

    override fun sessionInitialized() {
        client.start()
        client.request("initialize", JsonObject().apply {
            addProperty("clientID", "intellij-gad")
            addProperty("adapterID", "gad")
            addProperty("linesStartAt1", true)
            addProperty("columnsStartAt1", true)
            addProperty("pathFormat", "path")
        })
    }

    // --- DAP events -------------------------------------------------------

    override fun onEvent(event: String, body: JsonObject?) {
        when (event) {
            "initialized" -> onInitialized()
            "stopped" -> onStopped(body)
            "output" -> onOutput(body)
            "terminated", "exited" -> session.stop()
        }
    }

    override fun onClosed() {
        session.stop()
    }

    private fun onInitialized() {
        // Push all breakpoints, then finish configuration and launch.
        breakpoints.keys.forEach { sendBreakpoints(it) }
        client.request("configurationDone")
        launch()
    }

    private fun launch() {
        if (launched) return
        launched = true
        val args = JsonObject().apply {
            addProperty("program", profile.scriptPath)
            addProperty("stopOnEntry", false)
            if (profile.workingDirectory.isNotEmpty()) addProperty("cwd", profile.workingDirectory)
            if (profile.args.isNotEmpty()) {
                add("args", JsonArray().apply { profile.args.forEach { add(it) } })
            }
            if (profile.environment.isNotEmpty()) {
                add("env", JsonObject().apply { profile.environment.forEach { (k, v) -> addProperty(k, v) } })
            }
        }
        client.request("launch", args).exceptionally { err ->
            processHandler.output("launch failed: ${err.message}\n", true)
            session.stop()
            null
        }
    }

    private fun onOutput(body: JsonObject?) {
        val text = body?.get("output")?.asString ?: return
        val category = body.get("category")?.asString
        processHandler.output(text, category == "stderr")
    }

    private fun onStopped(body: JsonObject?) {
        val threadId = body?.get("threadId")?.asInt ?: 1
        val reason = body?.get("reason")?.asString
        val stArgs = JsonObject().apply { addProperty("threadId", threadId) }
        client.request("stackTrace", stArgs).whenComplete { st, err ->
            if (err != null) {
                thisLogger().debug("stackTrace failed", err)
                return@whenComplete
            }
            val frames = st?.getAsJsonArray("stackFrames")?.mapNotNull { toFrame(it.asJsonObject) } ?: emptyList()
            val context = GadSuspendContext(frames)
            val top = frames.firstOrNull()
            val bp = if (reason == "breakpoint") findBreakpoint(top) else null
            if (bp != null) {
                session.breakpointReached(bp, null, context)
            } else {
                session.positionReached(context)
            }
        }
    }

    private fun toFrame(o: JsonObject): GadStackFrame {
        val id = o.get("id")?.asInt ?: 0
        val name = o.get("name")?.asString ?: ""
        val source = o.getAsJsonObject("source")
        val path = source?.get("path")?.asString
        val line = o.get("line")?.asInt ?: 0
        return GadStackFrame(this, id, name, path, line)
    }

    private fun findBreakpoint(frame: GadStackFrame?): XLineBreakpoint<*>? {
        val pos = frame?.sourcePosition ?: return null
        val file = pos.file.path
        val line = pos.line // 0-based
        return breakpoints[file]?.firstOrNull { it.line == line }
    }

    // --- Stepping / control ----------------------------------------------

    override fun resume(context: XSuspendContext?) {
        client.request("continue", JsonObject().apply { addProperty("threadId", 1) })
    }

    override fun startStepOver(context: XSuspendContext?) {
        client.request("next", JsonObject().apply { addProperty("threadId", 1) })
    }

    override fun startStepInto(context: XSuspendContext?) {
        client.request("stepIn", JsonObject().apply { addProperty("threadId", 1) })
    }

    override fun startStepOut(context: XSuspendContext?) {
        client.request("stepOut", JsonObject().apply { addProperty("threadId", 1) })
    }

    override fun startPausing() {
        client.request("pause", JsonObject().apply { addProperty("threadId", 1) })
    }

    override fun stop() {
        try {
            client.request("disconnect", JsonObject().apply { addProperty("terminateDebuggee", true) })
        } catch (_: Exception) {
        } finally {
            client.close()
            if (process.isAlive) process.destroy()
        }
    }

    // --- Breakpoints ------------------------------------------------------

    private val breakpointHandler =
        object : XBreakpointHandler<XLineBreakpoint<XBreakpointProperties<*>>>(GadBreakpointType::class.java) {
            override fun registerBreakpoint(breakpoint: XLineBreakpoint<XBreakpointProperties<*>>) {
                val file = breakpoint.sourcePosition?.file?.path ?: return
                breakpoints.getOrPut(file) { mutableListOf() }.add(breakpoint)
                if (launched) sendBreakpoints(file)
            }

            override fun unregisterBreakpoint(
                breakpoint: XLineBreakpoint<XBreakpointProperties<*>>,
                temporary: Boolean,
            ) {
                val file = breakpoint.sourcePosition?.file?.path ?: return
                breakpoints[file]?.remove(breakpoint)
                if (launched) sendBreakpoints(file)
            }
        }

    private fun sendBreakpoints(file: String) {
        val list = breakpoints[file].orEmpty()
        val args = JsonObject().apply {
            add("source", JsonObject().apply {
                addProperty("path", file)
                addProperty("name", File(file).name)
            })
            add("breakpoints", JsonArray().apply {
                list.forEach { bp ->
                    add(JsonObject().apply {
                        addProperty("line", bp.line + 1) // DAP is 1-based
                        bp.conditionExpression?.expression?.let { addProperty("condition", it) }
                    })
                }
            })
        }
        client.request("setBreakpoints", args)
    }
}
