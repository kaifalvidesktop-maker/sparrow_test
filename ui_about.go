package main

// aboutPartialHTML shows app version (pulled live from Go via
// goGetAppVersion), a short description, and placeholder links for
// support/donate/privacy policy matching the reference screenshots.
const aboutPartialHTML = `
<div class="card-grid" style="grid-template-columns:1fr;">
  <div class="device-card">
    <div class="brand-logo" style="align-self:center;">
      <svg viewBox="0 0 64 64" width="48" height="48" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M8 40c8-2 14-8 16-16 4 10 14 16 24 14-6 4-8 10-6 16-8-4-16-2-22 4 0-8-4-14-12-18z"
              stroke="currentColor" stroke-width="2.5" stroke-linejoin="round" fill="none"/>
      </svg>
    </div>
    <div style="text-align:center; font-size:20px; font-weight:700;">SPARROW</div>
    <div style="text-align:center; color:var(--text-dim); font-size:13px;">
      Version <span id="aboutVersion">-</span>
    </div>
    <p class="muted-note" style="text-align:center; margin-top:10px;">
      Fast, private, local-network file sharing. No accounts, no cloud,
      no sync — everything happens directly between devices on your Wi-Fi.
    </p>
  </div>
</div>

<div class="settings-group" style="margin-top:24px;">
  <h3>Other</h3>
  <div class="settings-row">
    <div class="settings-label">Support Sparrow</div>
    <button class="pill-btn" id="btnDonate">Donate</button>
  </div>
  <div class="settings-row">
    <div class="settings-label">Privacy Policy</div>
    <button class="pill-btn" id="btnPrivacy">Open</button>
  </div>
  <div class="settings-row">
    <div class="settings-label">Check for Updates</div>
    <button class="pill-btn" id="btnCheckUpdate">Check Now</button>
  </div>
</div>

<script>
(function () {
  if (window.goGetAppVersion) {
    window.goGetAppVersion().then(function (v) {
      document.getElementById("aboutVersion").textContent = v;
    });
  }
  document.getElementById("btnDonate").addEventListener("click", function () {
    alert("Donate link goes here once you provide one.");
  });
  document.getElementById("btnPrivacy").addEventListener("click", function () {
    alert("Privacy policy text/page goes here.");
  });
  document.getElementById("btnCheckUpdate").addEventListener("click", function () {
    alert("Update checker arrives with the System-features part.");
  });
})();
</script>
`