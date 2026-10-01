const state = {
    monitors: [],
    editingId: null,
};

const monitorModal = document.querySelector("#monitor-modal");
const historyModal = document.querySelector("#history-modal");
const monitorForm = document.querySelector("#monitor-form");
const notice = document.querySelector("#notice");

document.addEventListener("DOMContentLoaded", () => {
    document.querySelector("#add-monitor").addEventListener("click", openCreateModal);
    document.querySelector("#monitors-body").addEventListener("click", handleMonitorAction);
    monitorForm.addEventListener("submit", saveMonitor);
    document.querySelectorAll("[data-close]").forEach((button) => {
        button.addEventListener("click", () => closeModal(document.getElementById(button.dataset.close)));
    });
    document.querySelectorAll(".modal").forEach((modal) => {
        modal.addEventListener("click", (event) => {
            if (event.target === modal) {
                closeModal(modal);
            }
        });
    });
    document.addEventListener("keydown", (event) => {
        if (event.key === "Escape") {
            document.querySelectorAll(".modal:not(.hidden)").forEach(closeModal);
        }
    });
    loadMonitors();
});

async function request(path, options = {}) {
    const response = await fetch(path, {
        headers: {"Content-Type": "application/json"},
        ...options,
    });
    if (!response.ok) {
        const payload = await response.json().catch(() => ({}));
        throw new Error(payload.error || `Request failed with status ${response.status}`);
    }
    if (response.status === 204) {
        return null;
    }
    return response.json();
}

async function loadMonitors() {
    try {
        state.monitors = await request("/api/v1/monitors");
        renderMonitors();
    } catch (error) {
        showNotice(error.message, true);
    }
}

function renderMonitors() {
    const body = document.querySelector("#monitors-body");
    const emptyState = document.querySelector("#empty-state");
    body.replaceChildren();
    emptyState.classList.toggle("hidden", state.monitors.length !== 0);

    state.monitors.forEach((monitor) => {
        const row = document.createElement("tr");
        row.dataset.id = monitor.id;
        row.appendChild(textCell(monitor.name));

        const urlCell = document.createElement("td");
        const link = document.createElement("a");
        link.href = monitor.url;
        link.target = "_blank";
        link.rel = "noreferrer";
        link.textContent = monitor.url;
        urlCell.appendChild(link);
        row.appendChild(urlCell);

        const enabledCell = document.createElement("td");
        enabledCell.textContent = monitor.enabled ? "Yes" : "No";
        row.appendChild(enabledCell);

        const statusCell = document.createElement("td");
        const status = document.createElement("span");
        status.className = `status status-${monitor.status}`;
        status.textContent = monitor.status.toUpperCase();
        statusCell.appendChild(status);
        row.appendChild(statusCell);

        row.appendChild(textCell(monitor.last_latency_ms === null ? "—" : `${monitor.last_latency_ms} ms`));
        row.appendChild(textCell(monitor.last_checked_at === null ? "—" : formatDate(monitor.last_checked_at)));

        const actions = document.createElement("td");
        actions.className = "actions";
        actions.appendChild(actionButton("Edit", "edit", monitor.id));
        actions.appendChild(actionButton("Delete", "delete", monitor.id, "button-danger"));
        actions.appendChild(actionButton("Check now", "check", monitor.id));
        actions.appendChild(actionButton("History", "history", monitor.id));
        actions.appendChild(actionButton(monitor.enabled ? "Disable" : "Enable", "toggle", monitor.id));
        row.appendChild(actions);
        body.appendChild(row);
    });
}

function textCell(value) {
    const cell = document.createElement("td");
    cell.textContent = value;
    return cell;
}

function actionButton(label, action, id, extraClass = "") {
    const button = document.createElement("button");
    button.className = `table-button ${extraClass}`;
    button.type = "button";
    button.dataset.action = action;
    button.dataset.id = id;
    button.textContent = label;
    return button;
}

async function handleMonitorAction(event) {
    const button = event.target.closest("button[data-action]");
    if (!button) {
        return;
    }

    const monitor = state.monitors.find((item) => item.id === button.dataset.id);
    if (!monitor) {
        return;
    }

    switch (button.dataset.action) {
        case "edit":
            openEditModal(monitor);
            break;
        case "delete":
            await deleteMonitor(monitor);
            break;
        case "check":
            await checkMonitor(button, monitor);
            break;
        case "history":
            await openHistoryModal(monitor);
            break;
        case "toggle":
            await toggleMonitor(monitor);
            break;
    }
}

function openCreateModal() {
    state.editingId = null;
    monitorForm.reset();
    monitorForm.elements.enabled.checked = true;
    document.querySelector("#monitor-modal-title").textContent = "Add monitor";
    document.querySelector("#monitor-submit").textContent = "Create";
    document.querySelector("#monitor-form-error").textContent = "";
    openModal(monitorModal);
}

function openEditModal(monitor) {
    state.editingId = monitor.id;
    monitorForm.elements.name.value = monitor.name;
    monitorForm.elements.url.value = monitor.url;
    monitorForm.elements.interval_seconds.value = monitor.interval_seconds;
    monitorForm.elements.timeout_seconds.value = monitor.timeout_seconds;
    monitorForm.elements.enabled.checked = monitor.enabled;
    document.querySelector("#monitor-modal-title").textContent = "Edit monitor";
    document.querySelector("#monitor-submit").textContent = "Save";
    document.querySelector("#monitor-form-error").textContent = "";
    openModal(monitorModal);
}

async function saveMonitor(event) {
    event.preventDefault();
    const formData = new FormData(monitorForm);
    const payload = {
        name: formData.get("name"),
        url: formData.get("url"),
        interval_seconds: Number(formData.get("interval_seconds")),
        timeout_seconds: Number(formData.get("timeout_seconds")),
        enabled: formData.get("enabled") === "on",
    };

    const path = state.editingId ? `/api/v1/monitors/${state.editingId}` : "/api/v1/monitors";
    const method = state.editingId ? "PUT" : "POST";
    try {
        await request(path, {method, body: JSON.stringify(payload)});
        closeModal(monitorModal);
        await loadMonitors();
        showNotice(state.editingId ? "Monitor updated" : "Monitor created", false);
    } catch (error) {
        document.querySelector("#monitor-form-error").textContent = error.message;
    }
}

async function deleteMonitor(monitor) {
    if (!window.confirm(`Delete monitor “${monitor.name}”?`)) {
        return;
    }
    try {
        await request(`/api/v1/monitors/${monitor.id}`, {method: "DELETE"});
        await loadMonitors();
        showNotice("Monitor deleted", false);
    } catch (error) {
        showNotice(error.message, true);
    }
}

async function toggleMonitor(monitor) {
    try {
        await request(`/api/v1/monitors/${monitor.id}`, {
            method: "PUT",
            body: JSON.stringify({
                name: monitor.name,
                url: monitor.url,
                interval_seconds: monitor.interval_seconds,
                timeout_seconds: monitor.timeout_seconds,
                enabled: !monitor.enabled,
            }),
        });
        await loadMonitors();
    } catch (error) {
        showNotice(error.message, true);
    }
}

async function checkMonitor(button, monitor) {
    button.disabled = true;
    try {
        await request(`/api/v1/monitors/${monitor.id}/check`, {method: "POST"});
        await loadMonitors();
    } catch (error) {
        showNotice(error.message, true);
    } finally {
        button.disabled = false;
    }
}

async function openHistoryModal(monitor) {
    document.querySelector("#history-modal-title").textContent = `Check history — ${monitor.name}`;
    const body = document.querySelector("#history-body");
    const emptyState = document.querySelector("#history-empty");
    body.replaceChildren();
    emptyState.textContent = "Loading...";
    emptyState.classList.remove("hidden");
    openModal(historyModal);

    try {
        const checks = await request(`/api/v1/monitors/${monitor.id}/checks?limit=100`);
        renderHistory(checks);
    } catch (error) {
        emptyState.textContent = error.message;
    }
}

function renderHistory(checks) {
    const body = document.querySelector("#history-body");
    const emptyState = document.querySelector("#history-empty");
    body.replaceChildren();
    emptyState.textContent = "No checks yet.";
    emptyState.classList.toggle("hidden", checks.length !== 0);

    checks.forEach((check) => {
        const row = document.createElement("tr");
        row.appendChild(textCell(formatDate(check.checked_at)));
        const statusCell = document.createElement("td");
        const status = document.createElement("span");
        status.className = `status status-${check.status}`;
        status.textContent = check.status.toUpperCase();
        statusCell.appendChild(status);
        row.appendChild(statusCell);
        row.appendChild(textCell(check.status_code === null ? "—" : String(check.status_code)));
        row.appendChild(textCell(`${check.latency_ms} ms`));
        row.appendChild(textCell(check.error || "—"));
        body.appendChild(row);
    });
}

function openModal(modal) {
    modal.classList.remove("hidden");
}

function closeModal(modal) {
    modal.classList.add("hidden");
}

function formatDate(value) {
    return new Date(value).toLocaleString();
}

function showNotice(message, isError) {
    notice.textContent = message;
    notice.className = `notice ${isError ? "notice-error" : "notice-success"}`;
    window.setTimeout(() => {
        notice.textContent = "";
        notice.className = "notice";
    }, 4000);
}
