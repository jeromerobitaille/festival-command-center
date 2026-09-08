/* Command Center — portail. Vanilla JS : composants UI réutilisables + un contrôleur par page. */

/* ---------- Icônes (outline, style Lucide) ---------- */
const ICONS = {
  dashboard: '<rect x="3" y="3" width="7" height="9" rx="1.5"/><rect x="14" y="3" width="7" height="5" rx="1.5"/><rect x="14" y="12" width="7" height="9" rx="1.5"/><rect x="3" y="16" width="7" height="5" rx="1.5"/>',
  devices: '<rect x="2" y="4" width="20" height="13" rx="2"/><path d="M8 21h8M12 17v4"/>',
  actions: '<path d="M13 2 4 14h7l-1 8 9-12h-7l1-8z"/>',
  bell: '<path d="M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9M13.7 21a2 2 0 0 1-3.4 0"/>',
  users: '<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.9M16 3.1a4 4 0 0 1 0 7.8"/>',
  settings: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/>',
  folder: '<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>',
  search: '<circle cx="11" cy="11" r="7"/><path d="m21 21-4.3-4.3"/>',
  menu: '<path d="M3 6h18M3 12h18M3 18h18"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  alert: '<path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0zM12 9v4M12 17h.01"/>',
  info: '<circle cx="12" cy="12" r="10"/><path d="M12 16v-4M12 8h.01"/>',
  x: '<path d="M18 6 6 18M6 6l12 12"/>',
  more: '<circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/>',
  monitor: '<rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/>',
  play: '<path d="m6 4 14 8-14 8z"/>',
  check: '<path d="M20 6 9 17l-5-5"/>',
};
function iconSVG(name) { return `<svg viewBox="0 0 24 24" aria-hidden="true">${ICONS[name] || ''}</svg>`; }
function renderIcons(root = document) { root.querySelectorAll('i[data-icon]').forEach(i => { if (!i.firstChild) i.innerHTML = iconSVG(i.dataset.icon); }); }

/* ---------- Noyau : API, toast, modal, menus ---------- */
const App = {
  async api(method, url, body) {
    const opt = { method, headers: {} };
    if (body !== undefined) { opt.headers['Content-Type'] = 'application/json'; opt.body = JSON.stringify(body); }
    const r = await fetch(url, opt);
    if (r.status === 401) { location.href = '/login?next=' + encodeURIComponent(location.pathname); throw new Error('session expirée'); }
    const text = await r.text();
    let data = null; try { data = text ? JSON.parse(text) : null; } catch { data = text; }
    if (!r.ok) { const msg = (data && data.error) || r.statusText; App.toast(msg, true); throw new Error(msg); }
    return data;
  },
  toast(msg, err) { const t = document.getElementById('toast'); t.textContent = msg; t.className = 'toast' + (err ? ' err' : ''); t.hidden = false; clearTimeout(App._t); App._t = setTimeout(() => t.hidden = true, err ? 6000 : 2800); },
  modal(html) { const m = document.getElementById('modal'); document.getElementById('modal-body').innerHTML = html; m.hidden = false; renderIcons(m); m.onclick = e => { if (e.target === m) App.closeModal(); }; const f = m.querySelector('input,select,textarea'); if (f) f.focus(); },
  closeModal() { document.getElementById('modal').hidden = true; },
  esc(s) { return String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); },
  copy(t) { navigator.clipboard.writeText(t).then(() => App.toast('Copié')); },
  age(sec) { if (sec == null) return '—'; if (sec < 60) return sec + ' s'; if (sec < 3600) return Math.floor(sec / 60) + ' min'; if (sec < 86400) return Math.floor(sec / 3600) + ' h'; return Math.floor(sec / 86400) + ' j'; },
  fmtDate(s) { if (!s) return '—'; const d = new Date(s); return isNaN(d) ? s : d.toLocaleString('fr-CA', { dateStyle: 'short', timeStyle: 'short' }); },
  rel(s) { const d = new Date(s); const sec = Math.max(0, Math.floor((Date.now() - d) / 1000)); return sec < 60 ? 'à l’instant' : 'il y a ' + App.age(sec); },
  val(id) { return document.getElementById(id).value; },
  confirm(msg) { return window.confirm(msg); },
  isAdmin() { return window.ME && window.ME.role === 'admin'; },
  /* menu déroulant ancré sous un élément */
  menu(anchor, items) {
    App.closeMenu();
    const m = document.createElement('div'); m.className = 'menu';
    m.innerHTML = items.map(it => it === '-' ? '<div class="sep"></div>' : it.href ? `<a href="${it.href}" class="${it.cls || ''}">${it.label}</a>` : `<button class="${it.cls || ''}" data-k="${it.k}">${it.label}</button>`).join('');
    document.body.appendChild(m);
    const r = anchor.getBoundingClientRect(); const below = r.bottom + 6 + window.scrollY;
    m.style.top = below + 'px'; m.style.left = Math.min(r.left + window.scrollX, window.innerWidth - m.offsetWidth - 12) + 'px';
    if (r.bottom + m.offsetHeight + 12 > window.innerHeight) { m.style.top = (r.top + window.scrollY - m.offsetHeight - 6) + 'px'; }
    m.querySelectorAll('button[data-k]').forEach(b => b.onclick = () => { const it = items.find(x => x.k === b.dataset.k); App.closeMenu(); it && it.onClick && it.onClick(); });
    App._menu = m; setTimeout(() => document.addEventListener('click', App._closeOnClick = e => { if (!m.contains(e.target)) App.closeMenu(); }), 0);
  },
  closeMenu() { if (App._menu) { App._menu.remove(); App._menu = null; document.removeEventListener('click', App._closeOnClick); } },
  /* en-tête : action principale et recherche */
  setPrimary(label, onClick, icon = 'plus') { const s = document.getElementById('page-actions'); s.innerHTML = `<button class="btn primary" id="primary-btn"><i data-icon="${icon}"></i>${App.esc(label)}</button>`; renderIcons(s); document.getElementById('primary-btn').onclick = onClick; },
  onSearch(fn) { const box = document.getElementById('search-box'); box.hidden = false; const inp = document.getElementById('global-search'); inp.oninput = () => fn(inp.value.trim().toLowerCase()); document.addEventListener('keydown', e => { if (e.key === '/' && document.activeElement.tagName !== 'INPUT' && document.activeElement.tagName !== 'TEXTAREA') { e.preventDefault(); inp.focus(); } }); },
  changePassword() {
    App.modal(`<h2>Changer mon mot de passe</h2>
      <label class="field"><span>Mot de passe actuel</span><input id="pw-cur" type="password" autocomplete="current-password"></label>
      <label class="field"><span>Nouveau mot de passe (8 caractères min.)</span><input id="pw-new" type="password" autocomplete="new-password"></label>
      <div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="App.savePassword()">Enregistrer</button></div>`);
  },
  async savePassword() { await App.api('POST', '/api/me/password', { current: App.val('pw-cur'), password: App.val('pw-new') }); App.closeModal(); App.toast('Mot de passe changé'); },
  newProject() {
    App.modal(`<h2>Nouveau projet</h2><label class="field"><span>Nom</span><input id="np-name" placeholder="Festival 2027"></label>
      <label class="field"><span>Fuseau horaire</span><input id="np-tz" value="America/Toronto"></label>
      <div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="App.createProject()">Créer</button></div>`);
  },
  async createProject() { const p = await App.api('POST', '/api/projects', { name: App.val('np-name'), timezone: App.val('np-tz') }); location.href = '/p/' + p.slug + '/settings'; },
  /* coquille : sidebar mobile, menus, badges */
  initShell() {
    renderIcons();
    const sb = document.getElementById('sidebar'), scrim = document.getElementById('scrim'), mb = document.getElementById('menu-btn');
    if (mb) mb.onclick = () => { sb.classList.add('open'); scrim.hidden = false; };
    if (scrim) scrim.onclick = () => { sb.classList.remove('open'); scrim.hidden = true; };
    const um = document.getElementById('user-menu');
    if (um) um.onclick = e => { e.stopPropagation(); App.menu(um, [
      { k: 'pw', label: 'Changer mon mot de passe', onClick: App.changePassword },
      ...(App.isAdmin() ? [{ href: '/admin/users', label: 'Utilisateurs' }] : []),
      '-', { k: 'out', label: 'Se déconnecter', cls: 'danger', onClick: () => { const f = document.createElement('form'); f.method = 'post'; f.action = '/logout'; document.body.appendChild(f); f.submit(); } },
    ]); };
    const sw = document.getElementById('project-switcher');
    if (sw) sw.onclick = e => { e.stopPropagation(); App.menu(sw, [
      ...(window.PROJECTS || []).map(p => ({ href: '/p/' + p.slug, label: App.esc(p.name), cls: PROJECT && p.id === PROJECT.id ? 'on' : '' })),
      '-', { href: '/', label: 'Tous les projets' }, ...(App.isAdmin() ? [{ k: 'new', label: 'Nouveau projet', onClick: App.newProject }] : []),
    ]); };
    if (window.PROJECT) App.pollBadges();
  },
  async pollBadges() {
    try {
      const key = 'fcc.seen.' + PROJECT.id; const seen = +(localStorage.getItem(key) || 0);
      const [ev, devs] = await Promise.all([App.api('GET', `/api/projects/${PROJECT.id}/events?after=${seen}&limit=100`), App.api('GET', `/api/projects/${PROJECT.id}/devices`)]);
      const unread = ev.length, pending = devs.filter(d => d.status === 'pending').length;
      const nb = document.getElementById('nav-unread'), bb = document.getElementById('bell-badge'), np = document.getElementById('nav-pending');
      if (nb) { nb.hidden = !unread; nb.textContent = unread > 99 ? '99+' : unread; } if (bb) bb.hidden = !unread;
      if (np) { np.hidden = !pending; np.textContent = pending; }
    } catch { }
    setTimeout(App.pollBadges, 30000);
  },
};

/* ---------- Composants ---------- */
const UI = {
  badge(kind, text) { return `<span class="badge ${kind}">${App.esc(text)}</span>`; },
  status(d) { if (d.status !== 'approved') return UI.badge(d.status === 'pending' ? 'warn' : '', { pending: 'En attente', rejected: 'Refusé', revoked: 'Révoqué' }[d.status] || d.status); return d.online ? UI.badge('ok', 'En ligne') : UI.badge('danger', 'Hors ligne ' + App.age(d.age_seconds)); },
  empty(title, text, action) { return `<div class="empty"><h3>${App.esc(title)}</h3><p>${text || ''}</p>${action ? `<button class="btn primary" onclick="${action.onclick}">${App.esc(action.label)}</button>` : ''}</div>`; },
  skeleton(n = 4) { return `<div class="table-wrap" style="padding:8px 16px">${'<div class="skeleton"></div>'.repeat(n)}</div>`; },
  table(cols, rows, empty) { if (!rows.length) return empty; return `<div class="table-wrap"><table class="table"><thead><tr>${cols.map(c => `<th${c.right ? ' class="right"' : ''}>${c}</th>`).join('')}</tr></thead><tbody>${rows.join('')}</tbody></table></div>`; },
  subDevicesOf(hb) { return hb.sub_devices || (hb.processor ? [Object.assign({ name: 'Processeur' }, hb.processor)] : []); },
  deviceTile(d) {
    const hb = d.last_heartbeat || {}; const subs = UI.subDevicesOf(hb); const relayed = (hb.peers || []).some(p => p.online && !p.direct);
    const lines = !d.online ? '' : subs.length ? subs.map(sd => `<span class="dot ${sd.reachable ? 'ok' : 'danger'}"></span>${App.esc(sd.name)}${sd.reachable ? ' · ' + sd.rtt_ms.toFixed(1) + ' ms' : ' · injoignable'}`).join('<br>') : '<span class="muted">aucun sous-appareil</span>';
    return `<div class="tile ${d.online ? '' : 'off'}"><b><span class="dot ${d.online ? 'ok' : 'danger'}"></span>${App.esc(d.name)}</b><div class="sub-line">${d.online ? 'en ligne' : 'hors ligne depuis ' + App.age(d.age_seconds)}${relayed ? ' · relais' : ''}</div><div>${lines}</div></div>`;
  },
  deviceDetail(d) {
    const hb = d.last_heartbeat || {}; const sys = hb.system || {}; const subs = UI.subDevicesOf(hb); const peers = (hb.peers || []).filter(p => p.online); const fwds = d.config.forwards || [];
    return `<dl class="kv">
      <dt>État</dt><dd>${UI.status(d)}${d.online ? ` <span class="muted small">vu il y a ${App.age(d.age_seconds)}</span>` : ''}</dd>
      <dt>IP réseau privé</dt><dd><code>${App.esc(d.tailnet_ip || '—')}</code></dd>
      <dt>Forwards</dt><dd>${d.tailnet_ip && fwds.length ? fwds.map(f => `<code>${App.esc(d.tailnet_ip)}:${f.listen}</code> <span class="muted small">${App.esc(f.name)} → ${App.esc(f.target)}</span>`).join('<br>') : '<span class="muted">aucun</span>'}</dd>
      <dt>Sous-appareils</dt><dd>${subs.length ? subs.map(sd => `<span class="dot ${sd.reachable ? 'ok' : 'danger'}"></span>${App.esc(sd.name)} <span class="muted small">${App.esc(sd.target)} · ${sd.reachable ? sd.rtt_ms.toFixed(1) + ' ms' : App.esc(sd.error || 'injoignable')}</span>`).join('<br>') : '<span class="muted">aucun</span>'}</dd>
      <dt>Chemin réseau</dt><dd>${peers.length ? peers.map(p => `${App.esc(p.hostname)} ${p.direct ? UI.badge('ok', 'direct') : UI.badge('warn', 'relais ' + p.relay)} <span class="muted small">${p.latency_ms ? p.latency_ms.toFixed(1) + ' ms' : ''}</span>`).join('<br>') : '<span class="muted">aucun pair en ligne</span>'}</dd>
      <dt>Machine</dt><dd>${App.esc(sys.hostname || d.hostname)} · ${App.esc(sys.os || d.os)}${sys.cpu_percent != null ? `<span class="muted small"> · CPU ${Math.round(sys.cpu_percent)} % · RAM ${Math.round(sys.mem_percent)} % · allumée depuis ${App.age(sys.uptime_seconds)}</span>` : ''}</dd>
      <dt>Agent</dt><dd>${App.esc(d.agent_version || '—')}</dd>
      <dt>Bureau à distance</dt><dd>${hb.remote_desktop && hb.remote_desktop.id ? `<a class="btn small" href="rustdesk://connection/new/${App.esc(hb.remote_desktop.id)}">Ouvrir dans RustDesk</a> <code>${App.esc(hb.remote_desktop.id)}</code>` : '<span class="muted">RustDesk non détecté</span>'}</dd>
    </dl>`;
  },
  eventRow(e) { const ic = e.level === 'error' ? 'alert' : e.level === 'warn' ? 'alert' : 'info'; return `<div class="event ${e.level}"><i data-icon="${ic}"></i><div class="msg">${App.esc(e.message)}</div><div class="when" title="${App.esc(e.created_at)}">${App.rel(e.created_at)}</div></div>`; },
};

/* ---------- Projets ---------- */
const Projects = {
  async init() {
    if (App.isAdmin()) App.setPrimary('Nouveau projet', App.newProject);
    const root = document.getElementById('projects-root'); root.innerHTML = UI.skeleton(3);
    const ps = await App.api('GET', '/api/projects');
    if (!ps.length) { root.innerHTML = UI.empty('Aucun projet', App.isAdmin() ? 'Créez votre premier projet pour commencer à y rattacher des appareils.' : 'Aucun projet ne vous a encore été attribué.', App.isAdmin() ? { label: 'Nouveau projet', onclick: 'App.newProject()' } : null); return; }
    root.innerHTML = `<div class="cards">${ps.map(p => `<a class="card" href="/p/${p.slug}"><h2>${App.esc(p.name)}</h2><div class="meta"><span>${p.online}/${p.devices} appareils en ligne</span>${p.pending ? UI.badge('warn', p.pending + ' en attente') : ''}<span>${App.esc(p.timezone)}</span></div></a>`).join('')}</div>`;
  },
};

/* ---------- Tableau de bord ---------- */
const Dash = {
  grid: null, editing: false, widgets: [], data: null, dashboards: [], current: null,
  async init() {
    GridStack.renderCB = (el, w) => { el.innerHTML = w.content || ''; };
    Dash.grid = GridStack.init({ column: 12, cellHeight: 64, margin: 6, float: true, disableDrag: true, disableResize: true, animate: false }, '#grid');
    Dash.grid.on('change', (e, items) => items.forEach(it => { const w = Dash.widgets.find(x => x.id === it.el.dataset.id); if (w) Object.assign(w, { x: it.x, y: it.y, w: it.w, h: it.h }); }));
    const nb = document.getElementById('dash-new'); if (nb) nb.onclick = Dash.create;
    await Dash.loadList(); Dash.refresh(); setInterval(Dash.refresh, 10000);
  },
  async loadList() {
    Dash.dashboards = await App.api('GET', `/api/projects/${PROJECT.id}/dashboards`);
    if (!Dash.dashboards.length && App.isAdmin()) { const d = await App.api('POST', `/api/projects/${PROJECT.id}/dashboards`, { name: 'Principal' }); Dash.dashboards = [d]; }
    const wanted = +(localStorage.getItem('fcc.dash.' + PROJECT.id) || 0);
    Dash.select(Dash.dashboards.find(d => d.id === wanted) || Dash.dashboards[0] || null);
  },
  renderTabs() {
    const t = document.getElementById('dash-tabs');
    t.innerHTML = Dash.dashboards.map(d => `<button class="dash-tab ${Dash.current && d.id === Dash.current.id ? 'on' : ''}" data-id="${d.id}">${App.esc(d.name)}</button>`).join('');
    t.querySelectorAll('.dash-tab').forEach(b => b.onclick = e => { const d = Dash.dashboards.find(x => x.id == b.dataset.id); if (Dash.current && d.id === Dash.current.id && App.isAdmin()) { Dash.tabMenu(b, d); } else Dash.select(d); });
  },
  tabMenu(anchor, d) {
    App.menu(anchor, [
      { k: 'edit', label: 'Modifier les widgets', onClick: Dash.toggleEdit },
      { k: 'rename', label: 'Renommer', onClick: () => Dash.rename(d) }, '-',
      { k: 'del', label: 'Supprimer ce tableau de bord', cls: 'danger', onClick: () => Dash.remove(d) },
    ]);
  },
  select(d) {
    if (Dash.editing) Dash.toggleEdit();
    Dash.current = d; if (d) localStorage.setItem('fcc.dash.' + PROJECT.id, d.id);
    Dash.widgets = d && Array.isArray(d.layout) ? JSON.parse(JSON.stringify(d.layout)) : [];
    Dash.renderTabs(); Dash.renderAll(); Dash.setHeader();
  },
  setHeader() {
    if (!App.isAdmin()) return;
    if (Dash.editing) { document.getElementById('page-actions').innerHTML = `<button class="btn ghost" onclick="Dash.addWidget()"><i data-icon="plus"></i>Widget</button><button class="btn ghost" onclick="Dash.toggleEdit()">Annuler</button><button class="btn primary" onclick="Dash.save()">Enregistrer</button>`; renderIcons(document.getElementById('page-actions')); }
    else App.setPrimary('Modifier', Dash.toggleEdit, 'settings');
  },
  renderAll() {
    Dash.grid.removeAll();
    const empty = document.getElementById('dash-empty');
    if (!Dash.current) { empty.hidden = false; empty.innerHTML = UI.empty('Aucun tableau de bord', 'Un administrateur peut en créer un.'); return; }
    if (!Dash.widgets.length) { empty.hidden = false; empty.innerHTML = UI.empty(App.esc(Dash.current.name) + ' est vide', 'Ajoutez des widgets : état des appareils, indicateurs, boutons d’action, notifications.', App.isAdmin() ? { label: 'Ajouter un widget', onclick: 'Dash.editing||Dash.toggleEdit();Dash.addWidget()' } : null); }
    else empty.hidden = true;
    Dash.widgets.forEach(w => Dash.grid.addWidget({ x: w.x, y: w.y, w: w.w || 4, h: w.h || 3, id: w.id, content: Dash.widgetHTML(w) }));
    Dash.grid.engine.nodes.forEach(n => { n.el.dataset.id = n.id; });
    Dash.fill();
  },
  widgetHTML(w) {
    const rm = App.isAdmin() ? `<button class="btn ghost small rm" onclick="Dash.remove_w('${w.id}')" ${Dash.editing ? '' : 'hidden'} aria-label="Retirer"><i data-icon="x"></i></button>` : '';
    const title = w.title || Dash.defaultTitle(w);
    return `<div class="widget-head">${title ? `<h3>${App.esc(title)}</h3>` : ''}${rm}</div><div class="widget-body" data-wid="${w.id}"><div class="skeleton"></div></div>`;
  },
  defaultTitle(w) { return { screens: 'Appareils', screen: 'Appareil', button: '', kpi: '', automations: 'Automatisations', events: 'Dernières notifications', note: '' }[w.type] ?? w.type; },
  async refresh() {
    try { const [ov, ev] = await Promise.all([App.api('GET', `/api/projects/${PROJECT.id}/overview`), App.api('GET', `/api/projects/${PROJECT.id}/events?limit=8`)]); Dash.data = ov; Dash.data.events = ev; } catch { return; }
    Dash.fill();
  },
  fill() { if (!Dash.data) return; document.querySelectorAll('.widget-body').forEach(el => { const w = Dash.widgets.find(x => x.id === el.dataset.wid); if (w) { el.innerHTML = Dash.body(w); renderIcons(el); } }); },
  body(w) {
    const D = Dash.data;
    switch (w.type) {
      case 'kpi': {
        const online = D.devices.filter(d => d.online).length, total = D.devices.length;
        const ko = D.devices.filter(d => d.online && UI.subDevicesOf(d.last_heartbeat || {}).some(s => !s.reachable)).length;
        const m = { online: [online + ' / ' + total, 'appareils en ligne', online < total ? 'warn' : ''], subko: [ko, 'sous-appareils injoignables', ko ? 'danger' : ''], pending: [D.pending, 'en attente d’approbation', D.pending ? 'warn' : ''], relayed: [D.devices.filter(d => d.online && (d.last_heartbeat.peers || []).some(p => p.online && !p.direct)).length, 'appareils relayés', ''] }[w.metric || 'online'];
        return `<div class="kpi"><div class="value ${m[2]}">${m[0]}</div><div class="label">${m[1]}</div></div>`;
      }
      case 'screens': return D.devices.length ? `<div class="tiles">${D.devices.map(UI.deviceTile).join('')}</div>` : '<span class="muted">Aucun appareil approuvé.</span>';
      case 'screen': { const d = D.devices.find(x => x.id == w.device_id); return d ? UI.deviceDetail(d) : '<span class="muted">Appareil introuvable ou non approuvé.</span>'; }
      case 'button': { const a = D.actions.find(x => x.id == w.action_id); if (!a) return '<span class="muted">Action introuvable.</span>'; return `<div class="bigbtn"><button class="btn primary" onclick="Dash.run(${a.id}, this)"><i data-icon="play"></i>${App.esc(a.name)}</button><div class="muted small">${a.last_run ? App.rel(a.last_run) + ' · ' + App.esc(a.last_result) : 'jamais exécutée'}</div></div>`; }
      case 'automations': return D.automations.length ? `<table class="table" style="font-size:13px">${D.automations.map(a => `<tr><td style="padding:6px 0"><span class="dot ${a.enabled ? 'ok' : ''}"></span>${App.esc(a.name)}</td><td class="muted" style="padding:6px 8px">${App.esc(a.action_name)}</td><td class="muted small" style="padding:6px 0;text-align:right">${a.enabled ? App.esc(a.next_run) : 'désactivée'}</td></tr>`).join('')}</table>` : '<span class="muted">Aucune automatisation.</span>';
      case 'events': return D.events.length ? `<div class="event-list">${D.events.map(UI.eventRow).join('')}</div>` : '<span class="muted">Aucune notification.</span>';
      case 'note': return `<div>${App.esc(w.text || '').replace(/\n/g, '<br>')}</div>`;
    }
    return '';
  },
  async run(id, btn) { btn.disabled = true; try { const r = await App.api('POST', `/api/actions/${id}/run`); App.toast(r.result); await Dash.refresh(); } finally { btn.disabled = false; } },
  toggleEdit() {
    Dash.editing = !Dash.editing; document.body.classList.toggle('editing', Dash.editing);
    Dash.grid.enableMove(Dash.editing); Dash.grid.enableResize(Dash.editing);
    document.querySelectorAll('.widget-head .rm').forEach(b => b.hidden = !Dash.editing);
    if (!Dash.editing) { Dash.widgets = Dash.current && Array.isArray(Dash.current.layout) ? JSON.parse(JSON.stringify(Dash.current.layout)) : []; Dash.renderAll(); }
    Dash.setHeader();
  },
  addWidget() {
    const D = Dash.data || { devices: [], actions: [] };
    App.modal(`<h2>Ajouter un widget</h2>
      <label class="field"><span>Type</span><select id="w-type" onchange="Dash.onType()">
        <option value="kpi">Indicateur</option><option value="screens">État de tous les appareils</option><option value="screen">Détail d'un appareil</option>
        <option value="button">Bouton d'action</option><option value="events">Dernières notifications</option><option value="automations">Automatisations</option><option value="note">Note</option></select></label>
      <label class="field" id="w-metric-l"><span>Indicateur</span><select id="w-metric"><option value="online">Appareils en ligne</option><option value="subko">Sous-appareils injoignables</option><option value="pending">En attente d'approbation</option><option value="relayed">Appareils relayés</option></select></label>
      <label class="field"><span>Titre (optionnel)</span><input id="w-title"></label>
      <label class="field" id="w-dev-l" hidden><span>Appareil</span><select id="w-dev">${D.devices.map(d => `<option value="${d.id}">${App.esc(d.name)}</option>`).join('')}</select></label>
      <label class="field" id="w-act-l" hidden><span>Action</span><select id="w-act">${D.actions.map(a => `<option value="${a.id}">${App.esc(a.name)}</option>`).join('')}</select></label>
      <label class="field" id="w-text-l" hidden><span>Texte</span><textarea id="w-text"></textarea></label>
      <div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="Dash.confirmAdd()">Ajouter</button></div>`);
  },
  onType() { const t = App.val('w-type'); document.getElementById('w-metric-l').hidden = t !== 'kpi'; document.getElementById('w-dev-l').hidden = t !== 'screen'; document.getElementById('w-act-l').hidden = t !== 'button'; document.getElementById('w-text-l').hidden = t !== 'note'; },
  confirmAdd() {
    const t = App.val('w-type'); const size = { kpi: [3, 2], screens: [8, 4], screen: [5, 5], button: [3, 2], events: [4, 5], automations: [5, 3], note: [4, 2] }[t];
    const w = { id: 'w' + Date.now().toString(36), type: t, title: App.val('w-title'), w: size[0], h: size[1] };
    if (t === 'kpi') w.metric = App.val('w-metric'); if (t === 'screen') w.device_id = +App.val('w-dev'); if (t === 'button') w.action_id = +App.val('w-act'); if (t === 'note') w.text = App.val('w-text');
    if ((t === 'screen' && !w.device_id) || (t === 'button' && !w.action_id)) { App.toast('Aucun élément disponible pour ce type', true); return; }
    Dash.widgets.push(w); App.closeModal(); document.getElementById('dash-empty').hidden = true;
    const el = Dash.grid.addWidget({ w: w.w, h: w.h, id: w.id, content: Dash.widgetHTML(w) }); el.dataset.id = w.id;
    const n = Dash.grid.engine.nodes.find(n => n.id === w.id); if (n) Object.assign(w, { x: n.x, y: n.y }); Dash.fill(); renderIcons(el);
  },
  remove_w(id) { Dash.widgets = Dash.widgets.filter(w => w.id !== id); const el = document.querySelector(`.grid-stack-item[data-id="${id}"]`); if (el) Dash.grid.removeWidget(el); },
  async save() {
    Dash.grid.engine.nodes.forEach(n => { const w = Dash.widgets.find(x => x.id === n.id); if (w) Object.assign(w, { x: n.x, y: n.y, w: n.w, h: n.h }); });
    const d = await App.api('PUT', `/api/dashboards/${Dash.current.id}`, { layout: Dash.widgets });
    Dash.current.layout = d.layout; Dash.dashboards = Dash.dashboards.map(x => x.id === d.id ? d : x); Dash.toggleEdit(); App.toast('Tableau de bord enregistré');
  },
  create() { App.modal(`<h2>Nouveau tableau de bord</h2><label class="field"><span>Nom</span><input id="nd-name" placeholder="Régie, Soirée, Techniciens…"></label><div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="Dash.confirmCreate()">Créer</button></div>`); },
  async confirmCreate() { const d = await App.api('POST', `/api/projects/${PROJECT.id}/dashboards`, { name: App.val('nd-name') }); App.closeModal(); Dash.dashboards.push(d); Dash.select(d); },
  rename(d) { App.modal(`<h2>Renommer</h2><label class="field"><span>Nom</span><input id="rd-name" value="${App.esc(d.name)}"></label><div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="Dash.confirmRename(${d.id})">Enregistrer</button></div>`); },
  async confirmRename(id) { const d = await App.api('PUT', `/api/dashboards/${id}`, { name: App.val('rd-name') }); App.closeModal(); Dash.dashboards = Dash.dashboards.map(x => x.id === d.id ? d : x); if (Dash.current.id === d.id) Dash.current = d; Dash.renderTabs(); },
  async remove(d) { if (!App.confirm(`Supprimer le tableau de bord « ${d.name} » ?`)) return; await App.api('DELETE', `/api/dashboards/${d.id}`); Dash.dashboards = Dash.dashboards.filter(x => x.id !== d.id); Dash.select(Dash.dashboards[0] || null); },
};

/* ---------- Appareils ---------- */
const Devices = {
  list: [], open: new Set(), q: '',
  init() { App.onSearch(q => { Devices.q = q; Devices.render(); }); Devices.refresh(); setInterval(Devices.refresh, 10000); },
  async refresh() { Devices.list = await App.api('GET', `/api/projects/${PROJECT.id}/devices`); Devices.render(); },
  render() {
    const q = Devices.q; const match = d => !q || (d.name + ' ' + d.slug + ' ' + d.hostname + ' ' + d.tailnet_ip).toLowerCase().includes(q);
    const pend = Devices.list.filter(d => d.status === 'pending' && match(d));
    document.getElementById('pending-section').hidden = pend.length === 0;
    document.getElementById('pending-list').innerHTML = pend.map(d => `<div class="card"><h2>${App.esc(d.slug)}</h2>
      <dl class="kv"><dt>Nom d'hôte</dt><dd>${App.esc(d.hostname)}</dd><dt>Système</dt><dd>${App.esc(d.os)} ${App.esc(d.arch)}</dd><dt>Agent</dt><dd>${App.esc(d.agent_version)}</dd><dt>Inscrit</dt><dd>${App.rel(d.enrolled_at)}</dd>
      <dt>Sous-appareils</dt><dd>${(d.config.sub_devices || []).map(sd => App.esc(sd.name + ' ' + sd.ip + ':' + sd.port)).join('<br>') || '—'}</dd></dl>
      ${App.isAdmin() ? `<div class="btnrow"><button class="btn primary" onclick="Devices.act(${d.id},'approve')"><i data-icon="check"></i>Approuver</button><button class="btn" onclick="Devices.act(${d.id},'reject')">Refuser</button></div>` : ''}</div>`).join('');
    const others = Devices.list.filter(d => d.status !== 'pending' && match(d));
    const host = document.getElementById('device-list');
    if (!others.length) { host.innerHTML = Devices.list.length ? UI.empty('Aucun résultat', 'Aucun appareil ne correspond à la recherche.') : UI.empty('Aucun appareil', 'Installez l’agent avec la clé du projet : l’appareil apparaîtra ici en attente d’approbation.'); }
    else host.innerHTML = others.map(Devices.card).join('');
    renderIcons(host); renderIcons(document.getElementById('pending-list'));
  },
  card(d) {
    const open = Devices.open.has(d.id); const subs = UI.subDevicesOf(d.last_heartbeat || {}); const ko = subs.filter(s => !s.reachable).length;
    return `<div class="device" data-id="${d.id}"><div class="head" onclick="Devices.toggle(${d.id})">
      <span class="dot ${d.status !== 'approved' ? 'warn' : d.online ? (ko ? 'warn' : 'ok') : 'danger'}"></span>
      <span><span class="name">${App.esc(d.name)}</span><span class="slug">${App.esc(d.slug)}</span></span>
      <span class="muted small hide-m">${subs.length ? (subs.length - ko) + '/' + subs.length + ' sous-appareils' : ''}</span>
      <span class="hide-m">${d.tailnet_ip ? `<code>${App.esc(d.tailnet_ip)}</code>` : ''}</span>
      ${UI.status(d)}</div>
      ${open ? `<div class="body"><div><h3>État</h3>${UI.deviceDetail(d)}<h3>Dernières commandes</h3><div class="cmds" id="cmds-${d.id}"><div class="skeleton"></div></div></div><div>${Devices.configForm(d)}</div></div>` : ''}</div>`;
  },
  toggle(id) { Devices.open.has(id) ? Devices.open.delete(id) : Devices.open.add(id); const d = Devices.list.find(x => x.id === id); const el = document.querySelector(`.device[data-id="${id}"]`); el.outerHTML = Devices.card(d); renderIcons(document.getElementById('device-list')); if (Devices.open.has(id)) Devices.loadCmds(id); },
  async loadCmds(id) { const cmds = await App.api('GET', `/api/devices/${id}/commands`); const el = document.getElementById('cmds-' + id); if (!el) return; el.innerHTML = cmds.length ? cmds.map(c => `<div>${UI.badge(c.status === 'done' ? 'ok' : c.status === 'failed' ? 'danger' : '', c.status)}<span>${App.esc(c.kind)}</span><span class="muted">${App.rel(c.created_at)}</span>${c.result ? `<span class="muted">· ${App.esc(c.result)}</span>` : ''}</div>`).join('') : '<span class="muted">aucune</span>'; },
  configForm(d) {
    const c = d.config; const ro = !App.isAdmin(); const dis = ro ? 'disabled' : '';
    return `<h3>Configuration à distance <span class="faint" style="text-transform:none;letter-spacing:0">· version ${d.config_version}</span></h3>
      <div class="row"><label class="field"><span>Nom affiché</span><input id="c-name" value="${App.esc(c.name)}" ${dis}></label><label class="field"><span>Heartbeat (s)</span><input id="c-hb" type="number" value="${c.heartbeat_seconds || 15}" ${dis}></label></div>
      <h3>Sous-appareils</h3>
      <div class="fwd-row sub hdr"><span>nom</span><span>IP</span><span>port</span><span></span></div>
      <div id="subs">${(c.sub_devices || []).map(sd => Devices.subRow(sd, ro)).join('')}</div>
      ${ro ? '' : `<button class="btn ghost small" onclick="Devices.addSub()"><i data-icon="plus"></i>Sous-appareil</button>`}
      <h3>Forwards <span class="faint" style="text-transform:none;letter-spacing:0">· IP privée de l'appareil → réseau local</span></h3>
      <div class="fwd-row hdr"><span>nom</span><span>proto</span><span>port</span><span>cible</span><span></span></div>
      <div id="fwds">${(c.forwards || []).map(f => Devices.fwdRow(f, ro)).join('')}</div>
      ${ro ? '' : `<button class="btn ghost small" onclick="Devices.addFwd()"><i data-icon="plus"></i>Forward</button>`}
      <h3>Mises à jour</h3>
      <div class="row"><label class="check"><input type="checkbox" id="c-upd" ${c.update && c.update.enabled ? 'checked' : ''} ${dis}> Mise à jour automatique</label><label class="field"><span>Vérification (heures)</span><input id="c-updh" type="number" value="${c.update && c.update.check_hours || 1}" ${dis}></label></div>
      ${ro ? '' : `<div class="btnrow"><button class="btn primary" onclick="Devices.saveConfig(${d.id})">Enregistrer et appliquer</button>
        <button class="btn" onclick="Devices.cmd(${d.id},'probe')">Sonder</button><button class="btn" onclick="Devices.cmd(${d.id},'restart')">Redémarrer l'agent</button><button class="btn" onclick="Devices.cmd(${d.id},'update')">Mettre à jour</button>
        ${d.status === 'approved' ? `<button class="btn danger-text" onclick="Devices.act(${d.id},'revoke')">Révoquer</button>` : `<button class="btn primary" onclick="Devices.act(${d.id},'approve')">Approuver</button>`}
        <button class="btn danger-text" onclick="Devices.remove(${d.id})">Supprimer</button></div>`}`;
  },
  subRow(sd, ro) { const dis = ro ? 'disabled' : ''; return `<div class="fwd-row sub"><input class="s-name" value="${App.esc(sd.name)}" placeholder="Processeur LED" ${dis}><input class="s-ip" value="${App.esc(sd.ip)}" placeholder="192.168.0.10" ${dis}><input class="s-port" type="number" value="${sd.port || ''}" placeholder="37564" ${dis}>${ro ? '<span></span>' : '<button class="icon-btn" onclick="this.parentNode.remove()" aria-label="Retirer"><i data-icon="x"></i></button>'}</div>`; },
  fwdRow(f, ro) { const dis = ro ? 'disabled' : ''; return `<div class="fwd-row"><input class="f-name" value="${App.esc(f.name)}" ${dis}><select class="f-proto" ${dis}><option ${f.proto === 'tcp' ? 'selected' : ''}>tcp</option><option ${f.proto === 'udp' ? 'selected' : ''}>udp</option></select><input class="f-listen" type="number" value="${f.listen || ''}" ${dis}><input class="f-target" value="${App.esc(f.target)}" placeholder="192.168.0.10:37564" ${dis}>${ro ? '<span></span>' : '<button class="icon-btn" onclick="this.parentNode.remove()" aria-label="Retirer"><i data-icon="x"></i></button>'}</div>`; },
  addSub() { const h = document.getElementById('subs'); h.insertAdjacentHTML('beforeend', Devices.subRow({ name: '', ip: '', port: '' }, false)); renderIcons(h); },
  addFwd() { const h = document.getElementById('fwds'); h.insertAdjacentHTML('beforeend', Devices.fwdRow({ name: '', proto: 'tcp', listen: '', target: '' }, false)); renderIcons(h); },
  async saveConfig(id) {
    const sub_devices = [...document.querySelectorAll('#subs .fwd-row.sub')].map(r => ({ name: r.querySelector('.s-name').value.trim(), ip: r.querySelector('.s-ip').value.trim(), port: +r.querySelector('.s-port').value })).filter(sd => sd.ip);
    const forwards = [...document.querySelectorAll('#fwds .fwd-row:not(.hdr)')].map(r => ({ name: r.querySelector('.f-name').value.trim(), proto: r.querySelector('.f-proto').value, listen: +r.querySelector('.f-listen').value, target: r.querySelector('.f-target').value.trim() }));
    await App.api('PUT', `/api/devices/${id}/config`, { name: App.val('c-name'), heartbeat_seconds: +App.val('c-hb'), sub_devices, forwards, update: { enabled: document.getElementById('c-upd').checked, check_hours: +App.val('c-updh') } });
    App.toast('Configuration enregistrée, appliquée au prochain heartbeat'); Devices.refresh();
  },
  async act(id, what) { if (what === 'revoke' && !App.confirm('Révoquer cet appareil ? Il sera retiré du réseau privé et devra être ré-approuvé.')) return; await App.api('POST', `/api/devices/${id}/${what}`); App.toast({ approve: 'Appareil approuvé : il rejoint le réseau dans quelques secondes', reject: 'Appareil refusé', revoke: 'Appareil révoqué' }[what]); Devices.refresh(); },
  async cmd(id, kind) { await App.api('POST', `/api/devices/${id}/command`, { kind }); App.toast('Commande envoyée, exécutée au prochain heartbeat'); setTimeout(() => Devices.loadCmds(id), 1500); },
  async remove(id) { if (!App.confirm('Supprimer définitivement cet appareil du projet ?')) return; await App.api('DELETE', `/api/devices/${id}`); Devices.open.delete(id); Devices.refresh(); },
};

/* ---------- Actions ---------- */
const Actions = {
  list: [], devices: [], q: '',
  async init() { if (App.isAdmin()) App.setPrimary('Nouvelle action', () => Actions.edit()); App.onSearch(q => { Actions.q = q; Actions.render(); }); await Actions.load(); },
  async load() { [Actions.list, Actions.devices] = await Promise.all([App.api('GET', `/api/projects/${PROJECT.id}/actions`), App.api('GET', `/api/projects/${PROJECT.id}/devices`)]); Actions.render(); },
  render() {
    const host = document.getElementById('action-list'); const rows = Actions.list.filter(a => !Actions.q || (a.name + ' ' + a.url).toLowerCase().includes(Actions.q));
    host.innerHTML = UI.table(['Action', 'Exécutée par', 'Requête', 'Dernier résultat', ''], rows.map(a => {
      const dev = a.device_id ? (Actions.devices.find(d => d.id === a.device_id) || {}).name : '';
      return `<tr><td><b>${App.esc(a.name)}</b></td><td>${a.kind === 'agent' ? 'agent de ' + App.esc(dev || '?') : 'hub'}</td><td><code>${App.esc(a.method)}</code> <span class="muted">${App.esc(a.url)}</span></td>
        <td class="muted small">${a.last_run ? App.rel(a.last_run) + '<span class="sub">' + App.esc(a.last_result) + '</span>' : '—'}</td>
        <td class="actions"><button class="btn small" onclick="Actions.run(${a.id})"><i data-icon="play"></i>Exécuter</button> ${App.isAdmin() ? `<button class="icon-btn" onclick="Actions.menu(event, ${a.id})" aria-label="Plus"><i data-icon="more"></i></button>` : ''}</td></tr>`; }),
      Actions.list.length ? UI.empty('Aucun résultat', 'Aucune action ne correspond à la recherche.') : UI.empty('Aucune action', 'Une action est une requête HTTP, exécutée par le hub (API externe) ou par l’agent d’un appareil (réseau local).', App.isAdmin() ? { label: 'Nouvelle action', onclick: 'Actions.edit()' } : null));
    renderIcons(host);
  },
  menu(e, id) { e.stopPropagation(); App.menu(e.currentTarget, [{ k: 'edit', label: 'Modifier', onClick: () => Actions.edit(id) }, '-', { k: 'del', label: 'Supprimer', cls: 'danger', onClick: () => Actions.remove(id) }]); },
  edit(id) {
    const a = Actions.list.find(x => x.id === id) || { kind: 'hub', method: 'GET', headers: {}, timeout_seconds: 10 };
    App.modal(`<h2>${id ? 'Modifier l’action' : 'Nouvelle action'}</h2>
      <label class="field"><span>Nom</span><input id="a-name" value="${App.esc(a.name || '')}"></label>
      <div class="row"><label class="field"><span>Exécutée par</span><select id="a-kind" onchange="document.getElementById('a-dev-l').hidden=this.value!=='agent'"><option value="hub" ${a.kind === 'hub' ? 'selected' : ''}>le hub (API externe)</option><option value="agent" ${a.kind === 'agent' ? 'selected' : ''}>l'agent d'un appareil</option></select></label>
      <label class="field" id="a-dev-l" ${a.kind === 'agent' ? '' : 'hidden'}><span>Appareil</span><select id="a-dev">${Actions.devices.filter(d => d.status === 'approved').map(d => `<option value="${d.id}" ${a.device_id === d.id ? 'selected' : ''}>${App.esc(d.name)}</option>`).join('')}</select></label></div>
      <div class="row"><label class="field"><span>Méthode</span><select id="a-method">${['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].map(m => `<option ${a.method === m ? 'selected' : ''}>${m}</option>`).join('')}</select></label><label class="field"><span>Délai (s)</span><input id="a-timeout" type="number" value="${a.timeout_seconds}"></label></div>
      <label class="field"><span>URL</span><input id="a-url" value="${App.esc(a.url || '')}" placeholder="http://192.168.0.10/api/…"></label>
      <label class="field"><span>En-têtes (JSON)</span><textarea id="a-headers">${App.esc(JSON.stringify(a.headers || {}, null, 1))}</textarea></label>
      <label class="field"><span>Corps</span><textarea id="a-body">${App.esc(a.body || '')}</textarea></label>
      <div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="Actions.save(${id || 0})">Enregistrer</button></div>`);
  },
  async save(id) {
    let headers; try { headers = JSON.parse(App.val('a-headers') || '{}'); } catch { App.toast('En-têtes : JSON invalide', true); return; }
    const a = { name: App.val('a-name'), kind: App.val('a-kind'), device_id: App.val('a-kind') === 'agent' ? +App.val('a-dev') || null : null, method: App.val('a-method'), url: App.val('a-url'), headers, body: App.val('a-body'), timeout_seconds: +App.val('a-timeout') };
    if (id) await App.api('PUT', `/api/actions/${id}`, a); else await App.api('POST', `/api/projects/${PROJECT.id}/actions`, a);
    App.closeModal(); Actions.load();
  },
  async run(id) { const r = await App.api('POST', `/api/actions/${id}/run`); App.toast(r.result); Actions.load(); },
  async remove(id) { if (!App.confirm('Supprimer cette action ?')) return; await App.api('DELETE', `/api/actions/${id}`); Actions.load(); },
};

/* ---------- Automatisations ---------- */
const Autos = {
  list: [], actions: [], days: ['dim', 'lun', 'mar', 'mer', 'jeu', 'ven', 'sam'],
  async init() { if (App.isAdmin()) App.setPrimary('Nouvelle automatisation', () => Autos.edit()); await Autos.load(); },
  async load() { [Autos.list, Autos.actions] = await Promise.all([App.api('GET', `/api/projects/${PROJECT.id}/automations`), App.api('GET', `/api/projects/${PROJECT.id}/actions`)]); Autos.render(); },
  render() {
    const host = document.getElementById('auto-list');
    host.innerHTML = UI.table(['Automatisation', 'Horaire', 'Action', 'Prochaine', 'Dernier résultat', ''], Autos.list.map(a =>
      `<tr><td><span class="dot ${a.enabled ? 'ok' : ''}"></span><b>${App.esc(a.name)}</b></td><td>${App.esc(Autos.describe(a.cron))}<span class="sub"><code>${App.esc(a.cron)}</code></span></td><td>${App.esc(a.action_name)}</td><td>${a.enabled ? App.esc(a.next_run) : '<span class="muted">désactivée</span>'}</td>
       <td class="muted small">${a.last_run ? App.rel(a.last_run) + '<span class="sub">' + App.esc(a.last_result) + '</span>' : '—'}</td>
       <td class="actions">${App.isAdmin() ? `<button class="icon-btn" onclick="Autos.menu(event, ${a.id})" aria-label="Plus"><i data-icon="more"></i></button>` : ''}</td></tr>`),
      UI.empty('Aucune automatisation', Autos.actions.length ? 'Exécutez une action selon un horaire, dans le fuseau du projet.' : 'Créez d’abord une action, puis planifiez-la ici.', App.isAdmin() && Autos.actions.length ? { label: 'Nouvelle automatisation', onclick: 'Autos.edit()' } : null));
    renderIcons(host);
  },
  menu(e, id) { e.stopPropagation(); App.menu(e.currentTarget, [{ k: 'edit', label: 'Modifier', onClick: () => Autos.edit(id) }, '-', { k: 'del', label: 'Supprimer', cls: 'danger', onClick: () => Autos.remove(id) }]); },
  describe(cron) { const p = cron.split(/\s+/); if (p.length !== 5 || !/^\d+$/.test(p[0]) || !/^\d+$/.test(p[1])) return 'expression cron'; const t = p[1].padStart(2, '0') + ':' + p[0].padStart(2, '0'); if (p[4] === '*' && p[2] === '*') return 'tous les jours à ' + t; if (p[2] === '*') return p[4].split(',').map(d => Autos.days[+d] ?? d).join(', ') + ' à ' + t; return 'le ' + p[2] + ' de chaque mois à ' + t; },
  edit(id) {
    const a = Autos.list.find(x => x.id === id) || { enabled: true, cron: '0 22 * * *', action_id: Autos.actions[0] && Autos.actions[0].id };
    const parts = a.cron.split(/\s+/); const simple = parts.length === 5 && /^\d+$/.test(parts[0]) && /^\d+$/.test(parts[1]) && parts[2] === '*'; const days = simple && parts[4] !== '*' ? parts[4].split(',') : ['0', '1', '2', '3', '4', '5', '6'];
    App.modal(`<h2>${id ? 'Modifier l’automatisation' : 'Nouvelle automatisation'}</h2>
      <label class="field"><span>Nom</span><input id="u-name" value="${App.esc(a.name || '')}" placeholder="Éteindre les appareils"></label>
      <label class="field"><span>Action</span><select id="u-action">${Autos.actions.map(x => `<option value="${x.id}" ${a.action_id === x.id ? 'selected' : ''}>${App.esc(x.name)}</option>`).join('')}</select></label>
      <div class="row"><label class="field"><span>Heure</span><input id="u-time" type="time" value="${simple ? parts[1].padStart(2, '0') + ':' + parts[0].padStart(2, '0') : '22:00'}" onchange="Autos.build()"></label><label class="check" style="margin-top:22px"><input type="checkbox" id="u-enabled" ${a.enabled ? 'checked' : ''}> Activée</label></div>
      <div class="field"><span>Jours</span><div style="display:flex;gap:10px;flex-wrap:wrap">${Autos.days.map((d, i) => `<label class="check"><input type="checkbox" class="u-day" value="${i}" ${days.includes(String(i)) ? 'checked' : ''} onchange="Autos.build()"> ${d}</label>`).join('')}</div></div>
      <label class="field"><span>Expression cron (modifiable)</span><input id="u-cron" value="${App.esc(a.cron)}"></label>
      <div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="Autos.save(${id || 0})">Enregistrer</button></div>`);
  },
  build() { const [h, m] = App.val('u-time').split(':'); const days = [...document.querySelectorAll('.u-day:checked')].map(c => c.value); document.getElementById('u-cron').value = `${+m} ${+h} * * ${days.length === 7 || !days.length ? '*' : days.join(',')}`; },
  async save(id) { const a = { name: App.val('u-name'), action_id: +App.val('u-action'), enabled: document.getElementById('u-enabled').checked, cron: App.val('u-cron') }; if (id) await App.api('PUT', `/api/automations/${id}`, a); else await App.api('POST', `/api/projects/${PROJECT.id}/automations`, a); App.closeModal(); Autos.load(); },
  async remove(id) { if (!App.confirm('Supprimer cette automatisation ?')) return; await App.api('DELETE', `/api/automations/${id}`); Autos.load(); },
};

/* ---------- Notifications ---------- */
const Notifs = {
  list: [], level: '', q: '',
  async init() {
    App.onSearch(q => { Notifs.q = q; Notifs.render(); });
    document.querySelectorAll('#notif-filters .chip').forEach(c => c.onclick = () => { document.querySelectorAll('#notif-filters .chip').forEach(x => x.classList.remove('on')); c.classList.add('on'); Notifs.level = c.dataset.level; Notifs.render(); });
    App.setPrimary('Tout marquer comme lu', Notifs.markRead, 'check');
    await Notifs.load(); setInterval(Notifs.load, 15000);
  },
  async load() { Notifs.list = await App.api('GET', `/api/projects/${PROJECT.id}/events?limit=300`); Notifs.render(); },
  render() {
    const rows = Notifs.list.filter(e => (!Notifs.level || e.level === Notifs.level) && (!Notifs.q || (e.message + ' ' + e.device_name).toLowerCase().includes(Notifs.q)));
    const host = document.getElementById('notif-list');
    host.innerHTML = rows.length ? `<div class="event-list event-page">${rows.map(UI.eventRow).join('')}</div>` : UI.empty('Aucune notification', 'Les événements des appareils, des actions et des automatisations apparaîtront ici.');
    renderIcons(host);
  },
  markRead() { const max = Math.max(0, ...Notifs.list.map(e => e.id)); localStorage.setItem('fcc.seen.' + PROJECT.id, max); ['nav-unread', 'bell-badge'].forEach(id => { const el = document.getElementById(id); if (el) el.hidden = true; }); App.toast('Notifications marquées comme lues'); },
};

/* ---------- Configuration du projet ---------- */
const Settings = {
  async init() {
    document.getElementById('p-toml').textContent = document.getElementById('p-toml').textContent.replace('{{ORIGIN}}', location.origin);
    const [users, members] = await Promise.all([App.api('GET', '/api/users'), App.api('GET', `/api/projects/${PROJECT.id}/members`)]);
    document.getElementById('members').innerHTML = users.map(u => `<label class="check"><input type="checkbox" class="m-user" value="${u.id}" ${members.includes(u.id) ? 'checked' : ''} ${u.role === 'admin' ? 'disabled checked' : ''}> ${App.esc(u.username)} <span class="muted small">${u.role === 'admin' ? 'administrateur' : 'utilisateur'}</span></label>`).join('') || '<span class="muted">Aucun utilisateur.</span>';
  },
  async save() { await App.api('PUT', `/api/projects/${PROJECT.id}`, { name: App.val('p-name'), timezone: App.val('p-tz') }); App.toast('Projet enregistré'); },
  async rotate() { if (!App.confirm('Régénérer la clé ? L’ancienne cessera d’accepter de nouveaux appareils.')) return; await App.api('POST', `/api/projects/${PROJECT.id}/rotate-key`); location.reload(); },
  async saveMembers() { const ids = [...document.querySelectorAll('.m-user:checked:not([disabled])')].map(c => +c.value); await App.api('PUT', `/api/projects/${PROJECT.id}/members`, ids); App.toast('Membres enregistrés'); },
  async remove() { if (!App.confirm(`Supprimer le projet « ${PROJECT.name} » et révoquer tous ses appareils ?`)) return; await App.api('DELETE', `/api/projects/${PROJECT.id}`); location.href = '/'; },
};

/* ---------- Utilisateurs ---------- */
const Users = {
  q: '',
  init() { App.setPrimary('Nouvel utilisateur', Users.create); App.onSearch(q => { Users.q = q; Users.render(); }); Users.render(); },
  render() {
    const rows = (window.USERS || []).filter(u => !Users.q || u.username.toLowerCase().includes(Users.q));
    document.getElementById('users-root').innerHTML = UI.table(['Utilisateur', 'Rôle', 'Créé', ''], rows.map(u => `<tr><td><span class="avatar" style="width:26px;height:26px;font-size:11px;margin-right:10px">${App.esc(u.username.slice(0, 2).toUpperCase())}</span><b>${App.esc(u.username)}</b></td><td>${UI.badge(u.role === 'admin' ? 'accent plain' : 'plain', u.role === 'admin' ? 'Administrateur' : 'Utilisateur')}</td><td class="muted small">${App.fmtDate(u.created_at)}</td>
      <td class="actions"><button class="icon-btn" onclick="Users.menu(event, ${u.id}, '${App.esc(u.username)}')" aria-label="Plus"><i data-icon="more"></i></button></td></tr>`), UI.empty('Aucun utilisateur', ''));
    renderIcons(document.getElementById('users-root'));
  },
  menu(e, id, name) { e.stopPropagation(); App.menu(e.currentTarget, [{ k: 'pw', label: 'Réinitialiser le mot de passe', onClick: () => Users.reset(id, name) }, ...(id !== ME.id ? ['-', { k: 'del', label: 'Supprimer', cls: 'danger', onClick: () => Users.remove(id, name) }] : [])]); },
  pw() { return Math.random().toString(36).slice(2, 8) + Math.random().toString(36).slice(2, 8); },
  create() { App.modal(`<h2>Nouvel utilisateur</h2><label class="field"><span>Nom d'utilisateur</span><input id="nu-name" autocomplete="off"></label><label class="field"><span>Mot de passe (8 caractères min.)</span><input id="nu-pw" type="text" value="${Users.pw()}"></label><label class="field"><span>Rôle</span><select id="nu-role"><option value="user">Utilisateur</option><option value="admin">Administrateur</option></select></label><div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="Users.save()">Créer</button></div>`); },
  async save() { await App.api('POST', '/api/users', { username: App.val('nu-name'), password: App.val('nu-pw'), role: App.val('nu-role') }); location.reload(); },
  reset(id, name) { App.modal(`<h2>Nouveau mot de passe pour ${App.esc(name)}</h2><label class="field"><span>Mot de passe</span><input id="rp-pw" type="text" value="${Users.pw()}"></label><div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="Users.saveReset(${id})">Enregistrer</button></div>`); },
  async saveReset(id) { await App.api('POST', `/api/users/${id}/password`, { password: App.val('rp-pw') }); App.closeModal(); App.toast('Mot de passe remplacé'); },
  async remove(id, name) { if (!App.confirm(`Supprimer l'utilisateur ${name} ?`)) return; await App.api('DELETE', `/api/users/${id}`); location.reload(); },
};

document.addEventListener('DOMContentLoaded', () => {
  App.initShell();
  const page = document.body.dataset.page;
  ({ projects: Projects, dashboard: Dash, devices: Devices, actions: Actions, automations: Autos, notifications: Notifs, settings: Settings, users: Users }[page] || { init() { } }).init();
});
