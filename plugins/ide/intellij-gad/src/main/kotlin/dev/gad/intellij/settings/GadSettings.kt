package dev.gad.intellij.settings

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.components.BaseState
import com.intellij.openapi.components.SimplePersistentStateComponent
import com.intellij.openapi.components.State
import com.intellij.openapi.components.Storage
import com.intellij.openapi.util.SystemInfo
import java.io.File

/** Application-level Gad settings (the `gad` binary location, default GADPATH). */
@State(name = "GadSettings", storages = [Storage("gad.xml")])
class GadSettings : SimplePersistentStateComponent<GadSettings.MyState>(MyState()) {

    class MyState : BaseState() {
        /** Path to the `gad` executable; blank means "resolve from PATH". */
        var gadPath by string("")

        /** Default module search path (GADPATH), OS path-list separated. */
        var gadPathEnv by string("")
    }

    var gadPath: String
        get() = state.gadPath.orEmpty()
        set(value) { state.gadPath = value }

    var gadPathEnv: String
        get() = state.gadPathEnv.orEmpty()
        set(value) { state.gadPathEnv = value }

    /** The resolved `gad` executable to launch (falls back to a bare `gad`). */
    fun resolveExecutable(): String {
        val configured = gadPath.trim()
        if (configured.isNotEmpty()) return configured
        return findOnPath() ?: DEFAULT_EXE
    }

    private fun findOnPath(): String? {
        val path = System.getenv("PATH") ?: return null
        val sep = File.pathSeparatorChar
        return path.split(sep).asSequence()
            .map { File(it, DEFAULT_EXE) }
            .firstOrNull { it.canExecute() }
            ?.absolutePath
    }

    companion object {
        val DEFAULT_EXE: String = if (SystemInfo.isWindows) "gad.exe" else "gad"

        @JvmStatic
        fun getInstance(): GadSettings =
            ApplicationManager.getApplication().getService(GadSettings::class.java)
    }
}
