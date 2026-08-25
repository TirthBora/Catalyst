import * as vscode from 'vscode';
import { spawn, ChildProcess } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

let catalystProcess: ChildProcess | null = null;
let statusBarItem: vscode.StatusBarItem;

export function activate(context: vscode.ExtensionContext) {
    vscode.window.showInformationMessage('Catalyst extension activated!');

    statusBarItem = vscode.window.createStatusBarItem(
        vscode.StatusBarAlignment.Right,
        100
    );

    context.subscriptions.push(statusBarItem);

    // Command to open the Catalyst UI
    context.subscriptions.push(
        vscode.commands.registerCommand('catalyst.openUI', () => {
            vscode.env.openExternal(
                vscode.Uri.parse('http://localhost:9999')
            );
        })
    );

    // Try to start Catalyst if a workspace is already open
    startFromCurrentWorkspace();

    // If the Extension Development Host starts without a workspace,
    // start Catalyst when a workspace is opened.
    context.subscriptions.push(
        vscode.workspace.onDidChangeWorkspaceFolders(() => {
            startFromCurrentWorkspace();
        })
    );
}

function startFromCurrentWorkspace() {
    if (catalystProcess) {
        return;
    }

    const workspaceFolders = vscode.workspace.workspaceFolders;

    if (!workspaceFolders || workspaceFolders.length === 0) {
        statusBarItem.text = '$(circle-slash) Catalyst: No Workspace';
        statusBarItem.tooltip = 'Open a workspace to start Catalyst';
        statusBarItem.show();

        return;
    }

    const rootPath = workspaceFolders[0].uri.fsPath;

    startCatalystDaemon(rootPath);
}

function startCatalystDaemon(rootPath: string) {
    statusBarItem.text = '$(sync~spin) Catalyst Booting...';
    statusBarItem.tooltip = 'Starting Catalyst...';
    statusBarItem.command = undefined;
    statusBarItem.show();

    const exePath = path.join(rootPath, 'catalyst.exe');

    console.log('=================================');
    console.log('CATALYST EXTENSION');
    console.log(`Workspace: ${rootPath}`);
    console.log(`Executable: ${exePath}`);
    console.log('=================================');

    vscode.window.showInformationMessage(
        `Catalyst found workspace: ${rootPath}`
    );

    // Check that catalyst.exe actually exists
    if (!fs.existsSync(exePath)) {
        const message = `catalyst.exe was not found at:\n${exePath}`;

        console.error(message);

        statusBarItem.text = '$(error) Catalyst Error';
        statusBarItem.tooltip = message;
        statusBarItem.show();

        vscode.window.showErrorMessage(message);

        return;
    }

    vscode.window.showInformationMessage(
        `Starting Catalyst: ${exePath}`
    );

    try {
        catalystProcess = spawn(exePath, [], {
            cwd: rootPath,
            windowsHide: true
        });
    } catch (error) {
        const message = `Failed to start Catalyst: ${error}`;

        console.error(message);

        statusBarItem.text = '$(error) Catalyst Error';
        statusBarItem.tooltip = message;
        statusBarItem.show();

        catalystProcess = null;

        return;
    }

    let stdoutBuffer = '';

    catalystProcess.stdout?.on('data', (data) => {
        const output = data.toString();

        console.log(`Catalyst stdout: ${output}`);

        stdoutBuffer += output;

        // Keep checking the complete accumulated output.
        if (stdoutBuffer.includes('CATALYST V1 IS ACTIVE')) {
            statusBarItem.text = '$(broadcast) Catalyst Active';
            statusBarItem.tooltip =
                'Click to open Catalyst Secondary Workspace';
            statusBarItem.command = 'catalyst.openUI';
            statusBarItem.show();

            vscode.window.showInformationMessage(
                'Catalyst engine started successfully!'
            );
        }
    });

    catalystProcess.stderr?.on('data', (data) => {
        const errorOutput = data.toString();

        console.error(`Catalyst stderr: ${errorOutput}`);
    });

    catalystProcess.on('error', (err) => {
        console.error(`Catalyst process error: ${err.message}`);

        statusBarItem.text = '$(error) Catalyst Error';
        statusBarItem.tooltip = err.message;
        statusBarItem.show();

        vscode.window.showErrorMessage(
            `Catalyst failed to start: ${err.message}`
        );

        catalystProcess = null;
    });

    catalystProcess.on('exit', (code) => {
        console.log(`Catalyst exited with code ${code}`);

        catalystProcess = null;

        if (code !== 0 && code !== null) {
            statusBarItem.text = '$(error) Catalyst Stopped';
            statusBarItem.tooltip =
                `Catalyst exited with code ${code}`;
            statusBarItem.show();
        }
    });
}

export function deactivate() {
    if (catalystProcess) {
        catalystProcess.kill();
        catalystProcess = null;
    }
}