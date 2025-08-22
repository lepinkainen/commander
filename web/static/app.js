// Commander Web Interface
class Commander {
    constructor() {
        this.ws = null;
        this.tasks = new Map();
        this.currentFilter = 'running';
        this.currentTheme = localStorage.getItem('commander-theme') || 'default';
        this.directories = [];
        this.files = [];
        this.selectedDirectory = null;
        this.selectedFiles = new Set();
        this.init();
    }

    async init() {
        this.initTheme();
        await this.loadTools();
        await this.loadTasks();
        await this.loadStats();
        await this.loadDirectories();
        this.connectWebSocket();
        this.setupEventListeners();
        
        // Refresh stats every 5 seconds
        setInterval(() => this.loadStats(), 5000);
    }

    initTheme() {
        // Apply saved theme
        document.body.setAttribute('data-theme', this.currentTheme);
        
        // Update theme selector UI
        document.querySelectorAll('.theme-icon').forEach(icon => {
            icon.classList.remove('active');
            if (icon.dataset.theme === this.currentTheme) {
                icon.classList.add('active');
            }
        });
    }

    switchTheme(themeName) {
        this.currentTheme = themeName;
        document.body.setAttribute('data-theme', themeName);
        localStorage.setItem('commander-theme', themeName);
        
        // Update theme selector UI
        document.querySelectorAll('.theme-icon').forEach(icon => {
            icon.classList.remove('active');
            if (icon.dataset.theme === themeName) {
                icon.classList.add('active');
            }
        });
    }

    async loadTools() {
        try {
            const response = await fetch('/api/tools');
            this.tools = await response.json();
            const toolButtonsContainer = document.getElementById('tool-buttons');
            const toolInput = document.getElementById('tool');
            
            toolButtonsContainer.innerHTML = '';
            
            this.tools.forEach((tool, index) => {
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
        } catch (error) {
            console.error('Failed to load tools:', error);
        }
    }

    async loadTasks() {
        try {
            const response = await fetch('/api/tasks');
            const tasks = await response.json();
            
            this.tasks.clear();
            tasks.forEach(task => {
                this.tasks.set(task.id, task);
            });
            
            this.renderTasks();
        } catch (error) {
            console.error('Failed to load tasks:', error);
        }
    }

    async loadStats() {
        try {
            const response = await fetch('/api/stats');
            const stats = await response.json();
            
            const statsContainer = document.getElementById('stats');
            statsContainer.innerHTML = '';
            
            // Calculate totals
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
            
            // Create stat cards
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
            
            // Add tool-specific stats
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
        } catch (error) {
            console.error('Failed to load stats:', error);
        }
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/ws`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.updateConnectionStatus(true);
        };
        
        this.ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            this.handleWebSocketMessage(data);
        };
        
        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            this.updateConnectionStatus(false);
            
            // Reconnect after 3 seconds
            setTimeout(() => this.connectWebSocket(), 3000);
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    handleWebSocketMessage(data) {
        const { task_id, type, data: content } = data;
        
        switch (type) {
            case 'created':
                // Reload tasks to get the new one
                this.loadTasks();
                this.loadStats();
                break;
                
            case 'status':
                // Update task status
                const task = this.tasks.get(task_id);
                if (task) {
                    task.status = content;
                    this.updateTaskElement(task_id);
                }
                this.loadStats();
                break;
                
            case 'output':
                // Append output to task
                const outputTask = this.tasks.get(task_id);
                if (outputTask) {
                    if (!outputTask.output) {
                        outputTask.output = [];
                    }
                    outputTask.output.push(content);
                    this.appendOutputToTask(task_id, content);
                }
                break;
                
            case 'files_discovered':
                // Handle file discovery notification
                this.handleFileDiscovery(task_id, content);
                break;
        }
    }

    handleFileDiscovery(taskId, files) {
        // Update task with associated files
        const task = this.tasks.get(taskId);
        if (task) {
            task.associated_files = files;
            this.updateTaskElement(taskId);
        }
        
        // Show notification about file discovery
        if (files && files.length > 0) {
            const fileCount = files.length;
            const message = `📁 ${fileCount} file${fileCount > 1 ? 's' : ''} discovered for task`;
            this.showNotification(message);
        }
        
        // Refresh file list if current directory matches any of the discovered files
        if (this.selectedDirectory && files && files.length > 0) {
            const hasMatchingFiles = files.some(file => 
                file.directory_id === this.selectedDirectory.id
            );
            if (hasMatchingFiles) {
                this.loadFiles(this.selectedDirectory.id);
            }
        }
    }

    setupEventListeners() {
        // Task form submission
        document.getElementById('taskForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const formData = new FormData(e.target);
            const tool = formData.get('tool');
            const args = formData.get('args').split(' ').filter(arg => arg.length > 0);
            
            try {
                const response = await fetch('/api/tasks', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        tool,
                        args,
                    }),
                });
                
                if (response.ok) {
                    document.getElementById('args').value = '';
                    this.showNotification('Task created successfully');
                    // Task will be added via WebSocket message
                } else {
                    this.showNotification('Failed to create task', true);
                }
            } catch (error) {
                console.error('Failed to create task:', error);
                this.showNotification('Failed to create task', true);
            }
        });
        
        // Filter buttons
        document.querySelectorAll('.filter-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                document.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active'));
                e.target.classList.add('active');
                this.currentFilter = e.target.dataset.filter;
                this.renderTasks();
            });
        });

        // Theme selector buttons
        document.querySelectorAll('.theme-icon').forEach(icon => {
            icon.addEventListener('click', (e) => {
                const themeName = e.target.dataset.theme;
                this.switchTheme(themeName);
            });
        });

        // File management event listeners
        this.setupFileEventListeners();

        // Event delegation for cancel buttons
        document.getElementById('taskList').addEventListener('click', (e) => {
            if (e.target.classList.contains('cancel-btn')) {
                const taskId = e.target.dataset.taskId;
                this.cancelTask(taskId);
            }
        });

        // Event delegation for tool buttons
        document.getElementById('tool-buttons').addEventListener('click', (e) => {
            if (e.target.classList.contains('tool-btn')) {
                document.querySelectorAll('.tool-btn').forEach(b => b.classList.remove('active'));
                e.target.classList.add('active');
                document.getElementById('tool').value = e.target.dataset.tool;
            }
        });
    }

    renderTasks() {
        const taskList = document.getElementById('taskList');
        taskList.innerHTML = '';
        
        // Sort tasks by creation time (newest first)
        const sortedTasks = Array.from(this.tasks.values()).sort((a, b) => {
            return new Date(b.created_at) - new Date(a.created_at);
        });
        
        // Filter tasks
        const filteredTasks = sortedTasks.filter(task => {
            if (this.currentFilter === 'all') return true;
            return task.status === this.currentFilter;
        });
        
        // Render each task
        filteredTasks.forEach(task => {
            const taskElement = this.createTaskElement(task);
            taskList.appendChild(taskElement);
        });
        
        if (filteredTasks.length === 0) {
            taskList.innerHTML = '<div style="text-align: center; color: #666; padding: 40px;">No tasks found</div>';
        }
    }

    createTaskElement(task) {
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
            <div class="task-command">${this.escapeHtml(command)}</div>
            ${task.error ? `<div style="color: #f56565; margin-top: 10px;">Error: ${this.escapeHtml(task.error)}</div>` : ''}
            ${hasOutput ? `
                <div class="task-output" id="output-${task.id}">
                    ${task.output.map(line => `
                        <div class="output-line ${line.startsWith('[ERROR]') ? 'error' : ''}">${this.escapeHtml(line)}</div>
                    `).join('')}
                </div>
            ` : ''}
            ${hasFiles ? `
                <div class="task-files">
                    <h4>📁 Associated Files (${task.associated_files.length})</h4>
                    <div class="task-file-list">
                        ${task.associated_files.map(file => `
                            <div class="task-file-item">
                                <div class="task-file-name">${this.escapeHtml(file.filename)}</div>
                                <div class="task-file-meta">
                                    ${this.formatFileSize(file.file_size)} • 
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

    updateTaskElement(taskId) {
        const task = this.tasks.get(taskId);
        if (!task) return;
        
        const element = document.getElementById(`task-${taskId}`);
        if (!element) {
            // If element doesn't exist and should be visible, re-render
            this.renderTasks();
            return;
        }
        
        // Update status
        const statusElement = element.querySelector('.task-status');
        if (statusElement) {
            statusElement.className = `task-status status-${task.status}`;
            statusElement.textContent = task.status.toUpperCase();
        }

        // Update cancel button
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
        
        // Update error if present
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

    appendOutputToTask(taskId, output) {
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
        
        // Auto-scroll to bottom
        outputContainer.scrollTop = outputContainer.scrollHeight;
    }

    async cancelTask(taskId) {
        try {
            const response = await fetch(`/api/tasks/${taskId}/cancel`, {
                method: 'POST',
            });
            
            if (!response.ok) {
                this.showNotification('Failed to cancel task', true);
            }
        } catch (error) {
            console.error('Failed to cancel task:', error);
            this.showNotification('Failed to cancel task', true);
        }
    }

    showNotification(message, isError = false) {
        const container = document.getElementById('notificationContainer');
        const notification = document.createElement('div');
        notification.className = `notification ${isError ? 'error' : 'success'}`;
        notification.textContent = message;
        
        container.appendChild(notification);
        
        // Trigger the animation
        setTimeout(() => {
            notification.classList.add('show');
        }, 10);
        
        // Remove the notification after 3 seconds
        setTimeout(() => {
            notification.classList.remove('show');
            setTimeout(() => {
                container.removeChild(notification);
            }, 500);
        }, 3000);
    }

    updateConnectionStatus(connected) {
        const status = document.getElementById('connectionStatus');
        if (connected) {
            status.className = 'connection-status connected';
            status.textContent = 'Connected';
        } else {
            status.className = 'connection-status disconnected';
            status.textContent = 'Disconnected';
        }
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // File Management Methods
    async loadDirectories() {
        try {
            const response = await fetch('/api/directories');
            this.directories = await response.json();
            this.renderDirectories();
        } catch (error) {
            console.error('Failed to load directories:', error);
        }
    }

    renderDirectories() {
        const container = document.getElementById('directoriesList');
        if (!this.directories || this.directories.length === 0) {
            container.innerHTML = '<div class="empty-state">No directories configured.<br>Click "Create Directory" to get started.</div>';
            return;
        }

        container.innerHTML = this.directories.map(dir => `
            <div class="directory-item ${this.selectedDirectory?.id === dir.id ? 'selected' : ''}" 
                 data-id="${dir.id}">
                <div class="directory-name">${this.escapeHtml(dir.name)}</div>
                <div class="directory-path">${this.escapeHtml(dir.path)}</div>
                <div class="directory-meta">
                    <span>${dir.tool_name || 'All tools'}</span>
                    <span>${dir.default_dir ? 'Default' : ''}</span>
                </div>
            </div>
        `).join('');

        // Add click handlers
        container.querySelectorAll('.directory-item').forEach(item => {
            item.addEventListener('click', () => {
                const dirId = item.dataset.id;
                this.selectDirectory(dirId);
            });
        });
    }

    async selectDirectory(dirId) {
        this.selectedDirectory = this.directories.find(d => d.id === dirId);
        this.renderDirectories();
        
        if (this.selectedDirectory) {
            await this.loadFiles(dirId);
        }
    }

    async loadFiles(directoryId = null) {
        try {
            let url = '/api/files';
            if (directoryId) {
                url += `?directory_id=${directoryId}`;
            }
            const response = await fetch(url);
            this.files = await response.json();
            this.renderFiles();
        } catch (error) {
            console.error('Failed to load files:', error);
        }
    }

    renderFiles() {
        const container = document.getElementById('filesList');
        if (!this.files || this.files.length === 0) {
            const message = this.selectedDirectory 
                ? 'No files found in this directory.<br>Click "Scan Directories" to discover files.'
                : 'Select a directory to view files.';
            container.innerHTML = `<div class="empty-state">${message}</div>`;
            this.updateBulkActionsVisibility();
            return;
        }

        container.innerHTML = this.files.map(file => `
            <div class="file-item" data-id="${file.id}">
                <div class="file-checkbox">
                    <input type="checkbox" 
                           id="file-${file.id}" 
                           class="file-select-checkbox" 
                           data-file-id="${file.id}"
                           ${this.selectedFiles.has(file.id) ? 'checked' : ''}>
                </div>
                <div class="file-content">
                    <div class="file-name">${this.escapeHtml(file.filename)}</div>
                    <div class="file-meta">Type: ${file.mime_type || 'Unknown'}</div>
                    <div class="file-meta">Size: ${this.formatFileSize(file.file_size)}</div>
                    <div class="file-meta">Created: ${new Date(file.created_at).toLocaleDateString()}</div>
                    ${file.tags && file.tags.length > 0 ? `<div class="file-meta">Tags: ${file.tags.join(', ')}</div>` : ''}
                    <div class="file-actions">
                        <button onclick="commander.downloadFile('${file.id}')">Download</button>
                        <button onclick="commander.deleteFile('${file.id}')" class="danger">Delete</button>
                    </div>
                </div>
            </div>
        `).join('');
        
        this.setupFileCheckboxListeners();
        this.updateBulkActionsVisibility();
    }

    formatFileSize(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    async createDirectory(formData) {
        try {
            const response = await fetch('/api/directories', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    name: formData.get('name'),
                    path: formData.get('path'),
                    tool_name: formData.get('tool') || null,
                    default_dir: formData.has('defaultDir')
                })
            });

            if (response.ok) {
                this.showNotification('Directory created successfully');
                await this.loadDirectories();
                this.hideDirectoryModal();
            } else {
                throw new Error('Failed to create directory');
            }
        } catch (error) {
            console.error('Failed to create directory:', error);
            this.showNotification('Failed to create directory', true);
        }
    }

    async scanDirectories() {
        if (!this.selectedDirectory) {
            this.showNotification('Please select a directory first', true);
            return;
        }

        try {
            const response = await fetch(`/api/directories/${this.selectedDirectory.id}/scan`, {
                method: 'POST'
            });

            if (response.ok) {
                this.showNotification('Directory scan completed');
                await this.loadFiles(this.selectedDirectory.id);
            } else {
                throw new Error('Failed to scan directory');
            }
        } catch (error) {
            console.error('Failed to scan directory:', error);
            this.showNotification('Failed to scan directory', true);
        }
    }

    async searchFiles(query) {
        try {
            const response = await fetch(`/api/files/search?q=${encodeURIComponent(query)}`);
            this.files = await response.json();
            this.renderFiles();
        } catch (error) {
            console.error('Failed to search files:', error);
            this.showNotification('Failed to search files', true);
        }
    }

    async downloadFile(fileId) {
        try {
            window.open(`/api/files/${fileId}/download`, '_blank');
        } catch (error) {
            console.error('Failed to download file:', error);
            this.showNotification('Failed to download file', true);
        }
    }

    async deleteFile(fileId) {
        if (!confirm('Are you sure you want to delete this file?')) {
            return;
        }

        try {
            const response = await fetch(`/api/files/${fileId}`, {
                method: 'DELETE'
            });

            if (response.ok) {
                this.showNotification('File deleted successfully');
                if (this.selectedDirectory) {
                    await this.loadFiles(this.selectedDirectory.id);
                } else {
                    await this.loadFiles();
                }
            } else {
                throw new Error('Failed to delete file');
            }
        } catch (error) {
            console.error('Failed to delete file:', error);
            this.showNotification('Failed to delete file', true);
        }
    }

    showDirectoryModal() {
        const modal = document.getElementById('directoryModal');
        modal.style.display = 'flex';
        
        // Populate tool dropdown
        const toolSelect = document.getElementById('directoryTool');
        toolSelect.innerHTML = '<option value="">All tools</option>';
        this.tools.forEach(tool => {
            toolSelect.innerHTML += `<option value="${tool.name}">${tool.name}</option>`;
        });
    }

    hideDirectoryModal() {
        const modal = document.getElementById('directoryModal');
        modal.style.display = 'none';
        document.getElementById('directoryForm').reset();
    }

    setupFileEventListeners() {
        // Create directory button
        document.getElementById('createDirectoryBtn').addEventListener('click', () => {
            this.showDirectoryModal();
        });

        // Scan directories button
        document.getElementById('scanDirectoriesBtn').addEventListener('click', () => {
            this.scanDirectories();
        });

        // File search
        document.getElementById('fileSearchBtn').addEventListener('click', () => {
            const query = document.getElementById('fileSearchInput').value.trim();
            if (query) {
                this.searchFiles(query);
            } else {
                this.loadFiles(this.selectedDirectory?.id);
            }
        });

        // Search on enter
        document.getElementById('fileSearchInput').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                document.getElementById('fileSearchBtn').click();
            }
        });

        // Directory form submission
        document.getElementById('directoryForm').addEventListener('submit', (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            this.createDirectory(formData);
        });

        // Modal close button
        document.querySelector('#directoryModal .close').addEventListener('click', () => {
            this.hideDirectoryModal();
        });

        // Cancel button
        document.querySelector('#directoryModal .cancel-btn').addEventListener('click', () => {
            this.hideDirectoryModal();
        });

        // Close modal on outside click
        document.getElementById('directoryModal').addEventListener('click', (e) => {
            if (e.target === e.currentTarget) {
                this.hideDirectoryModal();
            }
        });

        // Bulk actions
        this.setupBulkActionListeners();
    }

    setupBulkActionListeners() {
        // Select all button
        document.getElementById('selectAllBtn').addEventListener('click', () => {
            this.selectAllFiles();
        });

        // Clear selection button
        document.getElementById('clearSelectionBtn').addEventListener('click', () => {
            this.clearFileSelection();
        });

        // Bulk delete button
        document.getElementById('bulkDeleteBtn').addEventListener('click', () => {
            this.bulkDeleteFiles();
        });

        // Bulk move button
        document.getElementById('bulkMoveBtn').addEventListener('click', () => {
            this.showBulkMoveModal();
        });

        // Bulk tag button
        document.getElementById('bulkTagBtn').addEventListener('click', () => {
            this.showBulkTagModal();
        });

        // Bulk move form
        document.getElementById('bulkMoveForm').addEventListener('submit', (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            this.executeBulkMove(formData.get('directory'));
        });

        // Bulk tag form
        document.getElementById('bulkTagForm').addEventListener('submit', (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            this.executeBulkTag(formData.get('tags'));
        });

        // Bulk move modal controls
        document.querySelector('#bulkMoveModal .close').addEventListener('click', () => {
            this.hideBulkMoveModal();
        });
        document.querySelector('#bulkMoveModal .cancel-btn').addEventListener('click', () => {
            this.hideBulkMoveModal();
        });

        // Bulk tag modal controls
        document.querySelector('#bulkTagModal .close').addEventListener('click', () => {
            this.hideBulkTagModal();
        });
        document.querySelector('#bulkTagModal .cancel-btn').addEventListener('click', () => {
            this.hideBulkTagModal();
        });

        // Close modals on outside click
        document.getElementById('bulkMoveModal').addEventListener('click', (e) => {
            if (e.target === e.currentTarget) {
                this.hideBulkMoveModal();
            }
        });
        document.getElementById('bulkTagModal').addEventListener('click', (e) => {
            if (e.target === e.currentTarget) {
                this.hideBulkTagModal();
            }
        });
    }

    setupFileCheckboxListeners() {
        document.querySelectorAll('.file-select-checkbox').forEach(checkbox => {
            checkbox.addEventListener('change', (e) => {
                const fileId = e.target.dataset.fileId;
                if (e.target.checked) {
                    this.selectedFiles.add(fileId);
                } else {
                    this.selectedFiles.delete(fileId);
                }
                this.updateBulkActionsVisibility();
            });
        });
    }

    updateBulkActionsVisibility() {
        const bulkActions = document.getElementById('bulkActions');
        const selectedCount = document.getElementById('selectedCount');
        
        if (this.selectedFiles.size > 0) {
            bulkActions.style.display = 'flex';
            selectedCount.textContent = this.selectedFiles.size;
        } else {
            bulkActions.style.display = 'none';
        }
    }

    selectAllFiles() {
        this.files.forEach(file => {
            this.selectedFiles.add(file.id);
        });
        this.renderFiles();
    }

    clearFileSelection() {
        this.selectedFiles.clear();
        this.renderFiles();
    }

    async bulkDeleteFiles() {
        if (this.selectedFiles.size === 0) return;

        const confirmMessage = `Are you sure you want to delete ${this.selectedFiles.size} files? This action cannot be undone.`;
        if (!confirm(confirmMessage)) return;

        try {
            const response = await fetch('/api/files/bulk/delete', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    file_ids: Array.from(this.selectedFiles)
                })
            });

            if (response.ok) {
                this.showNotification(`${this.selectedFiles.size} files deleted successfully`);
                this.selectedFiles.clear();
                if (this.selectedDirectory) {
                    await this.loadFiles(this.selectedDirectory.id);
                } else {
                    await this.loadFiles();
                }
            } else {
                throw new Error('Failed to delete files');
            }
        } catch (error) {
            console.error('Failed to bulk delete files:', error);
            this.showNotification('Failed to delete files', true);
        }
    }

    showBulkMoveModal() {
        if (this.selectedFiles.size === 0) return;

        const modal = document.getElementById('bulkMoveModal');
        const directorySelect = document.getElementById('moveToDirectory');
        
        // Populate directory options
        directorySelect.innerHTML = '<option value="">Select directory...</option>';
        this.directories.forEach(dir => {
            directorySelect.innerHTML += `<option value="${dir.id}">${dir.name} (${dir.path})</option>`;
        });
        
        modal.style.display = 'flex';
    }

    hideBulkMoveModal() {
        document.getElementById('bulkMoveModal').style.display = 'none';
        document.getElementById('bulkMoveForm').reset();
    }

    async executeBulkMove(directoryId) {
        if (!directoryId || this.selectedFiles.size === 0) return;

        try {
            const response = await fetch('/api/files/bulk/move', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    file_ids: Array.from(this.selectedFiles),
                    directory_id: directoryId
                })
            });

            if (response.ok) {
                this.showNotification(`${this.selectedFiles.size} files moved successfully`);
                this.selectedFiles.clear();
                this.hideBulkMoveModal();
                if (this.selectedDirectory) {
                    await this.loadFiles(this.selectedDirectory.id);
                } else {
                    await this.loadFiles();
                }
            } else {
                throw new Error('Failed to move files');
            }
        } catch (error) {
            console.error('Failed to bulk move files:', error);
            this.showNotification('Failed to move files', true);
        }
    }

    showBulkTagModal() {
        if (this.selectedFiles.size === 0) return;
        document.getElementById('bulkTagModal').style.display = 'flex';
    }

    hideBulkTagModal() {
        document.getElementById('bulkTagModal').style.display = 'none';
        document.getElementById('bulkTagForm').reset();
    }

    async executeBulkTag(tagsInput) {
        if (this.selectedFiles.size === 0) return;

        const tags = tagsInput ? tagsInput.split(',').map(tag => tag.trim()).filter(tag => tag) : [];

        try {
            const response = await fetch('/api/files/bulk/tag', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    file_ids: Array.from(this.selectedFiles),
                    tags: tags
                })
            });

            if (response.ok) {
                this.showNotification(`${this.selectedFiles.size} files tagged successfully`);
                this.selectedFiles.clear();
                this.hideBulkTagModal();
                if (this.selectedDirectory) {
                    await this.loadFiles(this.selectedDirectory.id);
                } else {
                    await this.loadFiles();
                }
            } else {
                throw new Error('Failed to tag files');
            }
        } catch (error) {
            console.error('Failed to bulk tag files:', error);
            this.showNotification('Failed to tag files', true);
        }
    }
}

// Initialize the application
let commander;
document.addEventListener('DOMContentLoaded', () => {
    commander = new Commander();
});
