package dev.gad.intellij.settings

import com.intellij.execution.configurations.GeneralCommandLine
import com.intellij.execution.process.CapturingProcessHandler
import com.intellij.notification.NotificationGroupManager
import com.intellij.notification.NotificationType
import com.intellij.openapi.project.Project
import java.io.File

/**
 * Result of probing the configured `gad` binary: the resolved absolute path, its
 * `gad version` line, and a human-readable error when either step failed.
 */
data class GadProbe(
    val path: String?,
    val version: String?,
    val error: String?,
) {
    val ok: Boolean get() = path != null && version != null
}

/**
 * Resolution and probing of the `gad` executable, shared by the settings UI and
 * the Run / Transpile actions so a missing or broken binary fails with a clear
 * message instead of a silent no-op.
 */
object GadExecutable {

    /**
     * Resolve `configured` (the settings value) to an existing executable file:
     * the configured path when it points at one, else the first `gad` found on
     * PATH. Returns null when neither resolves — the caller reports the error.
     */
    fun resolvePath(configured: String): String? {
        val trimmed = configured.trim()
        if (trimmed.isNotEmpty()) {
            val f = File(trimmed)
            return if (f.canExecute()) f.absolutePath else null
        }
        val path = System.getenv("PATH") ?: return null
        val sep = File.pathSeparatorChar
        return path.split(sep).asSequence()
            .map { File(it, GadSettings.DEFAULT_EXE) }
            .firstOrNull { it.canExecute() }
            ?.absolutePath
    }

    /** Run `<gad> version` and capture the version line, or an error message. */
    fun probe(configured: String): GadProbe {
        val trimmed = configured.trim()
        val path = resolvePath(trimmed)
            ?: return GadProbe(
                path = null,
                version = null,
                error = if (trimmed.isNotEmpty()) {
                    "gad executable not found or not executable at '$trimmed'"
                } else {
                    "gad executable not found on PATH — set its location in " +
                        "Settings ▸ Build, Execution, Deployment ▸ Gad ▸ Executable"
                },
            )
        return try {
            val cmd = GeneralCommandLine(path, "version").withCharset(Charsets.UTF_8)
            val output = CapturingProcessHandler(cmd).runProcess(PROBE_TIMEOUT_MS)
            when {
                output.isTimeout ->
                    GadProbe(path, null, "gad version timed out")
                output.exitCode != 0 ->
                    GadProbe(path, null, "gad version exited with ${output.exitCode}: ${output.stderr.trim()}")
                else ->
                    GadProbe(path, output.stdout.lineSequence().firstOrNull { it.isNotBlank() }?.trim(), null)
            }
        } catch (e: Exception) {
            GadProbe(path, null, "failed to run gad: ${e.message}")
        }
    }

    /**
     * Resolve the executable for an action, posting an error notification and
     * returning null when it cannot be found — so Run / Transpile give feedback
     * instead of failing silently.
     */
    fun resolveForActionOrNotify(project: Project?, settings: GadSettings): String? {
        val path = resolvePath(settings.gadPath)
        if (path == null) {
            notifyError(
                project,
                "Gad executable not found",
                "Could not find the <code>gad</code> binary. Set its location in " +
                    "Settings ▸ Build, Execution, Deployment ▸ Gad ▸ Executable, " +
                    "or put <code>gad</code> on your PATH.",
            )
        }
        return path
    }

    fun notifyError(project: Project?, title: String, content: String) {
        NotificationGroupManager.getInstance()
            .getNotificationGroup("Gad")
            .createNotification(title, content, NotificationType.ERROR)
            .notify(project)
    }

    private const val PROBE_TIMEOUT_MS = 5000
}
