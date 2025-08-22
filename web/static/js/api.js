
export async function loadTasks() {
    const response = await fetch('/api/tasks');
    return await response.json();
}

export async function loadTools() {
    const response = await fetch('/api/tools');
    return await response.json();
}

export async function loadStats() {
    const response = await fetch('/api/stats');
    return await response.json();
}

export async function loadDirectories() {
    const response = await fetch('/api/directories');
    return await response.json();
}

export async function createDirectory(formData) {
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
    if (!response.ok) throw new Error('Failed to create directory');
}

export async function scanDirectories(directoryId) {
    const response = await fetch(`/api/directories/${directoryId}/scan`, {
        method: 'POST'
    });
    if (!response.ok) throw new Error('Failed to scan directory');
}

export async function searchFiles(query) {
    const response = await fetch(`/api/files/search?q=${encodeURIComponent(query)}`);
    return await response.json();
}

export function downloadFile(fileId) {
    window.open(`/api/files/${fileId}/download`, '_blank');
}

export async function deleteFile(fileId) {
    const response = await fetch(`/api/files/${fileId}`, {
        method: 'DELETE'
    });
    if (!response.ok) throw new Error('Failed to delete file');
}

export async function bulkDeleteFiles(fileIds) {
    const response = await fetch('/api/files/bulk/delete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ file_ids: fileIds })
    });
    if (!response.ok) throw new Error('Failed to delete files');
}

export async function executeBulkMove(fileIds, directoryId) {
    const response = await fetch('/api/files/bulk/move', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            file_ids: fileIds,
            directory_id: directoryId
        })
    });
    if (!response.ok) throw new Error('Failed to move files');
}

export async function executeBulkTag(fileIds, tags) {
    const response = await fetch('/api/files/bulk/tag', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            file_ids: fileIds,
            tags: tags
        })
    });
    if (!response.ok) throw new Error('Failed to tag files');
}

export async function cancelTask(taskId) {
    const response = await fetch(`/api/tasks/${taskId}/cancel`, {
        method: 'POST',
    });
    if (!response.ok) throw new Error('Failed to cancel task');
}

export async function createTask(tool, args) {
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
    if (!response.ok) throw new Error('Failed to create task');
}

export async function loadFiles(directoryId = null) {
    let url = '/api/files';
    if (directoryId) {
        url += `?directory_id=${directoryId}`;
    }
    const response = await fetch(url);
    return await response.json();
}
