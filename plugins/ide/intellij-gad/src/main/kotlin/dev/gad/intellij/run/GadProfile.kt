package dev.gad.intellij.run

import com.intellij.util.execution.ParametersListUtil
import dev.gad.intellij.settings.GadSettings
import java.io.File

/**
 * The resolved execution profile derived from a run configuration's options and
 * the application settings — shared by the plain run (command line) and the
 * debug launch (DAP configuration).
 */
data class GadProfile(
    val executable: String,
    val scriptPath: String,
    val args: List<String>,
    val workingDirectory: String,
    val environment: Map<String, String>,
) {
    companion object {
        fun from(options: GadRunConfigurationOptions): GadProfile {
            val settings = GadSettings.getInstance()

            val script = options.scriptPath.trim()
            val workDir = options.workingDirectory.trim().ifEmpty {
                File(script).parent ?: System.getProperty("user.dir")
            }

            val env = LinkedHashMap<String, String>()
            decodeEnv(options.envJson).forEach { (k, v) -> env[k] = v }
            // GADPATH: the profile override wins, else the application default.
            val gadPath = options.gadPath.trim().ifEmpty { settings.gadPathEnv.trim() }
            if (gadPath.isNotEmpty() && !env.containsKey("GADPATH")) {
                env["GADPATH"] = gadPath
            }

            return GadProfile(
                executable = settings.resolveExecutable(),
                scriptPath = script,
                args = ParametersListUtil.parse(options.programArguments),
                workingDirectory = workDir,
                environment = env,
            )
        }

        /** Decode the `{"K":"V"}` JSON env map without a JSON dependency. */
        fun decodeEnv(json: String): Map<String, String> {
            val trimmed = json.trim()
            if (trimmed.length < 2 || trimmed[0] != '{') return emptyMap()
            val out = LinkedHashMap<String, String>()
            var i = 1
            val s = trimmed
            fun readString(): String? {
                while (i < s.length && s[i] != '"') i++
                if (i >= s.length) return null
                i++ // opening quote
                val sb = StringBuilder()
                while (i < s.length && s[i] != '"') {
                    if (s[i] == '\\' && i + 1 < s.length) {
                        i++
                        sb.append(
                            when (s[i]) {
                                'n' -> '\n'; 't' -> '\t'; 'r' -> '\r'; else -> s[i]
                            },
                        )
                    } else {
                        sb.append(s[i])
                    }
                    i++
                }
                i++ // closing quote
                return sb.toString()
            }
            while (i < s.length) {
                val key = readString() ?: break
                while (i < s.length && s[i] != ':') i++
                i++
                val value = readString() ?: break
                out[key] = value
                while (i < s.length && s[i] != ',' && s[i] != '}') i++
                if (i >= s.length || s[i] == '}') break
                i++
            }
            return out
        }

        fun encodeEnv(env: Map<String, String>): String {
            if (env.isEmpty()) return ""
            fun esc(v: String) = buildString {
                for (c in v) when (c) {
                    '"' -> append("\\\"")
                    '\\' -> append("\\\\")
                    '\n' -> append("\\n")
                    '\t' -> append("\\t")
                    '\r' -> append("\\r")
                    else -> append(c)
                }
            }
            return env.entries.joinToString(",", "{", "}") { "\"${esc(it.key)}\":\"${esc(it.value)}\"" }
        }
    }
}
