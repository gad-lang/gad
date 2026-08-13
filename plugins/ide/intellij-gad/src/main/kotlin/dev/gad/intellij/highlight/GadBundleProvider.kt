package dev.gad.intellij.highlight

import com.intellij.openapi.diagnostic.thisLogger
import com.intellij.plugins.textmate.api.TextMateBundleProvider
import com.intellij.plugins.textmate.api.TextMateBundleProvider.PluginBundle
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardCopyOption

/**
 * Ships the Gad grammars (reused from the VS Code extension, assembled into a
 * TextMate bundle by the Gradle build) so `.gad` / `.gadt` / `.gadx` files get
 * syntax highlighting with no hand-written lexer.
 *
 * TextMate reads bundle files from disk, so the bundle — which lives inside the
 * plugin jar under `/bundles/gad` — is extracted once to a temp directory and
 * that path is registered.
 */
class GadBundleProvider : TextMateBundleProvider {

    override fun getBundles(): List<PluginBundle> {
        val dir = extractedBundle ?: return emptyList()
        return listOf(PluginBundle("Gad", dir))
    }

    private val extractedBundle: Path? by lazy { extract() }

    private fun extract(): Path? = try {
        val target = Files.createTempDirectory("gad-textmate-bundle")
        for (rel in BUNDLE_FILES) {
            val res = javaClass.getResourceAsStream("/bundles/gad/$rel") ?: continue
            val out = target.resolve(rel)
            Files.createDirectories(out.parent)
            res.use { Files.copy(it, out, StandardCopyOption.REPLACE_EXISTING) }
        }
        if (Files.exists(target.resolve("package.json"))) target else null
    } catch (e: Exception) {
        thisLogger().warn("failed to extract Gad TextMate bundle", e)
        null
    }

    private companion object {
        val BUNDLE_FILES = listOf(
            "package.json",
            "language-configuration.json",
            "gadx-language-configuration.json",
            "syntaxes/gad.tmLanguage.json",
            "syntaxes/gadx.tmLanguage.json",
        )
    }
}
