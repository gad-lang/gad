package dev.gad.intellij.run

import com.intellij.execution.Executor
import com.intellij.execution.configurations.ConfigurationFactory
import com.intellij.execution.configurations.RunConfiguration
import com.intellij.execution.configurations.RunConfigurationBase
import com.intellij.execution.configurations.RuntimeConfigurationError
import com.intellij.execution.runners.ExecutionEnvironment
import com.intellij.openapi.options.SettingsEditor
import com.intellij.openapi.project.Project
import java.io.File

class GadRunConfiguration(
    project: Project,
    factory: ConfigurationFactory,
    name: String,
) : RunConfigurationBase<GadRunConfigurationOptions>(project, factory, name) {

    public override fun getOptions(): GadRunConfigurationOptions =
        super.getOptions() as GadRunConfigurationOptions

    var scriptPath: String
        get() = options.scriptPath
        set(value) { options.scriptPath = value }

    override fun getConfigurationEditor(): SettingsEditor<out RunConfiguration> =
        GadRunConfigurationEditor()

    override fun checkConfiguration() {
        val script = options.scriptPath.trim()
        if (script.isEmpty()) throw RuntimeConfigurationError("No Gad script specified")
        if (!File(script).isFile) throw RuntimeConfigurationError("Script not found: $script")
    }

    override fun getState(executor: Executor, environment: ExecutionEnvironment) =
        GadCommandLineState(this, environment)

    fun profile(): GadProfile = GadProfile.from(options)
}
