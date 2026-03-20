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

    async delete(url) {
        const res = await fetch(url, { method: 'DELETE' });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.status === 204 ? null : res.json();
    }
};

// State
let currentQueueId = null;

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
        const queues = await api.get('/api/queues');
        renderQueues(queues);
    } catch (err) {
        console.error('Failed to load queues:', err);
        alert('Failed to load queues: ' + err.message);
    }
}

function renderQueues(queues) {
    if (queues.length === 0) {
        queuesList.innerHTML = `
            <div class="empty-state">
                <p>No queues yet. Create your first queue to get started!</p>
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
}

async function loadQueueDetail(queueId) {
    try {
        const [queue, tasks] = await Promise.all([
            api.get(`/api/queues/${queueId}`),
            api.get(`/api/queues/${queueId}/tasks`)
        ]);

        queueTitle.textContent = queue.name;
        renderQueueStats(queue.stats);
        renderTasks(tasks);
    } catch (err) {
        console.error('Failed to load queue:', err);
        alert('Failed to load queue: ' + err.message);
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
                <p>No tasks in this queue. Add your first task!</p>
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
                    <button class="btn btn-small btn-primary" onclick="startTask(${task.id})">Start</button>
                    <button class="btn btn-small btn-secondary" onclick="prioritizeTask(${task.id})">⬆️ Prioritize</button>
                ` : ''}
                ${task.status === 'doing' ? `
                    <button class="btn btn-small btn-primary" onclick="finishTask(${task.id})">Finish</button>
                ` : ''}
                ${task.status === 'finished' ? `
                    <button class="btn btn-small btn-secondary" onclick="resetTask(${task.id})">Reset</button>
                ` : ''}
                <button class="btn btn-small btn-danger" onclick="deleteTask(${task.id})">Delete</button>
            </div>
        </div>
    `).join('');
}

function showQueuesList() {
    currentQueueId = null;
    queueDetailSection.classList.add('hidden');
    queuesSection.classList.remove('hidden');
    loadQueues();
}

// Task actions
async function startTask(taskId) {
    try {
        await api.post(`/api/tasks/${taskId}/start`);
        loadQueueDetail(currentQueueId);
    } catch (err) {
        alert('Failed to start task: ' + err.message);
    }
}

async function finishTask(taskId) {
    try {
        await api.post(`/api/tasks/${taskId}/finish`);
        loadQueueDetail(currentQueueId);
    } catch (err) {
        alert('Failed to finish task: ' + err.message);
    }
}

async function resetTask(taskId) {
    try {
        await api.patch(`/api/tasks/${taskId}`, { status: 'pending' });
        loadQueueDetail(currentQueueId);
    } catch (err) {
        alert('Failed to reset task: ' + err.message);
    }
}

async function prioritizeTask(taskId) {
    try {
        await api.post(`/api/tasks/${taskId}/prioritize`, { position: 1 });
        loadQueueDetail(currentQueueId);
    } catch (err) {
        alert('Failed to prioritize task: ' + err.message);
    }
}

async function deleteTask(taskId) {
    if (!confirm('Are you sure you want to delete this task?')) return;

    try {
        await api.delete(`/api/tasks/${taskId}`);
        loadQueueDetail(currentQueueId);
    } catch (err) {
        alert('Failed to delete task: ' + err.message);
    }
}

async function deleteCurrentQueue() {
    if (!confirm('Are you sure you want to delete this queue and all its tasks?')) return;

    try {
        await api.delete(`/api/queues/${currentQueueId}`);
        showQueuesList();
    } catch (err) {
        alert('Failed to delete queue: ' + err.message);
    }
}

// Modal functions
function showQueueModal(queue = null) {
    modalTitle.textContent = queue ? 'Edit Queue' : 'Create Queue';
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
            await api.post('/api/queues', data);
            hideModal();
            loadQueues();
        } catch (err) {
            alert('Failed to create queue: ' + err.message);
        }
    };

    showModal();
}

function showTaskModal(task = null) {
    modalTitle.textContent = task ? 'Edit Task' : 'Create Task';
    modalForm.innerHTML = `
        <div class="form-group">
            <label for="title">Title *</label>
            <input type="text" id="title" name="title" required value="${task?.title || ''}">
        </div>
        <div class="form-group">
            <label for="description">Description</label>
            <textarea id="description" name="description" rows="3">${task?.description || ''}</textarea>
        </div>
        <div class="form-group">
            <label for="priority">Priority</label>
            <input type="number" id="priority" name="priority" value="${task?.priority || 0}" min="0">
        </div>
    `;

    modalSubmit.onclick = async () => {
        const formData = new FormData(modalForm);
        const data = {
            queue_id: currentQueueId,
            title: formData.get('title'),
            description: formData.get('description'),
            priority: parseInt(formData.get('priority')) || 0
        };

        try {
            await api.post('/api/tasks', data);
            hideModal();
            loadQueueDetail(currentQueueId);
        } catch (err) {
            alert('Failed to create task: ' + err.message);
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
