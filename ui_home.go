package main

const homePartialHTML = `
<div class="top-bar-row">
  <label class="switch-row">
    <span class="switch">
      <input type="checkbox" id="turboToggle" checked>
      <span class="slider"></span>
    </span>
    <span class="switch-label">Turbo Mode</span>
    <span class="switch-hint" id="turboHint">No encryption · Max speed</span>
  </label>
  <button class="lock-btn" id="lockAppBtn" title="Lock App">&#128274; Lock App</button>
</div>

<div class="drop-zone" id="dropZone">
  <div class="drop-zone-icon">&#8681;</div>
  <div class="drop-zone-title">Drag &amp; Drop Files Here</div>
  <div class="drop-zone-sub">or Click to Select</div>
</div>

<div class="action-row">
  <button class="pill-btn" id="btnSelectFiles">&#128196; Select Files</button>
  <button class="pill-btn" id="btnSelectFolder">&#128193; Select Folder</button>
  <button class="pill-btn" id="btnPasteLink">&#128203; Paste Text / Link</button>
  <button class="pill-btn" id="btnConnectPhone">&#9635; Connect Phone via QR</button>
  <button class="pill-btn" id="btnVoiceNote">&#127908; Voice Note</button>
</div>

<div id="qrModal" style="display:none; text-align:center; margin-bottom:18px; padding:20px; background:var(--card-bg); border:1px solid var(--card-border); border-radius:var(--radius);">
  <img id="qrImage" style="width:220px; height:220px; border-radius:8px; background:white; padding:8px;">
  <div id="qrUrlText" style="margin-top:12px; font-size:13px; color:var(--text-dim);"></div>
</div>

<div class="selected-list" id="selectedList">
  <div class="empty-hint">No files selected yet.</div>
</div>

<label style="display:flex; align-items:center; gap:8px; margin-bottom:10px; font-size:13px; color:var(--text-dim);">
  <input type="checkbox" id="groupSendToggle"> Send to multiple devices</label><div id="groupDeviceList" style="display:none; margin-bottom:12px;"></div>

<div class="send-controls">
  <select class="send-to-select" id="sendToSelect">
    <option value="">Send To: (no devices found yet)</option>
  </select>
  <button class="send-now-btn" id="sendNowBtn">SEND NOW</button>
</div>

<div id="activeTransfers" style="margin-top:22px;"></div>

<script>
(function () {
  var selectedFiles = [];
  var mediaRecorder = null;
  var audioChunks = [];

  function formatBytes(bytes) {
    if (!bytes || bytes === 0) return "0 B";
    var units = ["B", "KB", "MB", "GB"];
    var i = Math.floor(Math.log(bytes) / Math.log(1024));
    return (bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1) + " " + units[i];
  }

  function formatSpeed(bps) {
    if (!bps || bps <= 0) return "";
    return formatBytes(bps) + "/s";
  }

  function escapeHtml(str) {
    var div = document.createElement("div");
    div.textContent = str;
    return div.innerHTML;
  }

  function renderList() {
    var container = document.getElementById("selectedList");
    container.innerHTML = "";
    if (selectedFiles.length === 0) {
      container.innerHTML = '<div class="empty-hint">No files selected yet.</div>';
      return;
    }
    selectedFiles.forEach(function (file, index) {
      var row = document.createElement("div");
      row.className = "file-row";
      row.innerHTML =
        "<span>" + escapeHtml(file.name) +
        '<span class="file-meta">' + file.sizeLabel + "</span></span>" +
        '<button class="remove-btn" data-index="' + index + '" title="Remove">&times;</button>';
      container.appendChild(row);
    });
    container.querySelectorAll(".remove-btn").forEach(function (btn) {
      btn.addEventListener("click", function () {
        selectedFiles.splice(parseInt(btn.getAttribute("data-index"), 10), 1);
        renderList();
      });
    });
  }

  document.getElementById("dropZone").addEventListener("click", pickFilesNow);
  document.getElementById("btnSelectFiles").addEventListener("click", pickFilesNow);

  function pickFilesNow() {
    if (!window.goPickFiles) {
      alert("Native file picker binding not available.");
      return;
    }
    window.goPickFiles().then(function (paths) {
      if (!paths || paths.length === 0) return;
      paths.forEach(function (p) {
        var name = p.split("\\").pop().split("/").pop();
        selectedFiles.push({ path: p, name: name, sizeLabel: "" });
      });
      renderList();
    }).catch(function (err) {
      alert("File picker failed: " + err);
    });
  }

  document.getElementById("btnSelectFolder").addEventListener("click", function () {
    if (!window.goPickFolder) {
      alert("Native folder picker binding not available.");
      return;
    }
    window.goPickFolder().then(function (path) {
      if (!path) return;
      var name = path.split("\\").pop().split("/").pop();
      selectedFiles.push({ path: path, name: "[Folder] " + name, sizeLabel: "" });
      renderList();
    });
  });

  document.getElementById("btnPasteLink").addEventListener("click", function () {
    var text = prompt("Paste text or a link to send:");
    if (text && text.trim() !== "") {
      selectedFiles.push({ path: "", text: text, name: "[Text] " + text.slice(0, 40) + (text.length > 40 ? "..." : ""), sizeLabel: formatBytes(new Blob([text]).size) });
      renderList();
    }
  });

  document.getElementById("btnVoiceNote").addEventListener("click", function () {
    var btn = document.getElementById("btnVoiceNote");
    if (mediaRecorder && mediaRecorder.state === "recording") {
      mediaRecorder.stop();
      return;
    }
    navigator.mediaDevices.getUserMedia({ audio: true }).then(function (stream) {
      audioChunks = [];
      mediaRecorder = new MediaRecorder(stream);
      mediaRecorder.ondataavailable = function (e) { audioChunks.push(e.data); };
      mediaRecorder.onstop = function () {
        stream.getTracks().forEach(function (t) { t.stop(); });
        btn.textContent = "\uD83C\uDFA4 Voice Note";
        btn.style.color = "";
        var blob = new Blob(audioChunks, { type: "audio/webm" });
        var reader = new FileReader();
        reader.onloadend = function () {
          var base64 = reader.result.split(",")[1];
          var select = document.getElementById("sendToSelect");
          var opt = select.selectedOptions[0];
          if (!opt || !opt.dataset.ip) {
            alert("Choose a device first, then record again.");
            return;
          }
          if (window.goSendVoiceNote) {
            window.goSendVoiceNote(base64, opt.dataset.ip, parseInt(opt.dataset.port, 10)).then(function () {
              console.log("Voice note sent");
            }).catch(function (err) { alert("Failed to send voice note: " + err); });
          }
        };
        reader.readAsDataURL(blob);
      };
      mediaRecorder.start();
      btn.textContent = "\u23F9 Stop Recording";
      btn.style.color = "#e74c3c";
      setTimeout(function () {
        if (mediaRecorder && mediaRecorder.state === "recording") mediaRecorder.stop();
      }, 30000);
    }).catch(function (err) {
      alert("Microphone access denied or unavailable: " + err);
    });
  });

  function populateSendTo(devices) {
    var select = document.getElementById("sendToSelect");
    var currentValue = select.value;
    select.innerHTML = "";
    if (!devices || devices.length === 0) {
      select.innerHTML = '<option value="">Send To: (no devices found yet)</option>';
      return;
    }
    var placeholder = document.createElement("option");
    placeholder.value = "";
    placeholder.textContent = "Send To: choose a device";
    select.appendChild(placeholder);
    devices.forEach(function (d) {
      var opt = document.createElement("option");
      opt.value = d.id;
      opt.dataset.ip = d.ip;
      opt.dataset.port = d.port;
      opt.textContent = d.name + " (" + d.ip + ")";
      select.appendChild(opt);
    });
    if (window.__sparrowPreselectDeviceId) {
      select.value = window.__sparrowPreselectDeviceId;
      window.__sparrowPreselectDeviceId = null;
    } else if (currentValue) {
      select.value = currentValue;
    }
  }

  populateSendTo(window.__sparrowDevices);
  document.addEventListener("sparrow:devices-updated", function (e) {
    populateSendTo(e.detail);
  });

  document.getElementById("groupSendToggle").addEventListener("change", function (e) {
    document.getElementById("sendToSelect").style.display = e.target.checked ? "none" : "";
    document.getElementById("groupDeviceList").style.display = e.target.checked ? "block" : "none";
    if (e.target.checked) renderGroupDevices(window.__sparrowDevices);
  });

  function renderGroupDevices(devices) {
    var box = document.getElementById("groupDeviceList");
    box.innerHTML = "";
    (devices || []).forEach(function (d) {
      var row = document.createElement("label");
      row.style.cssText = "display:flex; align-items:center; gap:8px; padding:6px 0; font-size:13px;";
      row.innerHTML = '<input type="checkbox" class="group-device-cb" value="' + d.id + '"> ' + d.name + ' (' + d.ip + ')';
      box.appendChild(row);
    });
  }
  document.addEventListener("sparrow:devices-updated", function (e) {
    if (document.getElementById("groupSendToggle").checked) renderGroupDevices(e.detail);
  });

  document.getElementById("turboToggle").addEventListener("change", function (e) {
    var hint = document.getElementById("turboHint");
    hint.textContent = e.target.checked ? "No encryption · Max speed" : "AES encrypted · Slightly slower";
    if (window.goSetTurboMode) window.goSetTurboMode(e.target.checked);
  });

  document.getElementById("lockAppBtn").addEventListener("click", function () {
    if (window.goLockApp) window.goLockApp();
  });

  document.getElementById("btnConnectPhone").addEventListener("click", function () {
    var modal = document.getElementById("qrModal");
    if (modal.style.display === "block") {
      modal.style.display = "none";
      return;
    }
    modal.style.display = "block";
    if (window.goGetQRCodeBase64 && window.goGetLanURL) {
      window.goGetQRCodeBase64().then(function (b64) {
        document.getElementById("qrImage").src = "data:image/png;base64," + b64;
      });
      window.goGetLanURL().then(function (url) {
        document.getElementById("qrUrlText").textContent = url;
      });
    }
  });

  document.getElementById("sendNowBtn").addEventListener("click", function () {
    if (document.getElementById("groupSendToggle").checked) {
      var checked = Array.from(document.querySelectorAll(".group-device-cb:checked")).map(function (cb) { return cb.value; });
      if (checked.length === 0) { alert("Select at least one device."); return; }
      if (selectedFiles.length === 0) { alert("Select at least one file first."); return; }
      var f = selectedFiles.filter(function (x) { return x.path; })[0];
      if (!f) { alert("Text sending isn't supported for group-send yet."); return; }
      if (window.goSendFileToDevices) {
        window.goSendFileToDevices(f.path, checked).then(function () {
          selectedFiles = []; renderList();
        }).catch(function (err) { alert("Group send failed: " + err); });
      }
      return;
    }

    var select = document.getElementById("sendToSelect");
    var deviceId = select.value;
    if (selectedFiles.length === 0) {
      alert("Select at least one file or folder first.");
      return;
    }
    if (!deviceId) {
      alert("Choose a device to send to.");
      return;
    }
    var opt = select.querySelector('option[value="' + deviceId + '"]');
    var ip = opt.dataset.ip;
    var port = parseInt(opt.dataset.port, 10);
    if (!window.goSendFile) {
      alert("Transfer engine binding not available.");
      return;
    }
    var filesToSend = selectedFiles.filter(function (f) { return f.path; });
    if (filesToSend.length === 0) {
      alert("Text/link sending over the fast-transfer protocol arrives in a later part — use the Chat panel for text for now.");
      return;
    }
    filesToSend.forEach(function (f) {
      window.goSendFile(f.path, ip, port).then(function (transferId) {
        console.log("Started transfer", transferId, "for", f.name);
      }).catch(function (err) {
        alert("Failed to send " + f.name + ": " + err);
      });
    });
    selectedFiles = [];
    renderList();
  });

  function renderTransfers(transfers) {
    var box = document.getElementById("activeTransfers");
    if (!transfers || transfers.length === 0) {
      box.innerHTML = "";
      return;
    }
    box.innerHTML = '<h3 style="font-size:13px; text-transform:uppercase; letter-spacing:1px; color:var(--text-dim); margin-bottom:10px;">Transfers</h3>';
    transfers.forEach(function (t) {
      var pct = t.totalSize > 0 ? Math.min(100, Math.round((t.bytesDone / t.totalSize) * 100)) : 0;
      var row = document.createElement("div");
      row.style.cssText = "background:var(--card-bg); border:1px solid var(--card-border); border-radius:var(--radius); padding:12px 16px; margin-bottom:10px;";
      row.innerHTML =
        '<div style="display:flex; justify-content:space-between; font-size:13px; margin-bottom:6px;">' +
        '<span>' + (t.direction === "send" ? "&#8593; " : "&#8595; ") + escapeHtml(t.fileName) + '</span>' +
        '<span style="color:var(--text-dim);">' + pct + '% · ' + formatSpeed(t.speedBps) + '</span>' +
        '</div>' +
        '<div style="height:6px; border-radius:999px; background:var(--input-bg); overflow:hidden;">' +
        '<div style="height:100%; width:' + pct + '%; background:var(--accent); transition:width 0.2s ease;"></div>' +
        '</div>' +
        '<div style="display:flex; gap:8px; margin-top:8px;">' +
        (t.status === "active" ? '<button class="pill-btn" data-action="pause" data-id="' + t.id + '" style="padding:4px 12px; font-size:11px;">Pause</button>' : '') +
        (t.status === "paused" ? '<button class="pill-btn" data-action="resume" data-id="' + t.id + '" style="padding:4px 12px; font-size:11px;">Resume</button>' : '') +
        (t.status === "active" || t.status === "paused" ? '<button class="pill-btn" data-action="cancel" data-id="' + t.id + '" style="padding:4px 12px; font-size:11px;">Cancel</button>' : '') +
        '<span style="font-size:11px; color:var(--text-dim); align-self:center;">' + t.status + '</span>' +
        '</div>';
      box.appendChild(row);
    });
    box.querySelectorAll("button[data-action]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var id = btn.getAttribute("data-id");
        var action = btn.getAttribute("data-action");
        if (action === "pause" && window.goPauseTransfer) window.goPauseTransfer(id);
        if (action === "resume" && window.goResumeTransfer) window.goResumeTransfer(id);
        if (action === "cancel" && window.goCancelTransfer) window.goCancelTransfer(id);
      });
    });
  }

  var transferPollTimer = setInterval(function () {
    if (!document.getElementById("activeTransfers")) {
      clearInterval(transferPollTimer);
      return;
    }
    if (window.goListTransfers) {
      window.goListTransfers().then(renderTransfers);
    }
  }, 1000);

  renderList();
})();
</script>
`