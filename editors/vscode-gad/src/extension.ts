import * as vscode from "vscode";
import { execFile } from "child_process";
import * as path from "path";

// The Gad debug adapter is the `gad` CLI itself, run as `gad debug --dap`,
// speaking the Debug Adapter Protocol over stdio. This extension wires it into
// VS Code's debugging UI for files of the `gad` language.

// Run `gad fmt -` with `input` on stdin and return {stdout, stderr, code}.
function runGadFmt(
  gadPath: string,
  args: string[],
  input: string,
  cwd: string,
): Promise<{ stdout: string; stderr: string; code: number }> {
  return new Promise((resolve) => {
    const child = execFile(gadPath, args, { cwd }, (err, stdout, stderr) => {
      resolve({ stdout, stderr, code: (err as NodeJS.ErrnoException & { code?: number })?.code ?? 0 });
    });
    child.stdin?.write(input);
    child.stdin?.end();
  });
}

export function activate(context: vscode.ExtensionContext): void {
  // Fill in a default launch config (program = current file) when the user
  // starts debugging without a launch.json entry.
  const provider: vscode.DebugConfigurationProvider = {
    resolveDebugConfiguration(folder, config) {
      if (!config.type && !config.request && !config.name) {
        const editor = vscode.window.activeTextEditor;
        if (editor && editor.document.languageId === "gad") {
          config.type = "gad";
          config.request = "launch";
          config.name = "Debug Gad file";
          config.program = "${file}";
        }
      }
      if (!config.program) {
        return vscode.window
          .showInformationMessage("Cannot find a Gad program to debug")
          .then(() => undefined);
      }
      return config;
    },
  };
  context.subscriptions.push(
    vscode.debug.registerDebugConfigurationProvider("gad", provider),
  );

  // Spawn `gad debug --dap` as the debug adapter.
  const factory: vscode.DebugAdapterDescriptorFactory = {
    createDebugAdapterDescriptor() {
      const gadPath = vscode.workspace
        .getConfiguration("gad")
        .get<string>("path", "gad");
      return new vscode.DebugAdapterExecutable(gadPath, ["debug", "--dap"]);
    },
  };
  context.subscriptions.push(
    vscode.debug.registerDebugAdapterDescriptorFactory("gad", factory),
  );

  // Format-on-save: register a DocumentFormattingEditProvider that calls
  // `gad fmt -` (stdin/stdout mode). The `gad.format.useConfig` setting
  // controls whether the workspace .gad.yaml is applied; `gad.path` is the
  // executable path.
  context.subscriptions.push(
    vscode.languages.registerDocumentFormattingEditProvider("gad", {
      async provideDocumentFormattingEdits(
        document: vscode.TextDocument,
      ): Promise<vscode.TextEdit[]> {
        const cfg = vscode.workspace.getConfiguration("gad");
        const gadPath = cfg.get<string>("path", "gad");
        const useConfig = cfg.get<boolean>("format.useConfig", true);

        const args = ["fmt"];
        if (!useConfig) args.push("--no-config");
        args.push("-"); // stdin mode

        const workspaceFolder = vscode.workspace.getWorkspaceFolder(document.uri);
        const cwd = workspaceFolder?.uri.fsPath ?? path.dirname(document.uri.fsPath);

        const input = document.getText();
        const result = await runGadFmt(gadPath, args, input, cwd);

        if (result.code !== 0 || !result.stdout) {
          if (result.stderr) {
            void vscode.window.showErrorMessage(`gad fmt: ${result.stderr.trim()}`);
          }
          return [];
        }

        // Replace the entire document with the formatted output.
        const range = new vscode.Range(
          document.positionAt(0),
          document.positionAt(input.length),
        );
        return [vscode.TextEdit.replace(range, result.stdout)];
      },
    }),
  );
}

export function deactivate(): void {
  // nothing to clean up
}
