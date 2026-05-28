// Global State
let selectedNode = null;
let nodesCache = [];
let monitorInterval = null;

// DOM Elements
const syncRepoUrlInput = document.getElementById('sync-repo-url');
const btnSync = document.getElementById('btn-sync');
const syncStatusMsg = document.getElementById('sync-status-msg');

const nodesListContainer = document.getElementById('nodes-list');
const nodeActionsArea = document.getElementById('node-actions-area');
const nodeActionsPlaceholder = document.getElementById('node-actions-placeholder');
const selectedNodeName = document.getElementById('selected-node-name');
const btnActionKeygen = document.getElementById('btn-action-keygen');
const btnActionBootstrap = document.getElementById('btn-action-bootstrap');
const bootstrapForm = document.getElementById('bootstrap-form');
const nodeSshPasswordInput = document.getElementById('node-ssh-password');
const btnSubmitBootstrap = document.getElementById('btn-submit-bootstrap');
const nodeActionsStatus = document.getElementById('node-actions-status');

const catalogGrid = document.getElementById('catalog-grid');
const btnRefreshCatalog = document.getElementById('btn-refresh-catalog');

const monitorNodeSelect = document.getElementById('monitor-node-select');
const healthDashboard = document.getElementById('health-dashboard');

const terminalConsole = document.getElementById('terminal-console');
const btnClearConsole = document.getElementById('btn-clear-console');

// Modal Elements
const deployModal = document.getElementById('deploy-modal');
const deployModalAppInfo = document.getElementById('deploy-modal-app-info');
const deployNodeSelect = document.getElementById('deploy-node-select');
const deployComposePath = document.getElementById('deploy-compose-path');
const btnModalCancel = document.getElementById('btn-modal-cancel');
const btnModalSubmit = document.getElementById('btn-modal-submit');
let appToDeploy = null;

// Router Elements
const routerStatusIndicator = document.getElementById('router-status-indicator');
const routerStatusText = document.getElementById('router-status-text');
const btnRouterStart = document.getElementById('btn-router-start');
const btnRouterStop = document.getElementById('btn-router-stop');
const btnRouterReload = document.getElementById('btn-router-reload');
const routerControlStatus = document.getElementById('router-control-status');

const routeDomainInput = document.getElementById('route-domain');
const routeNodeSelect = document.getElementById('route-node');
const routePortInput = document.getElementById('route-port');
const btnAddRoute = document.getElementById('btn-add-route');
const routeCreateStatus = document.getElementById('route-create-status');
const routesListBody = document.getElementById('routes-list-body');

// Initialize on Load
window.addEventListener('DOMContentLoaded', () => {
  fetchNodes();
  fetchCatalog();
  setupEventListeners();
  setupTabNavigation();
  lucide.createIcons();
});

// Setup Notion Tab Navigation
function setupTabNavigation() {
  document.querySelectorAll('.nav-item').forEach(item => {
    item.addEventListener('click', (e) => {
      e.preventDefault();
      const tabName = item.getAttribute('data-tab');

      // Update active nav class
      document.querySelectorAll('.nav-item').forEach(el => el.classList.remove('active'));
      item.classList.add('active');

      // Show tab content
      document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.add('hidden');
      });
      document.getElementById(`tab-${tabName}`).classList.remove('hidden');

      // Sync and nodes loading refresh triggers
      if (tabName === 'nodes') {
        fetchNodes();
      } else if (tabName === 'catalog') {
        fetchCatalog();
      } else if (tabName === 'router') {
        fetchRouterStatus();
        fetchRouterRoutes();
      }
    });
  });
}

// Event Listeners Setup
function setupEventListeners() {
  btnSync.addEventListener('click', handleSyncCatalog);
  btnRefreshCatalog.addEventListener('click', fetchCatalog);
  btnClearConsole.addEventListener('click', () => {
    terminalConsole.innerHTML = '<div class="console-line system">Console cleared.</div>';
  });

  // Node Actions
  btnActionKeygen.addEventListener('click', handleKeyGen);
  btnActionBootstrap.addEventListener('click', () => {
    bootstrapForm.classList.toggle('hidden');
    nodeActionsStatus.className = 'status-msg';
    nodeActionsStatus.style.display = 'none';
  });
  btnSubmitBootstrap.addEventListener('click', handleBootstrapNode);

  // Monitor Dropdown Select
  monitorNodeSelect.addEventListener('change', (e) => {
    const node = e.target.value;
    if (monitorInterval) {
      clearInterval(monitorInterval);
      monitorInterval = null;
    }
    if (node) {
      fetchNodeStatus(node);
      // Poll node status every 5 seconds
      monitorInterval = setInterval(() => fetchNodeStatus(node), 5000);
    } else {
      healthDashboard.innerHTML = `
        <div class="placeholder-card notion-card">
          <i data-lucide="activity" class="lucide-big"></i>
          <p>Please select a node above to inspect active docker containers.</p>
        </div>
      `;
      lucide.createIcons();
    }
  });

  // Modal Cancel
  btnModalCancel.addEventListener('click', () => {
    deployModal.classList.add('hidden');
    appToDeploy = null;
  });

  // Modal Deploy Submit
  btnModalSubmit.addEventListener('click', handleDeploySubmit);

  // Router Controls
  btnRouterStart.addEventListener('click', handleRouterStart);
  btnRouterStop.addEventListener('click', handleRouterStop);
  btnRouterReload.addEventListener('click', handleRouterReload);
  btnAddRoute.addEventListener('click', handleCreateRoute);
}

// Helper: Log message to scrolling console
function logToConsole(message, type = 'log') {
  const line = document.createElement('div');
  line.className = `console-line ${type}`;
  line.textContent = `[${new Date().toLocaleTimeString()}] ${message}`;
  terminalConsole.appendChild(line);
  terminalConsole.scrollTop = terminalConsole.scrollHeight;
}

// Fetch Nodes
async function fetchNodes() {
  try {
    const response = await fetch('/api/nodes');
    const data = await response.json();
    if (response.ok && data.nodes) {
      nodesCache = data.nodes;
      renderNodesList();
      populateNodeDropdowns();
    } else {
      nodesListContainer.innerHTML = `<div class="status-msg error" style="display:block">Error: ${data.message || 'Failed to load nodes'}</div>`;
    }
  } catch (error) {
    nodesListContainer.innerHTML = `<div class="status-msg error" style="display:block">Network Error: ${error.message}</div>`;
  }
}

// Render Nodes List UI
function renderNodesList() {
  if (nodesCache.length === 0) {
    nodesListContainer.innerHTML = '<div class="placeholder-text" style="padding:1rem">No nodes configured. Run "init" or configure nodes.yaml</div>';
    return;
  }

  nodesListContainer.innerHTML = '';
  nodesCache.forEach(node => {
    const item = document.createElement('div');
    item.className = 'node-item';
    if (selectedNode && selectedNode.name === node.Name) {
      item.classList.add('selected');
    }

    item.innerHTML = `
      <div class="node-info-text">
        <h4>${node.Name}</h4>
        <span>${node.IP}</span>
      </div>
      <div class="node-connection-badge">ssh://${node.Username}</div>
    `;

    item.addEventListener('click', () => selectNode(node, item));
    nodesListContainer.appendChild(item);
  });
}

// Populate Dropdowns with nodes
function populateNodeDropdowns() {
  const currentMonitorValue = monitorNodeSelect.value;
  monitorNodeSelect.innerHTML = '<option value="">-- Select Node to Inspect --</option>';
  deployNodeSelect.innerHTML = '';
  routeNodeSelect.innerHTML = '';

  nodesCache.forEach(node => {
    // Monitor Dropdown
    const optMonitor = document.createElement('option');
    optMonitor.value = node.Name;
    optMonitor.textContent = node.Name;
    monitorNodeSelect.appendChild(optMonitor);

    // Deploy Dropdown
    const optDeploy = document.createElement('option');
    optDeploy.value = node.Name;
    optDeploy.textContent = node.Name;
    deployNodeSelect.appendChild(optDeploy);

    // Route Target Dropdown
    const optRoute = document.createElement('option');
    optRoute.value = node.Name;
    optRoute.textContent = node.Name;
    routeNodeSelect.appendChild(optRoute);
  });

  if (currentMonitorValue && nodesCache.some(n => n.Name === currentMonitorValue)) {
    monitorNodeSelect.value = currentMonitorValue;
  }
}

// Node Selection
function selectNode(node, element) {
  document.querySelectorAll('.node-item').forEach(el => el.classList.remove('selected'));

  if (selectedNode && selectedNode.Name === node.Name) {
    selectedNode = null;
    nodeActionsArea.classList.add('hidden');
    nodeActionsPlaceholder.classList.remove('hidden');
    bootstrapForm.classList.add('hidden');
  } else {
    selectedNode = node;
    element.classList.add('selected');
    selectedNodeName.textContent = `Configure Node: ${node.Name}`;
    nodeActionsArea.classList.remove('hidden');
    nodeActionsPlaceholder.classList.add('hidden');
    bootstrapForm.classList.add('hidden');
    nodeSshPasswordInput.value = '';
    nodeActionsStatus.style.display = 'none';
  }
}

// Fetch App Catalog
async function fetchCatalog() {
  try {
    catalogGrid.innerHTML = '<div class="loading-spinner">Loading catalog apps...</div>';
    const response = await fetch('/api/catalog');
    const data = await response.json();
    if (response.ok && data.apps) {
      renderCatalogGrid(data.apps);
    } else {
      catalogGrid.innerHTML = `<div class="status-msg error" style="display:block">Error: ${data.message || 'Failed to load catalog'}</div>`;
    }
  } catch (error) {
    catalogGrid.innerHTML = `<div class="status-msg error" style="display:block">Network Error: ${error.message}</div>`;
  }
}

// Render Catalog Grid UI
function renderCatalogGrid(apps) {
  if (apps.length === 0) {
    catalogGrid.innerHTML = '<div class="placeholder-card notion-card" style="grid-column: 1/-1"><i data-lucide="package" class="lucide-big"></i><p>Catalog is empty. Navigate to <strong>Settings & Sync</strong> to synchronize.</p></div>';
    lucide.createIcons();
    return;
  }

  catalogGrid.innerHTML = '';
  apps.forEach(app => {
    const card = document.createElement('div');
    card.className = 'app-card';

    const actionHtml = app.installed
      ? `<span class="app-status-badge healthy" style="font-size: 0.75rem;">Installed (${(app.installed_nodes || []).join(', ')})</span>`
      : `<button class="btn btn-primary btn-deploy" data-name="${app.name}">Deploy</button>`;

    card.innerHTML = `
      <div>
        <h3 class="app-card-title"><i data-lucide="package" class="app-card-icon"></i> ${app.title || app.name}</h3>
        <p class="app-card-desc">${app.description || 'No description available.'}</p>
      </div>
      <div class="app-card-meta">
        <span>v${app.version || '1.0.0'}</span>
        ${actionHtml}
      </div>
    `;

    if (!app.installed) {
      card.querySelector('.btn-deploy').addEventListener('click', () => openDeployModal(app));
    }
    catalogGrid.appendChild(card);
  });
  lucide.createIcons();
}

// Open Deploy Modal
function openDeployModal(app) {
  if (nodesCache.length === 0) {
    alert("Please register at least one node in nodes.yaml first!");
    return;
  }
  appToDeploy = app;
  deployModalAppInfo.innerHTML = `Deploying <strong>${app.title || app.name}</strong> to your cluster.`;
  deployComposePath.value = app.compose_path || '';
  const portsInput = document.getElementById('deploy-ports');
  if (portsInput) {
    portsInput.value = '';
  }
  deployModal.classList.remove('hidden');
}

// Sync Catalog Action
async function handleSyncCatalog() {
  const repo = syncRepoUrlInput.value.trim();
  if (!repo) {
    showStatusMsg(syncStatusMsg, 'Git repository URL is required', 'error');
    return;
  }

  showStatusMsg(syncStatusMsg, 'Synchronizing app catalog with Git...', 'loading');
  logToConsole(`Triggered app catalog sync for: ${repo}`, 'system');

  try {
    const response = await fetch('/api/sync', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ repo })
    });
    const data = await response.json();

    if (response.ok && data.status === 'success') {
      showStatusMsg(syncStatusMsg, 'Catalog synced successfully!', 'success');
      logToConsole(`Sync succeeded: ${data.message}`, 'success');
      fetchCatalog();
    } else {
      showStatusMsg(syncStatusMsg, data.message || 'Sync failed', 'error');
      logToConsole(`Sync failed: ${data.message}`, 'error');
    }
  } catch (error) {
    showStatusMsg(syncStatusMsg, `Network error: ${error.message}`, 'error');
    logToConsole(`Sync network error: ${error.message}`, 'error');
  }
}

// KeyGen Action
async function handleKeyGen() {
  if (!selectedNode) return;

  showStatusMsg(nodeActionsStatus, `Generating SSH keys for ${selectedNode.Name}...`, 'loading');
  logToConsole(`Generating Ed25519 keypair for: ${selectedNode.Name}`, 'system');

  try {
    const response = await fetch('/api/keygen', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ node: selectedNode.Name })
    });
    const data = await response.json();

    if (response.ok && data.status === 'success') {
      showStatusMsg(nodeActionsStatus, 'SSH key pair generated successfully!', 'success');
      logToConsole(`Key pair generated successfully for ${selectedNode.Name}. Path: ${data.private_key_path}`, 'success');
      logToConsole(`Public key: ${data.public_key}`, 'success');
    } else {
      showStatusMsg(nodeActionsStatus, data.message || 'Keygen failed', 'error');
      logToConsole(`Keygen failed: ${data.message}`, 'error');
    }
  } catch (error) {
    showStatusMsg(nodeActionsStatus, `Network error: ${error.message}`, 'error');
    logToConsole(`Keygen network error: ${error.message}`, 'error');
  }
}

// Bootstrap Node Action
async function handleBootstrapNode() {
  if (!selectedNode) return;
  const password = nodeSshPasswordInput.value;
  if (!password) {
    showStatusMsg(nodeActionsStatus, 'Password is required to bootstrap key', 'error');
    return;
  }

  showStatusMsg(nodeActionsStatus, `Provisioning SSH key to remote host ${selectedNode.Name}...`, 'loading');
  logToConsole(`Bootstrapping SSH authentication for node: ${selectedNode.Name}`, 'system');

  try {
    const response = await fetch('/api/bootstrap', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ node: selectedNode.Name, password })
    });
    const data = await response.json();

    if (response.ok && data.status === 'success') {
      showStatusMsg(nodeActionsStatus, 'Node bootstrapped successfully!', 'success');
      logToConsole(`Node ${selectedNode.Name} bootstrapped successfully! Key registered.`, 'success');
      bootstrapForm.classList.add('hidden');
      nodeSshPasswordInput.value = '';
    } else {
      showStatusMsg(nodeActionsStatus, data.message || 'Bootstrap failed', 'error');
      logToConsole(`Bootstrap failed: ${data.message}`, 'error');
    }
  } catch (error) {
    showStatusMsg(nodeActionsStatus, `Network error: ${error.message}`, 'error');
    logToConsole(`Bootstrap network error: ${error.message}`, 'error');
  }
}

// Deploy Submit
async function handleDeploySubmit() {
  if (!appToDeploy) return;
  const targetNode = deployNodeSelect.value;
  const composePath = deployComposePath.value;

  if (!targetNode || !composePath) {
    alert("Please configure target node and compose path!");
    return;
  }

  const portsValue = (document.getElementById('deploy-ports')?.value || '').trim();
  const portsArray = portsValue ? portsValue.split(',').map(p => p.trim()).filter(Boolean) : [];

  const appName = appToDeploy.name;
  deployModal.classList.add('hidden');
  
  // Switch to monitor tab so the user can see deployment logs in real-time
  switchTab('monitor');
  
  // Pre-select the deploying node in the monitor dropdown so they see its status
  monitorNodeSelect.value = targetNode;

  logToConsole(`Starting deployment of ${appName} on ${targetNode}...`, 'system');

  try {
    const response = await fetch('/api/deploy', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ node: targetNode, compose_path: composePath, ports: portsArray })
    });

    const responseText = await response.text();
    let data;
    try {
      data = JSON.parse(responseText);
    } catch (e) {
      data = {
        status: 'error',
        message: responseText.trim() || `Server returned status code ${response.status} (${response.statusText})`
      };
    }

    if (data.logs) {
      logToConsole(`--- Deployment Output Logs ---`, 'system');
      logToConsole(data.logs, 'log');
      logToConsole(`------------------------------`, 'system');
    }

    if (response.ok && data.status === 'success') {
      logToConsole(`Application ${appName} successfully deployed to ${targetNode}!`, 'success');
      alert(`Success: ${appName} successfully deployed to ${targetNode}!`);
      fetchNodeStatus(targetNode);
    } else {
      const errMsg = data.message || 'Unknown error';
      logToConsole(`Deployment failed: ${errMsg}`, 'error');
      alert(`Deployment Failed: ${errMsg}`);
    }
  } catch (error) {
    logToConsole(`Deployment network error: ${error.message}`, 'error');
    alert(`Network Error: ${error.message}`);
  } finally {
    appToDeploy = null;
  }
}

// Fetch Node Status
async function fetchNodeStatus(nodeName) {
  try {
    const response = await fetch(`/api/status?node=${nodeName}`);
    const data = await response.json();

    if (response.ok && data.status === 'success') {
      renderHealthDashboard(data.apps);
    } else {
      healthDashboard.innerHTML = `<div class="status-msg error" style="display:block">Error: ${data.message || 'Failed to fetch status'}</div>`;
    }
  } catch (error) {
    console.error("Health check fetch error:", error);
  }
}

// Render Health Dashboard UI (Notion style)
function renderHealthDashboard(apps) {
  if (!apps || apps.length === 0) {
    healthDashboard.innerHTML = '<div class="placeholder-card notion-card"><i data-lucide="activity" class="lucide-big"></i><p>No applications running or defined on this node.</p></div>';
    lucide.createIcons();
    return;
  }

  // Filter out system mock images if present
  const activeApps = apps.filter(app => app.app_name !== "catalog_cache-web-1" && !app.app_name.includes("api-test-node"));

  healthDashboard.innerHTML = '';

  // Group into managed (multiple services or service name != app name) vs standalone (unmanaged)
  const managedApps = [];
  const standaloneApps = [];

  activeApps.forEach(app => {
    const isStandalone = app.services && app.services.length === 1 && app.services[0].name === app.app_name;
    if (isStandalone) {
      standaloneApps.push(app);
    } else {
      managedApps.push(app);
    }
  });

  // 1. Managed Apps Section
  const titleManaged = document.createElement('h3');
  titleManaged.textContent = "Orchestrated Applications";
  healthDashboard.appendChild(titleManaged);

  const listManaged = document.createElement('div');
  listManaged.className = 'monitor-list';

  managedApps.forEach(app => {
    listManaged.appendChild(renderMonitorItem(app));
  });

  if (managedApps.length === 0) {
    const placeholder = document.createElement('div');
    placeholder.className = 'placeholder-card notion-card';
    placeholder.style.padding = "2rem 1rem";
    placeholder.innerHTML = '<i data-lucide="package" class="lucide-big"></i><p>No catalog services deployed.</p>';
    listManaged.appendChild(placeholder);
  }
  healthDashboard.appendChild(listManaged);

  // 2. Standalone Containers Section
  const titleStandalone = document.createElement('h3');
  titleStandalone.textContent = "Standalone Containers";
  healthDashboard.appendChild(titleStandalone);

  const listStandalone = document.createElement('div');
  listStandalone.className = 'monitor-list';

  standaloneApps.forEach(app => {
    listStandalone.appendChild(renderMonitorItem(app));
  });

  if (standaloneApps.length === 0) {
    const placeholder = document.createElement('div');
    placeholder.className = 'placeholder-card notion-card';
    placeholder.style.padding = "2rem 1rem";
    placeholder.innerHTML = '<i data-lucide="box" class="lucide-big"></i><p>No standalone docker containers detected.</p>';
    listStandalone.appendChild(placeholder);
  }
  healthDashboard.appendChild(listStandalone);
  lucide.createIcons();
}

// Render a single Monitor Item card
function renderMonitorItem(app) {
  const item = document.createElement('div');
  item.className = 'monitor-item';

  const badgeClass = (app.health || '').toLowerCase();

  item.innerHTML = `
    <div class="monitor-item-header">
      <span class="app-status-title"><i data-lucide="box" class="app-icon"></i> ${app.app_name}</span>
      <span class="app-status-badge ${badgeClass}">${app.health || ''}</span>
    </div>
    <div class="services-list">
      ${(app.services || []).map(svc => `
        <div class="service-row">
          <span class="service-name"><i data-lucide="terminal" class="svc-icon"></i> ${svc.name}</span>
          <span class="service-status ${svc.status}">${svc.status}</span>
        </div>
      `).join('')}
    </div>
  `;

  return item;
}

// Helper: Show status message on form fields
function showStatusMsg(el, message, type) {
  el.className = `status-msg ${type}`;
  el.textContent = message;
  el.style.display = 'block';
}

// Helper: Switch tab programmatically
function switchTab(tabName) {
  const navItem = document.querySelector(`.nav-item[data-tab="${tabName}"]`);
  if (navItem) {
    navItem.click();
  }
}

let currentRoutes = [];

// Fetch Router Status
async function fetchRouterStatus() {
  try {
    const response = await fetch('/api/router/status');
    const data = await response.json();
    if (response.ok && data.status === 'success') {
      const status = data.router_status.toLowerCase();
      routerStatusText.textContent = `Router Status: ${data.router_status.toUpperCase()}`;
      
      // Update badge indicator
      routerStatusIndicator.className = 'status-dot';
      if (status === 'running') {
        routerStatusIndicator.classList.add('online');
      } else {
        // stopped / offline
      }
    } else {
      routerStatusText.textContent = "Error loading status";
    }
  } catch (error) {
    routerStatusText.textContent = "Offline";
  }
}

// Fetch Router Routes
async function fetchRouterRoutes() {
  try {
    const response = await fetch('/api/router/routes');
    const data = await response.json();
    if (response.ok && data.status === 'success') {
      currentRoutes = data.routes || [];
      renderRoutesTable(currentRoutes);
    } else {
      routesListBody.innerHTML = `<tr><td colspan="5" class="status-msg error" style="display:table-cell">Failed to load routes: ${data.message}</td></tr>`;
    }
  } catch (error) {
    routesListBody.innerHTML = `<tr><td colspan="5" class="status-msg error" style="display:table-cell">Network Error: ${error.message}</td></tr>`;
  }
}

// Render Routes Table
function renderRoutesTable(routes) {
  if (routes.length === 0) {
    routesListBody.innerHTML = '<tr><td colspan="5" style="text-align:center; padding: 2rem; color: var(--text-muted);">No active routes mapped.</td></tr>';
    return;
  }

  // Sort routes so that domains are alphabetical
  routes.sort((a, b) => a.domain.localeCompare(b.domain));

  routesListBody.innerHTML = '';
  routes.forEach(route => {
    const tr = document.createElement('tr');
    
    const badgeTypeClass = route.auto_discovered ? 'auto' : 'custom';
    const badgeTypeText = route.auto_discovered ? 'Auto' : 'Custom';
    
    // Delete action only for custom routes
    const actionHtml = route.auto_discovered 
      ? `<span style="font-size:0.75rem; color: var(--text-muted); padding-right: 0.5rem;">Managed</span>`
      : `<button class="btn-delete-route" onclick="handleDeleteRoute('${route.domain}')">Delete</button>`;

    tr.innerHTML = `
      <td style="font-weight: 500; font-family: var(--font-sans);">${route.domain}</td>
      <td style="color: var(--text-main);">${route.node}</td>
      <td style="font-family: var(--font-mono); font-size: 0.75rem;">${route.port}</td>
      <td><span class="route-badge ${badgeTypeClass}">${badgeTypeText}</span></td>
      <td style="text-align: right;">${actionHtml}</td>
    `;
    routesListBody.appendChild(tr);
  });
}

// Start Router
async function handleRouterStart() {
  showStatusMsg(routerControlStatus, 'Starting inbound Nginx router...', 'loading');
  try {
    const response = await fetch('/api/router/start', { method: 'POST' });
    const data = await response.json();
    if (response.ok && data.status === 'success') {
      showStatusMsg(routerControlStatus, 'Router successfully started!', 'success');
      logToConsole('Inbound Nginx router successfully started.', 'success');
      fetchRouterStatus();
    } else {
      showStatusMsg(routerControlStatus, data.message || 'Failed to start router', 'error');
    }
  } catch (error) {
    showStatusMsg(routerControlStatus, error.message, 'error');
  }
}

// Stop Router
async function handleRouterStop() {
  showStatusMsg(routerControlStatus, 'Stopping inbound Nginx router...', 'loading');
  try {
    const response = await fetch('/api/router/stop', { method: 'POST' });
    const data = await response.json();
    if (response.ok && data.status === 'success') {
      showStatusMsg(routerControlStatus, 'Router successfully stopped.', 'success');
      logToConsole('Inbound Nginx router successfully stopped.', 'system');
      fetchRouterStatus();
    } else {
      showStatusMsg(routerControlStatus, data.message || 'Failed to stop router', 'error');
    }
  } catch (error) {
    showStatusMsg(routerControlStatus, error.message, 'error');
  }
}

// Reload Router Config
async function handleRouterReload() {
  showStatusMsg(routerControlStatus, 'Reloading Nginx config...', 'loading');
  try {
    const response = await fetch('/api/router/reload', { method: 'POST' });
    const data = await response.json();
    if (response.ok && data.status === 'success') {
      showStatusMsg(routerControlStatus, 'Config reloaded successfully!', 'success');
      logToConsole('Regenerated configurations and reloaded Nginx router.', 'success');
      fetchRouterStatus();
      fetchRouterRoutes();
    } else {
      showStatusMsg(routerControlStatus, data.message || 'Failed to reload config', 'error');
    }
  } catch (error) {
    showStatusMsg(routerControlStatus, error.message, 'error');
  }
}

// Create Custom Route
async function handleCreateRoute() {
  const domain = routeDomainInput.value.trim();
  const node = routeNodeSelect.value;
  const portVal = routePortInput.value.trim();

  if (!domain || !node || !portVal) {
    showStatusMsg(routeCreateStatus, 'All route parameters are required', 'error');
    return;
  }

  const port = parseInt(portVal, 10);
  if (isNaN(port) || port <= 0 || port > 65535) {
    showStatusMsg(routeCreateStatus, 'Port must be a valid number (1-65535)', 'error');
    return;
  }

  showStatusMsg(routeCreateStatus, 'Saving custom route...', 'loading');

  // Load existing custom routes from currentRoutes and filter out any existing entry for this domain
  const customOnly = currentRoutes
    .filter(r => !r.auto_discovered && r.domain !== domain);

  // Add the new route
  customOnly.push({
    domain: domain,
    node: node,
    port: port,
    auto_discovered: false
  });

  try {
    const response = await fetch('/api/router/routes', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ routes: customOnly })
    });
    const data = await response.json();
    if (response.ok && data.status === 'success') {
      showStatusMsg(routeCreateStatus, 'Route successfully created!', 'success');
      logToConsole(`Custom route created: ${domain} -> ${node}:${port}`, 'success');
      routeDomainInput.value = '';
      routePortInput.value = '';
      fetchRouterRoutes();
    } else {
      showStatusMsg(routeCreateStatus, data.message || 'Failed to save route', 'error');
    }
  } catch (error) {
    showStatusMsg(routeCreateStatus, error.message, 'error');
  }
}

// Delete Route
async function handleDeleteRoute(domain) {
  if (!confirm(`Are you sure you want to delete the route for ${domain}?`)) {
    return;
  }

  // Filter out the domain to delete
  const customOnly = currentRoutes
    .filter(r => !r.auto_discovered && r.domain !== domain);

  try {
    const response = await fetch('/api/router/routes', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ routes: customOnly })
    });
    const data = await response.json();
    if (response.ok && data.status === 'success') {
      logToConsole(`Custom route deleted for domain: ${domain}`, 'system');
      fetchRouterRoutes();
    } else {
      alert(`Failed to delete route: ${data.message}`);
    }
  } catch (error) {
    alert(`Network Error: ${error.message}`);
  }
}

// Bind handleDeleteRoute globally so HTML onclick handler can invoke it
window.handleDeleteRoute = handleDeleteRoute;

