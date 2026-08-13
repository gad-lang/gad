package dev.gad.intellij.debug

import com.intellij.execution.process.ProcessHandler
import com.intellij.execution.process.ProcessOutputTypes
import java.io.OutputStream

/**
 * Represents the `gad debug --dap` adapter process for the debug tool window.
 * The adapter's stdio is owned by the DAP client, so this handler does not read
 * the streams; program output arrives via DAP `output` events and is pushed here
 * with [output]. It waits for the process to exit to report termination.
 */
class GadDapProcessHandler(private val process: Process) : ProcessHandler() {

    override fun startNotify() {
        super.startNotify()
        Thread({
            val code = try { process.waitFor() } catch (_: InterruptedException) { -1 }
            notifyProcessTerminated(code)
        }, "gad-dap-waiter").apply { isDaemon = true; start() }
    }

    fun output(text: String, isStderr: Boolean) {
        notifyTextAvailable(text, if (isStderr) ProcessOutputTypes.STDERR else ProcessOutputTypes.STDOUT)
    }

    override fun destroyProcessImpl() {
        process.destroy()
    }

    override fun detachProcessImpl() {
        notifyProcessDetached()
    }

    override fun detachIsDefault(): Boolean = false

    override fun getProcessInput(): OutputStream? = null
}
