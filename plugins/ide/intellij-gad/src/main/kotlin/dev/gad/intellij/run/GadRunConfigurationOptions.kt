package dev.gad.intellij.run

import com.intellij.execution.configurations.RunConfigurationOptions

/**
 * Persisted fields of a Gad run configuration — the execution profile: the
 * script, its arguments, the working directory, environment variables and an
 * optional GADPATH override.
 */
class GadRunConfigurationOptions : RunConfigurationOptions() {
    private val scriptPathProp = string("").provideDelegate(this, "scriptPath")
    private val programArgsProp = string("").provideDelegate(this, "programArguments")
    private val workingDirProp = string("").provideDelegate(this, "workingDirectory")
    private val gadPathProp = string("").provideDelegate(this, "gadPath")
    private val passParentProp = property(true).provideDelegate(this, "passParentEnvs")
    private val envJsonProp = string("").provideDelegate(this, "envJson")

    var scriptPath: String
        get() = scriptPathProp.getValue(this).orEmpty()
        set(value) = scriptPathProp.setValue(this, value)

    var programArguments: String
        get() = programArgsProp.getValue(this).orEmpty()
        set(value) = programArgsProp.setValue(this, value)

    var workingDirectory: String
        get() = workingDirProp.getValue(this).orEmpty()
        set(value) = workingDirProp.setValue(this, value)

    /** GADPATH override for this profile (blank → use the application default). */
    var gadPath: String
        get() = gadPathProp.getValue(this).orEmpty()
        set(value) = gadPathProp.setValue(this, value)

    var passParentEnvs: Boolean
        get() = passParentProp.getValue(this)
        set(value) = passParentProp.setValue(this, value)

    /** Environment variables serialized as JSON (`{"K":"V",…}`). */
    var envJson: String
        get() = envJsonProp.getValue(this).orEmpty()
        set(value) = envJsonProp.setValue(this, value)
}
