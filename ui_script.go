package main

// appJS is the entire frontend logic: tab routing (fetches /partial/<tab>
// and injects it into #content-mount, then executes any inline <script>
// found inside that partial), theme handling, and the persistent shell
// features (right sidebar nearby devices + ephemeral chat).
const appJS = `
(function () {
  "use strict";

  var TAB_TITLES = {
    home: "Home",
    devices: "Connected Devices",
    history: "Transfer History",
    chat: "Ephemeral Chat",
    settings: "Settings",
    about: "About Sparrow"
  };

  /* ---------------- THEME (Dark/Light) ---------------- */
  function applyTheme(theme) {
    document.documentElement.setAttribute("data-theme", theme);
    localStorage.setItem("sparrow-theme", theme);
    if (window.goSetThemeMode) window.goSetThemeMode(theme);
    
    // Sync the settings dropdown if it exists on the current page
    var select = document.getElementById("settingThemeSelect");
    if (select) {
      select.value = theme;
    }
  }
  // Expose globally so Settings partial and other modules can call it.
  window.applyTheme = applyTheme;

  function initTheme() {
    var saved = localStorage.getItem("sparrow-theme") || "dark";
    applyTheme(saved);
    var toggle = document.getElementById("themeToggle");
    if (toggle) {
      toggle.addEventListener("click", function () {
        var current = document.documentElement.getAttribute("data-theme");
        applyTheme(current === "dark" ? "light" : "dark");
      });
    }
  }

  /* ---------------- COLOR THEME (accent color swatches) ---------------- */
  var COLOR_THEMES = {
    "violet-default": { accent: "#9B6DFF", strong: "#4B0082" },
    "violet-green":   { accent: "#2ecc71", strong: "#9B6DFF" },
    "violet-blue":    { accent: "#3498db", strong: "#9B6DFF" },
    "violet-red":     { accent: "#e74c3c", strong: "#9B6DFF" },
    "violet-gold":    { accent: "#f1c40f", strong: "#9B6DFF" },
    "violet-teal":    { accent: "#1abc9c", strong: "#9B6DFF" },
    "violet-pink":    { accent: "#ff6fb1", strong: "#9B6DFF" },
    "violet-orange":  { accent: "#e67e22", strong: "#9B6DFF" },
    "violet-mono":    { accent: "#B48CFF", strong: "#2c2c2c" },
    "violet-slate":   { accent: "#607d8b", strong: "#9B6DFF" }
  };

  function applyColorTheme(id) {
    var theme = COLOR_THEMES[id] || COLOR_THEMES["violet-default"];
    document.documentElement.style.setProperty("--accent", theme.accent);
    document.documentElement.style.setProperty("--accent-strong", theme.strong);
    localStorage.setItem("sparrow-color-theme", id);
  }
  window.applyColorTheme = applyColorTheme;

  function initColorTheme() {
    var saved = localStorage.getItem("sparrow-color-theme") || "violet-default";
    applyColorTheme(saved);
  }

  /* ---------------- SCRIPT EXECUTION HELPER ---------------- */
  function executeScripts(container) {
    var scripts = container.querySelectorAll("script");
    scripts.forEach(function (oldScript) {
      var newScript = document.createElement("script");
      Array.from(oldScript.attributes).forEach(function (attr) {
        newScript.setAttribute(attr.name, attr.value);
      });
      newScript.textContent = oldScript.textContent;
      oldScript.parentNode.replaceChild(newScript, oldScript);
    });
  }

  /* ---------------- TAB ROUTER ---------------- */
  function loadPartial(tabName) {
    var mount = document.getElementById("content-mount");
    mount.innerHTML = '<div class="empty-hint">Loading...</div>';

    fetch("/partial/" + tabName)
      .then(function (res) {
        if (!res.ok) throw new Error("Failed to load partial: " + tabName);
        return res.text();
      })
      .then(function (html) {
        mount.innerHTML = html;
        document.getElementById("tabTitle").textContent = TAB_TITLES[tabName] || tabName;
        executeScripts(mount);
      })
      .catch(function (err) {
        mount.innerHTML = '<div class="empty-hint">Failed to load tab: ' + err.message + '</div>';
      });
  }

  function initNav() {
    var navItems = document.querySelectorAll(".nav-item");
    navItems.forEach(function (btn) {
      btn.addEventListener("click", function () {
        var target = btn.getAttribute("data-tab");
        navItems.forEach(function (b) { b.classList.remove("active"); });
        btn.classList.add("active");
        loadPartial(target);
      });
    });
  }

  /* ---------------- SHELL: BACKEND INFO ---------------- */
  function initBackendInfo() {
    if (window.goGetDeviceName) {
      window.goGetDeviceName().then(function (name) {
        document.getElementById("deviceNameLabel").textContent = name;
      });
    }
    if (window.goGetLocalIP) {
      window.goGetLocalIP().then(function (ip) {
        document.getElementById("deviceIpLabel").textContent = "IP: " + ip;
      });
    }
  }

  /* ---------------- SHELL: NEARBY DEVICES (right sidebar + shared cache) ---------------- */
  window.__sparrowDevices = [];

  function renderNearbyList(devices) {
    var box = document.getElementById("nearbyList");
    if (!box) return;
    if (!devices || devices.length === 0) {
      box.innerHTML = '<div class="empty-hint small">Searching for devices on this Wi-Fi...</div>';
      return;
    }
    box.innerHTML = "";
    devices.forEach(function (d) {
      var chip = document.createElement("div");
      chip.className = "device-chip";
      var initial = (d.name || "?").charAt(0).toUpperCase();
      chip.innerHTML =
        '<span class="device-avatar">' + initial + '</span>' +
        '<span class="device-info">' +
        '<span class="device-name">' + escapeHtml(d.name) + '</span>' +
        '<span class="device-meta">' + escapeHtml(d.ip) + '</span>' +
        '</span>';
      chip.addEventListener("click", function () {
        window.__sparrowPreselectDeviceId = d.id;
        document.querySelector('.nav-item[data-tab="home"]').click();
      });
      box.appendChild(chip);
    });
  }

  function pollDevices() {
    if (!window.goListDevices) return;
    window.goListDevices().then(function (devices) {
      window.__sparrowDevices = devices || [];
      renderNearbyList(window.__sparrowDevices);
      document.dispatchEvent(new CustomEvent("sparrow:devices-updated", { detail: window.__sparrowDevices }));
    });
  }

  function initDevicePolling() {
    pollDevices();
    setInterval(pollDevices, 2500);
  }

  /* ---------------- SHELL: LOCK SCREEN ---------------- */
  function initLockScreen() {
    var overlay = document.getElementById("lockOverlay");
    if (!overlay) return;

    function checkLocked() {
      if (window.goIsLocked) {
        window.goIsLocked().then(function (locked) {
          overlay.style.display = locked ? "flex" : "none";
        });
      }
    }
    checkLocked();
    setInterval(checkLocked, 1500);

    document.getElementById("lockUnlockBtn").addEventListener("click", tryUnlock);
    document.getElementById("lockPinInput").addEventListener("keydown", function (e) {
      if (e.key === "Enter") tryUnlock();
    });

    function tryUnlock() {
      var pin = document.getElementById("lockPinInput").value;
      if (window.goUnlockApp) {
        window.goUnlockApp(pin).then(function (ok) {
          var err = document.getElementById("lockError");
          if (ok) {
            overlay.style.display = "none";
            document.getElementById("lockPinInput").value = "";
            err.style.display = "none";
          } else {
            err.style.display = "block";
          }
        });
      }
    }
  }

  /* ---------------- SHELL: PASSWORD GATE ---------------- */
  function initPasswordGate() {
    var gate = document.getElementById("passwordGate");
    if (!gate) return;

    if (window.goHasAppPassword) {
      window.goHasAppPassword().then(function (has) {
        gate.style.display = has ? "flex" : "none";
      });
    }

    document.getElementById("gateUnlockBtn").addEventListener("click", tryGate);
    document.getElementById("gatePasswordInput").addEventListener("keydown", function (e) {
      if (e.key === "Enter") tryGate();
    });

    function tryGate() {
      var pw = document.getElementById("gatePasswordInput").value;
      if (window.goVerifyAppPassword) {
        window.goVerifyAppPassword(pw).then(function (ok) {
          var err = document.getElementById("gateError");
          if (ok) {
            gate.style.display = "none";
            err.style.display = "none";
          } else {
            err.style.display = "block";
          }
        });
      }
    }
  }

  /* ---------------- SHELL: EPHEMERAL CHAT (right sidebar) ---------------- */
  function escapeHtml(str) {
    var div = document.createElement("div");
    div.textContent = str;
    return div.innerHTML;
  }

  function initShellChat() {
    var box = document.getElementById("chatMessages");
    var input = document.getElementById("chatInput");
    var sendBtn = document.getElementById("chatSendBtn");
    var micBtn = document.getElementById("chatMicBtn");

    function appendBubble(text, outgoing) {
      if (box.querySelector(".empty-hint")) box.innerHTML = "";
      var bubble = document.createElement("div");
      bubble.className = "chat-bubble " + (outgoing ? "outgoing" : "incoming");
      bubble.innerHTML = (outgoing ? "" : '<span class="chat-sender">Nearby Device</span>') + escapeHtml(text);
      box.appendChild(bubble);
      box.scrollTop = box.scrollHeight;
    }

    function send() {
      var text = input.value.trim();
      if (!text) return;
      appendBubble(text, true);
      input.value = "";
    }

    sendBtn.addEventListener("click", send);
    input.addEventListener("keydown", function (e) { if (e.key === "Enter") send(); });
    micBtn.addEventListener("click", function () {
      alert("Voice note recording will use MediaRecorder in a later part.");
    });
  }

  /* ---------------- INIT ---------------- */
  document.addEventListener("DOMContentLoaded", function () {
    initLockScreen();
    initPasswordGate();
    initTheme();
    initColorTheme();
    initNav();
    initBackendInfo();
    initShellChat();
    initDevicePolling();
    loadPartial("home");
  });
})();
`