import * as vscode from "vscode";

// The Gad debug adapter is the `gad` CLI itself, run as `gad debug --dap`,
// speaking the Debug Adapter Protocol over stdio. This extension wires it into
// VS Code's debugging UI for files of the `gad` language.

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
}

export function deactivate(): void {
  // nothing to clean up
}
