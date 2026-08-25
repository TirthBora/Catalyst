import * as vscode from 'vscode';
import { spawn, ChildProcess } from 'child_process';
import * as path from 'path';

let catalystProcess: ChildProcess | null = null;
let statusBarItem: vscode.StatusBarItem;

export function activate(context: vscode.ExtensionContext) {
    statusBarItem = vscode.window.createStatusBarItem(
        vscode.StatusBarAlignment.Right,
        100
    );

    context.subscriptions.push(statusBarItem);

    context.subscriptions.push(
        vscode.commands.registerCommand('catalyst.openUI', () => {
            vscode.env.openExternal(
                vscode.Uri.parse('http://localhost:9999')
            );
        })
    );

    // Start Catalyst immediately if a workspace is already open
    if (vscode.workspace.workspaceFolders) {
        const rootPath = vscode.workspace.workspaceFolders[0].uri.fsPath;
        startCatalystDaemon(rootPath);
    }

    // Start Catalyst when a workspace is opened later
    context.subscriptions.push(
        vscode.workspace.onDidChangeWorkspaceFolders(() => {
            if (!catalystProcess && vscode.workspace.workspaceFolders) {
                const rootPath =
                    vscode.workspace.workspaceFolders[0].uri.fsPath;

                startCatalystDaemon(rootPath);
            }
        })
    );
}

function startCatalystDaemon(rootPath: string) {
    statusBarItem.text = '$(sync~spin) Catalyst Booting...';
    statusBarItem.tooltip = 'Starting Catalyst...';
    statusBarItem.show();

    const exePath = path.join(rootPath, 'catalyst.exe');

    console.log(`Catalyst executable: ${exePath}`);

    catalystProcess = spawn(exePath, [], {
        cwd: rootPath
    });

    catalystProcess.stdout?.on('data', (data) => {
        const output = data.toString();

        console.log(`Catalyst: ${output}`);

        if (output.includes('CATALYST V1 IS ACTIVE')) {
            statusBarItem.text = '$(broadcast) Catalyst Active';
            statusBarItem.tooltip =
                'Click to open Catalyst Secondary Workspace';
            statusBarItem.command = 'catalyst.openUI';
        }
    });

    catalystProcess.stderr?.on('data', (data) => {
        console.error(`Catalyst Error: ${data.toString()}`);
    });

    catalystProcess.on('error', (err) => {
        vscode.window.showErrorMessage(
            `Catalyst failed to start: ${err.message}`
        );

        statusBarItem.text = '$(error) Catalyst Error';
        statusBarItem.tooltip = err.message;

        catalystProcess = null;
    });

    catalystProcess.on('exit', (code) => {
        console.log(`Catalyst exited with code ${code}`);

        catalystProcess = null;

        if (code !== 0) {
            statusBarItem.text = '$(error) Catalyst Stopped';
        }
    });
}

export function deactivate() {
    if (catalystProcess) {
        catalystProcess.kill();
        catalystProcess = null;
    }
}