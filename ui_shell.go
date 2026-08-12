package main

const lockOverlayHTML = `<div id="lockOverlay" style="display:none; position:fixed; inset:0; z-index:9999; background:var(--bg); display:none; flex-direction:column; align-items:center; justify-content:center; gap:16px;">  <div style="font-size:40px;">&#128038;</div>  <div style="font-size:18px; font-weight:700;">Sparrow Locked</div>  <input type="password" id="lockPinInput" maxlength="4" placeholder="Enter PIN" style="text-align:center; font-size:20px; letter-spacing:6px; width:140px; padding:10px; border-radius:10px; border:1px solid var(--card-border); background:var(--input-bg); color:var(--text);">  <button class="send-now-btn" id="lockUnlockBtn" style="width:140px;">Unlock</button>  <div id="lockError" style="color:#e74c3c; font-size:12px; display:none;">Incorrect PIN</div></div>`

const passwordGateHTML = `<div id="passwordGate" style="display:none; position:fixed; inset:0; z-index:10000; background:var(--bg); flex-direction:column; align-items:center; justify-content:center; gap:16px;">  <div style="font-size:40px;">&#128038;</div>  <div style="font-size:18px; font-weight:700;">Enter App Password</div>  <input type="password" id="gatePasswordInput" placeholder="Password" style="text-align:center; font-size:16px; width:220px; padding:10px; border-radius:10px; border:1px solid var(--card-border); background:var(--input-bg); color:var(--text);">  <button class="send-now-btn" id="gateUnlockBtn" style="width:220px;">Continue</button>  <div id="gateError" style="color:#e74c3c; font-size:12px; display:none;">Incorrect password</div></div>`

// shellHTML is the persistent app frame: left sidebar navigation, top bar
// (theme toggle), a content-mount area where tab partials get injected via
// fetch(), and the right sidebar (nearby devices + ephemeral chat) which
// stays visible no matter which tab is active — exactly like the
// screenshots. This loads first; app.js then fetches /partial/home into
// #content-mount automatically.
const shellHTML = `
<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Sparrow</title>
<link rel="stylesheet" href="/style.css">
</head>
<body>

` + lockOverlayHTML + passwordGateHTML + `
<div id="app">

  <aside class="sidebar">
    <div class="brand">
      <div class="brand-logo">
        <svg viewBox="0 0 64 64" width="42" height="42" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M8 40c8-2 14-8 16-16 4 10 14 16 24 14-6 4-8 10-6 16-8-4-16-2-22 4 0-8-4-14-12-18z"
                stroke="currentColor" stroke-width="2.5" stroke-linejoin="round" fill="none"/>
        </svg>
      </div>
      <div class="brand-name">SPARROW</div>
    </div>

    <nav class="nav-list" id="navList">
      <button class="nav-item active" data-tab="home">
        <span class="nav-icon">&#8962;</span>
        <span class="nav-labels">
          <span class="nav-title">Home</span>
          <span class="nav-sub">Send / Receive</span>
        </span>
      </button>
      <button class="nav-item" data-tab="devices">
        <span class="nav-icon">&#128421;</span>
        <span class="nav-labels">
          <span class="nav-title">Devices</span>
          <span class="nav-sub">Connected Devices</span>
        </span>
      </button>
      <button class="nav-item" data-tab="history">
        <span class="nav-icon">&#8635;</span>
        <span class="nav-labels">
          <span class="nav-title">History</span>
          <span class="nav-sub">All Transfers</span>
        </span>
      </button>
      <button class="nav-item" data-tab="chat">
        <span class="nav-icon">&#128172;</span>
        <span class="nav-labels">
          <span class="nav-title">Chat</span>
          <span class="nav-sub">Ephemeral Chat</span>
        </span>
      </button>
      <button class="nav-item" data-tab="settings">
        <span class="nav-icon">&#9881;</span>
        <span class="nav-labels">
          <span class="nav-title">Settings</span>
          <span class="nav-sub">App Settings</span>
        </span>
      </button>
      <button class="nav-item" data-tab="about">
        <span class="nav-icon">&#8505;</span>
        <span class="nav-labels">
          <span class="nav-title">About</span>
        </span>
      </button>
    </nav>

    <div class="sidebar-footer">
      <div class="footer-row">
        <span class="status-dot online"></span>
        <span id="deviceNameLabel" class="device-name-label">Loading...</span>
      </div>
      <div class="footer-ip" id="deviceIpLabel">IP: --.--.--.--</div>
    </div>
  </aside>

  <main class="content">
    <div class="tab-header">
      <h1 id="tabTitle">Home</h1>
      <div class="theme-toggle" id="themeToggle" title="Toggle light / dark mode">
        <span class="theme-icon moon">&#9789;</span>
        <span class="theme-icon sun">&#9728;</span>
      </div>
    </div>
    <div id="content-mount">
      <div class="empty-hint">Loading Sparrow...</div>
    </div>
  </main>

  <aside class="right-sidebar">
    <div class="right-section">
      <h2>Nearby Devices</h2>
      <div class="nearby-list" id="nearbyList">
        <div class="empty-hint small">Searching for devices on this Wi-Fi...</div>
      </div>
    </div>

    <div class="right-section chat-section">
      <h2>Ephemeral Chat</h2>
      <div class="chat-messages" id="chatMessages">
        <div class="empty-hint small">Messages are cleared when Sparrow closes.</div>
      </div>
      <div class="chat-input-row">
        <input type="text" id="chatInput" placeholder="Type a message...">
        <button id="chatSendBtn" title="Send">&#10148;</button>
        <button id="chatMicBtn" title="Voice note">&#127908;</button>
      </div>
    </div>
  </aside>

</div>

<script src="/app.js"></script>
</body>
</html>
`