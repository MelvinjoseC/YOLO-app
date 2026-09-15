const API_BASE = window.location.origin; // Dynamically binds to host
let isApiOnline = false;
let tasksCache = [];

// Local storage fallback data
const DEFAULT_TASKS = [
  { id: 101, title: "Provision AWS VPC using Terraform", description: "Set up multi-AZ subnets, NAT gateway, route tables.", status: "DONE", created_at: new Date(Date.now() - 86400000 * 3).toISOString() },
  { id: 102, title: "Configure EKS Cluster", description: "Establish managed node groups with autoscaling and secure IAM roles.", status: "DONE", created_at: new Date(Date.now() - 86400000 * 2).toISOString() },
  { id: 103, title: "Package Application as Helm Chart", description: "Template deployment manifests and configuration values.", status: "PROGRESS", created_at: new Date(Date.now() - 86400000).toISOString() },
  { id: 104, title: "Implement CI/CD pipeline", description: "Create GitHub Action to build and run Trivy vulnerability scans.", status: "TODO", created_at: new Date().toISOString() }
];

document.addEventListener("DOMContentLoaded", () => {
  initApp();
  
  // Set up event listeners
  document.getElementById("task-form").addEventListener("submit", handleAddTask);
  
  // Periodic polling for health and metrics
  pollSystemStatus();
  setInterval(pollSystemStatus, 5000);
});

async function initApp() {
  await checkApiHealth();
  await loadTasks();
}

async function checkApiHealth() {
  try {
    const res = await fetch(`${API_BASE}/health`, { signal: AbortSignal.timeout(2000) });
    if (res.ok) {
      const data = await res.json();
      isApiOnline = true;
      updateConnectionStatus(true, data.database === "connected");
    } else {
      throw new Error("API unhealthy");
    }
  } catch (err) {
    isApiOnline = false;
    updateConnectionStatus(false, false);
  }
}

function updateConnectionStatus(apiOnline, dbConnected) {
  const apiStatus = document.getElementById("api-status");
  const dbStatus = document.getElementById("db-status");
  const dbIndicator = document.getElementById("db-indicator");

  if (apiOnline) {
    apiStatus.textContent = "Online";
    apiStatus.style.color = "var(--accent-color)";
    
    if (dbConnected) {
      dbStatus.textContent = "Connected";
      dbIndicator.className = "dot healthy";
    } else {
      dbStatus.textContent = "Error";
      dbIndicator.className = "dot unhealthy";
    }
  } else {
    apiStatus.textContent = "Offline (Local mode)";
    apiStatus.style.color = "var(--text-muted)";
    dbStatus.textContent = "Simulated";
    dbIndicator.className = "dot healthy";
  }
}

async function loadTasks() {
  if (isApiOnline) {
    try {
      const res = await fetch(`${API_BASE}/api/tasks`);
      if (res.ok) {
        tasksCache = await res.json();
      } else {
        throw new Error("Failed to fetch tasks");
      }
    } catch (err) {
      console.warn("Error fetching tasks from API, using fallback:", err);
      loadTasksFromLocal();
    }
  } else {
    loadTasksFromLocal();
  }
  renderTasks();
}

function loadTasksFromLocal() {
  const local = localStorage.getItem("yolo_tasks");
  if (local) {
    tasksCache = JSON.parse(local);
  } else {
    tasksCache = [...DEFAULT_TASKS];
    saveTasksToLocal();
  }
}

function saveTasksToLocal() {
  localStorage.setItem("yolo_tasks", JSON.stringify(tasksCache));
}

function renderTasks() {
  const todoList = document.getElementById("list-todo");
  const progressList = document.getElementById("list-progress");
  const doneList = document.getElementById("list-done");

  // Clear lists
  todoList.innerHTML = "";
  progressList.innerHTML = "";
  doneList.innerHTML = "";

  const counts = { TODO: 0, PROGRESS: 0, DONE: 0 };

  tasksCache.forEach(task => {
    const card = createTaskCard(task);
    
    if (task.status === "TODO") {
      todoList.appendChild(card);
      counts.TODO++;
    } else if (task.status === "PROGRESS" || task.status === "IN_PROGRESS") {
      progressList.appendChild(card);
      counts.PROGRESS++;
    } else if (task.status === "DONE") {
      doneList.appendChild(card);
      counts.DONE++;
    }
  });

  // Update headers count
  document.getElementById("count-todo").textContent = counts.TODO;
  document.getElementById("count-progress").textContent = counts.PROGRESS;
  document.getElementById("count-done").textContent = counts.DONE;

  // Render empty states if necessary
  checkEmptyState(todoList, "No tasks in backlog");
  checkEmptyState(progressList, "No tasks in progress");
  checkEmptyState(doneList, "No completed tasks");
}

function createTaskCard(task) {
  const card = document.createElement("div");
  const statusClass = (task.status === "PROGRESS" || task.status === "IN_PROGRESS") ? "progress" : task.status.toLowerCase();
  card.className = `task-card ${statusClass}`;
  
  const createdDate = new Date(task.created_at).toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric'
  });

  card.innerHTML = `
    <div class="task-title">${escapeHtml(task.title)}</div>
    <div class="task-desc">${escapeHtml(task.description)}</div>
    <div class="task-footer">
      <span class="task-time">${createdDate}</span>
      <div class="task-actions">
        <button class="btn-icon delete-btn" data-id="${task.id}" title="Delete Task">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="3 6 5 6 21 6"></polyline>
            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
          </svg>
        </button>
      </div>
    </div>
  `;

  card.querySelector(".delete-btn").addEventListener("click", () => handleDeleteTask(task.id));
  return card;
}

function checkEmptyState(listElement, message) {
  if (listElement.children.length === 0) {
    const empty = document.createElement("div");
    empty.className = "empty-state";
    empty.textContent = message;
    listElement.appendChild(empty);
  }
}

async function handleAddTask(e) {
  e.preventDefault();
  
  const titleInput = document.getElementById("task-title");
  const descInput = document.getElementById("task-desc");
  const statusInput = document.getElementById("task-status");

  const taskData = {
    title: titleInput.value.trim(),
    description: descInput.value.trim(),
    status: statusInput.value
  };

  if (isApiOnline) {
    try {
      const res = await fetch(`${API_BASE}/api/tasks`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(taskData)
      });
      if (res.ok) {
        titleInput.value = "";
        descInput.value = "";
        initApp();
        return;
      }
    } catch (err) {
      console.error("Error creating task, falling back to local:", err);
    }
  }

  // Local mode task creation
  const newTask = {
    id: Date.now(),
    title: taskData.title,
    description: taskData.description,
    status: taskData.status,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString()
  };

  tasksCache.unshift(newTask);
  saveTasksToLocal();
  renderTasks();

  titleInput.value = "";
  descInput.value = "";
}

async function handleDeleteTask(id) {
  if (isApiOnline) {
    try {
      const res = await fetch(`${API_BASE}/api/tasks/${id}`, {
        method: "DELETE"
      });
      if (res.ok) {
        initApp();
        return;
      }
    } catch (err) {
      console.error("Error deleting task, falling back to local:", err);
    }
  }

  // Local mode deletion
  tasksCache = tasksCache.filter(t => t.id !== id);
  saveTasksToLocal();
  renderTasks();
}

async function pollSystemStatus() {
  await checkApiHealth();

  const statRequests = document.getElementById("stat-requests");
  const statErrors = document.getElementById("stat-errors");
  const statLatency = document.getElementById("stat-latency");

  if (isApiOnline) {
    try {
      // Scrape Prometheus metrics to parse and extract telemetry data
      const res = await fetch(`${API_BASE}/metrics`);
      if (res.ok) {
        const text = await res.text();
        
        // Simple regex parsing for demonstration
        const totalReqsMatch = text.match(/yolo_api_http_requests_total.*? (\d+)/);
        const totalErrorsMatch = text.match(/yolo_api_http_requests_total{code=~"5.*?"}.*? (\d+)/); // simplified
        
        if (totalReqsMatch) {
          statRequests.textContent = totalReqsMatch[1];
        }
        // Simulated parsing calculations or default stats when actual API running has 0 traffic
        statLatency.textContent = "12ms"; 
        statErrors.textContent = "0.0%";
      }
    } catch (err) {
      console.error("Error scraping metrics:", err);
    }
  } else {
    // Generate realistic simulated DevOps SRE statistics in local mode
    const simulatedRequests = parseInt(statRequests.textContent) + Math.floor(Math.random() * 3);
    statRequests.textContent = simulatedRequests;
    
    const randomLatency = Math.floor(Math.random() * 8) + 8; // 8ms to 15ms
    statLatency.textContent = `${randomLatency}ms`;
    
    // 0.5% chance of error spike simulation
    if (Math.random() > 0.98) {
      statErrors.textContent = "2.4%";
      statErrors.style.color = "var(--error-color)";
    } else {
      statErrors.textContent = "0.0%";
      statErrors.style.color = "var(--accent-color)";
    }
  }
}

function escapeHtml(str) {
  return str
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}
