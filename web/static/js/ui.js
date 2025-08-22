
import { escapeHtml, formatFileSize } from './utils.js';

export function initTheme(theme) {
    document.body.setAttribute('data-theme', theme);
    document.querySelectorAll('.theme-icon').forEach(icon => {
        icon.classList.remove('active');
        if (icon.dataset.theme === theme) {
            icon.classList.add('active');
        }
    });
}

export function switchTheme(theme) {
    document.body.setAttribute('data-theme', theme);
    localStorage.setItem('commander-theme', theme);
    document.querySelectorAll('.theme-icon').forEach(icon => {
        icon.classList.remove('active');
        if (icon.dataset.theme === theme) {
            icon.classList.add('active');
        }
    });
}

export function renderTasks(tasks, currentFilter) {
    const taskList = document.getElementById('taskList');
    taskList.innerHTML = '';
    
    const sortedTasks = Array.from(tasks.values()).sort((a, b) => {
        return new Date(b.created_at) - new Date(a.created_at);
    });
    
    const filteredTasks = sortedTasks.filter(task => {
        if (currentFilter === 'all') return true;
        return task.status === currentFilter;
    });
    
    filteredTasks.forEach(task => {
        const taskElement = createTaskElement(task);
        taskList.appendChild(taskElement);
    });
    
    if (filteredTasks.length === 0) {
        taskList.innerHTML = '<div style="text-align: center; color: #666; padding: 40px;">No tasks found</div>';
    }
}

export function createTaskElement(task) {
    const div = document.createElement('div');
    div.className = 'task-item';
    div.id = `task-${task.id}`;
    
    const command = `${task.command} ${task.args.join(' ')}`;
    const hasOutput = task.output && task.output.length > 0;
    const hasFiles = task.associated_files && task.associated_files.length > 0;
    const isCancelable = task.status === 'running' || task.status === 'queued';
    
    div.innerHTML = `
        <div class="task-header">
            <span class="task-tool">${task.tool}</span>
            <div class="task-actions">
                ${isCancelable ? `<button class="cancel-btn" data-task-id="${task.id}">Cancel</button>` : ''}
            </div>
            <span class="task-status status-${task.status}">${task.status.toUpperCase()}</span>
        </div>
        <div class="task-command">${escapeHtml(command)}</div>
        ${task.error ? `<div style="color: #f56565; margin-top: 10px;">Error: ${escapeHtml(task.error)}</div>` : ''}
        ${hasOutput ? `
            <div class="task-output" id="output-${task.id}">
                ${task.output.map(line => `
                    <div class="output-line ${line.startsWith('[ERROR]') ? 'error' : ''}">${escapeHtml(line)}</div>
                `).join('')}
            </div>
        ` : ''}
        ${hasFiles ? `
            <div class="task-files">
                <h4>📁 Associated Files (${task.associated_files.length})</h4>
                <div class="task-file-list">
                    ${task.associated_files.map(file => `
                        <div class="task-file-item">
                            <div class="task-file-name">${escapeHtml(file.filename)}</div>
                            <div class="task-file-meta">
                                ${formatFileSize(file.file_size)} • 
                                ${file.mime_type || 'Unknown type'} • 
                                <a href="/api/files/${file.id}/download" target="_blank">Download</a>
                            </div>
                        </div>
                    `).join('')}
                </div>
            </div>
        ` : ''}
    `;
    
    return div;
}

export function updateTaskElement(task) {
    const element = document.getElementById(`task-${task.id}`);
    if (!element) return;
    
    const statusElement = element.querySelector('.task-status');
    if (statusElement) {
        statusElement.className = `task-status status-${task.status}`;
        statusElement.textContent = task.status.toUpperCase();
    }

    const actionsContainer = element.querySelector('.task-actions');
    if (actionsContainer) {
        const isCancelable = task.status === 'running' || task.status === 'queued';
        if (isCancelable) {
            if (!actionsContainer.querySelector('.cancel-btn')) {
                actionsContainer.innerHTML = `<button class="cancel-btn" data-task-id="${task.id}">Cancel</button>`;
            }
        } else {
            actionsContainer.innerHTML = '';
        }
    }
    
    if (task.error) {
        const errorDiv = document.createElement('div');
        errorDiv.style.cssText = 'color: #f56565; margin-top: 10px;';
        errorDiv.textContent = `Error: ${task.error}`;
        
        const commandDiv = element.querySelector('.task-command');
        if (commandDiv && !element.querySelector('div[style*="color: #f56565"]')) {
            commandDiv.insertAdjacentElement('afterend', errorDiv);
        }
    }
}

export function appendOutputToTask(taskId, output) {
    let outputContainer = document.getElementById(`output-${taskId}`);
    
    if (!outputContainer) {
        const taskElement = document.getElementById(`task-${taskId}`);
        if (!taskElement) return;
        
        outputContainer = document.createElement('div');
        outputContainer.className = 'task-output';
        outputContainer.id = `output-${taskId}`;
        taskElement.appendChild(outputContainer);
    }
    
    const outputLine = document.createElement('div');
    outputLine.className = `output-line ${output.startsWith('[ERROR]') ? 'error' : ''}`;
    outputLine.textContent = output;
    outputContainer.appendChild(outputLine);
    
    outputContainer.scrollTop = outputContainer.scrollHeight;
}

export function showNotification(message, isError = false) {
    const container = document.getElementById('notificationContainer');
    const notification = document.createElement('div');
    notification.className = `notification ${isError ? 'error' : 'success'}`;
    notification.textContent = message;
    
    container.appendChild(notification);
    
    setTimeout(() => {
        notification.classList.add('show');
    }, 10);
    
    setTimeout(() => {
        notification.classList.remove('show');
        setTimeout(() => {
            container.removeChild(notification);
        }, 500);
    }, 3000);
}

export function updateConnectionStatus(connected) {
    const status = document.getElementById('connectionStatus');
    if (connected) {
        status.className = 'connection-status connected';
        status.textContent = 'Connected';
    } else {
        status.className = 'connection-status disconnected';
        status.textContent = 'Disconnected';
    }
}

export function renderDirectories(directories, selectedDirectory, selectDirectoryCallback) {
    const container = document.getElementById('directoriesList');
    if (!directories || directories.length === 0) {
        container.innerHTML = '<div class="empty-state">No directories configured.<br>Click "Create Directory" to get started.</div>';
        return;
    }

    container.innerHTML = directories.map(dir => `
        <div class="directory-item ${selectedDirectory?.id === dir.id ? 'selected' : ''}" 
             data-id="${dir.id}">
            <div class="directory-name">${escapeHtml(dir.name)}</div>
            <div class="directory-path">${escapeHtml(dir.path)}</div>
            <div class="directory-meta">
                <span>${dir.tool_name || 'All tools'}</span>
                <span>${dir.default_dir ? 'Default' : ''}</span>
            </div>
        </div>
    `).join('');

    container.querySelectorAll('.directory-item').forEach(item => {
        item.addEventListener('click', () => {
            selectDirectoryCallback(item.dataset.id);
        });
    });
}

export function renderFiles(files, selectedFiles, selectedDirectory, fileCheckboxCallback) {
    const container = document.getElementById('filesList');
    if (!files || files.length === 0) {
        const message = selectedDirectory 
            ? 'No files found in this directory.<br>Click "Scan Directories" to discover files.'
            : 'Select a directory to view files.';
        container.innerHTML = `<div class="empty-state">${message}</div>`;
        updateBulkActionsVisibility(selectedFiles.size);
        return;
    }

    container.innerHTML = files.map(file => `
        <div class="file-item" data-id="${file.id}">
            <div class="file-checkbox">
                <input type="checkbox" 
                       id="file-${file.id}" 
                       class="file-select-checkbox" 
                       data-file-id="${file.id}"
                       ${selectedFiles.has(file.id) ? 'checked' : ''}>
            </div>
            <div class="file-content">
                <div class="file-name">${escapeHtml(file.filename)}</div>
                <div class="file-meta">Type: ${file.mime_type || 'Unknown'}</div>
                <div class="file-meta">Size: ${formatFileSize(file.file_size)}</div>
                <div class="file-meta">Created: ${new Date(file.created_at).toLocaleDateString()}</div>
                ${file.tags && file.tags.length > 0 ? `<div class="file-meta">Tags: ${file.tags.join(', ')}</div>` : ''}
                <div class="file-actions">
                    <button onclick="commander.downloadFile('${file.id}')">Download</button>
                    <button onclick="commander.deleteFile('${file.id}')" class="danger">Delete</button>
                </div>
            </div>
        </div>
    `).join('');
    
    setupFileCheckboxListeners(fileCheckboxCallback);
    updateBulkActionsVisibility(selectedFiles.size);
}

export function showDirectoryModal(tools) {
    const modal = document.getElementById('directoryModal');
    modal.style.display = 'flex';
    
    const toolSelect = document.getElementById('directoryTool');
    toolSelect.innerHTML = '<option value="">All tools</option>';
    tools.forEach(tool => {
        toolSelect.innerHTML += `<option value="${tool.name}">${tool.name}</option>`;
    });
}

export function hideDirectoryModal() {
    const modal = document.getElementById('directoryModal');
    modal.style.display = 'none';
    document.getElementById('directoryForm').reset();
}

export function updateBulkActionsVisibility(selectedFilesSize) {
    const bulkActions = document.getElementById('bulkActions');
    const selectedCount = document.getElementById('selectedCount');
    
    if (selectedFilesSize > 0) {
        bulkActions.style.display = 'flex';
        selectedCount.textContent = selectedFilesSize;
    } else {
        bulkActions.style.display = 'none';
    }
}

export function showBulkMoveModal(directories) {
    const modal = document.getElementById('bulkMoveModal');
    const directorySelect = document.getElementById('moveToDirectory');
    
    directorySelect.innerHTML = '<option value="">Select directory...</option>';
    directories.forEach(dir => {
        directorySelect.innerHTML += `<option value="${dir.id}">${dir.name} (${dir.path})</option>`;
    });
    
    modal.style.display = 'flex';
}

export function hideBulkMoveModal() {
    document.getElementById('bulkMoveModal').style.display = 'none';
    document.getElementById('bulkMoveForm').reset();
}

export function showBulkTagModal() {
    document.getElementById('bulkTagModal').style.display = 'flex';
}

export function hideBulkTagModal() {
    document.getElementById('bulkTagModal').style.display = 'none';
    document.getElementById('bulkTagForm').reset();
}

export function renderTools(tools) {
    const toolButtonsContainer = document.getElementById('tool-buttons');
    const toolInput = document.getElementById('tool');
    
    toolButtonsContainer.innerHTML = '';
    
    tools.forEach((tool, index) => {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'tool-btn';
        button.textContent = tool.name;
        button.dataset.tool = tool.name;
        button.title = tool.description;
        
        toolButtonsContainer.appendChild(button);

        if (index === 0) {
            button.classList.add('active');
            toolInput.value = tool.name;
        }
    });
}

export function renderStats(stats) {
    const statsContainer = document.getElementById('stats');
    statsContainer.innerHTML = '';
    
    let totalPending = 0;
    let totalRunning = 0;
    let totalCompleted = 0;
    let totalFailed = 0;
    
    Object.values(stats).forEach(stat => {
        totalPending += stat.pending;
        totalRunning += stat.running;
        totalCompleted += stat.completed;
        totalFailed += stat.failed;
    });
    
    const statCards = [
        { label: 'Pending', value: totalPending },
        { label: 'Running', value: totalRunning },
        { label: 'Completed', value: totalCompleted },
        { label: 'Failed', value: totalFailed }
    ];
    
    statCards.forEach(stat => {
        const card = document.createElement('div');
        card.className = 'stat-card';
        card.innerHTML = `
            <h3>${stat.label}</h3>
            <div class="stat-value">${stat.value}</div>
        `;
        statsContainer.appendChild(card);
    });
    
    Object.entries(stats).forEach(([tool, stat]) => {
        if (stat.running > 0 || stat.pending > 0) {
            const card = document.createElement('div');
            card.className = 'stat-card';
            card.innerHTML = `
                <h3>${tool}</h3>
                <div class="stat-value">${stat.running}/${stat.pending}</div>
            `;
            statsContainer.appendChild(card);
        }
    });
}

function setupFileCheckboxListeners(callback) {
    document.querySelectorAll('.file-select-checkbox').forEach(checkbox => {
        checkbox.addEventListener('change', (e) => {
            const fileId = e.target.dataset.fileId;
            callback(fileId, e.target.checked);
        });
    });
}
