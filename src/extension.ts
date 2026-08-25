import * as vscode from 'vscode';
import { spawn, ChildProcess } from 'child_process';
import * as path from 'path';

let catalystProcess: ChildProcess | null = null;
let statusBarItem: vscode.StatusBarItem;

export function activate(context: vscode.ExtensionContext) {
    statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
    context.subscriptions.push(statusBarItem);

    const workspaceFolders = vscode.workspace.workspaceFolders;
    if (!workspaceFolders) return;

    const rootPath = workspaceFolders[0].uri.fsPath;
    
    // Command to open the browser
    context.subscriptions.push(vscode.commands.registerCommand('catalyst.openUI', () => {
        vscode.env.openExternal(vscode.Uri.parse('http://localhost:9999'));
    }));

    startCatalystDaemon(rootPath);
}

function startCatalystDaemon(rootPath: string) {
    statusBarItem.text = "$(sync~spin) Catalyst Booting...";
    statusBarItem.show();

    // Point this to where you put your catalyst.exe!
    // For now, we assume it is in the root of the opened VS Code folder
    const exePath = path.join(rootPath, 'catalyst.exe');

    catalystProcess = spawn(exePath, [], { cwd: rootPath });

    catalystProcess.stdout?.on('data', (data) => {
        const output = data.toString();
        if (output.includes('CATALYST V1 IS ACTIVE')) {
            statusBarItem.text = "$(broadcast) Catalyst Active";
            statusBarItem.tooltip = "Click to open Catalyst Secondary Workspace";
            statusBarItem.command = 'catalyst.openUI';
        }
    });

    catalystProcess.on('error', (err) => {
        vscode.window.showErrorMessage(`Catalyst failed to start: ${err.message}`);
        statusBarItem.text = "$(error) Catalyst Error";
    });
}

export function deactivate() {
    if (catalystProcess) {
        catalystProcess.kill();
    }
}