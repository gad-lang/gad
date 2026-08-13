package dev.gad.intellij.run

import com.intellij.execution.actions.ConfigurationContext
import com.intellij.execution.actions.LazyRunConfigurationProducer
import com.intellij.execution.configurations.ConfigurationFactory
import com.intellij.execution.configurations.ConfigurationTypeUtil
import com.intellij.openapi.util.Ref
import com.intellij.psi.PsiElement
import dev.gad.intellij.lang.GadFile

/** Creates a Gad run configuration from a right-click / gutter on a Gad file. */
class GadRunConfigurationProducer : LazyRunConfigurationProducer<GadRunConfiguration>() {

    override fun getConfigurationFactory(): ConfigurationFactory =
        ConfigurationTypeUtil.findConfigurationType(GadRunConfigurationType::class.java)
            .configurationFactories[0]

    override fun setupConfigurationFromContext(
        configuration: GadRunConfiguration,
        context: ConfigurationContext,
        sourceElement: Ref<PsiElement>,
    ): Boolean {
        val file = context.location?.virtualFile ?: return false
        if (!GadFile.isGadFile(file)) return false
        configuration.scriptPath = file.path
        configuration.name = file.name
        return true
    }

    override fun isConfigurationFromContext(
        configuration: GadRunConfiguration,
        context: ConfigurationContext,
    ): Boolean {
        val file = context.location?.virtualFile ?: return false
        return GadFile.isGadFile(file) && configuration.scriptPath == file.path
    }
}
