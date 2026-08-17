package dev.gad.intellij.doc

import com.intellij.execution.configurations.GeneralCommandLine
import com.intellij.openapi.Disposable
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.application.ModalityState
import com.intellij.openapi.editor.Document
import com.intellij.openapi.fileEditor.FileDocumentManager
import com.intellij.openapi.fileEditor.FileEditorManager
import com.intellij.openapi.project.Project
import com.intellij.openapi.util.Disposer
import com.intellij.openapi.vfs.VirtualFile
import com.intellij.ui.jcef.JBCefApp
import com.intellij.ui.jcef.JBCefBrowser
import com.intellij.util.Alarm
import com.intellij.util.ui.UIUtil
import dev.gad.intellij.lang.GadFile
import dev.gad.intellij.settings.GadExecutable
import dev.gad.intellij.settings.GadSettings
import java.awt.Color
import java.nio.charset.StandardCharsets
import java.util.concurrent.TimeUnit
import javax.swing.JComponent
import javax.swing.JEditorPane
import javax.swing.JScrollPane

/**
 * Live documentation preview of the active Gad editor. Every second it checks
 * the selected editor and, when the file or its (unsaved) content changed, pipes
 * the current buffer to `gad doc -name <file> -html -` and shows the rendered
 * HTML — without writing any file. Uses JCEF when available, else a basic Swing
 * HTML pane.
 */
class GadDocPanel(private val project: Project, parent: Disposable) {

    private val browser: JBCefBrowser? = if (JBCefApp.isSupported()) JBCefBrowser() else null
    private val fallback = JEditorPane("text/html", "").apply { isEditable = false }
    private val alarm = Alarm(Alarm.ThreadToUse.SWING_THREAD, parent)
    private var lastKey: String? = null

    val component: JComponent =
        browser?.component ?: JScrollPane(fallback)

    init {
        browser?.let { Disposer.register(parent, it) }
        showMessage("Open a .gad / .gadt / .gadx file to see its documentation.")
        schedule(0)
    }

    private fun schedule(delayMs: Int) {
        if (alarm.isDisposed) return
        alarm.addRequest({ tick() }, delayMs)
    }

    /** One refresh cycle: re-render only when the target/content changed. */
    private fun tick() {
        try {
            val editor = FileEditorManager.getInstance(project).selectedTextEditor
            val doc: Document? = editor?.document
            val file: VirtualFile? = doc?.let { FileDocumentManager.getInstance().getFile(it) }
            if (doc == null || file == null || !GadFile.isGadFile(file)) {
                if (lastKey != "none") {
                    lastKey = "none"
                    showMessage("Open a .gad / .gadt / .gadx file to see its documentation.")
                }
                return
            }
            val key = file.path + "@" + doc.modificationStamp
            if (key == lastKey) return
            lastKey = key
            render(file, doc.text)
        } finally {
            schedule(REFRESH_MS)
        }
    }

    /** Run `gad doc` off the EDT, then load the HTML back on the EDT. */
    private fun render(file: VirtualFile, text: String) {
        val exe = GadExecutable.resolvePath(GadSettings.getInstance().gadPath) ?: run {
            showMessage("gad executable not found — set it in Settings ▸ Build, Execution, Deployment ▸ Gad ▸ Executable.")
            return
        }
        // The generated docs mirror the source tree under the workspace `doc/`
        // directory, and `/PATH` in doc comments resolves against that `doc/` root.
        // Pass the source's project-relative path as --name so the module identity
        // matches, and base the preview at the file's `doc/` location so relative
        // asset links resolve the same way they do in the generated docs.
        val base = project.basePath
        val rel = if (base != null && file.path.startsWith("$base/")) file.path.removePrefix("$base/") else file.name
        val docBase = if (base != null) "file://$base/doc/${rel.substringBeforeLast('/', "")}".trimEnd('/') + "/" else null
        ApplicationManager.getApplication().executeOnPooledThread {
            val html = runDoc(exe, rel, text, docBase)
            ApplicationManager.getApplication().invokeLater({ show(html) }, ModalityState.any())
        }
    }

    private fun runDoc(exe: String, name: String, text: String, docBase: String?): String {
        return try {
            val cmd = GeneralCommandLine(exe, "doc", "-name", name, "-html", "-")
                .withCharset(StandardCharsets.UTF_8)
            val process = cmd.createProcess()
            process.outputStream.use { it.write(text.toByteArray(StandardCharsets.UTF_8)) }
            val out = process.inputStream.readBytes().toString(StandardCharsets.UTF_8)
            val err = process.errorStream.readBytes().toString(StandardCharsets.UTF_8)
            process.waitFor(10, TimeUnit.SECONDS)
            if (process.exitValue() != 0) {
                errorHtml(err.ifBlank { "gad doc exited with ${process.exitValue()}" })
            } else {
                page(out, docBase)
            }
        } catch (e: Exception) {
            errorHtml(e.message ?: e.toString())
        }
    }

    private fun show(html: String) {
        browser?.loadHTML(html) ?: run { fallback.text = html; fallback.caretPosition = 0 }
    }

    private fun showMessage(msg: String) = show(page("<p style='opacity:.7'>${escape(msg)}</p>", null))

    private fun errorHtml(msg: String) =
        page("<pre style='color:#e06c75;white-space:pre-wrap'>${escape(msg)}</pre>", null)

    /** Wrap an HTML fragment in a theme-aware page, basing relative URLs (asset
     * links in doc comments) at the file's `doc/` directory when known. */
    private fun page(body: String, docBase: String?): String {
        val bg = hex(UIUtil.getPanelBackground())
        val fg = hex(UIUtil.getLabelForeground())
        val link = hex(UIUtil.getLabelForeground())
        val border = hex(UIUtil.getBoundsColor())
        val mono = "SFMono-Regular, Consolas, 'Liberation Mono', Menlo, monospace"
        val baseTag = if (docBase != null) "<base href=\"$docBase\">" else ""
        return """
            <!doctype html><html><head><meta charset="utf-8">$baseTag<style>
              body { background:$bg; color:$fg; font-family:sans-serif; font-size:13px;
                     line-height:1.5; margin:0; padding:12px 16px; }
              h1 { font-size:1.5em; } h2 { font-size:1.25em; border-bottom:1px solid $border; padding-bottom:.2em; }
              h3 { font-size:1.05em; } a { color:$link; }
              code, pre { font-family:$mono; font-size:12px; }
              pre { background:rgba(127,127,127,.12); padding:8px 10px; border-radius:6px; overflow:auto; }
              code { background:rgba(127,127,127,.12); padding:1px 4px; border-radius:4px; }
              pre code { background:none; padding:0; }
              table { border-collapse:collapse; } th,td { border:1px solid $border; padding:4px 8px; }
              blockquote { margin:0; padding:0 12px; border-left:3px solid $border; opacity:.85; }
            </style></head><body>$body</body></html>
        """.trimIndent()
    }

    private fun hex(c: Color): String = "#%02x%02x%02x".format(c.red, c.green, c.blue)

    private fun escape(s: String): String =
        s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")

    companion object {
        private const val REFRESH_MS = 1000
    }
}
