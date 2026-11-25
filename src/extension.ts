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
            const config = vscode.workspace.getConfiguration('catalyst');
            const port = config.get<string>('port') || '9999';
            vscode.env.openExternal(
                vscode.Uri.parse(`http://localhost:${port}`)
            );
        })
    );

    // Try to start Catalyst if a workspace is already open
    startFromCurrentWorkspace(context);

    // If the Extension Development Host starts without a workspace,
    // start Catalyst when a workspace is opened.
    context.subscriptions.push(
        vscode.workspace.onDidChangeWorkspaceFolders(() => {
            startFromCurrentWorkspace(context);
        })
    );
}

function resolveCatalystBinary(context: vscode.ExtensionContext, rootPath: string): string | null {
    // 1. Explicitly configured path in VS Code settings
    const config = vscode.workspace.getConfiguration('catalyst');
    const configuredPath = config.get<string>('executablePath');
    if (configuredPath && configuredPath.trim().length > 0) {
        const expanded = configuredPath.replace(/^~/, process.env.HOME || process.env.USERPROFILE || '');
        if (fs.existsSync(expanded)) {
            return expanded;
        }
    }

    // 2. Candidate paths in workspace root
    const workspaceCandidates = [
        path.join(rootPath, 'catalyst.exe'),
        path.join(rootPath, 'catalyst'),
        path.join(rootPath, 'bin', 'catalyst.exe'),
        path.join(rootPath, 'bin', 'catalyst'),
    ];
    for (const candidate of workspaceCandidates) {
        if (fs.existsSync(candidate)) {
            return candidate;
        }
    }

    // 3. Bundled inside extension directory
    const extensionCandidates = [
        context.asAbsolutePath('bin/catalyst.exe'),
        context.asAbsolutePath('bin/catalyst'),
        context.asAbsolutePath('catalyst.exe'),
        context.asAbsolutePath('catalyst'),
    ];
    for (const candidate of extensionCandidates) {
        if (fs.existsSync(candidate)) {
            return candidate;
        }
    }

    // 4. System PATH check
    const pathDirs = (process.env.PATH || '').split(path.delimiter);
    const exeNames = process.platform === 'win32'
        ? ['catalyst.exe', 'catalyst.cmd', 'catalyst.bat', 'catalyst']
        : ['catalyst'];
    for (const dir of pathDirs) {
        for (const exe of exeNames) {
            const fullPath = path.join(dir, exe);
            if (fs.existsSync(fullPath)) {
                return fullPath;
            }
        }
    }

    return null;
}

function startFromCurrentWorkspace(context: vscode.ExtensionContext) {
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

    startCatalystDaemon(context, rootPath);
}

function startCatalystDaemon(context: vscode.ExtensionContext, rootPath: string) {
    statusBarItem.text = '$(sync~spin) Catalyst Booting...';
    statusBarItem.tooltip = 'Starting Catalyst...';
    statusBarItem.command = undefined;
    statusBarItem.show();

    const exePath = resolveCatalystBinary(context, rootPath);

    console.log('=================================');
    console.log('CATALYST EXTENSION');
    console.log(`Workspace: ${rootPath}`);
    console.log(`Executable: ${exePath || 'NOT FOUND'}`);
    console.log('=================================');

    // Check that a binary was resolved
    if (!exePath) {
        const message = 'Catalyst executable (catalyst.exe) was not found in settings, workspace, extension directory, or PATH.';

        console.error(message);

        statusBarItem.text = '$(error) Catalyst: Missing Binary';
        statusBarItem.tooltip = message;
        statusBarItem.show();

        vscode.window.showErrorMessage(
            message,
            'Configure Path'
        ).then((selection) => {
            if (selection === 'Configure Path') {
                vscode.commands.executeCommand(
                    'workbench.action.openSettings',
                    'catalyst.executablePath'
                );
            }
        });

        return;
    }

    const config = vscode.workspace.getConfiguration('catalyst');
    const port = config.get<string>('port') || '9999';
    const args = ['-port', port, '-dir', rootPath];

    vscode.window.showInformationMessage(
        `Starting Catalyst: ${exePath} (Port: ${port})`
    );

    try {
        catalystProcess = spawn(exePath, args, {
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
                `Click to open Catalyst Secondary Workspace (Port ${port})`;
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