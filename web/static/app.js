import { loadTasks, loadTools, loadStats, loadDirectories, createTask, cancelTask, scanDirectories, searchFiles, downloadFile, deleteFile, bulkDeleteFiles, executeBulkMove, executeBulkTag, createDirectory, loadFiles } from './js/api.js';
import { initTheme, switchTheme, renderTasks, updateTaskElement, appendOutputToTask, showNotification, updateConnectionStatus, renderDirectories, renderFiles, showDirectoryModal, hideDirectoryModal, updateBulkActionsVisibility, showBulkMoveModal, hideBulkMoveModal, showBulkTagModal, hideBulkTagModal, renderTools, renderStats } from './js/ui.js';
import { WebSocketManager } from './js/websocket.js';

class Commander {
    constructor() {
        this.wsManager = null;
        this.tasks = new Map();
        this.tools = [];
        this.currentFilter = 'running';
        this.currentTheme = localStorage.getItem('commander-theme') || 'default';
        this.directories = [];
        this.files = [];
        this.selectedDirectory = null;
        this.selectedFiles = new Set();
        this.init();
    }

    async init() {
        initTheme(this.currentTheme);
        try {
            this.tools = await loadTools();
            renderTools(this.tools);
        } catch (error) {
            console.error('Failed to load tools:', error);
        }
        await this.loadAndRenderTasks();
        await this.loadAndRenderStats();
        await this.loadAndRenderDirectories();
        
        this.wsManager = new WebSocketManager(
            (data) => this.handleWebSocketMessage(data),
            () => updateConnectionStatus(true),
            () => updateConnectionStatus(false)
        );
        this.wsManager.connect();
        
        this.setupEventListeners();
        
        setInterval(() => this.loadAndRenderStats(), 5000);
    }

    async loadAndRenderTasks() {
        try {
            const tasks = await loadTasks();
            this.tasks.clear();
            tasks.forEach(task => {
                this.tasks.set(task.id, task);
            });
            renderTasks(this.tasks, this.currentFilter);
        } catch (error) {
            console.error('Failed to load tasks:', error);
        }
    }

    async loadAndRenderStats() {
        try {
            const stats = await loadStats();
            renderStats(stats);
        } catch (error) {
            console.error('Failed to load stats:', error);
        }
    }

    async loadAndRenderDirectories() {
        try {
            this.directories = await loadDirectories();
            renderDirectories(this.directories, this.selectedDirectory, (dirId) => this.selectDirectory(dirId));
        } catch (error) {
            console.error('Failed to load directories:', error);
        }
    }

    async selectDirectory(dirId) {
        this.selectedDirectory = this.directories.find(d => d.id === dirId);
        renderDirectories(this.directories, this.selectedDirectory, (dirId) => this.selectDirectory(dirId));
        
        if (this.selectedDirectory) {
            await this.loadAndRenderFiles(dirId);
        }
    }

    async loadAndRenderFiles(directoryId = null) {
        try {
            this.files = await loadFiles(directoryId);
            this.renderFiles();
        } catch (error) {
            console.error('Failed to load files:', error);
        }
    }

    renderFiles() {
        renderFiles(this.files, this.selectedFiles, this.selectedDirectory, (fileId, checked) => {
            if (checked) {
                this.selectedFiles.add(fileId);
            } else {
                this.selectedFiles.delete(fileId);
            }
            updateBulkActionsVisibility(this.selectedFiles.size);
        });
    }

    handleWebSocketMessage(data) {
        const { task_id, type, data: content } = data;
        
        switch (type) {
            case 'created':
                this.loadAndRenderTasks();
                this.loadAndRenderStats();
                break;
                
            case 'status':
                const task = this.tasks.get(task_id);
                if (task) {
                    task.status = content;
                    updateTaskElement(task);
                }
                this.loadAndRenderStats();
                break;
                
            case 'output':
                const outputTask = this.tasks.get(task_id);
                if (outputTask) {
                    if (!outputTask.output) {
                        outputTask.output = [];
                    }
                    outputTask.output.push(content);
                    appendOutputToTask(task_id, content);
                }
                break;
                
            case 'files_discovered':
                this.handleFileDiscovery(task_id, content);
                break;
        }
    }

    handleFileDiscovery(taskId, files) {
        const task = this.tasks.get(taskId);
        if (task) {
            task.associated_files = files;
            updateTaskElement(task);
        }
        
        if (files && files.length > 0) {
            const fileCount = files.length;
            const message = `📁 ${fileCount} file${fileCount > 1 ? 's' : ''} discovered for task`;
            showNotification(message);
        }
        
        if (this.selectedDirectory && files && files.length > 0) {
            const hasMatchingFiles = files.some(file => 
                file.directory_id === this.selectedDirectory.id
            );
            if (hasMatchingFiles) {
                this.loadAndRenderFiles(this.selectedDirectory.id);
            }
        }
    }

    setupEventListeners() {
        document.getElementById('taskForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const formData = new FormData(e.target);
            const tool = formData.get('tool');
            const args = formData.get('args').split(' ').filter(arg => arg.length > 0);
            
            try {
                await createTask(tool, args);
                document.getElementById('args').value = '';
                showNotification('Task created successfully');
            } catch (error) {
                console.error('Failed to create task:', error);
                showNotification('Failed to create task', true);
            }
        });
        
        document.querySelectorAll('.filter-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                document.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active'));
                e.target.classList.add('active');
                this.currentFilter = e.target.dataset.filter;
                renderTasks(this.tasks, this.currentFilter);
            });
        });

        document.querySelectorAll('.theme-icon').forEach(icon => {
            icon.addEventListener('click', (e) => {
                const themeName = e.target.dataset.theme;
                this.currentTheme = themeName;
                switchTheme(themeName);
            });
        });

        document.getElementById('taskList').addEventListener('click', (e) => {
            if (e.target.classList.contains('cancel-btn')) {
                const taskId = e.target.dataset.taskId;
                this.cancelTask(taskId);
            }
        });

        document.getElementById('tool-buttons').addEventListener('click', (e) => {
            if (e.target.classList.contains('tool-btn')) {
                document.querySelectorAll('.tool-btn').forEach(b => b.classList.remove('active'));
                e.target.classList.add('active');
                document.getElementById('tool').value = e.target.dataset.tool;
            }
        });

        this.setupFileEventListeners();
    }

    setupFileEventListeners() {
        document.getElementById('createDirectoryBtn').addEventListener('click', () => {
            showDirectoryModal(this.tools);
        });

        document.getElementById('scanDirectoriesBtn').addEventListener('click', () => {
            this.scanDirectories();
        });

        document.getElementById('fileSearchBtn').addEventListener('click', () => {
            const query = document.getElementById('fileSearchInput').value.trim();
            if (query) {
                this.searchFiles(query);
            } else {
                this.loadAndRenderFiles(this.selectedDirectory?.id);
            }
        });

        document.getElementById('fileSearchInput').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                document.getElementById('fileSearchBtn').click();
            }
        });

        document.getElementById('directoryForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            try {
                await createDirectory(formData);
                showNotification('Directory created successfully');
                await this.loadAndRenderDirectories();
                hideDirectoryModal();
            } catch (error) {
                console.error('Failed to create directory:', error);
                showNotification('Failed to create directory', true);
            }
        });

        document.querySelector('#directoryModal .close').addEventListener('click', () => {
            hideDirectoryModal();
        });

        document.querySelector('#directoryModal .cancel-btn').addEventListener('click', () => {
            hideDirectoryModal();
        });

        document.getElementById('directoryModal').addEventListener('click', (e) => {
            if (e.target === e.currentTarget) {
                hideDirectoryModal();
            }
        });

        this.setupBulkActionListeners();
    }

    setupBulkActionListeners() {
        document.getElementById('selectAllBtn').addEventListener('click', () => {
            this.selectAllFiles();
        });

        document.getElementById('clearSelectionBtn').addEventListener('click', () => {
            this.clearFileSelection();
        });

        document.getElementById('bulkDeleteBtn').addEventListener('click', () => {
            this.bulkDeleteFiles();
        });

        document.getElementById('bulkMoveBtn').addEventListener('click', () => {
            if (this.selectedFiles.size === 0) return;
            showBulkMoveModal(this.directories);
        });

        document.getElementById('bulkTagBtn').addEventListener('click', () => {
            if (this.selectedFiles.size === 0) return;
            showBulkTagModal();
        });

        document.getElementById('bulkMoveForm').addEventListener('submit', (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            this.executeBulkMove(formData.get('directory'));
        });

        document.getElementById('bulkTagForm').addEventListener('submit', (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            this.executeBulkTag(formData.get('tags'));
        });

        document.querySelector('#bulkMoveModal .close').addEventListener('click', () => {
            hideBulkMoveModal();
        });
        document.querySelector('#bulkMoveModal .cancel-btn').addEventListener('click', () => {
            hideBulkMoveModal();
        });

        document.querySelector('#bulkTagModal .close').addEventListener('click', () => {
            hideBulkTagModal();
        });
        document.querySelector('#bulkTagModal .cancel-btn').addEventListener('click', () => {
            hideBulkTagModal();
        });

        document.getElementById('bulkMoveModal').addEventListener('click', (e) => {
            if (e.target === e.currentTarget) {
                hideBulkMoveModal();
            }
        });
        document.getElementById('bulkTagModal').addEventListener('click', (e) => {
            if (e.target === e.currentTarget) {
                hideBulkTagModal();
            }
        });
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

    async cancelTask(taskId) {
        try {
            await cancelTask(taskId);
        } catch (error) {
            console.error('Failed to cancel task:', error);
            showNotification('Failed to cancel task', true);
        }
    }

    async scanDirectories() {
        if (!this.selectedDirectory) {
            showNotification('Please select a directory first', true);
            return;
        }

        try {
            await scanDirectories(this.selectedDirectory.id);
            showNotification('Directory scan completed');
            await this.loadAndRenderFiles(this.selectedDirectory.id);
        } catch (error) {
            console.error('Failed to scan directory:', error);
            showNotification('Failed to scan directory', true);
        }
    }

    async searchFiles(query) {
        try {
            this.files = await searchFiles(query);
            this.renderFiles();
        } catch (error) {
            console.error('Failed to search files:', error);
            showNotification('Failed to search files', true);
        }
    }

    downloadFile(fileId) {
        try {
            downloadFile(fileId);
        } catch (error) {
            console.error('Failed to download file:', error);
            showNotification('Failed to download file', true);
        }
    }

    async deleteFile(fileId) {
        if (!confirm('Are you sure you want to delete this file?')) {
            return;
        }

        try {
            await deleteFile(fileId);
            showNotification('File deleted successfully');
            if (this.selectedDirectory) {
                await this.loadAndRenderFiles(this.selectedDirectory.id);
            } else {
                await this.loadAndRenderFiles();
            }
        } catch (error) {
            console.error('Failed to delete file:', error);
            showNotification('Failed to delete file', true);
        }
    }

    async bulkDeleteFiles() {
        if (this.selectedFiles.size === 0) return;

        const confirmMessage = `Are you sure you want to delete ${this.selectedFiles.size} files? This action cannot be undone.`;
        if (!confirm(confirmMessage)) return;

        try {
            await bulkDeleteFiles(Array.from(this.selectedFiles));
            showNotification(`${this.selectedFiles.size} files deleted successfully`);
            this.selectedFiles.clear();
            if (this.selectedDirectory) {
                await this.loadAndRenderFiles(this.selectedDirectory.id);
            } else {
                await this.loadAndRenderFiles();
            }
        } catch (error) {
            console.error('Failed to bulk delete files:', error);
            showNotification('Failed to delete files', true);
        }
    }

    async executeBulkMove(directoryId) {
        if (!directoryId || this.selectedFiles.size === 0) return;

        try {
            await executeBulkMove(Array.from(this.selectedFiles), directoryId);
            showNotification(`${this.selectedFiles.size} files moved successfully`);
            this.selectedFiles.clear();
            hideBulkMoveModal();
            if (this.selectedDirectory) {
                await this.loadAndRenderFiles(this.selectedDirectory.id);
            } else {
                await this.loadAndRenderFiles();
            }
        } catch (error) {
            console.error('Failed to bulk move files:', error);
            showNotification('Failed to move files', true);
        }
    }

    async executeBulkTag(tagsInput) {
        if (this.selectedFiles.size === 0) return;

        const tags = tagsInput ? tagsInput.split(',').map(tag => tag.trim()).filter(tag => tag) : [];

        try {
            await executeBulkTag(Array.from(this.selectedFiles), tags);
            showNotification(`${this.selectedFiles.size} files tagged successfully`);
            this.selectedFiles.clear();
            hideBulkTagModal();
            if (this.selectedDirectory) {
                await this.loadAndRenderFiles(this.selectedDirectory.id);
            } else {
                await this.loadAndRenderFiles();
            }
        } catch (error) {
            console.error('Failed to bulk tag files:', error);
            showNotification('Failed to tag files', true);
        }
    }
}

// Initialize the application
let commander;
document.addEventListener('DOMContentLoaded', () => {
    commander = new Commander();
    window.commander = commander; // Make it accessible for inline event handlers
});