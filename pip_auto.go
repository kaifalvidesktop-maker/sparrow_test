package main

import (
	"os"
	"strings"
)

// Windows system media card (SMTC) off. Chromium honours only ONE
// --disable-features switch, so we MERGE into the one flashfix.go sets.
func init() {
	const add = "GlobalMediaControls,HardwareMediaKeyHandling,UseSMTC,MediaSessionService"
	const key = "WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS"
	const p = "--disable-features="
	cur := strings.TrimSpace(os.Getenv(key))
	if i := strings.Index(cur, p); i >= 0 {
		j := i + len(p)
		cur = cur[:j] + add + "," + cur[j:]
	} else {
		cur = strings.TrimSpace(cur + " " + p + add)
	}
	os.Setenv(key, cur)
}

// Page side of the independent video card.
// App loses focus + video playing -> frames are sent to the native card window.
// App gets focus again            -> card hides.
// Card buttons call window.__sparrowPipCmd(cmd, value) back on this page.
const pipAutoJS = `
(function () {
  if (window.__sparrowPip || typeof window.pipOpen !== 'function') return;
  window.__sparrowPip = true;
  var cur = null, timer = null, cv = null, cx = null;

  function pick() {
    var vs = document.querySelectorAll('video'), i;
    for (i = 0; i < vs.length; i++) if (!vs[i].paused && !vs[i].ended && vs[i].readyState > 2) return vs[i];
    return null;
  }
  function stop() {
    if (timer) { clearInterval(timer); timer = null; }
    cur = null;
  }
  function tick() {
    if (!cur || !cur.isConnected) { stop(); window.pipClose(); return; }
    try {
      var vw = cur.videoWidth || 16, vh = cur.videoHeight || 9, W = Math.min(640, vw);
      cv.width = W; cv.height = Math.max(1, Math.round(W * vh / vw));
      cx.drawImage(cur, 0, 0, cv.width, cv.height);
      window.pipFrame(cv.toDataURL('image/jpeg', 0.6), JSON.stringify({
        t: cur.currentTime, d: isFinite(cur.duration) ? cur.duration : 0,
        p: cur.paused, m: cur.muted, v: cur.volume, r: cur.playbackRate
      }));
    } catch (e) {
      var v = cur; stop(); window.pipClose();
      try { if (document.pictureInPictureEnabled && !document.pictureInPictureElement) v.requestPictureInPicture(); } catch (e2) {}
    }
  }
  function start() {
    if (timer) return;
    var v = pick(); if (!v) return;
    cur = v;
    cv = cv || document.createElement('canvas');
    cx = cx || cv.getContext('2d');
    window.pipOpen();
    timer = setInterval(tick, 60);
  }
  function hide() { if (timer) { stop(); window.pipClose(); } }

  window.__sparrowPipStop = stop;
  window.__sparrowPipCmd = function (c, val) {
    if (!cur) return;
    switch (c) {
      case 'toggle': cur.paused ? cur.play() : cur.pause(); break;
      case 'seek': if (isFinite(cur.duration)) cur.currentTime = cur.duration * val; break;
      case 'skip': cur.currentTime = Math.max(0, cur.currentTime + val); break;
      case 'mute': cur.muted = !cur.muted; break;
      case 'volume': cur.volume = val; cur.muted = val === 0; break;
      case 'rate': cur.playbackRate = val; break;
      case 'close': cur.pause(); stop(); break;
    }
  };

  window.addEventListener('blur', function () { setTimeout(function () { if (!document.hasFocus()) start(); }, 150); });
  window.addEventListener('focus', hide);
  document.addEventListener('visibilitychange', function () { document.hidden ? start() : hide(); });

  // keep the Windows media card empty
  function hideSmtc() { try { navigator.mediaSession.metadata = null; navigator.mediaSession.playbackState = 'none'; } catch (e) {} }
  hideSmtc();
  document.addEventListener('play', hideSmtc, true);
  document.addEventListener('playing', hideSmtc, true);
  setInterval(hideSmtc, 1000);
})();
`
