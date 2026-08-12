package main

const settingsPartialHTML = `
<div class="settings-group">
  <h3>General</h3>
  <div class="settings-row">
    <div><div class="settings-label">Theme</div><div class="settings-desc">Switch between Light and Dark mode</div></div>
    <select id="settingThemeSelect"><option value="dark">Dark</option><option value="light">Light</option></select>
  </div>
  <div class="settings-row" style="flex-direction:column; align-items:stretch;">
    <div class="settings-label">Color Theme</div>
    <div class="theme-swatch-row" id="themeSwatchRow">
      <div class="theme-swatch" data-theme-id="violet-default" style="background:linear-gradient(135deg,#9B6DFF,#4B0082);"></div>
      <div class="theme-swatch" data-theme-id="violet-green" style="background:linear-gradient(135deg,#9B6DFF,#2ecc71);"></div>
      <div class="theme-swatch" data-theme-id="violet-blue" style="background:linear-gradient(135deg,#9B6DFF,#3498db);"></div>
      <div class="theme-swatch" data-theme-id="violet-red" style="background:linear-gradient(135deg,#9B6DFF,#e74c3c);"></div>
      <div class="theme-swatch" data-theme-id="violet-gold" style="background:linear-gradient(135deg,#9B6DFF,#f1c40f);"></div>
      <div class="theme-swatch" data-theme-id="violet-teal" style="background:linear-gradient(135deg,#9B6DFF,#1abc9c);"></div>
      <div class="theme-swatch" data-theme-id="violet-pink" style="background:linear-gradient(135deg,#9B6DFF,#ff6fb1);"></div>
      <div class="theme-swatch" data-theme-id="violet-orange" style="background:linear-gradient(135deg,#9B6DFF,#e67e22);"></div>
      <div class="theme-swatch" data-theme-id="violet-mono" style="background:linear-gradient(135deg,#B48CFF,#2c2c2c);"></div>
      <div class="theme-swatch" data-theme-id="violet-slate" style="background:linear-gradient(135deg,#9B6DFF,#607d8b);"></div>
    </div>
  </div>
  <div class="settings-row">
    <div><div class="settings-label">Minimize to Tray</div><div class="settings-desc">Keep Sparrow running in the tray when closed</div></div>
    <label class="switch"><input type="checkbox" id="settingMinimizeTray"><span class="slider"></span></label>
  </div>
  <div class="settings-row">
    <div><div class="settings-label">Autostart on Boot</div><div class="settings-desc">Launch Sparrow automatically at login</div></div>
    <label class="switch"><input type="checkbox" id="settingAutostart"><span class="slider"></span></label>
  </div>
  <div class="settings-row">
    <div><div class="settings-label">Share with Sparrow (right-click menu)</div><div class="settings-desc">Add to Windows right-click menu for files/folders</div></div>
    <label class="switch"><input type="checkbox" id="settingShareTarget"><span class="slider"></span></label>
  </div>
</div>

<div class="settings-group">
  <h3>Receive</h3>
  <div class="settings-row">
    <div><div class="settings-label">Save Folder</div><div class="settings-desc" id="saveFolderPath">Loading...</div></div>
    <button class="pill-btn" id="btnChangeFolder">Change</button>
  </div>
  <div class="settings-row">
    <div><div class="settings-label">Require PIN on Receive</div><div class="settings-desc">Sender must match your PIN before sending</div></div>
    <label class="switch"><input type="checkbox" id="settingRequirePin"><span class="slider"></span></label>
  </div>
  <div class="settings-row">
    <div><div class="settings-label">Set 4-digit PIN</div><div class="settings-desc">Used for receive protection and app unlock</div></div>
    <input type="text" maxlength="4" placeholder="••••" id="settingPinInput" style="width:80px; text-align:center;">
  </div>
</div>

<div class="settings-group">
  <h3>Privacy &amp; Identity</h3>
  <div class="settings-row">
    <div><div class="settings-label">Device Name</div><div class="settings-desc">Shown to other devices on the network</div></div>
    <input type="text" id="settingDeviceName" placeholder="My Device">
  </div>
  <div class="settings-row">
    <div><div class="settings-label">App Password</div><div class="settings-desc">Require a password to open Sparrow</div></div>
    <input type="text" id="settingAppPassword" placeholder="Set password" style="width:140px;">
  </div>
  <div class="settings-row">
    <div><div class="settings-label">Clear History</div><div class="settings-desc">Delete all saved transfer history</div></div>
    <button class="pill-btn" id="btnClearHistorySettings">Clear Now</button>
  </div>
</div>

<script>
(function () {
  var cfg = null;

  function load() {
    if (!window.goGetConfig) return;
    window.goGetConfig().then(function (c) {
      cfg = c;
      document.getElementById("settingThemeSelect").value = c.themeMode;
      document.getElementById("settingMinimizeTray").checked = c.minimizeToTray;
      document.getElementById("settingAutostart").checked = c.autostartEnabled;
      document.getElementById("settingShareTarget").checked = c.shareTargetEnabled;
      document.getElementById("saveFolderPath").textContent = c.downloadDir;
      document.getElementById("settingRequirePin").checked = c.requirePinReceive;
      document.getElementById("settingDeviceName").value = c.deviceName;
      document.querySelectorAll(".theme-swatch").forEach(function (sw) {
        sw.classList.toggle("selected", sw.getAttribute("data-theme-id") === c.colorTheme);
      });
    });
  }
  load();

  document.getElementById("settingThemeSelect").addEventListener("change", function (e) {
    document.documentElement.setAttribute("data-theme", e.target.value);
    localStorage.setItem("sparrow-theme", e.target.value);
    if (window.goSetThemeMode) window.goSetThemeMode(e.target.value);
  });

  document.querySelectorAll(".theme-swatch").forEach(function (sw) {
    sw.addEventListener("click", function () {
      document.querySelectorAll(".theme-swatch").forEach(function (s) { s.classList.remove("selected"); });
      sw.classList.add("selected");
      var id = sw.getAttribute("data-theme-id");
      if (window.applyColorTheme) window.applyColorTheme(id);
      if (window.goSetColorTheme) window.goSetColorTheme(id);
    });
  });

  document.getElementById("settingMinimizeTray").addEventListener("change", function (e) {
    if (window.goSetMinimizeToTray) window.goSetMinimizeToTray(e.target.checked);
  });

  document.getElementById("settingAutostart").addEventListener("change", function (e) {
    if (window.goSetAutostart) window.goSetAutostart(e.target.checked).catch(function (err) {
      alert("Failed to set autostart: " + err);
      e.target.checked = !e.target.checked;
    });
  });

  document.getElementById("settingShareTarget").addEventListener("change", function (e) {
    if (window.goSetShareTarget) window.goSetShareTarget(e.target.checked).catch(function (err) {
      alert("Failed to update right-click menu: " + err);
      e.target.checked = !e.target.checked;
    });
  });

  document.getElementById("btnChangeFolder").addEventListener("click", function () {
    if (!window.goPickFolder) return;
    window.goPickFolder().then(function (path) {
      if (!path) return;
      document.getElementById("saveFolderPath").textContent = path;
      if (window.goSetDownloadDir) window.goSetDownloadDir(path);
    });
  });

  document.getElementById("settingRequirePin").addEventListener("change", function (e) {
    if (window.goSetRequirePin) window.goSetRequirePin(e.target.checked);
  });

  document.getElementById("settingDeviceName").addEventListener("change", function (e) {
    if (window.goSetDeviceName) window.goSetDeviceName(e.target.value);
    var label = document.getElementById("deviceNameLabel");
    if (label) label.textContent = e.target.value;
  });

  document.getElementById("settingPinInput").addEventListener("change", function (e) {
    if (window.goSetPin) window.goSetPin(e.target.value);
  });

  document.getElementById("settingAppPassword").addEventListener("change", function (e) {
    if (window.goSetAppPassword) window.goSetAppPassword(e.target.value);
  });

  document.getElementById("btnClearHistorySettings").addEventListener("click", function () {
    if (confirm("Clear all transfer history?") && window.goClearHistory) {
      window.goClearHistory().then(function () { alert("History cleared."); });
    }
  });
})();
</script>
`