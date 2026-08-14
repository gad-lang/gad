package dev.gad.intellij.settings

import com.intellij.openapi.fileChooser.FileChooserDescriptorFactory
import com.intellij.openapi.options.BoundConfigurable
import com.intellij.openapi.options.Configurable
import com.intellij.openapi.options.SearchableConfigurable
import com.intellij.openapi.ui.DialogPanel
import com.intellij.openapi.ui.TextFieldWithBrowseButton
import com.intellij.ui.dsl.builder.AlignX
import com.intellij.ui.dsl.builder.bindSelected
import com.intellij.ui.dsl.builder.bindText
import com.intellij.ui.dsl.builder.panel
import javax.swing.JComponent

/**
 * Parent of the "Gad" settings group (Settings ▸ Tools ▸ Gad). It is a container
 * for the three child pages — Executable, GADPATH and Formatting — registered
 * with `parentId` in plugin.xml.
 */
class GadSettingsConfigurable : SearchableConfigurable {
    override fun getId(): String = ID
    override fun getDisplayName(): String = "Gad"

    override fun createComponent(): JComponent = panel {
        row {
            comment(
                "Configure the Gad language tools. Expand this node for the " +
                    "<b>Executable</b>, <b>GADPATH</b> and <b>Formatting</b> pages.",
            )
        }
    }

    override fun isModified(): Boolean = false
    override fun apply() {}

    companion object {
        const val ID = "dev.gad.intellij.settings"
    }
}

/** Settings ▸ Tools ▸ Gad ▸ Executable — the `gad` binary location. */
class GadExecutableConfigurable : BoundConfigurable("Executable"), Configurable {
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
                    .align(AlignX.FILL)
                    .comment("Leave blank to resolve <code>gad</code> from PATH.")
                    .bindText(settings::gadPath)
            }
        }
    }
}

/** Settings ▸ Tools ▸ Gad ▸ GADPATH — the default module search path. */
class GadPathConfigurable : BoundConfigurable("GADPATH"), Configurable {
    override fun createPanel(): DialogPanel {
        val settings = GadSettings.getInstance()
        return panel {
            row("Default GADPATH:") {
                textField()
                    .align(AlignX.FILL)
                    .comment(
                        "Module search path (like PYTHONPATH), OS path-list separated. " +
                            "Applied to runs and debug sessions; a run configuration may override it.",
                    )
                    .bindText(settings::gadPathEnv)
            }
        }
    }
}

/** Settings ▸ Tools ▸ Gad ▸ Formatting — the `gad fmt` options. */
class GadFormattingConfigurable : BoundConfigurable("Formatting"), Configurable {
    override fun createPanel(): DialogPanel {
        val settings = GadSettings.getInstance()
        return panel {
            group("gad fmt") {
                row {
                    checkBox("Disable all multi-line formatting")
                        .comment("<code>-no-format</code>")
                        .bindSelected(settings::fmtNoFormat)
                }
                row {
                    checkBox("Keep array items on a single line")
                        .comment("<code>-no-array-item-in-new-line</code>")
                        .bindSelected(settings::fmtNoArrayItemNewLine)
                }
                row {
                    checkBox("Keep call params on a single line")
                        .comment("<code>-no-call-params-in-new-line</code>")
                        .bindSelected(settings::fmtNoCallParamsNewLine)
                }
                row {
                    checkBox("Keep declaration items on a single line")
                        .comment("<code>-no-decl-item-in-new-line</code>")
                        .bindSelected(settings::fmtNoDeclItemNewLine)
                }
                row {
                    checkBox("Keep dict items on a single line")
                        .comment("<code>-no-dict-item-in-new-line</code>")
                        .bindSelected(settings::fmtNoDictItemNewLine)
                }
                row {
                    checkBox("Keep keyValueArray items on a single line")
                        .comment("<code>-no-key-value-array-item-in-new-line</code>")
                        .bindSelected(settings::fmtNoKeyValueArrayItemNewLine)
                }
                row {
                    checkBox("Keep param values on a single line")
                        .comment("<code>-no-parem-values-in-new-line</code>")
                        .bindSelected(settings::fmtNoParamValuesNewLine)
                }
                row {
                    checkBox("Back up each file before formatting")
                        .comment("<code>-backup</code>")
                        .bindSelected(settings::fmtBackup)
                }
            }
        }
    }
}
