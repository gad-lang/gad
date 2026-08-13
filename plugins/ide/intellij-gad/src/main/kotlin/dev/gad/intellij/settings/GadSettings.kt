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

        // Formatting options — the `gad fmt` toggles. Defaults match `gad fmt`'s own
        // defaults (multi-line formatting on, no backup).
        var fmtNoFormat by property(false)
        var fmtNoArrayItemNewLine by property(false)
        var fmtNoCallParamsNewLine by property(false)
        var fmtNoDeclItemNewLine by property(false)
        var fmtNoDictItemNewLine by property(false)
        var fmtNoKeyValueArrayItemNewLine by property(false)
        var fmtNoParamValuesNewLine by property(false)
        var fmtBackup by property(false)
    }

    var gadPath: String
        get() = state.gadPath.orEmpty()
        set(value) { state.gadPath = value }

    var gadPathEnv: String
        get() = state.gadPathEnv.orEmpty()
        set(value) { state.gadPathEnv = value }

    // Formatting options.
    var fmtNoFormat: Boolean
        get() = state.fmtNoFormat
        set(value) { state.fmtNoFormat = value }
    var fmtNoArrayItemNewLine: Boolean
        get() = state.fmtNoArrayItemNewLine
        set(value) { state.fmtNoArrayItemNewLine = value }
    var fmtNoCallParamsNewLine: Boolean
        get() = state.fmtNoCallParamsNewLine
        set(value) { state.fmtNoCallParamsNewLine = value }
    var fmtNoDeclItemNewLine: Boolean
        get() = state.fmtNoDeclItemNewLine
        set(value) { state.fmtNoDeclItemNewLine = value }
    var fmtNoDictItemNewLine: Boolean
        get() = state.fmtNoDictItemNewLine
        set(value) { state.fmtNoDictItemNewLine = value }
    var fmtNoKeyValueArrayItemNewLine: Boolean
        get() = state.fmtNoKeyValueArrayItemNewLine
        set(value) { state.fmtNoKeyValueArrayItemNewLine = value }
    var fmtNoParamValuesNewLine: Boolean
        get() = state.fmtNoParamValuesNewLine
        set(value) { state.fmtNoParamValuesNewLine = value }
    var fmtBackup: Boolean
        get() = state.fmtBackup
        set(value) { state.fmtBackup = value }

    /** The `gad fmt` flags implied by the current formatting options. */
    fun fmtFlags(): List<String> = buildList {
        if (fmtNoFormat) add("-no-format")
        if (fmtNoArrayItemNewLine) add("-no-array-item-in-new-line")
        if (fmtNoCallParamsNewLine) add("-no-call-params-in-new-line")
        if (fmtNoDeclItemNewLine) add("-no-decl-item-in-new-line")
        if (fmtNoDictItemNewLine) add("-no-dict-item-in-new-line")
        if (fmtNoKeyValueArrayItemNewLine) add("-no-key-value-array-item-in-new-line")
        if (fmtNoParamValuesNewLine) add("-no-parem-values-in-new-line")
        if (fmtBackup) add("-backup")
    }

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
