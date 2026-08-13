package dev.gad.intellij.lang

import com.intellij.openapi.fileTypes.LanguageFileType
import com.intellij.openapi.fileTypes.PlainTextLikeFileType
import javax.swing.Icon

/**
 * The three Gad dialect file types. They register real file types (so `.gad` /
 * `.gadt` / `.gadx` appear in Settings > Editor > File Types with the Gad icon
 * and give run/debug a solid file association). Highlighting is delegated to the
 * bundled TextMate grammars via a syntax-highlighter factory registered for the
 * shared [GadLanguage] — TextMate resolves the grammar by file name.
 *
 * Each has a public no-arg constructor so the platform can instantiate it from
 * the `<fileType implementationClass=...>` registration; a static `INSTANCE` is
 * kept for programmatic use.
 */

class GadFileType : LanguageFileType(GadLanguage), PlainTextLikeFileType {
    override fun getName(): String = "Gad"
    override fun getDescription(): String = "Gad script"
    override fun getDefaultExtension(): String = "gad"
    override fun getIcon(): Icon = GadIcons.GAD_FILE

    companion object {
        @JvmField
        val INSTANCE = GadFileType()
    }
}

class GadTemplateFileType : LanguageFileType(GadLanguage), PlainTextLikeFileType {
    override fun getName(): String = "Gad Template"
    override fun getDescription(): String = "Gad mixed-mode template"
    override fun getDefaultExtension(): String = "gadt"
    override fun getIcon(): Icon = GadIcons.GADT_FILE

    companion object {
        @JvmField
        val INSTANCE = GadTemplateFileType()
    }
}

class GadxFileType : LanguageFileType(GadLanguage), PlainTextLikeFileType {
    override fun getName(): String = "Gadx"
    override fun getDescription(): String = "Gadx indentation template"
    override fun getDefaultExtension(): String = "gadx"
    override fun getIcon(): Icon = GadIcons.GADX_FILE

    companion object {
        @JvmField
        val INSTANCE = GadxFileType()
    }
}
