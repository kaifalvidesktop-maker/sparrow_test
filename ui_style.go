package main

// appCSS holds every style rule for Sparrow: light/dark violet theme
// variables, glassmorphism cards, rounded corners, and the 3-column
// responsive layout (sidebar / content / right sidebar).
const appCSS = `
:root {
  --radius: 12px;
  --sidebar-w: 240px;
  --right-w: 300px;
  --font: "Segoe UI", "Inter", -apple-system, sans-serif;
}

html[data-theme="dark"] {
  --bg: #000000;
  --text: #E6E6FA;
  --text-dim: #B9AEDD;
  --card-bg: rgba(230, 230, 250, 0.06);
  --card-border: rgba(230, 230, 250, 0.14);
  --accent: #9B6DFF;
  --accent-strong: #B48CFF;
  --sidebar-bg: rgba(20, 12, 34, 0.55);
  --hover-bg: rgba(230, 230, 250, 0.10);
  --input-bg: rgba(230, 230, 250, 0.07);
  --shadow: 0 8px 24px rgba(0, 0, 0, 0.45);
}

html[data-theme="light"] {
  --bg: #FFFFFF;
  --text: #4B0082;
  --text-dim: #6E3FA3;
  --card-bg: rgba(75, 0, 130, 0.05);
  --card-border: rgba(75, 0, 130, 0.14);
  --accent: #6A0DAD;
  --accent-strong: #4B0082;
  --sidebar-bg: rgba(240, 230, 250, 0.55);
  --hover-bg: rgba(75, 0, 130, 0.08);
  --input-bg: rgba(75, 0, 130, 0.05);
  --shadow: 0 8px 24px rgba(75, 0, 130, 0.12);
}

* { box-sizing: border-box; }

html, body {
  margin: 0; padding: 0; height: 100%;
  font-family: var(--font);
  background: var(--bg);
  color: var(--text);
  overflow: hidden;
  user-select: none;
}

button, input, select { font-family: inherit; color: inherit; }

#app {
  display: grid;
  grid-template-columns: var(--sidebar-w) 1fr var(--right-w);
  height: 100vh; width: 100vw;
}

.sidebar {
  background: var(--sidebar-bg);
  backdrop-filter: blur(18px);
  border-right: 1px solid var(--card-border);
  display: flex; flex-direction: column;
  padding: 24px 16px;
}

.brand { display: flex; flex-direction: column; align-items: center; gap: 6px; margin-bottom: 28px; }
.brand-logo { color: var(--accent-strong); }
.brand-name { font-size: 22px; font-weight: 700; letter-spacing: 2px; }

.nav-list { display: flex; flex-direction: column; gap: 6px; flex: 1; }

.nav-item {
  display: flex; align-items: center; gap: 12px;
  padding: 10px 12px; border-radius: var(--radius);
  border: none; background: transparent; color: var(--text);
  cursor: pointer; text-align: left;
  transition: background 0.15s ease;
  width: 100%;
}
.nav-item:hover { background: var(--hover-bg); }
.nav-item.active {
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  box-shadow: var(--shadow);
}

.nav-icon { font-size: 18px; width: 22px; text-align: center; }
.nav-labels { display: flex; flex-direction: column; line-height: 1.25; }
.nav-title { font-size: 14px; font-weight: 600; }
.nav-sub { font-size: 11px; color: var(--text-dim); }

.sidebar-footer { border-top: 1px solid var(--card-border); padding-top: 12px; font-size: 12px; color: var(--text-dim); }
.footer-row { display: flex; align-items: center; gap: 6px; }
.status-dot { width: 8px; height: 8px; border-radius: 50%; background: #666; }
.status-dot.online { background: #3ddc84; }
.footer-ip { margin-top: 4px; font-size: 11px; }

.content { padding: 28px 32px; overflow-y: auto; }

.tab-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 20px; }
.tab-header h1 { margin: 0; font-size: 24px; font-weight: 700; }

.theme-toggle {
  display: flex; align-items: center; gap: 8px;
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: 999px; padding: 6px 10px; cursor: pointer;
}
.theme-icon { font-size: 16px; opacity: 0.5; }
html[data-theme="dark"] .theme-icon.moon { opacity: 1; }
html[data-theme="light"] .theme-icon.sun { opacity: 1; }

.muted-note { color: var(--text-dim); font-size: 13px; }
.empty-hint { color: var(--text-dim); font-size: 13px; text-align: center; padding: 10px 0; }
.empty-hint.small { font-size: 12px; padding: 6px 0; }

.top-bar-row {
  display: flex; align-items: center; justify-content: space-between;
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius); padding: 12px 18px; margin-bottom: 20px;
}
.switch-row { display: flex; align-items: center; gap: 10px; cursor: pointer; }
.switch-label { font-size: 14px; font-weight: 600; }
.switch-hint { font-size: 11px; color: var(--text-dim); }

.switch { position: relative; display: inline-block; width: 42px; height: 24px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; inset: 0; background: var(--card-border); border-radius: 999px; transition: 0.2s; }
.slider::before { content: ""; position: absolute; height: 18px; width: 18px; left: 3px; bottom: 3px; background: white; border-radius: 50%; transition: 0.2s; }
.switch input:checked + .slider { background: var(--accent); }
.switch input:checked + .slider::before { transform: translateX(18px); }

.lock-btn {
  background: var(--card-bg); border: 1px solid var(--card-border); color: var(--text);
  border-radius: var(--radius); padding: 8px 16px; cursor: pointer; font-weight: 600;
}
.lock-btn:hover { background: var(--hover-bg); }

.drop-zone {
  border: 2px dashed var(--card-border); border-radius: var(--radius);
  padding: 46px 20px; text-align: center; cursor: pointer;
  background: var(--card-bg); transition: border-color 0.15s ease, background 0.15s ease;
  margin-bottom: 18px;
}
.drop-zone.dragover { border-color: var(--accent); background: var(--hover-bg); }
.drop-zone-icon { font-size: 34px; color: var(--accent-strong); margin-bottom: 8px; }
.drop-zone-title { font-size: 17px; font-weight: 700; }
.drop-zone-sub { font-size: 13px; color: var(--text-dim); margin-top: 4px; }

.action-row { display: flex; gap: 10px; flex-wrap: wrap; margin-bottom: 18px; }
.pill-btn {
  background: var(--card-bg); border: 1px solid var(--card-border); color: var(--text);
  border-radius: var(--radius); padding: 9px 16px; font-size: 13px; font-weight: 600;
  cursor: pointer; transition: background 0.15s ease, transform 0.1s ease;
}
.pill-btn:hover { background: var(--hover-bg); transform: translateY(-1px); }

.selected-list {
  min-height: 90px; background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius); padding: 14px; margin-bottom: 18px;
  display: flex; flex-direction: column; gap: 8px;
}
.file-row { display: flex; align-items: center; justify-content: space-between; background: var(--input-bg); border-radius: 8px; padding: 8px 12px; font-size: 13px; }
.file-row .file-meta { color: var(--text-dim); font-size: 12px; margin-left: 8px; }
.file-row .remove-btn { background: transparent; border: none; color: var(--text-dim); cursor: pointer; font-size: 15px; }
.file-row .remove-btn:hover { color: var(--accent-strong); }

.send-controls { display: flex; gap: 12px; align-items: center; }
.send-to-select { flex: 1; background: var(--input-bg); border: 1px solid var(--card-border); border-radius: var(--radius); padding: 12px 14px; font-size: 14px; }
.send-now-btn {
  background: var(--accent); color: white; border: none; border-radius: var(--radius);
  padding: 13px 28px; font-size: 14px; font-weight: 700; letter-spacing: 0.5px;
  cursor: pointer; transition: filter 0.15s ease;
}
.send-now-btn:hover { filter: brightness(1.1); }

.card-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 14px; }
.device-card {
  background: var(--card-bg); border: 1px solid var(--card-border); border-radius: var(--radius);
  padding: 16px; display: flex; flex-direction: column; gap: 8px;
}
.device-card .device-avatar-lg {
  width: 44px; height: 44px; border-radius: 50%; background: var(--accent); color: white;
  display: flex; align-items: center; justify-content: center; font-weight: 700; font-size: 16px;
}

.history-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.history-table th { text-align: left; color: var(--text-dim); font-weight: 600; padding: 8px 10px; border-bottom: 1px solid var(--card-border); }
.history-table td { padding: 10px; border-bottom: 1px solid var(--card-border); }
.history-search { width: 100%; background: var(--input-bg); border: 1px solid var(--card-border); border-radius: var(--radius); padding: 10px 14px; margin-bottom: 16px; font-size: 13px; }

.settings-group { margin-bottom: 26px; }
.settings-group h3 { font-size: 14px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-dim); margin-bottom: 10px; }
.settings-row {
  display: flex; align-items: center; justify-content: space-between;
  background: var(--card-bg); border: 1px solid var(--card-border); border-radius: var(--radius);
  padding: 14px 18px; margin-bottom: 8px;
}
.settings-row .settings-label { font-size: 14px; font-weight: 600; }
.settings-row .settings-desc { font-size: 12px; color: var(--text-dim); margin-top: 2px; }
.settings-row select, .settings-row input[type=text] {
  background: var(--input-bg); border: 1px solid var(--card-border); border-radius: 8px;
  padding: 8px 12px; font-size: 13px;
}
.theme-swatch-row { display: flex; gap: 10px; margin-top: 10px; flex-wrap: wrap; }
.theme-swatch { width: 40px; height: 40px; border-radius: 10px; cursor: pointer; border: 2px solid transparent; }
.theme-swatch.selected { border-color: var(--accent-strong); }

.right-sidebar {
  background: var(--sidebar-bg); backdrop-filter: blur(18px);
  border-left: 1px solid var(--card-border);
  display: flex; flex-direction: column; padding: 22px 18px;
  overflow-y: auto; gap: 20px;
}
.right-section h2 { font-size: 15px; margin: 0 0 12px 0; font-weight: 700; }
.nearby-list, .chat-messages { display: flex; flex-direction: column; gap: 8px; }
.device-chip {
  display: flex; align-items: center; gap: 10px; background: var(--card-bg);
  border: 1px solid var(--card-border); border-radius: var(--radius); padding: 8px 10px; cursor: pointer;
}
.device-chip:hover { background: var(--hover-bg); }
.device-avatar { width: 32px; height: 32px; border-radius: 50%; background: var(--accent); color: white; display: flex; align-items: center; justify-content: center; font-size: 13px; font-weight: 700; flex-shrink: 0; }
.device-info { display: flex; flex-direction: column; line-height: 1.2; }
.device-info .device-name { font-size: 13px; font-weight: 600; }
.device-info .device-meta { font-size: 11px; color: var(--text-dim); }

.chat-section { flex: 1; display: flex; flex-direction: column; min-height: 0; }
.chat-messages { flex: 1; overflow-y: auto; padding-right: 2px; }
.chat-bubble { max-width: 85%; padding: 8px 12px; border-radius: 12px; font-size: 13px; line-height: 1.35; }
.chat-bubble.incoming { align-self: flex-start; background: var(--card-bg); border: 1px solid var(--card-border); border-bottom-left-radius: 2px; }
.chat-bubble.outgoing { align-self: flex-end; background: var(--accent); color: white; border-bottom-right-radius: 2px; }
.chat-bubble .chat-sender { display: block; font-size: 10px; opacity: 0.7; margin-bottom: 2px; }

.chat-input-row { display: flex; gap: 6px; margin-top: 10px; }
.chat-input-row input { flex: 1; background: var(--input-bg); border: 1px solid var(--card-border); border-radius: 999px; padding: 9px 14px; font-size: 13px; }
.chat-input-row button { background: var(--card-bg); border: 1px solid var(--card-border); color: var(--text); width: 36px; height: 36px; border-radius: 50%; cursor: pointer; flex-shrink: 0; }
.chat-input-row button:hover { background: var(--hover-bg); }

::-webkit-scrollbar { width: 8px; height: 8px; }
::-webkit-scrollbar-thumb { background: var(--card-border); border-radius: 999px; }
::-webkit-scrollbar-track { background: transparent; }

@media (max-width: 900px) {
  #app { grid-template-columns: 76px 1fr; }
  .right-sidebar { display: none; }
  .nav-labels { display: none; }
  .brand-name { display: none; }
}
`