package dev.gad.intellij.debug

import com.intellij.lang.Language
import com.intellij.openapi.editor.Document
import com.intellij.openapi.fileTypes.FileType
import com.intellij.openapi.fileTypes.PlainTextLanguage
import com.intellij.openapi.project.Project
import com.intellij.psi.PsiDocumentManager
import com.intellij.psi.PsiFileFactory
import com.intellij.testFramework.LightVirtualFile
import com.intellij.xdebugger.XExpression
import com.intellij.xdebugger.XSourcePosition
import com.intellij.xdebugger.evaluation.EvaluationMode
import com.intellij.xdebugger.evaluation.XDebuggerEditorsProvider

/**
 * Editor for breakpoint conditions and evaluate/watch expressions. Gad has no
 * PSI in this plugin, so expressions are edited as plain text (still sent to the
 * adapter for real evaluation).
 */
class GadDebuggerEditorsProvider : XDebuggerEditorsProvider() {
    override fun getFileType(): FileType = com.intellij.openapi.fileTypes.PlainTextFileType.INSTANCE

    override fun createDocument(
        project: Project,
        expression: XExpression,
        sourcePosition: XSourcePosition?,
        mode: EvaluationMode,
    ): Document {
        val language: Language = expression.language ?: PlainTextLanguage.INSTANCE
        val file = PsiFileFactory.getInstance(project).createFileFromText(
            "gad-expression.txt",
            language,
            expression.expression,
            false,
            true,
        )
        return PsiDocumentManager.getInstance(project).getDocument(file)
            ?: LightVirtualFile("gad-expression.txt", expression.expression).let {
                com.intellij.openapi.fileEditor.FileDocumentManager.getInstance().getDocument(it)!!
            }
    }
}
