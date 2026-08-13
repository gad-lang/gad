package dev.gad.intellij.run

import com.intellij.execution.configuration.EnvironmentVariablesData
import com.intellij.execution.configuration.EnvironmentVariablesTextFieldWithBrowseButton
import com.intellij.openapi.fileChooser.FileChooserDescriptorFactory
import com.intellij.openapi.options.SettingsEditor
import com.intellij.openapi.ui.TextFieldWithBrowseButton
import com.intellij.ui.RawCommandLineEditor
import com.intellij.ui.components.JBTextField
import com.intellij.util.ui.FormBuilder
import javax.swing.JComponent
import javax.swing.JPanel

/** The run configuration form: script, arguments, working dir, GADPATH, env. */
class GadRunConfigurationEditor : SettingsEditor<GadRunConfiguration>() {

    private val scriptPath = TextFieldWithBrowseButton().apply {
        addBrowseFolderListener(
            "Gad Script",
            "Select the .gad / .gadt / .gadx file to run",
            null,
            FileChooserDescriptorFactory.createSingleFileDescriptor(),
        )
    }
    private val programArgs = RawCommandLineEditor()
    private val workingDir = TextFieldWithBrowseButton().apply {
        addBrowseFolderListener(
            "Working Directory",
            "Directory the script runs in (imports resolve against it)",
            null,
            FileChooserDescriptorFactory.createSingleFolderDescriptor(),
        )
    }
    private val gadPath = JBTextField()
    private val env = EnvironmentVariablesTextFieldWithBrowseButton()

    private val panel: JPanel = FormBuilder.createFormBuilder()
        .addLabeledComponent("Script:", scriptPath)
        .addLabeledComponent("Program arguments:", programArgs)
        .addLabeledComponent("Working directory:", workingDir)
        .addLabeledComponent("GADPATH:", gadPath)
        .addLabeledComponent("Environment variables:", env)
        .panel

    override fun resetEditorFrom(config: GadRunConfiguration) {
        val o = config.options
        scriptPath.text = o.scriptPath
        programArgs.text = o.programArguments
        workingDir.text = o.workingDirectory
        gadPath.text = o.gadPath
        env.data = EnvironmentVariablesData.create(GadProfile.decodeEnv(o.envJson), o.passParentEnvs)
    }

    override fun applyEditorTo(config: GadRunConfiguration) {
        val o = config.options
        o.scriptPath = scriptPath.text.trim()
        o.programArguments = programArgs.text.trim()
        o.workingDirectory = workingDir.text.trim()
        o.gadPath = gadPath.text.trim()
        o.envJson = GadProfile.encodeEnv(env.data.envs)
        o.passParentEnvs = env.data.isPassParentEnvs
    }

    override fun createEditor(): JComponent = panel
}
