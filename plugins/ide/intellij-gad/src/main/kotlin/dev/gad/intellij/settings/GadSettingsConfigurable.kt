package dev.gad.intellij.settings

import com.intellij.openapi.fileChooser.FileChooserDescriptorFactory
import com.intellij.openapi.options.BoundConfigurable
import com.intellij.openapi.ui.DialogPanel
import com.intellij.openapi.ui.TextFieldWithBrowseButton
import com.intellij.ui.dsl.builder.bindText
import com.intellij.ui.dsl.builder.panel

/** The Settings ▸ Tools ▸ Gad page: `gad` executable path and default GADPATH. */
class GadSettingsConfigurable : BoundConfigurable("Gad") {

    override fun createPanel(): DialogPanel {
        val settings = GadSettings.getInstance()
        return panel {
            row("Gad executable:") {
                val field = TextFieldWithBrowseButton().apply {
                    addBrowseFolderListener(
                        "Gad Executable",
                        "Select the gad binary (leave blank to resolve from PATH)",
                        null,
                        FileChooserDescriptorFactory.createSingleFileDescriptor(),
                    )
                }
                cell(field)
                    .comment("Leave blank to resolve <code>gad</code> from PATH.")
                    .bindText(settings::gadPath)
            }
            row("GADPATH:") {
                textField()
                    .comment("Module search path (like PYTHONPATH), OS path-list separated. Applied to runs/debugs.")
                    .bindText(settings::gadPathEnv)
            }
        }
    }
}
