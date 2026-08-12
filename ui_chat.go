package main

const chatPartialHTML = `
<p class="muted-note" style="margin-bottom:14px;">Messages send over the real LAN transfer engine. Cleared when Sparrow closes.</p>

<select class="send-to-select" id="chatDeviceSelect" style="margin-bottom:14px;">
  <option value="">Choose a device to chat with</option>
</select>

<div class="chat-messages" id="fullChatMessages" style="min-height:340px; border:1px solid var(--card-border); border-radius:var(--radius); padding:14px; background:var(--card-bg);">
  <div class="empty-hint small">No messages yet.</div>
</div>

<div class="chat-input-row" style="margin-top:14px;">
  <input type="text" id="fullChatInput" placeholder="Type a message...">
  <button id="fullChatSendBtn" title="Send">&#10148;</button>
</div>

<script>
(function () {
  function escapeHtml(str) { var d = document.createElement("div"); d.textContent = str; return d.innerHTML; }

  function populateDevices(devices) {
    var select = document.getElementById("chatDeviceSelect");
    var current = select.value;
    select.innerHTML = '<option value="">Choose a device to chat with</option>';
    (devices || []).forEach(function (d) {
      var opt = document.createElement("option");
      opt.value = d.id; opt.dataset.ip = d.ip; opt.dataset.port = d.port;
      opt.textContent = d.name + " (" + d.ip + ")";
      select.appendChild(opt);
    });
    if (current) select.value = current;
  }
  populateDevices(window.__sparrowDevices);
  document.addEventListener("sparrow:devices-updated", function (e) { populateDevices(e.detail); });

  function render(messages) {
    var box = document.getElementById("fullChatMessages");
    if (!messages || messages.length === 0) {
      box.innerHTML = '<div class="empty-hint small">No messages yet.</div>';
      return;
    }
    box.innerHTML = "";
    messages.forEach(function (m) {
      var bubble = document.createElement("div");
      bubble.className = "chat-bubble " + (m.outgoing ? "outgoing" : "incoming");
      bubble.innerHTML = (m.outgoing ? "" : '<span class="chat-sender">' + escapeHtml(m.peerName) + '</span>') + escapeHtml(m.content);
      box.appendChild(bubble);
    });
    box.scrollTop = box.scrollHeight;
  }

  function poll() {
    if (window.goListTexts) window.goListTexts().then(render);
  }
  poll();
  var timer = setInterval(function () {
    if (!document.getElementById("fullChatMessages")) { clearInterval(timer); return; }
    poll();
  }, 2000);

  function send() {
    var input = document.getElementById("fullChatInput");
    var text = input.value.trim();
    if (!text) return;
    var select = document.getElementById("chatDeviceSelect");
    var opt = select.selectedOptions[0];
    if (!opt || !opt.dataset.ip) { alert("Choose a device first."); return; }
    if (window.goSendText) {
      window.goSendText(text, opt.dataset.ip, parseInt(opt.dataset.port, 10)).then(function () {
        input.value = "";
        poll();
      }).catch(function (err) { alert("Failed to send: " + err); });
    }
  }

  document.getElementById("fullChatSendBtn").addEventListener("click", send);
  document.getElementById("fullChatInput").addEventListener("keydown", function (e) { if (e.key === "Enter") send(); });
})();
</script>
`