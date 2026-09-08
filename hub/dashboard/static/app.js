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
  chevron: '<path d="m6 9 6 6 6-6"/>',
  braces: '<path d="M8 3H7a2 2 0 0 0-2 2v4a2 2 0 0 1-2 2 2 2 0 0 1 2 2v4a2 2 0 0 0 2 2h1M16 3h1a2 2 0 0 1 2 2v4a2 2 0 0 0 2 2 2 2 0 0 0-2 2v4a2 2 0 0 1-2 2h-1"/>',
};
function iconSVG(name) { return `<svg viewBox="0 0 24 24" aria-hidden="true">${ICONS[name] || ''}</svg>`; }
function renderIcons(root = document) { root.querySelectorAll('i[data-icon]').forEach(i => { if (!i.firstChild) i.innerHTML = iconSVG(i.dataset.icon); }); }

/* ---------- Noyau : API, toast, modal, menus ---------- */
const App = {
  async api(method, url, body) {
    const opt = { method, headers: {} };
    if (body !== undefined) { opt.headers['Content-Type'] = 'application/json'; opt.body = JSON.stringify(body); }
    opt.signal = AbortSignal.timeout(30000);
    let r;
    try { r = await fetch(url, opt); }
    catch (e) { const m = e.name === 'TimeoutError' ? 'Le serveur ne répond pas (30 s)' : 'Requête impossible : ' + e.message; App.toast(m, true); throw new Error(m); }
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
  devState(d) {
    const hb = d.last_heartbeat || {}; const sys = hb.system || {}; const subs = UI.subDevicesOf(hb);
    const peers = (hb.peers || []).filter(p => p.online); const free = (d.config.forwards || []).filter(f => !f.sub);
    const rd = hb.remote_desktop && hb.remote_desktop.id;
    const subLine = sd => {
      const cfg = (d.config.sub_devices || []).find(x => x.name === sd.name);
      const addr = cfg && cfg.expose && d.tailnet_ip ? `<code>${App.esc(d.tailnet_ip)}:${cfg.listen || cfg.port}</code>` : '<span class="faint">non exposé</span>';
      return `<div class="sub-stat"><span class="dot ${sd.reachable ? 'ok' : 'danger'}"></span><span>${App.esc(sd.name)}</span><span class="muted">${App.esc(sd.target)}</span>
        <span class="${sd.reachable ? 'muted' : 'danger-text'}">${sd.reachable ? sd.rtt_ms.toFixed(1) + ' ms' : App.esc(sd.error || 'injoignable')}</span><span>${addr}</span></div>`;
    };
    return `<h3>Connexion</h3>
      <dl class="kv">
        <dt>État</dt><dd>${UI.status(d)}${d.online ? ` <span class="muted small">vu il y a ${App.age(d.age_seconds)}</span>` : ''}</dd>
        <dt>Adresse privée</dt><dd><code>${App.esc(d.tailnet_ip || '—')}</code></dd>
        <dt>Réseau local</dt><dd>${sys.lan_ip ? `<code>${App.esc(sys.lan_ip)}</code> <span class="muted small">${App.esc(sys.lan_cidr || '')}${sys.lan_iface ? ' · ' + App.esc(sys.lan_iface) : ''}</span>` : '<span class="muted">inconnu — agent trop ancien</span>'}</dd>
        <dt>Chemin réseau</dt><dd>${peers.length ? peers.map(p => `${App.esc(p.hostname)} ${p.direct ? UI.badge('ok', 'direct') : UI.badge('warn', 'relais ' + p.relay)} <span class="muted small">${p.latency_ms ? p.latency_ms.toFixed(1) + ' ms' : ''}</span>`).join('<br>') : '<span class="muted">aucun pair en ligne</span>'}</dd>
      </dl>
      <h3>Sous-appareils</h3>
      ${subs.length ? `<div class="sub-stat hdr"><span></span><span>nom</span><span>cible</span><span>latence</span><span>accès</span></div>${subs.map(subLine).join('')}`
        : '<p class="muted small">Aucun sous-appareil détecté. Déclarez-en un dans la configuration à droite.</p>'}
      <h3>Accès à distance</h3>
      <dl class="kv">
        <dt>Bureau à distance</dt><dd>${rd ? `<a class="btn small" href="rustdesk://connection/new/${App.esc(hb.remote_desktop.id)}">Ouvrir dans RustDesk</a> <code>${App.esc(hb.remote_desktop.id)}</code>` : '<span class="muted">RustDesk non détecté</span>'}</dd>
        ${free.length ? `<dt>Forwards libres</dt><dd>${free.map(f => `<code>${App.esc(d.tailnet_ip)}:${f.listen}</code> <span class="muted small">${App.esc(f.name)} → ${App.esc(f.target)}</span>`).join('<br>')}</dd>` : ''}
      </dl>
      <h3>Machine</h3>
      <dl class="kv">
        <dt>Hôte</dt><dd>${App.esc(sys.hostname || d.hostname)} <span class="muted small">${App.esc(sys.os || d.os)}</span></dd>
        ${sys.cpu_percent != null ? `<dt>Charge</dt><dd>CPU ${Math.round(sys.cpu_percent)} % · RAM ${Math.round(sys.mem_percent)} % <span class="muted small">· allumée depuis ${App.age(sys.uptime_seconds)}</span></dd>` : ''}
        <dt>Agent</dt><dd>${App.esc(d.agent_version || '—')}</dd>
      </dl>`;
  },
  eventRow(e) { const ic = e.level === 'error' ? 'alert' : e.level === 'warn' ? 'alert' : 'info'; return `<div class="event ${e.level}"><i data-icon="${ic}"></i><div class="msg">${App.esc(e.message)}</div><div class="when" title="${App.esc(e.created_at)}">${App.rel(e.created_at)}</div></div>`; },
};

/* ---------- Variables ---------- */
// Le catalogue vient du serveur (/api/projects/{id}/variables) : le navigateur ne calcule
// aucune valeur, il substitue. Une seule source de vérité, partagée avec l'exécution des actions.
const Vars = {
  list: [], map: {},
  async load() {
    try { Vars.list = await App.api('GET', `/api/projects/${PROJECT.id}/variables`); } catch { return; }
    Vars.map = {}; Vars.list.forEach(v => Vars.map[v.key] = v.value);
  },
  re: /\{\{\s*([a-zA-Z0-9_.]+)\s*\}\}/g,
  // Texte brut : une clé inconnue reste littérale.
  resolve(text) { return String(text ?? '').replace(Vars.re, (m, k) => k in Vars.map ? Vars.map[k] : m); },
  // HTML échappé ; une clé inconnue est surlignée pour signaler la faute de frappe.
  resolveHTML(text) {
    const src = String(text ?? ''); let out = '', last = 0, m; Vars.re.lastIndex = 0;
    while ((m = Vars.re.exec(src))) {
      out += App.esc(src.slice(last, m.index));
      out += m[1] in Vars.map ? App.esc(Vars.map[m[1]]) : `<span class="var-unknown" title="Variable inconnue">${App.esc(m[0])}</span>`;
      last = m.index + m[0].length;
    }
    return out + App.esc(src.slice(last));
  },
  btn(targetId) { return `<button type="button" class="btn ghost small vp-btn" onclick="Vars.picker(this, '${targetId}')"><i data-icon="braces"></i>Variables</button>`; },
  insert(target, txt) {
    const el = typeof target === 'string' ? document.getElementById(target) : target; if (!el) return;
    const a = el.selectionStart ?? el.value.length, b = el.selectionEnd ?? a;
    el.value = el.value.slice(0, a) + txt + el.value.slice(b);
    el.focus(); el.selectionStart = el.selectionEnd = a + txt.length;
    el.dispatchEvent(new Event('input', { bubbles: true }));
  },
  picker(anchor, target) {
    App.closeMenu(); Vars.close();
    const box = document.createElement('div'); box.className = 'varpick';
    box.innerHTML = `<div class="vp-head"><input class="vp-q" placeholder="Filtrer…" autocomplete="off"></div><div class="vp-list"></div>`;
    document.body.appendChild(box);
    const listEl = box.querySelector('.vp-list');
    const draw = q => {
      q = q.trim().toLowerCase();
      const hit = Vars.list.filter(v => !q || (v.key + ' ' + v.label + ' ' + v.group).toLowerCase().includes(q));
      if (!hit.length) { listEl.innerHTML = '<p class="muted small" style="padding:12px">Aucune variable ne correspond.</p>'; return; }
      let g = null, html = '';
      hit.forEach(v => {
        if (v.group !== g) { g = v.group; html += `<div class="vp-group">${App.esc(g)}</div>`; }
        html += `<button type="button" class="vp-row" data-k="${App.esc(v.key)}"><code>${App.esc(v.key)}</code><span class="vp-lbl">${App.esc(v.label)}</span><span class="vp-val">${App.esc(v.value || '—')}</span></button>`;
      });
      listEl.innerHTML = html;
      listEl.querySelectorAll('.vp-row').forEach(b => b.onclick = () => { Vars.insert(target, '{{ ' + b.dataset.k + ' }}'); Vars.close(); });
    };
    draw('');
    box.querySelector('.vp-q').oninput = e => draw(e.target.value);
    const r = anchor.getBoundingClientRect();
    box.style.left = Math.max(8, Math.min(r.left + window.scrollX, window.innerWidth - box.offsetWidth - 12)) + 'px';
    box.style.top = (r.bottom + 6 + window.scrollY) + 'px';
    if (r.bottom + box.offsetHeight + 12 > window.innerHeight) box.style.top = Math.max(8, r.top + window.scrollY - box.offsetHeight - 6) + 'px';
    Vars._box = box; box.querySelector('.vp-q').focus();
    setTimeout(() => document.addEventListener('click', Vars._onClick = e => { if (!box.contains(e.target) && !anchor.contains(e.target)) Vars.close(); }), 0);
  },
  close() { if (Vars._box) { Vars._box.remove(); Vars._box = null; document.removeEventListener('click', Vars._onClick); } },
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
    Dash.grid = GridStack.init({ column: 12, cellHeight: 64, margin: 6, float: true, disableDrag: true, disableResize: true, animate: false, columnOpts: { breakpoints: [{ w: 768, c: 1 }, { w: 1024, c: 6 }] } }, '#grid');
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
    renderIcons(document.getElementById('grid')); // les en-têtes aussi : fill() ne couvre que les corps
    Dash.fill();
  },
  widgetHTML(w) {
    const tools = App.isAdmin() ? `<div class="widget-tools" ${Dash.editing ? '' : 'hidden'}>
      <button class="icon-btn" onclick="Dash.editWidget('${w.id}')" aria-label="Modifier ce widget" title="Modifier"><i data-icon="settings"></i></button>
      <button class="icon-btn" onclick="Dash.remove_w('${w.id}')" aria-label="Retirer ce widget" title="Retirer"><i data-icon="x"></i></button></div>` : '';
    const title = w.title || Dash.defaultTitle(w);
    return `<div class="widget-head">${title ? `<h3 data-wtitle="${w.id}">${Vars.resolveHTML(title)}</h3>` : ''}${tools}</div><div class="widget-body" data-wid="${w.id}"><div class="skeleton"></div></div>`;
  },
  defaultTitle(w) { return { screens: 'Appareils', screen: 'Appareil', button: '', kpi: '', value: '', automations: 'Automatisations', events: 'Dernières notifications', note: '' }[w.type] ?? w.type; },
  async refresh() {
    try { const [ov, ev] = await Promise.all([App.api('GET', `/api/projects/${PROJECT.id}/overview`), App.api('GET', `/api/projects/${PROJECT.id}/events?limit=8`), Vars.load()]); Dash.data = ov; Dash.data.events = ev; } catch { return; }
    Dash.fill();
  },
  fill() {
    if (!Dash.data) return;
    document.querySelectorAll('.widget-body').forEach(el => { const w = Dash.widgets.find(x => x.id === el.dataset.wid); if (w) { el.innerHTML = Dash.body(w); renderIcons(el); } });
    document.querySelectorAll('[data-wtitle]').forEach(el => { const w = Dash.widgets.find(x => x.id === el.dataset.wtitle); if (w) el.innerHTML = Vars.resolveHTML(w.title || Dash.defaultTitle(w)); });
  },
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
      case 'screen': { const d = D.devices.find(x => x.id == w.device_id); return d ? UI.devState(d) : '<span class="muted">Appareil introuvable ou non approuvé.</span>'; }
      case 'button': { const a = D.actions.find(x => x.id == w.action_id); if (!a) return '<span class="muted">Action introuvable.</span>'; return `<div class="bigbtn"><button class="btn primary" onclick="Dash.run(${a.id}, this)"><i data-icon="play"></i>${App.esc(a.name)}</button><div class="muted small">${a.last_run ? App.rel(a.last_run) + ' · ' + App.esc(a.last_result) : 'jamais exécutée'}</div></div>`; }
      case 'automations': return D.automations.length ? `<table class="table" style="font-size:13px">${D.automations.map(a => `<tr><td style="padding:6px 0"><span class="dot ${a.enabled ? 'ok' : ''}"></span>${App.esc(a.name)}</td><td class="muted" style="padding:6px 8px">${App.esc(a.action_name)}</td><td class="muted small" style="padding:6px 0;text-align:right">${a.enabled ? App.esc(a.next_run) : 'désactivée'}</td></tr>`).join('')}</table>` : '<span class="muted">Aucune automatisation.</span>';
      case 'events': return D.events.length ? `<div class="event-list">${D.events.map(UI.eventRow).join('')}</div>` : '<span class="muted">Aucune notification.</span>';
      case 'note': return `<div>${Vars.resolveHTML(w.text || '').replace(/\n/g, '<br>')}</div>`;
      case 'value': {
        const v = Vars.resolveHTML(w.expr || '');
        return `<div class="kpi"><div class="value">${v || '<span class="muted">—</span>'}${w.unit ? ` <span class="unit">${App.esc(w.unit)}</span>` : ''}</div>${w.label ? `<div class="label">${Vars.resolveHTML(w.label)}</div>` : ''}</div>`;
      }
    }
    return '';
  },
  async run(id, btn) { btn.disabled = true; try { const r = await App.api('POST', `/api/actions/${id}/run`); App.toast(r.result); await Dash.refresh(); } finally { btn.disabled = false; } },
  toggleEdit() {
    Dash.editing = !Dash.editing; document.body.classList.toggle('editing', Dash.editing);
    Dash.grid.enableMove(Dash.editing); Dash.grid.enableResize(Dash.editing);
    document.querySelectorAll('.widget-tools').forEach(b => b.hidden = !Dash.editing);
    if (!Dash.editing) { Dash.widgets = Dash.current && Array.isArray(Dash.current.layout) ? JSON.parse(JSON.stringify(Dash.current.layout)) : []; Dash.renderAll(); }
    Dash.setHeader();
  },
  addWidget() { Dash.editWidget(null); },
  // Même formulaire pour la création et la modification : un widget en place s'ouvre pré-rempli.
  editWidget(id) {
    const D = Dash.data || { devices: [], actions: [] };
    const w = id ? Dash.widgets.find(x => x.id === id) : null;
    const t = w ? w.type : 'kpi';
    const sel = (v, cur) => v === cur ? 'selected' : '';
    const types = { kpi: 'Indicateur', screens: 'État de tous les appareils', screen: "Détail d'un appareil", button: "Bouton d'action", events: 'Dernières notifications', automations: 'Automatisations', value: 'Valeur (variable)', note: 'Note' };
    App.modal(`<h2>${w ? 'Modifier le widget' : 'Ajouter un widget'}</h2>
      <label class="field"><span>Type</span><select id="w-type" onchange="Dash.onType()">${Object.entries(types).map(([k, lbl]) => `<option value="${k}" ${sel(k, t)}>${lbl}</option>`).join('')}</select></label>
      <label class="field" id="w-metric-l"><span>Indicateur</span><select id="w-metric">${[['online', 'Appareils en ligne'], ['subko', 'Sous-appareils injoignables'], ['pending', "En attente d'approbation"], ['relayed', 'Appareils relayés']].map(([k, lbl]) => `<option value="${k}" ${sel(k, w && w.metric)}>${lbl}</option>`).join('')}</select></label>
      <label class="field"><span>Titre (optionnel)</span><input id="w-title" value="${App.esc(w && w.title || '')}"></label>
      <label class="field" id="w-dev-l" hidden><span>Appareil</span><select id="w-dev">${D.devices.map(d => `<option value="${d.id}" ${w && w.device_id === d.id ? 'selected' : ''}>${App.esc(d.name)}</option>`).join('')}</select></label>
      <label class="field" id="w-act-l" hidden><span>Action</span><select id="w-act">${D.actions.map(a => `<option value="${a.id}" ${w && w.action_id === a.id ? 'selected' : ''}>${App.esc(a.name)}</option>`).join('')}</select></label>
      <label class="field" id="w-text-l" hidden><span>Texte <span class="faint">· {{ variables }} acceptées</span></span><textarea id="w-text">${App.esc(w && w.text || '')}</textarea>${Vars.btn('w-text')}</label>
      <label class="field" id="w-expr-l" hidden><span>Valeur <span class="faint">· une ou plusieurs variables</span></span><input id="w-expr" value="${App.esc(w && w.expr || '')}" placeholder="{{ ecran_01.cpu }}">${Vars.btn('w-expr')}</label>
      <div class="row" id="w-vrow" hidden><label class="field"><span>Unité (optionnel)</span><input id="w-unit" value="${App.esc(w && w.unit || '')}" placeholder="%"></label><label class="field"><span>Légende (optionnel)</span><input id="w-label" value="${App.esc(w && w.label || '')}" placeholder="CPU régie"></label></div>
      <div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="Dash.confirmWidget(${w ? `'${w.id}'` : 'null'})">${w ? 'Enregistrer' : 'Ajouter'}</button></div>`);
    Dash.onType();
  },
  onType() { const t = App.val('w-type'); const h = (id, on) => document.getElementById(id).hidden = !on; h('w-metric-l', t === 'kpi'); h('w-dev-l', t === 'screen'); h('w-act-l', t === 'button'); h('w-text-l', t === 'note'); h('w-expr-l', t === 'value'); h('w-vrow', t === 'value'); },
  confirmWidget(id) {
    const t = App.val('w-type');
    const cur = id ? Dash.widgets.find(x => x.id === id) : null;
    const size = { kpi: [3, 2], screens: [8, 4], screen: [5, 5], button: [3, 2], events: [4, 5], automations: [5, 3], note: [4, 2], value: [3, 2] }[t];
    // Objet reconstruit à neuf : un changement de type ne doit pas laisser traîner les champs de l'ancien.
    const w = { id: cur ? cur.id : 'w' + Date.now().toString(36), type: t, title: App.val('w-title') };
    if (cur) Object.assign(w, { x: cur.x, y: cur.y, w: cur.w, h: cur.h });
    else Object.assign(w, { w: size[0], h: size[1] });
    if (t === 'kpi') w.metric = App.val('w-metric');
    if (t === 'screen') w.device_id = +App.val('w-dev');
    if (t === 'button') w.action_id = +App.val('w-act');
    if (t === 'note') w.text = App.val('w-text');
    if (t === 'value') { w.expr = App.val('w-expr'); w.unit = App.val('w-unit'); w.label = App.val('w-label'); if (!w.expr.trim()) { App.toast('Indiquez au moins une variable', true); return; } }
    if ((t === 'screen' && !w.device_id) || (t === 'button' && !w.action_id)) { App.toast('Aucun élément disponible pour ce type', true); return; }
    App.closeModal();
    if (cur) {
      Dash.widgets = Dash.widgets.map(x => x.id === w.id ? w : x);
      Dash.renderAll();
      if (Dash.editing) { Dash.grid.enableMove(true); Dash.grid.enableResize(true); }
      return;
    }
    Dash.widgets.push(w); document.getElementById('dash-empty').hidden = true;
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
  list: [], open: new Set(), q: '', cmds: {}, shown: '', dirty: new Set(),
  init() {
    App.onSearch(q => { Devices.q = q; Devices.render(); });
    const host = document.getElementById('device-list');
    const edited = e => { const c = e.target.closest && e.target.closest('.pane-config'); if (c) Devices.markDirty(+c.closest('.device').dataset.id); };
    host.addEventListener('input', edited); host.addEventListener('change', edited);
    window.addEventListener('beforeunload', e => { if (Devices.dirty.size) { e.preventDefault(); e.returnValue = ''; } });
    Devices.refresh(); setInterval(Devices.refresh, 10000);
  },
  cardEl(id) { return document.querySelector(`.device[data-id="${id}"]`); },
  markDirty(id) { Devices.dirty.add(id); const el = Devices.cardEl(id); if (el) el.classList.add('dirty'); },
  // Sauvegarde puis restaure le champ actif : reconstruire la liste déplace les nœuds conservés.
  snapFocus() {
    const el = document.activeElement;
    if (!el || !el.closest || !el.closest('.pane-config')) return null;
    let s = null, e = null; try { s = el.selectionStart; e = el.selectionEnd; } catch { }
    return { el, s, e };
  },
  restoreFocus(f) {
    if (!f || !f.el.isConnected) return;
    f.el.focus(); if (f.s != null) { try { f.el.setSelectionRange(f.s, f.e); } catch { } }
  },
  async refresh() { Devices.list = await App.api('GET', `/api/projects/${PROJECT.id}/devices`); Devices.render(); [...Devices.open].forEach(Devices.loadCmds); },
  render() {
    const q = Devices.q; const match = d => !q || (d.name + ' ' + d.slug + ' ' + d.hostname + ' ' + d.tailnet_ip).toLowerCase().includes(q);
    const pend = Devices.list.filter(d => d.status === 'pending' && match(d));
    document.getElementById('pending-section').hidden = pend.length === 0;
    document.getElementById('pending-list').innerHTML = pend.map(d => `<div class="card"><h2>${App.esc(d.slug)}</h2>
      <dl class="kv"><dt>Nom d'hôte</dt><dd>${App.esc(d.hostname)}</dd><dt>Système</dt><dd>${App.esc(d.os)} ${App.esc(d.arch)}</dd><dt>Agent</dt><dd>${App.esc(d.agent_version)}</dd><dt>Inscrit</dt><dd>${App.rel(d.enrolled_at)}</dd>
      <dt>Sous-appareils</dt><dd>${(d.config.sub_devices || []).map(sd => App.esc(sd.name + ' ' + sd.ip + ':' + sd.port)).join('<br>') || '—'}</dd></dl>
      ${App.isAdmin() ? `<div class="btnrow"><button class="btn primary" onclick="Devices.act(${d.id},'approve',this)"><i data-icon="check"></i>Approuver</button><button class="btn" onclick="Devices.act(${d.id},'reject',this)">Refuser</button></div>` : ''}</div>`).join('');
    const others = Devices.list.filter(d => d.status !== 'pending' && match(d));
    const host = document.getElementById('device-list');
    if (!others.length) { host.innerHTML = Devices.list.length ? UI.empty('Aucun résultat', 'Aucun appareil ne correspond à la recherche.') : UI.empty('Aucun appareil', 'Installez l’agent avec la clé du projet : l’appareil apparaîtra ici en attente d’approbation.'); Devices.shown = ''; }
    else if (others.map(d => d.id).join(',') === Devices.shown) others.forEach(d => Devices.patch(d));
    else {
      const keep = new Map();
      host.querySelectorAll('.device').forEach(el => { const pane = el.querySelector('.pane-config'); if (pane && Devices.dirty.has(+el.dataset.id)) keep.set(+el.dataset.id, pane); });
      const focus = Devices.snapFocus();
      host.innerHTML = others.map(Devices.card).join('');
      Devices.shown = others.map(d => d.id).join(',');
      keep.forEach((pane, id) => { const fresh = host.querySelector(`.device[data-id="${id}"] .pane-config`); if (fresh) fresh.replaceWith(pane); });
      Devices.restoreFocus(focus);
      others.forEach(d => { if (Devices.open.has(d.id)) Devices.loadCmds(d.id); });
    }
    renderIcons(host); renderIcons(document.getElementById('pending-list'));
  },
  card(d) {
    const open = Devices.open.has(d.id);
    return `<div class="device${open ? ' open' : ''}${Devices.dirty.has(d.id) ? ' dirty' : ''}" data-id="${d.id}"><div class="head" onclick="Devices.toggle(${d.id})">${Devices.headInner(d)}</div>
      ${open ? `<div class="body"><div class="pane pane-state">${Devices.stateInner(d)}</div><div class="pane pane-config">${Devices.configForm(d)}</div><div class="scan-pane" id="scan-${d.id}">${Devices.scanInner(d.id)}</div>${Devices.footInner(d)}</div>` : ''}</div>`;
  },
  headInner(d) {
    const subs = UI.subDevicesOf(d.last_heartbeat || {}); const ko = subs.filter(s => !s.reachable).length;
    return `<span class="dot ${d.status !== 'approved' ? 'warn' : d.online ? (ko ? 'warn' : 'ok') : 'danger'}"></span>
      <span><span class="name">${App.esc(d.name)}</span><span class="slug">${App.esc(d.slug)}</span></span>
      <span class="muted small hide-m">${subs.length ? (subs.length - ko) + '/' + subs.length + ' sous-appareils' : ''}</span>
      <span class="hide-m">${d.tailnet_ip ? `<code>${App.esc(d.tailnet_ip)}</code>` : ''}</span>
      ${UI.status(d)}<i class="chev" data-icon="chevron"></i>`;
  },
  stateInner(d) { return `${UI.devState(d)}<h3>Dernières commandes</h3><div class="cmds" id="cmds-${d.id}">${Devices.cmdsInner(d.id)}</div>`; },
  // Le scan porte un résultat structuré : on montre le dernier en tableau plutôt qu'en ligne de journal.
  scanInner(id) {
    const c = (Devices.cmds[id] || []).find(x => x.kind === 'scan' && x.scan);
    if (!c) { const p = (Devices.cmds[id] || []).find(x => x.kind === 'scan' && x.status === 'queued'); return p ? '<h3>Découverte réseau</h3><p class="muted small">Scan en file, exécuté au prochain heartbeat…</p>' : ''; }
    const sc = c.scan;
    const rows = sc.hosts.map(h => {
      const port = (h.ports || [])[0] || '';
      const quoi = h.known ? `<span class="badge ok">${App.esc(h.known)}</span>`
        : h.vendor ? App.esc(h.vendor)
        : h.random ? '<span class="muted">MAC aléatoire</span>' : '<span class="faint">—</span>';
      const add = App.isAdmin() && !h.known
        ? `<button class="btn ghost small" onclick="Devices.addFromScan(${id},'${App.esc(h.ip)}',${port || 0},'${App.esc((h.hostname || h.vendor || '').replace(/'/g, ''))}')">+ ajouter</button>` : '';
      return `<div class="scan-row"><code>${App.esc(h.ip)}</code>
        <span>${quoi}${h.hostname ? ` <span class="muted small">· ${App.esc(h.hostname)}</span>` : ''}</span>
        <span class="muted small">${(h.ports || []).map(p => p + '/tcp').join(' · ') || '<span class="faint">aucun port ouvert</span>'}</span>
        <span class="right">${add}</span></div>`;
    }).join('');
    return `<h3>Découverte réseau <span class="faint" style="text-transform:none;letter-spacing:0">· ${App.rel(c.created_at)}</span></h3>
      <p class="muted small">${App.esc(c.result)}</p>
      <div class="scan-list">${rows || '<p class="muted small">Aucun hôte trouvé.</p>'}</div>`;
  },
  async scan(id) {
    if (!App.confirm('Lancer un balayage du réseau local de cet appareil ?\n\nL’agent teste chaque adresse de son sous-réseau. Sur un réseau fourni par un diffuseur, prévenez l’équipe technique : un balayage peut déclencher une alerte de sécurité.')) return;
    await App.api('POST', `/api/devices/${id}/command`, { kind: 'scan' });
    App.toast('Scan demandé, exécuté au prochain heartbeat (jusqu’à une minute)');
    setTimeout(() => Devices.loadCmds(id), 1500);
  },
  // Pré-remplit une ligne de sous-appareil à partir d'un hôte découvert.
  addFromScan(id, ip, port, nom) {
    const card = Devices.cardEl(id); if (!card) return;
    const h = card.querySelector('.subs');
    h.insertAdjacentHTML('beforeend', Devices.subRow({ name: nom || '', ip, port: port || '', expose: true }, false, ''));
    renderIcons(h); Devices.markDirty(id);
    const row = h.lastElementChild;
    row.scrollIntoView({ block: 'center', behavior: 'smooth' });
    row.querySelector('.s-name').focus();
    App.toast('Ligne ajoutée : complétez le nom puis enregistrez');
  },
  footInner(d) {
    if (!App.isAdmin()) return '';
    return `<div class="foot">
      <div class="btnrow"><span class="foot-label">Commandes</span><button class="btn" onclick="Devices.cmd(${d.id},'probe')">Sonder</button><button class="btn" onclick="Devices.scan(${d.id})">Scanner le réseau</button><button class="btn" onclick="Devices.cmd(${d.id},'restart')">Redémarrer l'agent</button><button class="btn" onclick="Devices.cmd(${d.id},'update')">Mettre à jour</button></div>
      <div class="btnrow">${d.status === 'approved' ? `<button class="btn" onclick="Devices.rekey(${d.id},this)" title="Quand l'agent affiche « approuvé mais aucune clé réseau reçue »">Nouvelle clé réseau</button><button class="btn danger-text" onclick="Devices.act(${d.id},'revoke',this)">Révoquer</button>` : `<button class="btn primary" onclick="Devices.act(${d.id},'approve',this)">Approuver</button>`}<button class="btn danger-text" onclick="Devices.remove(${d.id},this)">Supprimer</button></div></div>`;
  },
  // Rafraîchit tête et colonne d'état sans toucher au formulaire : sinon le poll écrase la saisie en cours.
  patch(d) {
    const el = document.querySelector(`.device[data-id="${d.id}"]`); if (!el) return;
    el.querySelector('.head').innerHTML = Devices.headInner(d);
    const st = el.querySelector('.pane-state'); if (st) st.innerHTML = Devices.stateInner(d);
    const sc = el.querySelector('.scan-pane'); if (sc) sc.innerHTML = Devices.scanInner(d.id);
    renderIcons(el);
  },
  rerender(id) {
    const d = Devices.list.find(x => x.id === id); const el = document.querySelector(`.device[data-id="${id}"]`); if (!d || !el) return;
    el.outerHTML = Devices.card(d); renderIcons(document.getElementById('device-list'));
    if (Devices.open.has(id)) Devices.loadCmds(id);
  },
  toggle(id) {
    if (Devices.open.has(id)) {
      if (Devices.dirty.has(id) && !App.confirm('Fermer sans enregistrer ? Les modifications seront perdues.')) return;
      Devices.open.delete(id); Devices.dirty.delete(id);
    } else Devices.open.add(id);
    Devices.rerender(id);
  },
  cmdsInner(id) {
    const cmds = Devices.cmds[id];
    if (!cmds) return '<div class="skeleton"></div>';
    if (!cmds.length) return '<p class="muted small">Aucune commande envoyée.</p>';
    return cmds.map(c => `<div>${UI.badge(c.status === 'done' ? 'ok' : c.status === 'failed' ? 'danger' : '', c.status)}<span>${App.esc(c.kind)}</span><span class="muted">${App.rel(c.created_at)}</span>${c.result ? `<span class="muted">· ${App.esc(c.result)}</span>` : ''}</div>`).join('');
  },
  async loadCmds(id) {
    Devices.cmds[id] = await App.api('GET', `/api/devices/${id}/commands`);
    const el = document.getElementById('cmds-' + id); if (el) el.innerHTML = Devices.cmdsInner(id);
    const sc = document.getElementById('scan-' + id); if (sc) sc.innerHTML = Devices.scanInner(id);
  },
  configForm(d) {
    const c = d.config; const ro = !App.isAdmin(); const dis = ro ? 'disabled' : '';
    return `<h3>Configuration à distance <span class="faint" style="text-transform:none;letter-spacing:0">· version ${d.config_version}</span></h3>
      <div class="row"><label class="field"><span>Nom affiché</span><input class="c-name" value="${App.esc(c.name)}" ${dis}></label><label class="field"><span>Heartbeat (s)</span><input class="c-hb" type="number" value="${c.heartbeat_seconds || 15}" ${dis}></label></div>
      <h3>Sous-appareils <span class="faint" style="text-transform:none;letter-spacing:0">· équipements du réseau local, surveillés par l'agent</span></h3>
      <div class="fwd-row sub hdr"><span>nom</span><span>IP</span><span>port</span><span title="Ouvre un accès à cet équipement depuis le réseau privé">accès à distance</span><span>port d'écoute</span><span></span></div>
      <div class="subs">${(c.sub_devices || []).map(sd => Devices.subRow(sd, ro, d.tailnet_ip)).join('')}</div>
      ${ro ? '' : `<button class="btn ghost small" onclick="Devices.addSub(${d.id})"><i data-icon="plus"></i>Sous-appareil</button>`}
      <details class="details" ${(c.forwards || []).some(f => !f.sub) ? 'open' : ''}><summary>Forwards libres (avancé)</summary>
        <p class="muted small" style="margin-bottom:8px">Relais vers une cible qui n'est pas un sous-appareil, ou en UDP.</p>
        <div class="fwd-row hdr"><span>nom</span><span>proto</span><span>port</span><span>cible</span><span></span></div>
        <div class="fwds">${(c.forwards || []).filter(f => !f.sub).map(f => Devices.fwdRow(f, ro)).join('')}</div>
        ${ro ? '' : `<button class="btn ghost small" onclick="Devices.addFwd(${d.id})"><i data-icon="plus"></i>Forward</button>`}
      </details>
      <h3>Mises à jour</h3>
      <div class="row"><label class="check"><input type="checkbox" class="c-upd" ${c.update && c.update.enabled ? 'checked' : ''} ${dis}> Mise à jour automatique</label><label class="field"><span>Vérification (heures)</span><input class="c-updh" type="number" value="${c.update && c.update.check_hours || 1}" ${dis}></label></div>
      ${ro ? '' : `<div class="btnrow save"><button class="btn primary" onclick="Devices.saveConfig(${d.id})">Enregistrer et appliquer</button>
        <span class="muted small save-hint">appliqué au prochain heartbeat</span><span class="small dirty-flag">modifications non enregistrées</span></div>`}`;
  },
  subRow(sd, ro, ip) {
    const dis = ro ? 'disabled' : '';
    const title = sd.expose && ip ? `title="Joignable sur ${ip}:${sd.listen || sd.port}"` : '';
    return `<div class="fwd-row sub"><input class="s-name" value="${App.esc(sd.name)}" placeholder="Processeur LED" ${dis}><input class="s-ip" value="${App.esc(sd.ip)}" placeholder="192.168.0.10" ${dis}><input class="s-port" type="number" value="${sd.port || ''}" placeholder="37564" ${dis}>
      <label class="check center" ${title}><input type="checkbox" class="s-expose" ${sd.expose ? 'checked' : ''} ${dis} onchange="Devices.toggleExpose(this)"></label>
      <input class="s-listen" type="number" value="${sd.expose && sd.listen && sd.listen !== sd.port ? sd.listen : ''}" placeholder="${sd.port || 'idem'}" ${sd.expose ? '' : 'disabled'} ${dis}>
      ${ro ? '<span></span>' : '<button class="icon-btn" onclick="this.parentNode.remove()" aria-label="Retirer"><i data-icon="x"></i></button>'}</div>`;
  },
  toggleExpose(cb) { const row = cb.closest('.fwd-row'); const l = row.querySelector('.s-listen'); l.disabled = !cb.checked; l.placeholder = row.querySelector('.s-port').value || 'idem'; if (!cb.checked) l.value = ''; },
  fwdRow(f, ro) { const dis = ro ? 'disabled' : ''; return `<div class="fwd-row"><input class="f-name" value="${App.esc(f.name)}" ${dis}><select class="f-proto" ${dis}><option ${f.proto === 'tcp' ? 'selected' : ''}>tcp</option><option ${f.proto === 'udp' ? 'selected' : ''}>udp</option></select><input class="f-listen" type="number" value="${f.listen || ''}" ${dis}><input class="f-target" value="${App.esc(f.target)}" placeholder="192.168.0.10:37564" ${dis}>${ro ? '<span></span>' : '<button class="icon-btn" onclick="this.parentNode.remove()" aria-label="Retirer"><i data-icon="x"></i></button>'}</div>`; },
  addSub(id) { const h = Devices.cardEl(id).querySelector('.subs'); h.insertAdjacentHTML('beforeend', Devices.subRow({ name: '', ip: '', port: '', expose: true }, false, '')); renderIcons(h); Devices.markDirty(id); },
  addFwd(id) { const h = Devices.cardEl(id).querySelector('.fwds'); h.insertAdjacentHTML('beforeend', Devices.fwdRow({ name: '', proto: 'tcp', listen: '', target: '' }, false)); renderIcons(h); Devices.markDirty(id); },
  async saveConfig(id) {
    const card = Devices.cardEl(id); if (!card) return;
    const v = sel => { const el = card.querySelector(sel); return el ? el.value : ''; };
    const sub_devices = [...card.querySelectorAll('.subs .fwd-row.sub:not(.hdr)')].map(r => ({
      name: r.querySelector('.s-name').value.trim(), ip: r.querySelector('.s-ip').value.trim(), port: +r.querySelector('.s-port').value,
      expose: r.querySelector('.s-expose').checked, listen: +r.querySelector('.s-listen').value || 0,
    })).filter(sd => sd.ip);
    const forwards = [...card.querySelectorAll('.fwds .fwd-row:not(.hdr)')].map(r => ({ name: r.querySelector('.f-name').value.trim(), proto: r.querySelector('.f-proto').value, listen: +r.querySelector('.f-listen').value, target: r.querySelector('.f-target').value.trim() }));
    await App.api('PUT', `/api/devices/${id}/config`, { name: v('.c-name'), heartbeat_seconds: +v('.c-hb'), sub_devices, forwards, update: { enabled: card.querySelector('.c-upd').checked, check_hours: +v('.c-updh') } });
    Devices.dirty.delete(id);
    App.toast('Configuration enregistrée, appliquée au prochain heartbeat'); await Devices.refresh(); Devices.rerender(id);
  },
  async act(id, what, btn) {
    if (what === 'revoke' && !App.confirm('Révoquer cet appareil ? Il sera retiré du réseau privé et devra être ré-approuvé.')) return;
    await Devices.busy(btn, async () => {
      const r = await App.api('POST', `/api/devices/${id}/${what}`);
      App.toast({ approve: 'Appareil approuvé : il rejoint le réseau dans quelques secondes', reject: 'Appareil refusé', revoke: 'Appareil révoqué' }[what]);
      if (r && r.warning) App.toast(r.warning, true);
      Devices.dirty.delete(id); await Devices.refresh();
    });
  },
  // Émet une nouvelle clé réseau sans révoquer : cas de l'agent approuvé qui a perdu son état local.
  async rekey(id, btn) {
    if (!App.confirm('Émettre une nouvelle clé réseau pour cet appareil ?\n\nÀ utiliser quand l’agent affiche « approuvé mais aucune clé réseau reçue ». L’appareil reste approuvé et n’est pas retiré du réseau.')) return;
    await Devices.busy(btn, async () => {
      await App.api('POST', `/api/devices/${id}/rekey`);
      App.toast('Nouvelle clé émise : l’agent la récupérera à sa prochaine tentative');
      await Devices.refresh();
    });
  },
  async busy(btn, fn) {
    const card = btn && btn.closest('.device'); const all = card ? [...card.querySelectorAll('.foot button')] : (btn ? [btn] : []);
    const was = all.map(b => b.disabled); all.forEach(b => b.disabled = true);
    if (btn) { btn.dataset.label = btn.textContent; btn.textContent = 'En cours…'; }
    try { await fn(); } catch { /* le message a déjà été affiché */ }
    finally {
      all.forEach((b, i) => { if (b.isConnected) b.disabled = was[i]; });
      if (btn && btn.isConnected && btn.dataset.label) { btn.textContent = btn.dataset.label; delete btn.dataset.label; }
    }
  },
  async cmd(id, kind) { await App.api('POST', `/api/devices/${id}/command`, { kind }); App.toast('Commande envoyée, exécutée au prochain heartbeat'); setTimeout(() => Devices.loadCmds(id), 1500); },
  async remove(id, btn) {
    if (!App.confirm('Supprimer définitivement cet appareil du projet ?')) return;
    await Devices.busy(btn, async () => {
      const r = await App.api('DELETE', `/api/devices/${id}`);
      if (r && r.warning) App.toast(r.warning, true);
      Devices.open.delete(id); Devices.dirty.delete(id); Devices.shown = ''; await Devices.refresh();
    });
  },
};

/* ---------- Actions ---------- */
const Actions = {
  list: [], devices: [], q: '',
  async init() { if (App.isAdmin()) App.setPrimary('Nouvelle action', () => Actions.edit()); App.onSearch(q => { Actions.q = q; Actions.render(); }); await Promise.all([Actions.load(), Vars.load()]); },
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
      <label class="field"><span>URL <span class="faint">· {{ variables }} acceptées</span></span><input id="a-url" value="${App.esc(a.url || '')}" placeholder="http://{{ ecran_01.subs.processeur.ip }}/api/…">${Vars.btn('a-url')}</label>
      <label class="field"><span>En-têtes (JSON)</span><textarea id="a-headers">${App.esc(JSON.stringify(a.headers || {}, null, 1))}</textarea>${Vars.btn('a-headers')}</label>
      <label class="field"><span>Corps</span><textarea id="a-body">${App.esc(a.body || '')}</textarea>${Vars.btn('a-body')}</label>
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
