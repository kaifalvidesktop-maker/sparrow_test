package main

const devicesPartialHTML = `
<p class="muted-note">Sparrow scans your Wi-Fi automatically every few seconds.</p>
<div class="action-row"><button class="pill-btn" id="btnRefreshDevices">&#128260; Refresh</button></div>
<div class="card-grid" id="devicesGrid"><div class="empty-hint">Searching...</div></div>

<script>
(function () {
  function escapeHtml(str) { var d = document.createElement("div"); d.textContent = str; return d.innerHTML; }

  function render(devices) {
    var grid = document.getElementById("devicesGrid");
    if (!devices || devices.length === 0) {
      grid.innerHTML = '<div class="empty-hint">No devices found. Make sure Sparrow is open on the other device on the same Wi-Fi.</div>';
      return;
    }
    grid.innerHTML = "";
    devices.forEach(function (d) {
      var card = document.createElement("div");
      card.className = "device-card";
      var initial = (d.name || "?").charAt(0).toUpperCase();
      card.innerHTML =
        '<span class="device-avatar-lg">' + initial + '</span>' +
        '<div style="font-weight:700;">' + escapeHtml(d.name) + '</div>' +
        '<div class="muted-note">' + escapeHtml(d.ip) + '</div>';
      grid.appendChild(card);
    });
  }

  render(window.__sparrowDevices || []);
  document.addEventListener("sparrow:devices-updated", function (e) { render(e.detail); });

  document.getElementById("btnRefreshDevices").addEventListener("click", function () {
    document.getElementById("devicesGrid").innerHTML = '<div class="empty-hint">Scanning...</div>';
    if (window.goListDevices) window.goListDevices().then(render);
  });
})();
</script>
`