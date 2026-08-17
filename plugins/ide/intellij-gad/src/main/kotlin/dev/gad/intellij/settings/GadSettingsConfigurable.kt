package dev.gad.intellij.settings

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.fileChooser.FileChooserDescriptorFactory
import com.intellij.openapi.options.BoundConfigurable
import com.intellij.openapi.options.Configurable
import com.intellij.openapi.options.SearchableConfigurable
import com.intellij.openapi.ui.DialogPanel
import com.intellij.openapi.ui.TextFieldWithBrowseButton
import com.intellij.ui.components.JBLabel
import com.intellij.ui.dsl.builder.AlignX
import com.intellij.ui.dsl.builder.bindSelected
import com.intellij.ui.dsl.builder.bindText
import com.intellij.ui.dsl.builder.panel
import com.intellij.util.ui.UIUtil
import javax.swing.JComponent

/**
 * Parent of the "Gad" settings group (Settings ▸ Build, Execution, Deployment ▸ Gad). It is a container
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

/** Settings ▸ Build, Execution, Deployment ▸ Gad ▸ Executable — the `gad` binary location, version and status. */
class GadExecutableConfigurable : BoundConfigurable("Executable"), Configurable {
    private lateinit var field: TextFieldWithBrowseButton
    private val statusLabel = JBLabel()

    override fun createPanel(): DialogPanel {
        val settings = GadSettings.getInstance()
        field = TextFieldWithBrowseButton().apply {
            addBrowseFolderListener(
                "Gad Executable",
                "Select the gad binary (leave blank to resolve from PATH)",
                null,
                FileChooserDescriptorFactory.createSingleFileDescriptor(),
            )
        }
        val panel = panel {
            row("Gad executable:") {
                cell(field)
                    .align(AlignX.FILL)
                    .comment("Leave blank to resolve <code>gad</code> from PATH.")
                    .bindText(settings::gadPath)
            }
            row("Status:") {
                cell(statusLabel).align(AlignX.FILL)
                button("Test") { refreshStatus() }
            }
        }
        refreshStatus()
        return panel
    }

    /**
     * Probe the binary named in the field (off the EDT — it launches
     * `gad version`) and show the resolved path + version, or the error.
     */
    private fun refreshStatus() {
        val configured = if (::field.isInitialized) field.text else ""
        statusLabel.text = "Checking…"
        ApplicationManager.getApplication().executeOnPooledThread {
            val probe = GadExecutable.probe(configured)
            ApplicationManager.getApplication().invokeLater {
                statusLabel.text = renderProbe(probe)
            }
        }
    }

    private fun renderProbe(probe: GadProbe): String {
        val err = UIUtil.getErrorForeground().let { "#%02x%02x%02x".format(it.red, it.green, it.blue) }
        return if (probe.ok) {
            "<html><b>${probe.version}</b><br/>" +
                "<span style='color:gray'>${probe.path}</span></html>"
        } else {
            "<html><span style='color:$err'>${probe.error ?: "gad not available"}</span></html>"
        }
    }
}

/** Settings ▸ Build, Execution, Deployment ▸ Gad ▸ GADPATH — the default module search path. */
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

/** Settings ▸ Build, Execution, Deployment ▸ Gad ▸ Formatting — the `gad fmt` options. */
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
