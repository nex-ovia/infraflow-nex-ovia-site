async function fetchProjectToml() {
  const response = await fetch('project.toml');
  if (!response.ok) throw new Error('Unable to load project.toml');
  return response.text();
}

function parseToml(tomlText) {
  const data = {};
  let currentSection = null;
  let currentObject = null;

  const trimQuotes = (value) => {
    if (value.startsWith('"') && value.endsWith('"')) {
      return value.slice(1, -1).replace(/\\"/g, '"');
    }
    if (value === 'true') return true;
    if (value === 'false') return false;
    if (/^-?\d+(\.\d+)?$/.test(value)) return Number(value);
    return value;
  };

  tomlText.split(/\r?\n/).forEach((rawLine) => {
    const line = rawLine.trim();
    if (!line || line.startsWith('#')) return;

    if (line.startsWith('[[') && line.endsWith(']]')) {
      const sectionName = line.slice(2, -2).trim();
      const parts = sectionName.split('.');
      
      if (parts.length === 1) {
        // Simple array section like [[node]]
        if (!Array.isArray(data[parts[0]])) {
          data[parts[0]] = [];
        }
        currentObject = {};
        data[parts[0]].push(currentObject);
      } else {
        // Nested array section like [[section.subsection]]
        const parent = parts[0];
        const name = parts[1];
        if (!data[parent]) data[parent] = {};
        if (!Array.isArray(data[parent][name])) {
          data[parent][name] = [];
        }
        currentObject = {};
        data[parent][name].push(currentObject);
      }
      return;
    }

    if (line.startsWith('[') && line.endsWith(']')) {
      const sectionName = line.slice(1, -1).trim();
      if (!data[sectionName]) data[sectionName] = {};
      currentSection = sectionName;
      currentObject = data[sectionName];
      return;
    }

    const equalsIndex = line.indexOf('=');
    if (equalsIndex === -1 || !currentObject) return;
    const key = line.slice(0, equalsIndex).trim();
    const value = line.slice(equalsIndex + 1).trim();
    currentObject[key] = trimQuotes(value);
  });

  return data;
}

function createNodeCard(node) {
  const wrapper = document.createElement('div');
  wrapper.className = 'graph-node';

  const box = document.createElement('div');
  box.className = 'node-box';
  box.title = node.description || node.purpose || '';

  const type = document.createElement('div');
  type.className = `node-type ${node.type ? `node-${node.type}` : ''}`;
  type.textContent = node.type || 'node';

  const name = document.createElement('div');
  name.className = 'node-name';
  name.textContent = node.id || node.source || node.path || 'unknown';

  const desc = document.createElement('div');
  desc.className = 'node-desc';
  desc.textContent = node.description || node.purpose || '';

  box.append(type, name, desc);
  wrapper.appendChild(box);
  wrapper.addEventListener('click', () => {
    document.querySelectorAll('.node-box.active').forEach((el) => el.classList.remove('active'));
    box.classList.add('active');
    showNodeDetails(node);
  });
  return wrapper;
}

function createEdgeLabel(edge) {
  const wrapper = document.createElement('div');
  wrapper.className = 'graph-edge';

  const line = document.createElement('div');
  line.className = 'edge-line';

  const label = document.createElement('div');
  label.className = 'edge-label';
  label.textContent = edge.label || `${edge.from} → ${edge.to}`;

  wrapper.append(line, label);
  return wrapper;
}

function showNodeDetails(node) {
  const details = document.getElementById('project-graph-details');
  if (!details) return;

  details.innerHTML = `
    <h3 class="graph-detail-title">${node.id || node.path || node.source}</h3>
    <p class="graph-detail-body">${node.description || node.purpose || 'No details available.'}</p>
    <dl class="graph-detail-meta">
      ${node.source ? `<dt>Source</dt><dd>${node.source}</dd>` : ''}
      ${node.path ? `<dt>Path</dt><dd>${node.path}</dd>` : ''}
      ${node.output ? `<dt>Output</dt><dd>${node.output}</dd>` : ''}
      ${node.role ? `<dt>Role</dt><dd>${node.role}</dd>` : ''}
      ${node.change_guidance ? `<dt>Change guidance</dt><dd>${node.change_guidance}</dd>` : ''}
    </dl>
  `;
}

async function loadProjectGraph() {
  try {
    const tomlText = await fetchProjectToml();
    const data = parseToml(tomlText);
    renderProjectGraph(data);
  } catch (error) {
    console.error('Project graph failed to load:', error);
    const details = document.getElementById('project-graph-details');
    if (details) {
      details.innerHTML = `<p class="graph-detail-body" style="color: #ff6b6b;">Graph failed to load: ${error.message}</p>`;
    }
  }
}

function renderProjectGraph(projectData) {
  const root = document.getElementById('project-graph-root');
  const details = document.getElementById('project-graph-details');
  if (!root || !details) return;

  const nodes = (projectData.node) || [];
  const edges = (projectData.edge) || [];
  const nodeMap = {};
  
  nodes.forEach((node) => {
    nodeMap[node.id] = node;
  });

  root.innerHTML = '';
  const container = document.createElement('div');
  container.className = 'graph-explorer';

  const rootNode = nodeMap['index.html'];
  if (!rootNode) {
    root.innerHTML = '<p class="graph-detail-body" style="color: #ff6b6b;">Root node (index.html) not found</p>';
    return;
  }

  const edgesBySource = {};
  edges.forEach((edge) => {
    if (!edgesBySource[edge.from]) edgesBySource[edge.from] = [];
    edgesBySource[edge.from].push(edge);
  });

  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  svg.setAttribute('class', 'graph-svg');
  svg.setAttribute('width', '100%');
  svg.setAttribute('height', '500');
  svg.setAttribute('viewBox', '0 0 800 500');
  svg.style.background = 'transparent';

  const center = { x: 400, y: 250 };
  const radius = 200;
  const rootDeps = edgesBySource['index.html'] || [];
  const angleSlice = (Math.PI * 2) / Math.max(rootDeps.length, 1);

  const defs = document.createElementNS('http://www.w3.org/2000/svg', 'defs');
  const marker = document.createElementNS('http://www.w3.org/2000/svg', 'marker');
  marker.setAttribute('id', 'arrowhead');
  marker.setAttribute('markerWidth', '10');
  marker.setAttribute('markerHeight', '10');
  marker.setAttribute('refX', '9');
  marker.setAttribute('refY', '3');
  marker.setAttribute('orient', 'auto');
  const polygon = document.createElementNS('http://www.w3.org/2000/svg', 'polygon');
  polygon.setAttribute('points', '0 0, 10 3, 0 6');
  polygon.setAttribute('fill', 'rgba(108, 99, 255, 0.4)');
  marker.appendChild(polygon);
  defs.appendChild(marker);
  svg.appendChild(defs);

  const g = document.createElementNS('http://www.w3.org/2000/svg', 'g');

  rootDeps.forEach((edge, index) => {
    const angle = angleSlice * index - Math.PI / 2;
    const x = center.x + radius * Math.cos(angle);
    const y = center.y + radius * Math.sin(angle);

    const line = document.createElementNS('http://www.w3.org/2000/svg', 'line');
    line.setAttribute('x1', center.x);
    line.setAttribute('y1', center.y);
    line.setAttribute('x2', x);
    line.setAttribute('y2', y);
    line.setAttribute('stroke', 'rgba(108, 99, 255, 0.2)');
    line.setAttribute('stroke-width', '2');
    line.setAttribute('marker-end', 'url(#arrowhead)');
    g.appendChild(line);

    const depNode = nodeMap[edge.to];
    if (depNode) {
      const text = document.createElementNS('http://www.w3.org/2000/svg', 'text');
      text.setAttribute('x', center.x + (x - center.x) * 0.4);
      text.setAttribute('y', center.y + (y - center.y) * 0.4);
      text.setAttribute('font-family', "'JetBrains Mono', monospace");
      text.setAttribute('font-size', '10');
      text.setAttribute('fill', 'rgba(0, 212, 170, 0.6)');
      text.setAttribute('text-anchor', 'middle');
      text.textContent = edge.label;
      g.appendChild(text);
    }
  });

  svg.appendChild(g);
  container.appendChild(svg);

  const nodesList = document.createElement('div');
  nodesList.className = 'graph-nodes-list';

  const rootNodeEl = createExplorerNodeCard(rootNode, true);
  nodesList.appendChild(rootNodeEl);

  const depsSection = document.createElement('div');
  depsSection.className = 'graph-deps-section';
  const depsTitle = document.createElement('div');
  depsTitle.className = 'deps-section-title';
  depsTitle.textContent = `Dependencies (${rootDeps.length})`;
  depsSection.appendChild(depsTitle);

  const depsList = document.createElement('div');
  depsList.className = 'graph-deps-list';
  
  rootDeps.forEach((edge) => {
    const depNode = nodeMap[edge.to];
    if (depNode) {
      const depEl = createExplorerNodeCard(depNode, false, edge.label);
      depsList.appendChild(depEl);
    }
  });

  depsSection.appendChild(depsList);
  nodesList.appendChild(depsSection);

  container.appendChild(nodesList);
  root.appendChild(container);

  const detailIntro = document.createElement('div');
  detailIntro.className = 'graph-detail-intro';
  detailIntro.innerHTML = '<p>Click any node to explore its dependencies and see where to make changes.</p>';
  details.innerHTML = '';
  details.appendChild(detailIntro);
}

function createExplorerNodeCard(node, isRoot, edgeLabel) {
  const card = document.createElement('div');
  card.className = `graph-explorer-node ${isRoot ? 'root-node' : ''}`;
  
  const header = document.createElement('div');
  header.className = 'explorer-node-header';
  
  const typeTag = document.createElement('span');
  typeTag.className = `node-type-tag node-${node.type || 'unknown'}`;
  typeTag.textContent = node.type || 'unknown';
  
  const idText = document.createElement('span');
  idText.className = 'node-id-text';
  idText.textContent = node.id;
  
  header.append(typeTag, idText);
  
  if (edgeLabel) {
    const edgeText = document.createElement('span');
    edgeText.className = 'edge-relation';
    edgeText.textContent = edgeLabel;
    header.appendChild(edgeText);
  }

  const body = document.createElement('div');
  body.className = 'explorer-node-body';
  
  const role = document.createElement('div');
  role.className = 'node-role';
  role.textContent = node.role;
  
  const desc = document.createElement('div');
  desc.className = 'node-details';
  desc.innerHTML = `<strong>Path:</strong> ${node.path}<br><strong>About:</strong> ${node.description}`;

  body.append(role, desc);
  
  card.append(header, body);
  
  card.addEventListener('click', () => {
    document.querySelectorAll('.graph-explorer-node.active').forEach((el) => {
      el.classList.remove('active');
    });
    card.classList.add('active');
    showNodeDetails(node);
  });

  return card;
}

window.addEventListener('DOMContentLoaded', loadProjectGraph);
