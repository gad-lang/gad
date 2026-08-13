package dev.gad.intellij.run

import com.intellij.execution.configurations.ConfigurationFactory
import com.intellij.execution.configurations.ConfigurationTypeBase
import com.intellij.execution.configurations.RunConfiguration
import com.intellij.openapi.project.Project
import dev.gad.intellij.lang.GadIcons

class GadRunConfigurationType : ConfigurationTypeBase(
    ID,
    "Gad",
    "Run or debug a Gad script (.gad / .gadt / .gadx)",
    GadIcons.FILE,
) {
    init {
        addFactory(GadConfigurationFactory(this))
    }

    companion object {
        const val ID = "GadRunConfiguration"
    }
}

class GadConfigurationFactory(type: GadRunConfigurationType) : ConfigurationFactory(type) {
    override fun getId(): String = "Gad"

    override fun createTemplateConfiguration(project: Project): RunConfiguration =
        GadRunConfiguration(project, this, "Gad")

    override fun getOptionsClass(): Class<GadRunConfigurationOptions> =
        GadRunConfigurationOptions::class.java
}
