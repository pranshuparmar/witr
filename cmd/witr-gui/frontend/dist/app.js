document.addEventListener('DOMContentLoaded', () => {
  // DOM Elements
  const searchForm = document.getElementById('searchForm');
  const searchInput = document.getElementById('searchInput');
  const clearSearchBtn = document.getElementById('clearSearchBtn');
  const exactMatchCheck = document.getElementById('exactMatchCheck');
  const targetTypeSelector = document.getElementById('targetTypeSelector');
  const categoryFilterBar = document.getElementById('categoryFilterBar');
  const inspectorTabBar = document.getElementById('inspectorTabBar');
  const tableHeaderRow = document.getElementById('tableHeaderRow');

  const refreshBtn = document.getElementById('refreshBtn');
  const refreshIcon = document.getElementById('refreshIcon');

  // Metrics
  const metricTotalProc = document.getElementById('metricTotalProc');
  const metricUserProc = document.getElementById('metricUserProc');
  const metricSysProc = document.getElementById('metricSysProc');
  const metricPorts = document.getElementById('metricPorts');

  // Why Hero & Details
  const whyHeroCard = document.getElementById('whyHeroCard');
  const whyExplanationText = document.getElementById('whyExplanationText');
  const targetNameTag = document.getElementById('targetNameTag');
  const sourceBadge = document.getElementById('sourceBadge');

  const treeContainer = document.getElementById('treeContainer');
  const detailContainer = document.getElementById('detailContainer');
  const processTableBody = document.getElementById('processTableBody');
  const processCountBadge = document.getElementById('processCountBadge');
  const systemStatus = document.getElementById('systemStatus');
  const socketsTabCount = document.getElementById('socketsTabCount');
  const envTabCount = document.getElementById('envTabCount');

  let activeTargetType = 'auto';
  let activeCategoryFilter = 'all';
  let activeInspectorTab = 'overview';
  let currentSortColumn = 'pid';
  let currentSortDirection = 'asc';
  let cachedProcesses = [];
  let currentGUIResult = null;

  // Wails Go IPC Call Helper
  const callWails = async (method, ...args) => {
    try {
      if (window.go && window.go.gui && window.go.gui.App && typeof window.go.gui.App[method] === 'function') {
        return await window.go.gui.App[method](...args);
      } else {
        return await mockWailsMethod(method, ...args);
      }
    } catch (err) {
      console.error(`Wails IPC Error [${method}]:`, err);
      throw err;
    }
  };

  // Mock Fallback
  const mockWailsMethod = async (method, ...args) => {
    if (method === 'Search' || method === 'GetProcessDetails') {
      const query = args[0] || 'Blip.exe';
      return {
        whyExplanation: `Process '${query}' (PID 23260) is running as a background Windows App.`,
        startedFormatted: '7 days ago (Tue 2026-08-04 08:40:40)',
        workingDir: 'C:\\WINDOWS\\system32\\',
        socketsList: ['192.168.1.244:62522 -> 0.0.0.0:0 (TCP | ESTABLISHED)'],
        cpuFormatted: '0.3%',
        memoryVirtual: '242.2 MB',
        memoryResident: '107.9 MB',
        memoryPrivate: '242.2 MB',
        ioRead: '19.1 MB (13865 ops)',
        ioWrite: '44.5 MB (2312 ops)',
        handlesCount: '1308 / unlimited',
        threadCount: '70',
        envVars: {
          'ALLUSERSPROFILE': 'C:\\ProgramData',
          'ANDROID_HOME': 'C:\\Android',
          'APPDATA': 'C:\\Users\\Vinrox\\AppData\\Roaming',
          'COMPUTERNAME': 'VINROX-85',
          'ComSpec': 'C:\\WINDOWS\\system32\\cmd.exe'
        },
        Target: { Type: 'name', Value: query },
        ResolvedTarget: query,
        Process: { PID: 23260, Command: query, Cmdline: '"C:\\Program Files\\WindowsApps\\BlipStudioInc.BlipApp\\bin\\Blip.exe" /background', User: 'VINROX-85\\Vinrox', Health: 'healthy' },
        Ancestry: [
          { PID: 4, Command: 'System', Cmdline: 'System', User: 'NT AUTHORITY\\SYSTEM' },
          { PID: 688, Command: 'services.exe', Cmdline: 'C:\\Windows\\System32\\services.exe', User: 'SYSTEM' },
          { PID: 23260, Command: query, Cmdline: '"C:\\Program Files\\WindowsApps\\BlipStudioInc.BlipApp\\bin\\Blip.exe" /background', User: 'VINROX-85\\Vinrox', PPID: 688 }
        ],
        Source: { Type: 'windows-app', Name: 'Windows Application' },
        Warnings: []
      };
    }
    if (method === 'ListRunningProcesses') {
      return [
        { PID: 29172, Command: 'Antigravity.exe', PPID: 1020, User: 'VINROX-85\\Vinrox', Cmdline: 'Antigravity.exe', CPUPercent: 1.5, MemoryRSS: 357040000, cpuFormatted: '1.5%', memFormatted: '340.5 MB', memPercentStr: '(4.2%)', isSystem: false, hasSockets: true },
        { PID: 23260, Command: 'Blip.exe', PPID: 688, User: 'VINROX-85\\Vinrox', Cmdline: 'Blip.exe /background', CPUPercent: 0.3, MemoryRSS: 112720000, cpuFormatted: '0.3%', memFormatted: '107.5 MB', memPercentStr: '(1.3%)', isSystem: false, hasSockets: true },
        { PID: 688, Command: 'services.exe', PPID: 4, User: 'SYSTEM', Cmdline: 'C:\\Windows\\System32\\services.exe', CPUPercent: 0.0, MemoryRSS: 14890000, cpuFormatted: '0.0%', memFormatted: '14.2 MB', memPercentStr: '(0.2%)', isSystem: true, hasSockets: false }
      ];
    }
    if (method === 'GetSystemAnalytics') {
      return { totalProcesses: 299, userProcesses: 171, systemProcesses: 128, listeningPorts: 58 };
    }
    return null;
  };

  // Column Header Sorting Listener
  tableHeaderRow.addEventListener('click', (e) => {
    const th = e.target.closest('.sortable-th');
    if (th) {
      const col = th.dataset.sort;
      if (col === currentSortColumn) {
        currentSortDirection = currentSortDirection === 'asc' ? 'desc' : 'asc';
      } else {
        currentSortColumn = col;
        currentSortDirection = (col === 'cpu' || col === 'mem') ? 'desc' : 'asc';
      }
      applyFilters();
    }
  });

  // Search input ONLY filters table as user types (NO searchbar modification or auto-tracing)
  searchInput.addEventListener('input', () => {
    const val = searchInput.value;
    clearSearchBtn.style.display = val ? 'block' : 'none';
    applyFilters();
  });

  clearSearchBtn.addEventListener('click', () => {
    searchInput.value = '';
    clearSearchBtn.style.display = 'none';
    applyFilters();
    searchInput.focus();
  });

  targetTypeSelector.addEventListener('click', (e) => {
    if (e.target.classList.contains('type-pill')) {
      document.querySelectorAll('.type-pill').forEach(p => p.classList.remove('active'));
      e.target.classList.add('active');
      activeTargetType = e.target.dataset.type;
    }
  });

  categoryFilterBar.addEventListener('click', (e) => {
    if (e.target.classList.contains('cat-filter-btn')) {
      document.querySelectorAll('.cat-filter-btn').forEach(b => b.classList.remove('active'));
      e.target.classList.add('active');
      activeCategoryFilter = e.target.dataset.filter;
      applyFilters();
    }
  });

  inspectorTabBar.addEventListener('click', (e) => {
    if (e.target.classList.contains('tab-btn')) {
      document.querySelectorAll('#inspectorTabBar .tab-btn').forEach(b => b.classList.remove('active'));
      e.target.classList.add('active');
      activeInspectorTab = e.target.dataset.tab;
      if (currentGUIResult) {
        renderInspectorContent(currentGUIResult);
      }
    }
  });

  // ONLY trace when user explicitly submits search or clicks Trace/Why?
  searchForm.addEventListener('submit', (e) => {
    e.preventDefault();
    const query = searchInput.value.trim();
    if (query) {
      performSearch(query, activeTargetType);
    }
  });

  // Refresh Button Action
  refreshBtn.addEventListener('click', async () => {
    refreshIcon.classList.add('spin');
    systemStatus.textContent = 'Refreshing active processes & metrics...';
    try {
      await loadActiveProcesses();
      await loadAnalytics();
      systemStatus.textContent = 'Refreshed successfully';
    } catch (e) {
      console.error(e);
      systemStatus.textContent = 'Refresh failed';
    } finally {
      setTimeout(() => refreshIcon.classList.remove('spin'), 600);
    }
  });

  // Apply Search + Category Filters + Column Sorting
  function applyFilters() {
    const term = searchInput.value.trim().toLowerCase();
    let filtered = [...cachedProcesses];

    // Filter by category
    if (activeCategoryFilter === 'user') {
      filtered = filtered.filter(p => !p.isSystem);
    } else if (activeCategoryFilter === 'system') {
      filtered = filtered.filter(p => p.isSystem);
    } else if (activeCategoryFilter === 'ports') {
      filtered = filtered.filter(p => p.hasSockets);
    }

    // Filter by search text
    if (term) {
      filtered = filtered.filter(p => 
        (p.Command && p.Command.toLowerCase().includes(term)) ||
        (p.PID && p.PID.toString().includes(term)) ||
        (p.User && p.User.toLowerCase().includes(term)) ||
        (p.Cmdline && p.Cmdline.toLowerCase().includes(term))
      );
    }

    // Sort column
    filtered.sort((a, b) => {
      let valA, valB;
      switch (currentSortColumn) {
        case 'pid':
          valA = a.PID || 0;
          valB = b.PID || 0;
          break;
        case 'name':
          valA = (a.Command || '').toLowerCase();
          valB = (b.Command || '').toLowerCase();
          break;
        case 'user':
          valA = (a.User || '').toLowerCase();
          valB = (b.User || '').toLowerCase();
          break;
        case 'cpu':
          valA = a.CPUPercent || 0;
          valB = b.CPUPercent || 0;
          break;
        case 'mem':
          valA = a.MemoryRSS || 0;
          valB = b.MemoryRSS || 0;
          break;
        default:
          valA = a.PID || 0;
          valB = b.PID || 0;
      }

      if (valA < valB) return currentSortDirection === 'asc' ? -1 : 1;
      if (valA > valB) return currentSortDirection === 'asc' ? 1 : -1;
      return 0;
    });

    // Update Header Sort Icons
    document.querySelectorAll('.sortable-th').forEach(th => {
      const col = th.dataset.sort;
      const iconSpan = th.querySelector('.sort-icon');
      if (col === currentSortColumn) {
        th.classList.add('active');
        if (iconSpan) iconSpan.textContent = currentSortDirection === 'asc' ? ' ▲' : ' ▼';
      } else {
        th.classList.remove('active');
        if (iconSpan) iconSpan.textContent = '';
      }
    });

    renderProcessTable(filtered);
  }

  // Perform Causality Search (Only triggered on explicit click/Enter)
  async function performSearch(query, type) {
    systemStatus.textContent = `Tracing ${query}...`;
    targetNameTag.textContent = `${type.toUpperCase()}: ${query}`;
    whyExplanationText.textContent = `Analyzing why '${query}' is running...`;

    treeContainer.innerHTML = '<div class="placeholder-box"><p>Loading execution chain...</p></div>';
    detailContainer.innerHTML = '<div class="placeholder-box"><p>Loading details...</p></div>';

    try {
      const isExact = exactMatchCheck.checked;
      const guiRes = await callWails('Search', query, type, isExact);
      if (guiRes) {
        currentGUIResult = guiRes;
        renderWhyHero(guiRes);
        renderCausalityTree(guiRes);
        renderProcessDetail(guiRes);
        systemStatus.textContent = `Traced ${query} successfully`;
      } else {
        showError('No matching process found.');
      }
    } catch (err) {
      showError(err.toString());
      systemStatus.textContent = 'Search failed';
    }
  }

  // Render Hero "Why is this running?" Card with Syntax Highlighted Colors
  function renderWhyHero(guiRes) {
    const rawExplanation = guiRes.whyExplanation || buildFallbackExplanation(guiRes);
    whyExplanationText.innerHTML = formatExplanationHTML(rawExplanation);
    targetNameTag.textContent = guiRes.ResolvedTarget || (guiRes.Process && guiRes.Process.Command) || 'Target';
    sourceBadge.textContent = (guiRes.Source && guiRes.Source.Name) || 'Standard Process';
  }

  // Syntax Highlighting for Explanation Text
  function formatExplanationHTML(str) {
    if (!str) return '';
    let html = escapeHtml(str);

    // Highlight Process 'name'
    html = html.replace(/Process '([^']+)'/g, 'Process <span class="hl-cmd">\'$1\'</span>');
    // Highlight (PID 1234)
    html = html.replace(/\(PID (\d+)\)/g, '(<span class="hl-pid">PID $1</span>)');
    // Highlight service 'name' or container 'name'
    html = html.replace(/service '([^']+)'/g, 'service <span class="hl-svc">\'$1\'</span>');
    html = html.replace(/container '([^']+)'/g, 'container <span class="hl-svc">\'$1\'</span>');
    html = html.replace(/shell '([^']+)'/g, 'shell <span class="hl-svc">\'$1\'</span>');
    // Highlight user 'name'
    html = html.replace(/user '([^']+)'/g, 'user <span class="hl-user">\'$1\'</span>');
    // Highlight parent process 'name'
    html = html.replace(/parent process '([^']+)'/g, 'parent process <span class="hl-svc">\'$1\'</span>');

    return html;
  }

  function buildFallbackExplanation(guiRes) {
    const p = guiRes.Process || {};
    const anc = guiRes.Ancestry || [];
    let parentStr = anc.length > 1 ? anc[anc.length - 2].Command : '';

    if (parentStr) {
      return `Process '${p.Command}' (PID ${p.PID}) was launched by parent process '${parentStr}' (PPID ${p.PPID}) under user '${p.User || 'SYSTEM'}'.`;
    }
    return `Process '${p.Command}' (PID ${p.PID}) is running under account '${p.User || 'SYSTEM'}'.`;
  }

  // Render Causality Ancestry Tree
  function renderCausalityTree(guiRes) {
    const anc = guiRes.Ancestry || [];
    if (anc.length === 0) {
      treeContainer.innerHTML = '<div class="placeholder-box"><p>No process ancestry found.</p></div>';
      return;
    }

    let html = '<div class="tree-node-list">';
    const targetPID = guiRes.Process ? guiRes.Process.PID : -1;

    anc.forEach((proc, index) => {
      const isTarget = proc.PID === targetPID;
      const isRoot = index === 0;
      const nodeClass = isTarget ? 'tree-node-item is-target' : 'tree-node-item';
      
      let badgeLabel = 'SPAWNER';
      if (isRoot) badgeLabel = 'ORIGIN';
      if (isTarget) badgeLabel = 'TARGET';

      html += `
        <div class="${nodeClass}">
          <div class="tree-node-header">
            <span class="tree-node-title">
              ${isRoot ? '🚀' : '⚡'} ${escapeHtml(proc.Command)}
              <span class="pid-tag">PID ${proc.PID}</span>
            </span>
            <span class="pid-tag" style="background: rgba(16,185,129,0.15); color: #34d399; font-weight:600;">${badgeLabel}</span>
          </div>
          <div class="tree-node-cmd">${escapeHtml(proc.Cmdline || proc.Command)}</div>
        </div>
      `;

      if (index < anc.length - 1) {
        html += '<div class="tree-link"></div>';
      }
    });

    html += '</div>';
    treeContainer.innerHTML = html;
  }

  // Render Process Details & Multi-Tab Inspector
  function renderProcessDetail(guiRes) {
    // Update Tab Counts
    const sockCount = (guiRes.socketsList && guiRes.socketsList.length) || 0;
    const envCount = (guiRes.envVars && Object.keys(guiRes.envVars).length) || 0;
    socketsTabCount.textContent = sockCount;
    envTabCount.textContent = envCount;

    renderInspectorContent(guiRes);
  }

  function renderInspectorContent(guiRes) {
    let html = '';

    // Security Warnings Banner
    if (guiRes.Warnings && guiRes.Warnings.length > 0) {
      html += `
        <div style="background: rgba(245,158,11,0.1); border: 1px solid var(--accent-amber); padding: 6px 8px; border-radius: 4px; margin-bottom: 8px;">
          <div style="color: var(--accent-amber); font-weight: 600; font-size: 0.76rem; margin-bottom: 2px;">⚠️ Warnings (${guiRes.Warnings.length})</div>
          <ul style="padding-left: 14px; color: #fcd34d; font-size: 0.72rem;">
            ${guiRes.Warnings.map(w => `<li>${escapeHtml(w)}</li>`).join('')}
          </ul>
        </div>
      `;
    }

    const p = guiRes.Process || {};
    const src = guiRes.Source || {};

    if (activeInspectorTab === 'overview') {
      // TAB 1: OVERVIEW & RICH METRICS
      html += `
        <div class="metrics-mini-grid">
          <div class="mini-metric-card">
            <span class="mini-metric-label">CPU Usage</span>
            <span class="mini-metric-val" style="color: var(--accent-sky);">${guiRes.cpuFormatted || '0.0%'}</span>
          </div>
          <div class="mini-metric-card">
            <span class="mini-metric-label">Resident Mem (RSS)</span>
            <span class="mini-metric-val" style="color: var(--accent-emerald);">${guiRes.memoryResident || '0 MB'}</span>
          </div>
          <div class="mini-metric-card">
            <span class="mini-metric-label">Virtual Mem (VMS)</span>
            <span class="mini-metric-val">${guiRes.memoryVirtual || '0 MB'}</span>
          </div>
          <div class="mini-metric-card">
            <span class="mini-metric-label">Private Memory</span>
            <span class="mini-metric-val">${guiRes.memoryPrivate || '0 MB'}</span>
          </div>
          <div class="mini-metric-card">
            <span class="mini-metric-label">I/O Storage Read</span>
            <span class="mini-metric-val">${guiRes.ioRead || 'N/A'}</span>
          </div>
          <div class="mini-metric-card">
            <span class="mini-metric-label">I/O Storage Write</span>
            <span class="mini-metric-val">${guiRes.ioWrite || 'N/A'}</span>
          </div>
          <div class="mini-metric-card">
            <span class="mini-metric-label">Handles / FDs</span>
            <span class="mini-metric-val">${guiRes.handlesCount || 'N/A'}</span>
          </div>
          <div class="mini-metric-card">
            <span class="mini-metric-label">Thread Count</span>
            <span class="mini-metric-val">${guiRes.threadCount || 'N/A'}</span>
          </div>
        </div>

        <table class="kv-table">
          <tr>
            <td class="kv-key">Target:</td>
            <td class="kv-val" style="color: var(--accent-sky); font-weight:700;">${escapeHtml(guiRes.ResolvedTarget || p.Command || 'N/A')}</td>
          </tr>
          <tr>
            <td class="kv-key">PID / PPID:</td>
            <td class="kv-val"><span style="color:var(--accent-emerald);">${p.PID || 'N/A'}</span> / ${p.PPID || '0'}</td>
          </tr>
          <tr>
            <td class="kv-key">User Account:</td>
            <td class="kv-val">${escapeHtml(p.User || 'SYSTEM')}</td>
          </tr>
          <tr>
            <td class="kv-key">Started At:</td>
            <td class="kv-val">${escapeHtml(guiRes.startedFormatted || 'N/A')}</td>
          </tr>
          <tr>
            <td class="kv-key">Working Dir:</td>
            <td class="kv-val">${escapeHtml(guiRes.workingDir || 'N/A')}</td>
          </tr>
          <tr>
            <td class="kv-key">Command Line:</td>
            <td class="kv-val">${escapeHtml(p.Cmdline || 'N/A')}</td>
          </tr>
          <tr>
            <td class="kv-key">Origin Source:</td>
            <td class="kv-val">${escapeHtml(src.Name || src.Type || 'Standard Process')}</td>
          </tr>
          <tr>
            <td class="kv-key">Health Status:</td>
            <td class="kv-val" style="color: var(--accent-emerald); font-weight:600;">${escapeHtml(p.Health || 'healthy')}</td>
          </tr>
        </table>
      `;
    } else if (activeInspectorTab === 'sockets') {
      // TAB 2: NETWORK SOCKETS
      const list = guiRes.socketsList || [];
      if (list.length === 0) {
        html += '<div class="placeholder-box"><p>No active network sockets bound to this process.</p></div>';
      } else {
        html += '<table class="kv-table"><thead><tr><th style="padding:4px; color:var(--text-muted);">Index</th><th style="padding:4px; color:var(--text-muted);">Connection & State</th></tr></thead><tbody>';
        list.forEach((s, idx) => {
          html += `
            <tr>
              <td style="color:var(--accent-emerald); width:40px;">#${idx+1}</td>
              <td class="kv-val" style="color:#38bdf8;">${escapeHtml(s)}</td>
            </tr>
          `;
        });
        html += '</tbody></table>';
      }
    } else if (activeInspectorTab === 'env') {
      // TAB 3: ENVIRONMENT VARIABLES
      const envs = guiRes.envVars || {};
      const keys = Object.keys(envs).sort();
      if (keys.length === 0) {
        html += '<div class="placeholder-box"><p>No environment variables captured for this process.</p></div>';
      } else {
        html += '<table class="kv-table"><thead><tr><th style="padding:4px; color:var(--text-muted); width:40%;">Variable</th><th style="padding:4px; color:var(--text-muted);">Value</th></tr></thead><tbody>';
        keys.forEach(k => {
          html += `
            <tr>
              <td class="kv-key" style="color:var(--accent-amber);">${escapeHtml(k)}</td>
              <td class="kv-val">${escapeHtml(envs[k])}</td>
            </tr>
          `;
        });
        html += '</tbody></table>';
      }
    }

    detailContainer.innerHTML = html;
  }

  // Load Active Processes List with CPU% & Mem Columns
  async function loadActiveProcesses() {
    try {
      const list = await callWails('ListRunningProcesses');
      if (list && Array.isArray(list)) {
        cachedProcesses = list;
        applyFilters();
      }
    } catch (e) {
      console.error('Failed to load active processes:', e);
    }
  }

  function renderProcessTable(list) {
    processCountBadge.textContent = `${list.length} Listed`;

    if (!list || list.length === 0) {
      processTableBody.innerHTML = '<tr><td colspan="6" class="table-loading">No active processes match filter.</td></tr>';
      return;
    }

    let html = '';
    list.slice(0, 150).forEach(p => {
      html += `
        <tr onclick="inspectPID(${p.PID})">
          <td style="font-family: var(--font-mono); color: var(--accent-cyan); font-weight:600;">${p.PID}</td>
          <td style="font-weight: 600; color: #f1f5f9;">${escapeHtml(p.Command)}</td>
          <td style="font-size: 0.72rem;">${escapeHtml(truncateUser(p.User || 'SYSTEM'))}</td>
          <td style="font-family: var(--font-mono); color: var(--accent-sky); text-align: right;">${p.cpuFormatted || '0.0%'}</td>
          <td style="font-family: var(--font-mono); color: var(--accent-emerald); text-align: right;">${p.memFormatted || '0 MB'}</td>
          <td style="text-align: center;">
            <button class="btn btn-primary btn-sm" onclick="event.stopPropagation(); inspectPID(${p.PID})">Why?</button>
          </td>
        </tr>
      `;
    });

    processTableBody.innerHTML = html;
  }

  // Load Analytics Bar Metrics
  async function loadAnalytics() {
    try {
      const analytics = await callWails('GetSystemAnalytics');
      if (analytics) {
        metricTotalProc.textContent = analytics.totalProcesses || '--';
        metricUserProc.textContent = analytics.userProcesses || '0';
        metricSysProc.textContent = analytics.systemProcesses || '0';
        metricPorts.textContent = analytics.listeningPorts || '0';
      }
    } catch (e) {
      console.error('Failed to load system analytics:', e);
    }
  }

  // Keep searchInput clean when inspecting PID from table
  window.inspectPID = (pid) => {
    performSearch(pid.toString(), 'pid');
  };

  function showError(msg) {
    whyExplanationText.innerHTML = `<span style="color: var(--accent-rose);">❌ ${escapeHtml(msg)}</span>`;
    targetNameTag.textContent = 'ERROR';
    treeContainer.innerHTML = '<div class="placeholder-box"><p>Process not found.</p></div>';
    detailContainer.innerHTML = '<div class="placeholder-box"><p>No details available.</p></div>';
  }

  function escapeHtml(str) {
    if (!str) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  function truncateUser(user) {
    if (!user) return 'SYSTEM';
    if (user.includes('\\')) {
      return user.split('\\')[1];
    }
    return user;
  }

  // Initial loads
  loadActiveProcesses();
  loadAnalytics();
});
