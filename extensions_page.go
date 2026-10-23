package main

// sparrow://extensions - install, enable, disable and remove extension zips.
// __EXT_TOKEN__ is replaced with the session token when the page is served.
const extensionsPageHTML = `
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>SPARROW Extensions</title>
<style>
  * { box-sizing: border-box; }
  body { margin: 0; background: #0b0b0d; color: #e8e8e8; font-family: 'Segoe UI', sans-serif; padding: 40px 60px; }
  h1 { font-size: 26px; margin: 0 0 6px; }
  .sub { color: #888; font-size: 13px; margin-bottom: 24px; max-width: 560px; }
  .card { background: #17181a; border: 1px solid #232323; border-radius: 14px; padding: 20px 24px; max-width: 640px; margin-bottom: 16px; box-shadow: 0 6px 20px rgba(0,0,0,0.35); }
  .card-title { font-size: 14px; font-weight: 600; color: #fff; margin-bottom: 6px; }
  .card-desc { font-size: 12px; color: #888; margin-bottom: 14px; }
  .row { display: flex; gap: 10px; align-items: center; }
  input[type=text] { flex: 1; background: #0f1012; border: 1px solid #2c2c2e; color: #e8e8e8; padding: 10px 12px; border-radius: 8px; font-size: 13px; outline: none; }
  input[type=text]:focus { border-color: #4d90fe; }
  .btn { background: #4d90fe; color: #000; border: none; padding: 10px 18px; border-radius: 8px; font-size: 13px; font-weight: 600; cursor: pointer; white-space: nowrap; }
  .btn-secondary { background: #232323; color: #ccc; border: 1px solid #2c2c2e; padding: 10px 18px; border-radius: 8px; font-size: 13px; cursor: pointer; white-space: nowrap; }
  .btn-danger { background: transparent; color: #ff6b6b; border: 1px solid #3a2222; padding: 6px 12px; border-radius: 8px; font-size: 12px; cursor: pointer; }
  .status { font-size: 12px; margin-top: 12px; min-height: 16px; }
  .status.ok { color: #3ddc84; } .status.err { color: #ff6b6b; }
  .ext { display: flex; gap: 14px; align-items: center; }
  .ext-icon { width: 44px; height: 44px; border-radius: 10px; background: #232323; display: flex; align-items: center; justify-content: center; font-size: 22px; flex: none; }
  .ext-body { flex: 1; min-width: 0; }
  .ext-name { font-size: 14px; font-weight: 600; color: #fff; }
  .ext-ver { font-size: 11px; color: #777; margin-left: 6px; font-weight: 400; }
  .ext-desc { font-size: 12px; color: #999; margin-top: 3px; }
  .ext-meta { font-size: 11px; color: #666; margin-top: 4px; }
  .switch { position: relative; width: 42px; height: 24px; flex: none; }
  .switch input { opacity: 0; width: 0; height: 0; }
  .slider { position: absolute; inset: 0; background: #333; border-radius: 24px; cursor: pointer; transition: .2s; }
  .slider:before { content: ""; position: absolute; width: 18px; height: 18px; left: 3px; top: 3px; background: #fff; border-radius: 50%; transition: .2s; }
  .switch input:checked + .slider { background: #4d90fe; }
  .switch input:checked + .slider:before { transform: translateX(18px); }
  .empty { color: #777; font-size: 13px; padding: 10px 0; }
  .note { font-size: 11px; color: #777; margin-top: 10px; }
  body[data-theme="light"] { background: #f5f7fa; color: #202124; }
  body[data-theme="light"] .card { background: #fff; border-color: #e1e4e8; box-shadow: 0 2px 8px rgba(0,0,0,0.06); }
  body[data-theme="light"] .card-title, body[data-theme="light"] .ext-name { color: #202124; }
  body[data-theme="light"] input[type=text] { background: #fff; color: #202124; border-color: #d0d4d9; }
  body[data-theme="light"] .btn-secondary { background: #eef0f3; color: #333; border-color: #d0d4d9; }
  body[data-theme="light"] .ext-icon { background: #eef0f3; }
</style>
</head>
<body>
  <h1>Extensions</h1>
  <div class="sub">Extensions are zip packages. When you turn one on, it is extracted and starts working on matching pages. Turning it off deletes the extracted files.</div>

  <div class="card">
    <div class="card-title">Add an extension</div>
    <div class="card-desc">Paste a link to an extension .zip (for example from your app store), or choose a .zip file from your computer.</div>
    <div class="row">
      <input type="text" id="zipUrl" placeholder="https://.../extension.zip">
      <button class="btn" id="btnUrl">Download</button>
    </div>
    <div class="row" style="margin-top:10px">
      <button class="btn-secondary" id="btnFile">Choose .zip file</button>
      <input type="file" id="fileInput" accept=".zip" style="display:none">
    </div>
    <div class="status" id="status"></div>
    <div class="note">Only install extensions you trust. An enabled extension can read and change the pages it runs on.</div>
  </div>

  <div id="list"></div>

<script>
  var TOKEN = "__EXT_TOKEN__";
  var listEl = document.getElementById('list');
  var statusEl = document.getElementById('status');

  function setStatus(msg, ok) {
    statusEl.textContent = msg || '';
    statusEl.className = 'status ' + (msg ? (ok ? 'ok' : 'err') : '');
  }
  function parse(raw) { try { return JSON.parse(raw); } catch (e) { return { ok: false, error: 'unexpected response' }; } }
  function esc(s) { var d = document.createElement('div'); d.textContent = s == null ? '' : String(s); return d.innerHTML; }

  async function refresh() {
    var res = parse(await window.extList(TOKEN));
    listEl.innerHTML = '';
    if (!res.ok) { listEl.innerHTML = '<div class="empty">' + esc(res.error) + '</div>'; return; }
    var items = res.data || [];
    if (!items.length) { listEl.innerHTML = '<div class="empty">No extensions yet. Add a .zip above.</div>'; return; }
    items.forEach(function (e) {
      var card = document.createElement('div');
      card.className = 'card';
      card.innerHTML =
        '<div class="ext">' +
          '<div class="ext-icon">' + esc(e.icon || '\uD83E\uDDE9') + '</div>' +
          '<div class="ext-body">' +
            '<div class="ext-name">' + esc(e.name) + '<span class="ext-ver">v' + esc(e.version) + '</span></div>' +
            '<div class="ext-desc">' + esc(e.description) + '</div>' +
            '<div class="ext-meta">' + (e.author ? esc(e.author) + ' \u00B7 ' : '') + esc(e.sizeKB) + ' KB \u00B7 ' + (e.enabled ? 'On' : 'Off') + '</div>' +
          '</div>' +
          '<label class="switch"><input type="checkbox" ' + (e.enabled ? 'checked' : '') + '><span class="slider"></span></label>' +
          '<button class="btn-danger">Remove</button>' +
        '</div>';
      card.querySelector('input').addEventListener('change', async function (ev) {
        var r = parse(await window.extSetEnabled(TOKEN, e.id, ev.target.checked));
        if (!r.ok) { setStatus(r.error, false); } else { setStatus(e.name + (ev.target.checked ? ' is on. Reload open pages to use it.' : ' is off.'), true); }
        refresh();
      });
      card.querySelector('.btn-danger').addEventListener('click', async function () {
        if (!confirm('Remove "' + e.name + '"?')) return;
        var r = parse(await window.extRemove(TOKEN, e.id));
        setStatus(r.ok ? e.name + ' removed.' : r.error, r.ok);
        refresh();
      });
      listEl.appendChild(card);
    });
  }

  document.getElementById('btnUrl').addEventListener('click', async function () {
    var url = document.getElementById('zipUrl').value.trim();
    if (!url) { setStatus('Paste a .zip link first.', false); return; }
    setStatus('Downloading...', true);
    var r = parse(await window.extInstallURL(TOKEN, url));
    setStatus(r.ok ? r.data.name + ' added. Turn it on below.' : r.error, r.ok);
    if (r.ok) { document.getElementById('zipUrl').value = ''; }
    refresh();
  });

  document.getElementById('btnFile').addEventListener('click', function () { document.getElementById('fileInput').click(); });
  document.getElementById('fileInput').addEventListener('change', function (ev) {
    var file = ev.target.files && ev.target.files[0];
    if (!file) return;
    var reader = new FileReader();
    reader.onload = async function () {
      var b64 = String(reader.result).split(',')[1] || '';
      var r = parse(await window.extInstallB64(TOKEN, b64));
      setStatus(r.ok ? r.data.name + ' added. Turn it on below.' : r.error, r.ok);
      ev.target.value = '';
      refresh();
    };
    reader.readAsDataURL(file);
  });

  refresh();
</script>
</body>
</html>
`
