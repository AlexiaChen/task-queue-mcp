// API client
const api = {
    async get(url) {
        const res = await fetch(url);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
    },

    async post(url, data) {
        const res = await fetch(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || `HTTP ${res.status}`);
        }
        return res.json();
    },

    async patch(url, data) {
        const res = await fetch(url, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || `HTTP ${res.status}`);
        }
        return res.json();
    },

    async put(url, data) {
        const res = await fetch(url, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || `HTTP ${res.status}`);
        }
        return res.json();
    },

    async delete(url) {
        const res = await fetch(url, { method: 'DELETE' });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.status === 204 ? null : res.json();
    }
};

// State
let currentQueueId = null;
let autoRefreshTimer = null;
const AUTO_REFRESH_INTERVAL = 5000; // 5 seconds

// DOM Elements
const queuesSection = document.getElementById('queues-section');
const queueDetailSection = document.getElementById('queue-detail-section');
const queuesList = document.getElementById('queues-list');
const tasksList = document.getElementById('tasks-list');
const queueTitle = document.getElementById('queue-title');
const queueStats = document.getElementById('queue-stats');
const modal = document.getElementById('modal');
const modalTitle = document.getElementById('modal-title');
const modalForm = document.getElementById('modal-form');
const modalSubmit = document.getElementById('modal-submit');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadQueues();

    // Event listeners
    document.getElementById('refresh-btn').addEventListener('click', () => {
        if (currentQueueId) {
            loadQueueDetail(currentQueueId);
        } else {
            loadQueues();
        }
    });

    document.getElementById('add-queue-btn').addEventListener('click', () => showQueueModal());
    document.getElementById('add-task-btn').addEventListener('click', () => showTaskModal());
    document.getElementById('back-btn').addEventListener('click', showQueuesList);
    document.getElementById('delete-queue-btn').addEventListener('click', deleteCurrentQueue);

    // Modal close
    document.querySelector('.modal-close').addEventListener('click', hideModal);
    document.querySelector('.modal-cancel').addEventListener('click', hideModal);
    modal.addEventListener('click', (e) => {
        if (e.target === modal) hideModal();
    });
});

// Queue functions
async function loadQueues() {
    try {
        const queues = await api.get('/api/projects');
        renderQueues(queues);
    } catch (err) {
        console.error('Failed to load projects:', err);
        alert('Failed to load projects: ' + err.message);
    }
}

function renderQueues(queues) {
    if (queues.length === 0) {
        queuesList.innerHTML = `
            <div class="empty-state">
                <p>No projects yet. Create your first project to get started!</p>
            </div>
        `;
        return;
    }

    queuesList.innerHTML = queues.map(q => `
        <div class="queue-card" onclick="showQueueDetail(${q.id})">
            <h3>${escapeHtml(q.name)}</h3>
            ${q.description ? `<p>${escapeHtml(q.description)}</p>` : ''}
            <div class="queue-stats-mini">
                <span class="stat-badge pending">${q.stats?.pending || 0} pending</span>
                <span class="stat-badge doing">${q.stats?.doing || 0} doing</span>
                <span class="stat-badge finished">${q.stats?.finished || 0} finished</span>
            </div>
        </div>
    `).join('');
}

async function showQueueDetail(queueId) {
    currentQueueId = queueId;
    queuesSection.classList.add('hidden');
    queueDetailSection.classList.remove('hidden');

    await loadQueueDetail(queueId);
    startAutoRefresh();
}

async function loadQueueDetail(queueId) {
    try {
        const [queue, tasks] = await Promise.all([
            api.get(`/api/projects/${queueId}`),
            api.get(`/api/projects/${queueId}/issues`)
        ]);

        queueTitle.textContent = queue.name;
        renderQueueStats(queue.stats);
        renderTasks(tasks);
    } catch (err) {
        console.error('Failed to load project:', err);
        alert('Failed to load project: ' + err.message);
    }
}

function renderQueueStats(stats) {
    queueStats.innerHTML = `
        <div class="stat-item">
            <span class="label">Total</span>
            <span class="value">${stats?.total || 0}</span>
        </div>
        <div class="stat-item">
            <span class="label">Pending</span>
            <span class="value">${stats?.pending || 0}</span>
        </div>
        <div class="stat-item">
            <span class="label">Doing</span>
            <span class="value">${stats?.doing || 0}</span>
        </div>
        <div class="stat-item">
            <span class="label">Finished</span>
            <span class="value">${stats?.finished || 0}</span>
        </div>
    `;
}

function renderTasks(tasks) {
    if (tasks.length === 0) {
        tasksList.innerHTML = `
            <div class="empty-state">
                <p>No issues in this project. Add your first issue!</p>
            </div>
        `;
        return;
    }

    tasksList.innerHTML = tasks.map(task => `
        <div class="task-card ${task.status}">
            <div class="task-info">
                <h4>${escapeHtml(task.title)} ${task.priority > 0 ? `<span class="priority-high">⚡${task.priority}</span>` : ''}</h4>
                ${task.description ? `<p>${escapeHtml(task.description)}</p>` : ''}
                <div class="task-meta">
                    <span>ID: ${task.id}</span>
                    <span>Position: ${task.position}</span>
                    <span>Created: ${formatDate(task.created_at)}</span>
                </div>
            </div>
            <span class="task-status ${task.status}">${task.status}</span>
            <div class="task-actions">
                ${task.status === 'pending' ? `
                    <button class="btn btn-small btn-secondary" onclick="editTask(${task.id})">✏️ Edit</button>
                    <button class="btn btn-small btn-secondary" onclick="prioritizeTask(${task.id})">⬆️ Prioritize</button>
                ` : ''}
                <button class="btn btn-small btn-danger" onclick="deleteTask(${task.id})">Delete</button>
            </div>
        </div>
    `).join('');
}

function showQueuesList() {
    stopAutoRefresh();
    currentQueueId = null;
    queueDetailSection.classList.add('hidden');
    queuesSection.classList.remove('hidden');
    loadQueues();
}

function startAutoRefresh() {
    stopAutoRefresh();
    autoRefreshTimer = setInterval(() => {
        if (currentQueueId) {
            loadQueueDetail(currentQueueId);
        }
    }, AUTO_REFRESH_INTERVAL);
}

function stopAutoRefresh() {
    if (autoRefreshTimer !== null) {
        clearInterval(autoRefreshTimer);
        autoRefreshTimer = null;
    }
}

// Task actions
async function editTask(taskId) {
    try {
        const task = await api.get(`/api/issues/${taskId}`);
        showTaskModal(task);
    } catch (err) {
        alert('Failed to load issue: ' + err.message);
    }
}

async function prioritizeTask(taskId) {
    try {
        await api.post(`/api/issues/${taskId}/prioritize`, { position: 1 });
        loadQueueDetail(currentQueueId);
    } catch (err) {
        alert('Failed to prioritize issue: ' + err.message);
    }
}

async function deleteTask(taskId) {
    if (!confirm('Are you sure you want to delete this issue?')) return;

    try {
        await api.delete(`/api/issues/${taskId}`);
        loadQueueDetail(currentQueueId);
    } catch (err) {
        alert('Failed to delete issue: ' + err.message);
    }
}

async function deleteCurrentQueue() {
    if (!confirm('Are you sure you want to delete this project and all its issues?')) return;

    try {
        await api.delete(`/api/projects/${currentQueueId}`);
        showQueuesList();
    } catch (err) {
        alert('Failed to delete project: ' + err.message);
    }
}

// Modal functions
function showQueueModal(queue = null) {
    modalTitle.textContent = queue ? 'Edit Project' : 'Create Project';
    modalForm.innerHTML = `
        <div class="form-group">
            <label for="name">Name *</label>
            <input type="text" id="name" name="name" required value="${queue?.name || ''}">
        </div>
        <div class="form-group">
            <label for="description">Description</label>
            <textarea id="description" name="description" rows="3">${queue?.description || ''}</textarea>
        </div>
    `;

    modalSubmit.onclick = async () => {
        const formData = new FormData(modalForm);
        const data = {
            name: formData.get('name'),
            description: formData.get('description')
        };

        try {
            await api.post('/api/projects', data);
            hideModal();
            loadQueues();
        } catch (err) {
            alert('Failed to create project: ' + err.message);
        }
    };

    showModal();
}

function showTaskModal(task = null) {
    modalTitle.textContent = task ? 'Edit Issue' : 'Create Issue';
    modalForm.innerHTML = `
        <div class="form-group">
            <label for="title">Title *</label>
            <input type="text" id="title" name="title" required value="${task ? escapeHtml(task.title) : ''}">
        </div>
        <div class="form-group">
            <label for="description">Description</label>
            <textarea id="description" name="description" rows="3">${task ? escapeHtml(task.description || '') : ''}</textarea>
        </div>
        <div class="form-group">
            <label for="priority">Priority</label>
            <input type="number" id="priority" name="priority" value="${task ? task.priority : 0}" min="0">
        </div>
    `;

    modalSubmit.onclick = async () => {
        const formData = new FormData(modalForm);

        try {
            if (task) {
                await api.put(`/api/issues/${task.id}`, {
                    title: formData.get('title'),
                    description: formData.get('description'),
                    priority: parseInt(formData.get('priority')) || 0
                });
            } else {
                await api.post('/api/issues', {
                    queue_id: currentQueueId,
                    title: formData.get('title'),
                    description: formData.get('description'),
                    priority: parseInt(formData.get('priority')) || 0
                });
            }
            hideModal();
            loadQueueDetail(currentQueueId);
        } catch (err) {
            alert((task ? 'Failed to update issue: ' : 'Failed to create issue: ') + err.message);
        }
    };

    showModal();
}

function showModal() {
    modal.classList.remove('hidden');
}

function hideModal() {
    modal.classList.add('hidden');
}

// Utility functions
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatDate(dateStr) {
    const date = new Date(dateStr);
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
}
