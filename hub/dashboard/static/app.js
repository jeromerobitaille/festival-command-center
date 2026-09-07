/* Festival Command Center — portail. Vanilla JS, une page = un objet. */
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
  toast(msg, err) {
    const t = document.getElementById('toast'); t.textContent = msg; t.className = 'toast' + (err ? ' err' : ''); t.hidden = false;
    clearTimeout(App._t); App._t = setTimeout(() => t.hidden = true, err ? 6000 : 3000);
  },
  modal(html) {
    const m = document.getElementById('modal'); document.getElementById('modal-body').innerHTML = html; m.hidden = false;
    m.onclick = e => { if (e.target === m) App.closeModal(); };
    const first = m.querySelector('input,select,textarea'); if (first) first.focus();
  },
  closeModal() { document.getElementById('modal').hidden = true; },
  esc(s) { return String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); },
  copy(text) { navigator.clipboard.writeText(text).then(() => App.toast('Copié')); },
  age(sec) { if (sec == null) return '—'; if (sec < 60) return sec + ' s'; if (sec < 3600) return Math.floor(sec / 60) + ' min'; if (sec < 86400) return Math.floor(sec / 3600) + ' h'; return Math.floor(sec / 86400) + ' j'; },
  fmtDate(s) { if (!s) return '—'; const d = new Date(s); return isNaN(d) ? s : d.toLocaleString('fr-CA', { dateStyle: 'short', timeStyle: 'short' }); },
  val(id) { return document.getElementById(id).value; },
  confirm(msg) { return window.confirm(msg); },
  changePassword() {
    App.modal(`<h2>Changer mon mot de passe</h2>
      <label>Mot de passe actuel <input id="pw-cur" type="password" autocomplete="current-password"></label>
      <label>Nouveau (8 caractères min.) <input id="pw-new" type="password" autocomplete="new-password"></label>
      <div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="App.savePassword()">Enregistrer</button></div>`);
  },
  async savePassword() { await App.api('POST', '/api/me/password', { current: App.val('pw-cur'), password: App.val('pw-new') }); App.closeModal(); App.toast('Mot de passe changé'); },
  newProject() {
    App.modal(`<h2>Nouveau projet</h2><label>Nom <input id="np-name" placeholder="Festival 2027"></label>
      <label>Fuseau horaire <input id="np-tz" value="America/Toronto"></label>
      <div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="App.createProject()">Créer</button></div>`);
  },
  async createProject() { const p = await App.api('POST', '/api/projects', { name: App.val('np-name'), timezone: App.val('np-tz') }); location.href = '/p/' + p.slug + '/settings'; },
  isAdmin() { return window.ME && window.ME.role === 'admin'; },
};

/* ---------- rendu partagé d'un écran ---------- */
function subDevicesOf(hb) { return hb.sub_devices || (hb.processor ? [Object.assign({ name: 'Processeur' }, hb.processor)] : []); }
function deviceTile(d) {
  const hb = d.last_heartbeat || {};
  const subs = subDevicesOf(hb);
  const procTxt = !d.online ? '' : subs.length ? subs.map(sd => `<span class="dot ${sd.reachable ? 'ok' : 'ko'}"></span>${App.esc(sd.name)}${sd.reachable ? ' ' + sd.rtt_ms.toFixed(1) + ' ms' : ' injoignable'}`).join('<br>') : '<span class="muted">aucun sous-appareil</span>';
  const relayed = (hb.peers || []).some(p => p.online && !p.direct);
  return `<div class="tile ${d.online ? '' : 'off'}"><b><span class="dot ${d.online ? 'ok' : 'ko'}"></span>${App.esc(d.name)}</b>
    <div>${d.online ? 'en ligne' : 'hors ligne depuis ' + App.age(d.age_seconds)}${relayed ? ' <span class="pill warn">relais</span>' : ''}</div>
    <div>${procTxt}</div></div>`;
}

function deviceDetail(d) {
  const hb = d.last_heartbeat || {};
  const sys = hb.system || {}; const subs = subDevicesOf(hb); const peers = (hb.peers || []).filter(p => p.online);
  const fwds = (d.config.forwards || []);
  const fwdAddr = d.tailnet_ip && fwds.length ? fwds.map(f => `<code>${App.esc(d.tailnet_ip)}:${f.listen}</code> <span class="muted">${App.esc(f.name)} → ${App.esc(f.target)}</span>`).join('<br>') : '<span class="muted">aucun forward</span>';
  return `<dl class="kv">
    <dt>État</dt><dd>${d.online ? '<span class="ok-text">en ligne</span>, vu il y a ' + App.age(d.age_seconds) : '<span class="err">hors ligne</span> depuis ' + App.age(d.age_seconds)}</dd>
    <dt>IP réseau privé</dt><dd><code>${App.esc(d.tailnet_ip || '—')}</code></dd>
    <dt>Adresses des forwards</dt><dd>${fwdAddr}</dd>
    <dt>Sous-appareils</dt><dd>${subs.length ? subs.map(sd => sd.reachable ? `<span class="dot ok"></span>${App.esc(sd.name)} · ${App.esc(sd.target)} · ${sd.rtt_ms.toFixed(1)} ms` : `<span class="dot ko"></span>${App.esc(sd.name)} · ${App.esc(sd.target)} · ${App.esc(sd.error || 'injoignable')}`).join('<br>') : '—'}</dd>
    <dt>Chemin réseau</dt><dd>${peers.length ? peers.map(p => `${App.esc(p.hostname)} ${p.direct ? '<span class="pill ok">direct</span>' : '<span class="pill warn">relais ' + App.esc(p.relay) + '</span>'} ${p.latency_ms ? p.latency_ms.toFixed(1) + ' ms' : ''}`).join('<br>') : '<span class="muted">aucun pair en ligne</span>'}</dd>
    <dt>Laptop</dt><dd>${App.esc(sys.hostname || d.hostname)} · ${App.esc(sys.os || d.os)} ${sys.cpu_percent != null ? `· CPU ${Math.round(sys.cpu_percent)} % · RAM ${Math.round(sys.mem_percent)} % · allumé depuis ${App.age(sys.uptime_seconds)}` : ''}</dd>
    <dt>Agent</dt><dd>${App.esc(d.agent_version || '—')}</dd>
    <dt>RustDesk</dt><dd>${hb.remote_desktop && hb.remote_desktop.id ? `<a class="btn small primary" href="rustdesk://connection/new/${App.esc(hb.remote_desktop.id)}">Bureau à distance</a> <code>${App.esc(hb.remote_desktop.id)}</code>` : '<span class="muted">non détecté</span>'}</dd>
  </dl>`;
}

/* ---------- Tableau de bord ---------- */
const Dash = {
  grid: null, editing: false, widgets: [], data: null, timer: null,
  init() {
    Dash.widgets = Array.isArray(PROJECT.dashboard) ? PROJECT.dashboard : [];
    GridStack.renderCB = (el, w) => { el.innerHTML = w.content || ''; }; // Gridstack ≥ 11 échappe le HTML par défaut
    Dash.grid = GridStack.init({ column: 12, cellHeight: 70, margin: 6, float: true, disableDrag: true, disableResize: true, animate: false }, '#grid');
    Dash.grid.on('change', (e, items) => { items.forEach(it => { const w = Dash.widgets.find(x => x.id === it.el.dataset.id); if (w) Object.assign(w, { x: it.x, y: it.y, w: it.w, h: it.h }); }); });
    Dash.renderAll(); Dash.refresh(); Dash.timer = setInterval(Dash.refresh, 10000);
  },
  renderAll() {
    Dash.grid.removeAll();
    document.getElementById('dash-empty').hidden = Dash.widgets.length > 0;
    Dash.widgets.forEach(w => Dash.grid.addWidget({ x: w.x, y: w.y, w: w.w || 4, h: w.h || 3, id: w.id, content: Dash.widgetHTML(w) }));
    Dash.grid.engine.nodes.forEach(n => { n.el.dataset.id = n.id; });
    Dash.fill();
  },
  widgetHTML(w) {
    const rm = App.isAdmin() ? `<button class="btn small danger rm" onclick="Dash.remove('${w.id}')" ${Dash.editing ? '' : 'hidden'}>Retirer</button>` : '';
    return `<div class="widget-head"><h3>${App.esc(w.title || Dash.defaultTitle(w))}</h3>${rm}</div><div class="widget-body" data-wid="${w.id}"><span class="muted">…</span></div>`;
  },
  defaultTitle(w) { return { screens: 'Appareils', screen: 'Appareil', button: 'Action', automations: 'Automatisations', note: 'Note' }[w.type] || w.type; },
  async refresh() {
    try { Dash.data = await App.api('GET', `/api/projects/${PROJECT.id}/overview`); } catch { return; }
    const b = document.getElementById('pending-banner');
    if (Dash.data.pending) { b.hidden = false; b.innerHTML = `<a href="/p/${PROJECT.slug}/devices">${Dash.data.pending} appareil(s) en attente d'approbation</a>`; } else b.hidden = true;
    Dash.fill();
  },
  fill() {
    if (!Dash.data) return;
    document.querySelectorAll('.widget-body').forEach(el => {
      const w = Dash.widgets.find(x => x.id === el.dataset.wid); if (!w) return;
      el.innerHTML = Dash.body(w);
    });
  },
  body(w) {
    const D = Dash.data;
    switch (w.type) {
      case 'screens': {
        if (!D.devices.length) return '<span class="muted">Aucun appareil approuvé.</span>';
        const on = D.devices.filter(d => d.online).length;
        return `<div class="muted small" style="margin-bottom:6px">${on}/${D.devices.length} en ligne</div><div class="tiles">${D.devices.map(deviceTile).join('')}</div>`;
      }
      case 'screen': { const d = D.devices.find(x => x.id == w.device_id); return d ? deviceDetail(d) : '<span class="muted">Appareil introuvable ou non approuvé.</span>'; }
      case 'button': {
        const a = D.actions.find(x => x.id == w.action_id); if (!a) return '<span class="muted">Action introuvable.</span>';
        return `<div class="bigbtn"><button class="btn primary" onclick="Dash.run(${a.id}, this)">${App.esc(a.name)}</button><div class="muted small" id="res-${w.id}">${a.last_run ? App.fmtDate(a.last_run) + ' · ' + App.esc(a.last_result) : 'jamais exécutée'}</div></div>`;
      }
      case 'automations': {
        if (!D.automations.length) return '<span class="muted">Aucune automatisation.</span>';
        return `<table class="table small">${D.automations.map(a => `<tr><td><span class="dot ${a.enabled ? 'ok' : ''}"></span>${App.esc(a.name)}</td><td class="muted">${App.esc(a.action_name)}</td><td class="muted">${a.enabled ? 'prochaine : ' + App.esc(a.next_run) : 'désactivée'}</td></tr>`).join('')}</table>`;
      }
      case 'note': return `<div>${App.esc(w.text || '').replace(/\n/g, '<br>')}</div>`;
    }
    return '';
  },
  async run(actionId, btn) {
    btn.disabled = true; try { const r = await App.api('POST', `/api/actions/${actionId}/run`); App.toast(r.result); await Dash.refresh(); } finally { btn.disabled = false; }
  },
  toggleEdit() {
    Dash.editing = !Dash.editing;
    document.body.classList.toggle('editing', Dash.editing);
    Dash.grid.enableMove(Dash.editing); Dash.grid.enableResize(Dash.editing);
    document.getElementById('btn-add').hidden = !Dash.editing; document.getElementById('btn-save').hidden = !Dash.editing;
    document.getElementById('btn-edit').textContent = Dash.editing ? 'Annuler' : 'Modifier';
    document.querySelectorAll('.widget-head .rm').forEach(b => b.hidden = !Dash.editing);
    if (!Dash.editing) { Dash.widgets = Array.isArray(PROJECT.dashboard) ? JSON.parse(JSON.stringify(PROJECT.dashboard)) : []; Dash.renderAll(); }
  },
  addWidget() {
    const D = Dash.data || { devices: [], actions: [] };
    App.modal(`<h2>Ajouter un widget</h2>
      <label>Type <select id="w-type" onchange="Dash.onType()">
        <option value="screens">Statut de tous les appareils</option><option value="screen">Détail d'un appareil</option>
        <option value="button">Bouton d'action</option><option value="automations">Liste des automatisations</option><option value="note">Note</option></select></label>
      <label>Titre (optionnel) <input id="w-title"></label>
      <label id="w-dev-l" hidden>Appareil <select id="w-dev">${D.devices.map(d => `<option value="${d.id}">${App.esc(d.name)}</option>`).join('')}</select></label>
      <label id="w-act-l" hidden>Action <select id="w-act">${D.actions.map(a => `<option value="${a.id}">${App.esc(a.name)}</option>`).join('')}</select></label>
      <label id="w-text-l" hidden>Texte <textarea id="w-text"></textarea></label>
      <div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="Dash.confirmAdd()">Ajouter</button></div>`);
  },
  onType() { const t = App.val('w-type'); document.getElementById('w-dev-l').hidden = t !== 'screen'; document.getElementById('w-act-l').hidden = t !== 'button'; document.getElementById('w-text-l').hidden = t !== 'note'; },
  confirmAdd() {
    const t = App.val('w-type');
    const w = { id: 'w' + Date.now().toString(36), type: t, title: App.val('w-title'), w: t === 'button' ? 3 : t === 'screens' ? 8 : 4, h: t === 'button' ? 2 : 4 };
    if (t === 'screen') w.device_id = +App.val('w-dev'); if (t === 'button') w.action_id = +App.val('w-act'); if (t === 'note') w.text = App.val('w-text');
    if ((t === 'screen' && !w.device_id) || (t === 'button' && !w.action_id)) { App.toast('Aucun élément disponible pour ce type', true); return; }
    Dash.widgets.push(w); App.closeModal();
    const el = Dash.grid.addWidget({ w: w.w, h: w.h, id: w.id, content: Dash.widgetHTML(w) }); el.dataset.id = w.id;
    const n = Dash.grid.engine.nodes.find(n => n.id === w.id); if (n) Object.assign(w, { x: n.x, y: n.y });
    document.getElementById('dash-empty').hidden = true; Dash.fill();
  },
  remove(id) { Dash.widgets = Dash.widgets.filter(w => w.id !== id); const el = document.querySelector(`.grid-stack-item[data-id="${id}"]`); if (el) Dash.grid.removeWidget(el); },
  async save() {
    Dash.grid.engine.nodes.forEach(n => { const w = Dash.widgets.find(x => x.id === n.id); if (w) Object.assign(w, { x: n.x, y: n.y, w: n.w, h: n.h }); });
    await App.api('PUT', `/api/projects/${PROJECT.id}/dashboard`, Dash.widgets);
    PROJECT.dashboard = JSON.parse(JSON.stringify(Dash.widgets)); Dash.toggleEdit(); App.toast('Tableau de bord enregistré');
  },
};

/* ---------- Écrans ---------- */
const Devices = {
  list: [], open: new Set(), timer: null,
  init() { Devices.refresh(); Devices.timer = setInterval(Devices.refresh, 10000); },
  async refresh() {
    Devices.list = await App.api('GET', `/api/projects/${PROJECT.id}/devices`);
    document.getElementById('dev-refresh').textContent = 'mis à jour à ' + new Date().toLocaleTimeString('fr-CA');
    const pend = Devices.list.filter(d => d.status === 'pending');
    document.getElementById('pending-section').hidden = pend.length === 0;
    document.getElementById('pending-list').innerHTML = pend.map(d => `<div class="card"><h2>${App.esc(d.slug)}</h2>
      <div class="kv"><dt>Nom d'hôte</dt><dd>${App.esc(d.hostname)}</dd><dt>Système</dt><dd>${App.esc(d.os)} ${App.esc(d.arch)}</dd><dt>Agent</dt><dd>${App.esc(d.agent_version)}</dd><dt>Inscrit</dt><dd>${App.fmtDate(d.enrolled_at)}</dd>
      <dt>Sous-appareils proposés</dt><dd>${(d.config.sub_devices || []).map(sd => App.esc(sd.name + ' ' + sd.ip + ':' + sd.port)).join('<br>') || '—'}</dd></div>
      ${App.isAdmin() ? `<div class="btnrow"><button class="btn primary" onclick="Devices.act(${d.id},'approve')">Approuver</button><button class="btn danger" onclick="Devices.act(${d.id},'reject')">Refuser</button></div>` : ''}</div>`).join('');
    const others = Devices.list.filter(d => d.status !== 'pending');
    const host = document.getElementById('device-list');
    if (!others.length) { host.innerHTML = '<div class="empty">Aucun appareil. Installez l\'agent avec la clé du projet : il apparaîtra ici en attente d\'approbation.</div>'; return; }
    host.innerHTML = others.map(d => Devices.card(d)).join('');
  },
  card(d) {
    const open = Devices.open.has(d.id);
    const status = d.status === 'approved' ? (d.online ? '<span class="dot ok"></span>' : '<span class="dot ko"></span>') : '<span class="dot warn"></span>';
    const st = d.status === 'approved' ? (d.online ? 'en ligne' : 'hors ligne ' + App.age(d.age_seconds)) : d.status === 'revoked' ? 'révoqué' : 'refusé';
    return `<div class="device" data-id="${d.id}"><div class="head" onclick="Devices.toggle(${d.id})">${status}<span class="name">${App.esc(d.name)}</span><span class="muted small">${App.esc(d.slug)}</span>
      <span class="grow"></span><span class="muted small">${st}</span>${d.tailnet_ip ? `<code>${App.esc(d.tailnet_ip)}</code>` : ''}<span class="muted">${open ? '▾' : '▸'}</span></div>
      ${open ? `<div class="body"><div>${deviceDetail(d)}<h3>Dernières commandes</h3><div class="cmds" id="cmds-${d.id}">…</div></div><div>${Devices.configForm(d)}</div></div>` : ''}</div>`;
  },
  toggle(id) { Devices.open.has(id) ? Devices.open.delete(id) : Devices.open.add(id); const d = Devices.list.find(x => x.id === id); const el = document.querySelector(`.device[data-id="${id}"]`); el.outerHTML = Devices.card(d); if (Devices.open.has(id)) Devices.loadCmds(id); },
  async loadCmds(id) {
    const cmds = await App.api('GET', `/api/devices/${id}/commands`); const el = document.getElementById('cmds-' + id); if (!el) return;
    el.innerHTML = cmds.length ? cmds.map(c => `<div><span class="pill ${c.status === 'done' ? 'ok' : c.status === 'failed' ? 'ko' : ''}">${c.status}</span> ${App.esc(c.kind)} · ${App.fmtDate(c.created_at)} ${c.result ? '· ' + App.esc(c.result) : ''}</div>`).join('') : '<span class="muted">aucune</span>';
  },
  configForm(d) {
    const c = d.config; const ro = !App.isAdmin(); const dis = ro ? 'disabled' : '';
    const fwd = (c.forwards || []).map((f, i) => Devices.fwdRow(f, i, ro)).join('');
    const subs = (c.sub_devices || []).map(sd => Devices.subRow(sd, ro)).join('');
    return `<h3>Configuration (appliquée à distance, version ${d.config_version})</h3>
      <div id="cfg-${d.id}">
      <div class="row"><label>Nom affiché <input id="c-name" value="${App.esc(c.name)}" ${dis}></label><label>Heartbeat (s) <input id="c-hb" type="number" value="${c.heartbeat_seconds || 15}" ${dis}></label></div>
      <label>Sous-appareils (équipements sur le réseau local de l'appareil, surveillés par l'agent)</label>
      <div class="fwd-row sub muted small"><span>nom</span><span>IP</span><span>port</span><span></span></div>
      <div id="subs">${subs}</div>
      ${ro ? '' : `<button class="btn small" onclick="Devices.addSub()">Ajouter un sous-appareil</button>`}
      <label>Forwards (écoute sur l'IP privée de l'appareil → cible sur son réseau local)</label>
      <div class="fwd-row muted small"><span>nom</span><span>proto</span><span>port</span><span>cible</span><span></span></div>
      <div id="fwds">${fwd}</div>
      ${ro ? '' : `<button class="btn small" onclick="Devices.addFwd()">Ajouter un forward</button>`}
      <div class="row"><label class="check"><input type="checkbox" id="c-upd" ${c.update && c.update.enabled ? 'checked' : ''} ${dis}> Mise à jour automatique</label><label>Vérification (heures) <input id="c-updh" type="number" value="${c.update && c.update.check_hours || 1}" ${dis}></label></div>
      </div>
      ${ro ? '' : `<div class="btnrow"><button class="btn primary" onclick="Devices.saveConfig(${d.id})">Enregistrer et appliquer</button>
        <button class="btn" onclick="Devices.cmd(${d.id},'probe')" title="L'agent teste lui-même ses sous-appareils et remonte le résultat">Sonder</button>
        <button class="btn" onclick="Devices.cmd(${d.id},'restart')">Redémarrer l'agent</button>
        <button class="btn" onclick="Devices.cmd(${d.id},'update')">Mettre à jour maintenant</button>
        ${d.status === 'approved' ? `<button class="btn danger" onclick="Devices.act(${d.id},'revoke')">Révoquer</button>` : `<button class="btn primary" onclick="Devices.act(${d.id},'approve')">Approuver</button>`}
        <button class="btn danger" onclick="Devices.remove(${d.id})">Supprimer</button></div>`}`;
  },
  subRow(sd, ro) {
    const dis = ro ? 'disabled' : '';
    return `<div class="fwd-row sub"><input class="s-name" value="${App.esc(sd.name)}" placeholder="Processeur LED" ${dis}><input class="s-ip" value="${App.esc(sd.ip)}" placeholder="192.168.0.10" ${dis}><input class="s-port" type="number" value="${sd.port || ''}" placeholder="37564" ${dis}>${ro ? '<span></span>' : '<button class="btn small danger" onclick="this.parentNode.remove()">×</button>'}</div>`;
  },
  addSub() { document.getElementById('subs').insertAdjacentHTML('beforeend', Devices.subRow({ name: '', ip: '', port: '' }, false)); },
  fwdRow(f, i, ro) {
    const dis = ro ? 'disabled' : '';
    return `<div class="fwd-row"><input class="f-name" value="${App.esc(f.name)}" ${dis}><select class="f-proto" ${dis}><option ${f.proto === 'tcp' ? 'selected' : ''}>tcp</option><option ${f.proto === 'udp' ? 'selected' : ''}>udp</option></select>
      <input class="f-listen" type="number" value="${f.listen || ''}" ${dis}><input class="f-target" value="${App.esc(f.target)}" placeholder="192.168.0.10:37564" ${dis}>${ro ? '<span></span>' : '<button class="btn small danger" onclick="this.parentNode.remove()">×</button>'}</div>`;
  },
  addFwd() { document.getElementById('fwds').insertAdjacentHTML('beforeend', Devices.fwdRow({ name: '', proto: 'tcp', listen: '', target: '' }, 0, false)); },
  async saveConfig(id) {
    const forwards = [...document.querySelectorAll('#fwds .fwd-row')].map(r => ({ name: r.querySelector('.f-name').value.trim(), proto: r.querySelector('.f-proto').value, listen: +r.querySelector('.f-listen').value, target: r.querySelector('.f-target').value.trim() }));
    const sub_devices = [...document.querySelectorAll('#subs .fwd-row.sub')].map(r => ({ name: r.querySelector('.s-name').value.trim(), ip: r.querySelector('.s-ip').value.trim(), port: +r.querySelector('.s-port').value })).filter(sd => sd.ip);
    const cfg = { name: App.val('c-name'), heartbeat_seconds: +App.val('c-hb'), sub_devices, forwards, update: { enabled: document.getElementById('c-upd').checked, check_hours: +App.val('c-updh') } };
    await App.api('PUT', `/api/devices/${id}/config`, cfg); App.toast('Configuration enregistrée, appliquée au prochain heartbeat'); Devices.refresh();
  },
  async act(id, what) {
    if (what === 'revoke' && !App.confirm('Révoquer cet appareil ? Il sera retiré du réseau privé et devra être ré-approuvé.')) return;
    await App.api('POST', `/api/devices/${id}/${what}`); App.toast({ approve: 'Appareil approuvé : il rejoint le réseau dans quelques secondes', reject: 'Appareil refusé', revoke: 'Appareil révoqué' }[what]); Devices.refresh();
  },
  async cmd(id, kind) { await App.api('POST', `/api/devices/${id}/command`, { kind }); App.toast('Commande envoyée, exécutée au prochain heartbeat'); setTimeout(() => Devices.loadCmds(id), 1500); },
  async remove(id) { if (!App.confirm('Supprimer définitivement cet appareil du projet ?')) return; await App.api('DELETE', `/api/devices/${id}`); Devices.open.delete(id); Devices.refresh(); },
};

/* ---------- Actions ---------- */
const Actions = {
  list: [], devices: [],
  async init() { [Actions.list, Actions.devices] = await Promise.all([App.api('GET', `/api/projects/${PROJECT.id}/actions`), App.api('GET', `/api/projects/${PROJECT.id}/devices`)]); Actions.render(); },
  render() {
    const host = document.getElementById('action-list');
    if (!Actions.list.length) { host.innerHTML = '<div class="empty">Aucune action définie.</div>'; return; }
    host.innerHTML = `<table class="table"><thead><tr><th>Nom</th><th>Exécutée par</th><th>Requête</th><th>Dernier résultat</th><th></th></tr></thead><tbody>${Actions.list.map(a => {
      const dev = a.device_id ? (Actions.devices.find(d => d.id === a.device_id) || {}).name : '';
      return `<tr><td><b>${App.esc(a.name)}</b></td><td>${a.kind === 'agent' ? 'agent de ' + App.esc(dev || '?') : 'hub'}</td><td><code>${App.esc(a.method)}</code> ${App.esc(a.url)}</td>
        <td class="small muted">${a.last_run ? App.fmtDate(a.last_run) + '<br>' + App.esc(a.last_result) : '—'}</td>
        <td class="right"><button class="btn small primary" onclick="Actions.run(${a.id})">Exécuter</button> ${App.isAdmin() ? `<button class="btn small" onclick="Actions.edit(${a.id})">Modifier</button><button class="btn small danger" onclick="Actions.remove(${a.id})">Supprimer</button>` : ''}</td></tr>`; }).join('')}</tbody></table>`;
  },
  edit(id) {
    const a = Actions.list.find(x => x.id === id) || { kind: 'hub', method: 'GET', headers: {}, timeout_seconds: 10 };
    App.modal(`<h2>${id ? 'Modifier' : 'Nouvelle'} action</h2>
      <label>Nom <input id="a-name" value="${App.esc(a.name || '')}"></label>
      <div class="row"><label>Exécutée par <select id="a-kind" onchange="document.getElementById('a-dev-l').hidden=this.value!=='agent'"><option value="hub" ${a.kind === 'hub' ? 'selected' : ''}>le hub (API externe)</option><option value="agent" ${a.kind === 'agent' ? 'selected' : ''}>l'agent d'un appareil (réseau local)</option></select></label>
      <label id="a-dev-l" ${a.kind === 'agent' ? '' : 'hidden'}>Appareil <select id="a-dev">${Actions.devices.filter(d => d.status === 'approved').map(d => `<option value="${d.id}" ${a.device_id === d.id ? 'selected' : ''}>${App.esc(d.name)}</option>`).join('')}</select></label></div>
      <div class="row"><label>Méthode <select id="a-method">${['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].map(m => `<option ${a.method === m ? 'selected' : ''}>${m}</option>`).join('')}</select></label><label>Délai (s) <input id="a-timeout" type="number" value="${a.timeout_seconds}"></label></div>
      <label>URL <input id="a-url" value="${App.esc(a.url || '')}" placeholder="http://192.168.0.10/api/..."></label>
      <label>En-têtes (JSON) <textarea id="a-headers">${App.esc(JSON.stringify(a.headers || {}, null, 1))}</textarea></label>
      <label>Corps <textarea id="a-body">${App.esc(a.body || '')}</textarea></label>
      <div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="Actions.save(${id || 0})">Enregistrer</button></div>`);
  },
  async save(id) {
    let headers; try { headers = JSON.parse(App.val('a-headers') || '{}'); } catch { App.toast('En-têtes : JSON invalide', true); return; }
    const a = { name: App.val('a-name'), kind: App.val('a-kind'), device_id: App.val('a-kind') === 'agent' ? +App.val('a-dev') || null : null, method: App.val('a-method'), url: App.val('a-url'), headers, body: App.val('a-body'), timeout_seconds: +App.val('a-timeout') };
    if (id) await App.api('PUT', `/api/actions/${id}`, a); else await App.api('POST', `/api/projects/${PROJECT.id}/actions`, a);
    App.closeModal(); Actions.init();
  },
  async run(id) { const r = await App.api('POST', `/api/actions/${id}/run`); App.toast(r.result); Actions.init(); },
  async remove(id) { if (!App.confirm('Supprimer cette action ?')) return; await App.api('DELETE', `/api/actions/${id}`); Actions.init(); },
};

/* ---------- Automatisations ---------- */
const Autos = {
  list: [], actions: [],
  async init() { [Autos.list, Autos.actions] = await Promise.all([App.api('GET', `/api/projects/${PROJECT.id}/automations`), App.api('GET', `/api/projects/${PROJECT.id}/actions`)]); Autos.render(); },
  render() {
    const host = document.getElementById('auto-list');
    if (!Autos.list.length) { host.innerHTML = '<div class="empty">Aucune automatisation.' + (Autos.actions.length ? '' : ' Créez d\'abord une action.') + '</div>'; return; }
    host.innerHTML = `<table class="table"><thead><tr><th>Nom</th><th>Horaire</th><th>Action</th><th>Prochaine</th><th>Dernier résultat</th><th></th></tr></thead><tbody>${Autos.list.map(a =>
      `<tr><td><span class="dot ${a.enabled ? 'ok' : ''}"></span><b>${App.esc(a.name)}</b></td><td>${App.esc(Autos.describe(a.cron))}<br><code>${App.esc(a.cron)}</code></td><td>${App.esc(a.action_name)}</td><td>${a.enabled ? App.esc(a.next_run) : '<span class="muted">désactivée</span>'}</td>
       <td class="small muted">${a.last_run ? App.fmtDate(a.last_run) + '<br>' + App.esc(a.last_result) : '—'}</td>
       <td class="right">${App.isAdmin() ? `<button class="btn small" onclick="Autos.edit(${a.id})">Modifier</button><button class="btn small danger" onclick="Autos.remove(${a.id})">Supprimer</button>` : ''}</td></tr>`).join('')}</tbody></table>`;
  },
  days: ['dim', 'lun', 'mar', 'mer', 'jeu', 'ven', 'sam'],
  describe(cron) {
    const p = cron.split(/\s+/); if (p.length !== 5 || !/^\d+$/.test(p[0]) || !/^\d+$/.test(p[1])) return 'expression cron';
    const t = p[1].padStart(2, '0') + ':' + p[0].padStart(2, '0');
    if (p[4] === '*' && p[2] === '*') return 'tous les jours à ' + t;
    if (p[2] === '*') return p[4].split(',').map(d => Autos.days[+d] ?? d).join(', ') + ' à ' + t;
    return 'le ' + p[2] + ' de chaque mois à ' + t;
  },
  edit(id) {
    const a = Autos.list.find(x => x.id === id) || { enabled: true, cron: '0 22 * * *', action_id: Autos.actions[0] && Autos.actions[0].id };
    const parts = a.cron.split(/\s+/); const simple = parts.length === 5 && /^\d+$/.test(parts[0]) && /^\d+$/.test(parts[1]) && parts[2] === '*';
    const days = simple && parts[4] !== '*' ? parts[4].split(',') : ['0', '1', '2', '3', '4', '5', '6'];
    App.modal(`<h2>${id ? 'Modifier' : 'Nouvelle'} automatisation</h2>
      <label>Nom <input id="u-name" value="${App.esc(a.name || '')}" placeholder="Éteindre les écrans"></label>
      <label>Action <select id="u-action">${Autos.actions.map(x => `<option value="${x.id}" ${a.action_id === x.id ? 'selected' : ''}>${App.esc(x.name)}</option>`).join('')}</select></label>
      <label class="check"><input type="checkbox" id="u-enabled" ${a.enabled ? 'checked' : ''}> Activée</label>
      <div class="row"><label>Heure <input id="u-time" type="time" value="${simple ? parts[1].padStart(2, '0') + ':' + parts[0].padStart(2, '0') : '22:00'}" onchange="Autos.build()"></label></div>
      <label>Jours</label><div class="row" style="flex-wrap:wrap">${Autos.days.map((d, i) => `<label class="check" style="flex:0 0 auto"><input type="checkbox" class="u-day" value="${i}" ${days.includes(String(i)) ? 'checked' : ''} onchange="Autos.build()"> ${d}</label>`).join('')}</div>
      <label>Expression cron (min heure jour mois jour-semaine, modifiable) <input id="u-cron" value="${App.esc(a.cron)}"></label>
      <div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="Autos.save(${id || 0})">Enregistrer</button></div>`);
  },
  build() {
    const [h, m] = App.val('u-time').split(':'); const days = [...document.querySelectorAll('.u-day:checked')].map(c => c.value);
    document.getElementById('u-cron').value = `${+m} ${+h} * * ${days.length === 7 || !days.length ? '*' : days.join(',')}`;
  },
  async save(id) {
    const a = { name: App.val('u-name'), action_id: +App.val('u-action'), enabled: document.getElementById('u-enabled').checked, cron: App.val('u-cron') };
    if (id) await App.api('PUT', `/api/automations/${id}`, a); else await App.api('POST', `/api/projects/${PROJECT.id}/automations`, a);
    App.closeModal(); Autos.init();
  },
  async remove(id) { if (!App.confirm('Supprimer cette automatisation ?')) return; await App.api('DELETE', `/api/automations/${id}`); Autos.init(); },
};

/* ---------- Configuration du projet ---------- */
const Settings = {
  async init() {
    document.getElementById('p-toml').textContent = document.getElementById('p-toml').textContent.replace('{{ORIGIN}}', location.origin);
    const [users, members] = await Promise.all([App.api('GET', '/api/users'), App.api('GET', `/api/projects/${PROJECT.id}/members`)]);
    document.getElementById('members').innerHTML = users.map(u => `<label class="check"><input type="checkbox" class="m-user" value="${u.id}" ${members.includes(u.id) ? 'checked' : ''} ${u.role === 'admin' ? 'disabled checked' : ''}> ${App.esc(u.username)} <span class="pill">${u.role}</span></label>`).join('') || '<span class="muted">Aucun utilisateur.</span>';
  },
  async save() { await App.api('PUT', `/api/projects/${PROJECT.id}`, { name: App.val('p-name'), timezone: App.val('p-tz') }); App.toast('Projet enregistré'); },
  async rotate() { if (!App.confirm('Régénérer la clé ? L\'ancienne cessera d\'accepter de nouveaux écrans.')) return; const r = await App.api('POST', `/api/projects/${PROJECT.id}/rotate-key`); location.reload(); },
  async saveMembers() { const ids = [...document.querySelectorAll('.m-user:checked:not([disabled])')].map(c => +c.value); await App.api('PUT', `/api/projects/${PROJECT.id}/members`, ids); App.toast('Membres enregistrés'); },
  async remove() { if (!App.confirm(`Supprimer le projet « ${PROJECT.name} » et révoquer tous ses appareils ?`)) return; await App.api('DELETE', `/api/projects/${PROJECT.id}`); location.href = '/'; },
};

/* ---------- Utilisateurs ---------- */
const Users = {
  create() {
    App.modal(`<h2>Nouvel utilisateur</h2><label>Nom d'utilisateur <input id="nu-name" autocomplete="off"></label><label>Mot de passe (8 caractères min.) <input id="nu-pw" type="text" value="${Math.random().toString(36).slice(2, 8) + Math.random().toString(36).slice(2, 8)}"></label>
      <label>Rôle <select id="nu-role"><option value="user">user</option><option value="admin">admin</option></select></label>
      <div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="Users.save()">Créer</button></div>`);
  },
  async save() { await App.api('POST', '/api/users', { username: App.val('nu-name'), password: App.val('nu-pw'), role: App.val('nu-role') }); location.reload(); },
  reset(id, name) {
    App.modal(`<h2>Nouveau mot de passe pour ${App.esc(name)}</h2><label>Mot de passe <input id="rp-pw" type="text" value="${Math.random().toString(36).slice(2, 8) + Math.random().toString(36).slice(2, 8)}"></label>
      <div class="modal-foot"><button class="btn" onclick="App.closeModal()">Annuler</button><button class="btn primary" onclick="Users.saveReset(${id})">Enregistrer</button></div>`);
  },
  async saveReset(id) { await App.api('POST', `/api/users/${id}/password`, { password: App.val('rp-pw') }); App.closeModal(); App.toast('Mot de passe remplacé'); },
  async remove(id, name) { if (!App.confirm(`Supprimer l'utilisateur ${name} ?`)) return; await App.api('DELETE', `/api/users/${id}`); location.reload(); },
};

document.addEventListener('DOMContentLoaded', () => {
  const page = document.body.dataset.page;
  ({ dashboard: Dash, devices: Devices, actions: Actions, automations: Autos, settings: Settings }[page] || { init() {} }).init();
});
