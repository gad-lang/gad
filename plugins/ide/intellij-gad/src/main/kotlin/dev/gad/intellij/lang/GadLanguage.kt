package dev.gad.intellij.lang

import com.intellij.lang.Language

/**
 * The Gad language. Shared by the three dialect file types (.gad / .gadt /
 * .gadx). Highlighting is provided by the bundled TextMate grammars (resolved by
 * file name), so no PSI/lexer is defined here.
 */
object GadLanguage : Language("Gad") {
    private fun readResolve(): Any = GadLanguage
    override fun getDisplayName(): String = "Gad"
}
