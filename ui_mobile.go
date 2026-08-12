package main

// mobileHTML is the page phones load when they scan Sparrow's QR code or
// type the LAN URL manually. It is completely self-contained (CSS + JS
// inline) since phones only ever load this one page — no separate
// stylesheet/script requests needed.
const mobileHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0">
<title>Sparrow — Connect</title>
<style>
  * { box-sizing: border-box; }
  html, body {
    margin: 0; padding: 0;
    background: #000000; color: #E6E6FA;
    font-family: "Segoe UI", -apple-system, sans-serif;
  }
  .wrap { max-width: 480px; margin: 0 auto; padding: 20px 16px 60px; }
  header { text-align: center; margin-bottom: 22px; }
  header h1 { margin: 8px 0 2px; font-size: 22px; letter-spacing: 1px; }
  header p { margin: 0; color: #B9AEDD; font-size: 12px; }

  .card {
    background: rgba(230,230,250,0.06);
    border: 1px solid rgba(230,230,250,0.14);
    border-radius: 14px;
    padding: 16px;
    margin-bottom: 16px;
  }
  .card h2 { font-size: 14px; margin: 0 0 12px; text-transform: uppercase; letter-spacing: 1px; color: #B9AEDD; }

  .drop-area {
    border: 2px dashed rgba(230,230,250,0.25);
    border-radius: 12px;
    padding: 28px 14px;
    text-align: center;
    font-size: 13px;
    color: #B9AEDD;
  }
  input[type=file] { width: 100%; margin-top: 12px; font-size: 13px; color: #E6E6FA; }

  textarea {
    width: 100%; min-height: 70px;
    background: rgba(230,230,250,0.07);
    border: 1px solid rgba(230,230,250,0.14);
    border-radius: 10px;
    color: #E6E6FA;
    padding: 10px; font-size: 13px;
    resize: vertical;
  }

  button {
    background: #9B6DFF; color: white; border: none;
    border-radius: 10px; padding: 11px 18px;
    font-size: 14px; font-weight: 700;
    width: 100%; margin-top: 10px;
    cursor: pointer;
  }
  button:active { filter: brightness(0.9); }

  .progress-track {
    width: 100%; height: 8px; border-radius: 999px;
    background: rgba(230,230,250,0.1); margin-top: 10px; overflow: hidden; display: none;
  }
  .progress-fill { height: 100%; width: 0%; background: #9B6DFF; transition: width 0.15s ease; }

  .file-item {
    display: flex; justify-content: space-between; align-items: center;
    background: rgba(230,230,250,0.07);
    border-radius: 10px; padding: 10px 12px; margin-bottom: 8px; font-size: 13px;
  }
  .file-item button { width: auto; margin: 0; padding: 6px 14px; font-size: 12px; }

  .empty-hint { color: #7d739b; font-size: 12px; text-align: center; padding: 10px 0; }
  .status-msg { font-size: 12px; margin-top: 8px; color: #3ddc84; display: none; }
</style>
</head>
<body>
<div class="wrap">
  <header>
    <div style="font-size:34px;">&#128038;</div>
    <h1>SPARROW</h1>
    <p id="whoamiLine">Connecting...</p>
  </header>

  <div class="card">
    <h2>Send to PC</h2>
    <div class="drop-area">
      Tap below to choose a photo, video, or any file from your phone
      <input type="file" id="fileInput" multiple>
    </div>
    <div class="progress-track" id="progressTrack">
      <div class="progress-fill" id="progressFill"></div>
    </div>
    <button id="uploadBtn">Send to PC</button>
    <div class="status-msg" id="uploadStatus"></div>
  </div>

  <div class="card">
    <h2>Send Text / Link</h2>
    <textarea id="textInput" placeholder="Paste text or a link here..."></textarea>
    <button id="textSendBtn">Send Text</button>
    <div class="status-msg" id="textStatus"></div>
  </div>

  <div class="card">
    <h2>Receive from PC</h2>
    <div id="outboxList">
      <div class="empty-hint">Nothing shared from PC yet.</div>
    </div>
    <button id="refreshBtn" style="background:transparent;border:1px solid rgba(230,230,250,0.2);">Refresh</button>
  </div>
</div>

<script>
(function () {
  fetch("/api/whoami").then(function (r) { return r.json(); }).then(function (d) {
    document.getElementById("whoamiLine").textContent = "Connected to " + d.deviceName;
  }).catch(function () {
    document.getElementById("whoamiLine").textContent = "Connected";
  });

  function formatBytes(bytes) {
    if (bytes === 0) return "0 B";
    var units = ["B", "KB", "MB", "GB"];
    var i = Math.floor(Math.log(bytes) / Math.log(1024));
    return (bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1) + " " + units[i];
  }

  var fileInput = document.getElementById("fileInput");
  var uploadBtn = document.getElementById("uploadBtn");
  var progressTrack = document.getElementById("progressTrack");
  var progressFill = document.getElementById("progressFill");
  var uploadStatus = document.getElementById("uploadStatus");

  uploadBtn.addEventListener("click", function () {
    var files = fileInput.files;
    if (!files || files.length === 0) {
      alert("Choose at least one file first.");
      return;
    }
    uploadNext(0, files);
  });

  function uploadNext(index, files) {
    if (index >= files.length) {
      uploadStatus.textContent = "All files sent successfully.";
      uploadStatus.style.display = "block";
      progressTrack.style.display = "none";
      return;
    }
    var file = files[index];
    var formData = new FormData();
    formData.append("file", file);

    var xhr = new XMLHttpRequest();
    xhr.open("POST", "/api/upload");

    progressTrack.style.display = "block";
    xhr.upload.onprogress = function (e) {
      if (e.lengthComputable) {
        var pct = Math.round((e.loaded / e.total) * 100);
        progressFill.style.width = pct + "%";
      }
    };

    xhr.onload = function () {
      uploadNext(index + 1, files);
    };
    xhr.onerror = function () {
      uploadStatus.textContent = "Failed to send " + file.name;
      uploadStatus.style.display = "block";
    };

    xhr.send(formData);
  }

  document.getElementById("textSendBtn").addEventListener("click", function () {
    var text = document.getElementById("textInput").value.trim();
    if (!text) return;
    fetch("/api/paste", { method: "POST", body: text }).then(function () {
      var status = document.getElementById("textStatus");
      status.textContent = "Sent to PC.";
      status.style.display = "block";
      document.getElementById("textInput").value = "";
    });
  });

  function loadOutbox() {
    fetch("/api/outbox").then(function (r) { return r.json(); }).then(function (files) {
      var box = document.getElementById("outboxList");
      if (!files || files.length === 0) {
        box.innerHTML = '<div class="empty-hint">Nothing shared from PC yet.</div>';
        return;
      }
      box.innerHTML = "";
      files.forEach(function (f) {
        var row = document.createElement("div");
        row.className = "file-item";
        row.innerHTML =
          "<span>" + f.name + " (" + formatBytes(f.size) + ")</span>" +
          '<button onclick="window.location.href=\'/api/download?name=' + encodeURIComponent(f.name) + '\'">Download</button>';
        box.appendChild(row);
      });
    });
  }

  document.getElementById("refreshBtn").addEventListener("click", loadOutbox);
  loadOutbox();
  setInterval(loadOutbox, 5000);
})();
</script>
</body>
</html>
`