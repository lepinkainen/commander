// Commander Web Interface
class Commander {
    constructor() {
        this.ws = null;
        this.tasks = new Map();
        this.currentFilter = 'all';
        this.init();
    }

    async init() {
        await this.loadTools();
        await this.loadTasks();
        await this.loadStats();
        this.connectWebSocket();
        this.setupEventListeners();
        
        // Refresh stats every 5 seconds
        setInterval(() => this.loadStats(), 5000);
    }

    async loadTools() {
        try {
            const response = await fetch('/api/tools');
            const tools = await response.json();
            
            const select = document.getElementById('tool');
            select.innerHTML = '';
            
            tools.forEach(tool => {
                const option = document.createElement('option');
                option.value = tool.name;
                option.textContent = `${tool.name} - ${tool.description}`;
                select.appendChild(option);
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
                    // Task will be added via WebSocket message
                } else {
                    alert('Failed to create task');
                }
            } catch (error) {
                console.error('Failed to create task:', error);
                alert('Failed to create task');
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
        
        div.innerHTML = `
            <div class="task-header">
                <span class="task-tool">${task.tool}</span>
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
}

// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    new Commander();
});
