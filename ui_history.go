package main

const historyPartialHTML = `
<input type="text" class="history-search" id="historySearch" placeholder="Search by file name or device...">
<table class="history-table">
  <thead><tr><th>File</th><th>Device</th><th>Direction</th><th>Size</th><th>Date</th><th>Action</th></tr></thead>
  <tbody id="historyBody"><tr><td colspan="6" class="empty-hint">Loading...</td></tr></tbody>
</table>
<div class="action-row" style="margin-top:18px;"><button class="pill-btn" id="btnClearHistory">&#128465; Clear History</button></div>

<script>
(function () {
  function escapeHtml(str) { var d = document.createElement("div"); d.textContent = str; return d.innerHTML; }

  function formatBytes(b) {
    if (!b) return "0 B";
    var u = ["B","KB","MB","GB"]; var i = Math.floor(Math.log(b)/Math.log(1024));
    return (b/Math.pow(1024,i)).toFixed(i===0?0:1) + " " + u[i];
  }

  function render(entries) {
    var body = document.getElementById("historyBody");
    if (!entries || entries.length === 0) {
      body.innerHTML = '<tr><td colspan="6" class="empty-hint">No transfer history yet.</td></tr>';
      return;
    }
    body.innerHTML = "";
    entries.forEach(function (e) {
      var date = new Date(e.timestampMs).toLocaleString();
      var tr = document.createElement("tr");
      tr.innerHTML =
        "<td>" + escapeHtml(e.fileName) + "</td>" +
        "<td>" + escapeHtml(e.peerName) + "</td>" +
        "<td>" + (e.direction === "send" ? "Sent" : "Received") + "</td>" +
        "<td>" + formatBytes(e.totalSize) + "</td>" +
        "<td>" + date + "</td>" +
        '<td><button class="pill-btn" data-path="' + encodeURIComponent(e.filePath) + '" style="padding:4px 10px; font-size:11px;">Open</button></td>';
      body.appendChild(tr);
    });
    body.querySelectorAll("button[data-path]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var path = decodeURIComponent(btn.getAttribute("data-path"));
        if (window.goOpenFolder) window.goOpenFolder(path).catch(function (err) { alert("Could not open: " + err); });
      });
    });
  }

  function load() {
    if (window.goListHistory) window.goListHistory().then(render);
  }
  load();

  document.getElementById("historySearch").addEventListener("input", function (e) {
    if (window.goSearchHistory) window.goSearchHistory(e.target.value).then(render);
  });

  document.getElementById("btnClearHistory").addEventListener("click", function () {
    if (confirm("Clear all transfer history? This cannot be undone.") && window.goClearHistory) {
      window.goClearHistory().then(load);
    }
  });
})();
</script>
`