package dev.gad.intellij.run

import com.intellij.execution.configurations.GeneralCommandLine
import com.intellij.execution.configurations.PtyCommandLine
import com.intellij.execution.process.KillableColoredProcessHandler
import com.intellij.execution.process.ProcessHandler
import com.intellij.execution.process.ProcessTerminatedListener
import com.intellij.execution.runners.ExecutionEnvironment
import com.intellij.execution.states.CommandLineState
import java.io.File

/** Runs `gad <script> [args…]` for the normal Run executor. */
class GadCommandLineState(
    private val configuration: GadRunConfiguration,
    environment: ExecutionEnvironment,
) : CommandLineState(environment) {

    override fun startProcess(): ProcessHandler {
        val profile = configuration.profile()
        val cmd = PtyCommandLine(buildCommandLine(profile))
        val handler = KillableColoredProcessHandler(cmd)
        ProcessTerminatedListener.attach(handler)
        return handler
    }

    private fun buildCommandLine(profile: GadProfile): GeneralCommandLine {
        val cmd = GeneralCommandLine(profile.executable)
        cmd.addParameter(profile.scriptPath)
        cmd.addParameters(profile.args)
        cmd.workDirectory = File(profile.workingDirectory)
        cmd.withEnvironment(profile.environment)
        cmd.withParentEnvironmentType(
            if (configuration.options.passParentEnvs)
                GeneralCommandLine.ParentEnvironmentType.CONSOLE
            else
                GeneralCommandLine.ParentEnvironmentType.NONE,
        )
        return cmd
    }
}
