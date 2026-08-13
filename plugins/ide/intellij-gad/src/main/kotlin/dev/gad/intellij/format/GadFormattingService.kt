package dev.gad.intellij.format

import com.intellij.execution.configurations.GeneralCommandLine
import com.intellij.formatting.service.AsyncDocumentFormattingService
import com.intellij.formatting.service.AsyncFormattingRequest
import com.intellij.formatting.service.FormattingService
import com.intellij.psi.PsiFile
import dev.gad.intellij.lang.GadFile
import dev.gad.intellij.settings.GadSettings

/**
 * Runs `gad fmt` on Reformat Code (Code ▸ Reformat Code) for Gad files, honoring
 * the options from Settings ▸ Tools ▸ Gad ▸ Formatting.
 *
 * `gad fmt -` reads the source on stdin and writes the formatted result to
 * stdout, so the whole document is piped through it; the configured
 * `-no-*-in-new-line` / `-no-format` / `-backup` flags are passed on the command
 * line.
 */
class GadFormattingService : AsyncDocumentFormattingService() {

    override fun getName(): String = "gad fmt"

    override fun getNotificationGroupId(): String = "Gad"

    override fun getFeatures(): Set<FormattingService.Feature> = emptySet()

    override fun canFormat(file: PsiFile): Boolean =
        GadFile.isGadFile(file.virtualFile)

    override fun createFormattingTask(request: AsyncFormattingRequest): FormattingTask? {
        val settings = GadSettings.getInstance()
        val exe = settings.resolveExecutable()

        val command = mutableListOf(exe, "fmt")
        // -backup is meaningless when formatting stdin (there is no file to back
        // up), so pass only the layout flags here.
        command.addAll(settings.fmtFlags().filter { it != "-backup" })
        command.add("-") // read stdin, write stdout

        val cmd = GeneralCommandLine(command)
        request.ioFile?.parent?.let { cmd.setWorkDirectory(it) }
        cmd.charset = Charsets.UTF_8

        return object : FormattingTask {
            @Volatile
            private var process: Process? = null

            override fun run() {
                try {
                    val p = cmd.createProcess().also { process = it }
                    p.outputStream.use { it.write(request.documentText.toByteArray(Charsets.UTF_8)) }
                    val out = p.inputStream.readBytes().toString(Charsets.UTF_8)
                    val err = p.errorStream.readBytes().toString(Charsets.UTF_8)
                    val code = p.waitFor()
                    if (code == 0) {
                        request.onTextReady(out)
                    } else {
                        request.onError(
                            "Gad formatting failed",
                            err.ifBlank { "gad fmt exited with code $code" },
                        )
                    }
                } catch (e: Exception) {
                    request.onError("Gad formatting failed", e.message ?: e.toString())
                }
            }

            override fun cancel(): Boolean {
                process?.destroy()
                return true
            }

            override fun isRunUnderProgress(): Boolean = true
        }
    }
}
