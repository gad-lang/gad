package dev.gad.intellij.debug

import com.intellij.execution.configurations.RunProfile
import com.intellij.execution.configurations.RunProfileState
import com.intellij.execution.executors.DefaultDebugExecutor
import com.intellij.execution.runners.ExecutionEnvironment
import com.intellij.execution.runners.GenericProgramRunner
import com.intellij.execution.ui.RunContentDescriptor
import com.intellij.openapi.fileEditor.FileDocumentManager
import com.intellij.xdebugger.XDebugProcess
import com.intellij.xdebugger.XDebugProcessStarter
import com.intellij.xdebugger.XDebugSession
import com.intellij.xdebugger.XDebuggerManager
import dev.gad.intellij.run.GadRunConfiguration

/** Handles the Debug executor for Gad run configurations by starting a DAP-backed session. */
class GadDebugRunner : GenericProgramRunner<com.intellij.execution.configurations.RunnerSettings>() {

    override fun getRunnerId(): String = "GadDebugRunner"

    override fun canRun(executorId: String, profile: RunProfile): Boolean =
        executorId == DefaultDebugExecutor.EXECUTOR_ID && profile is GadRunConfiguration

    override fun doExecute(state: RunProfileState, environment: ExecutionEnvironment): RunContentDescriptor {
        FileDocumentManager.getInstance().saveAllDocuments()
        val configuration = environment.runProfile as GadRunConfiguration
        val session = XDebuggerManager.getInstance(environment.project).startSession(
            environment,
            object : XDebugProcessStarter() {
                override fun start(session: XDebugSession): XDebugProcess =
                    GadDebugProcess(session, configuration.profile())
            },
        )
        return session.runContentDescriptor
    }
}
