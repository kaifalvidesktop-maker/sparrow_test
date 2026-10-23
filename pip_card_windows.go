//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"syscall"
	"unsafe"

	webview "github.com/webview/webview_go"
)

var (
	pcUser32   = syscall.NewLazyDLL("user32.dll")
	pcGdi32    = syscall.NewLazyDLL("gdi32.dll")
	pcKernel32 = syscall.NewLazyDLL("kernel32.dll")

	pcRegisterClass = pcUser32.NewProc("RegisterClassExW")
	pcCreateWindow  = pcUser32.NewProc("CreateWindowExW")
	pcDefWndProc    = pcUser32.NewProc("DefWindowProcW")
	pcShowWindow    = pcUser32.NewProc("ShowWindow")
	pcMoveWindow    = pcUser32.NewProc("MoveWindow")
	pcFindWindowEx  = pcUser32.NewProc("FindWindowExW")
	pcSysParams     = pcUser32.NewProc("SystemParametersInfoW")
	pcSetFG         = pcUser32.NewProc("SetForegroundWindow")
	pcReleaseCap    = pcUser32.NewProc("ReleaseCapture")
	pcSendMsg       = pcUser32.NewProc("SendMessageW")
	pcLoadCursor    = pcUser32.NewProc("LoadCursorW")
	pcSetWindowPos  = pcUser32.NewProc("SetWindowPos")
	pcStockObject   = pcGdi32.NewProc("GetStockObject")
	pcModuleHandle  = pcKernel32.NewProc("GetModuleHandleW")
)

type pcWndClass struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

var (
	pcOuter   uintptr
	pcMain    uintptr
	pcCard    webview.WebView
	pcMainWV  webview.WebView
	pcBusy    int32
	pcTopmost = ^uintptr(0) // HWND_TOPMOST (-1)
)

func pcWndProc(hwnd, msg, wp, lp uintptr) uintptr {
	switch msg {
	case 0x0005: // WM_SIZE
		child, _, _ := pcFindWindowEx.Call(hwnd, 0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("webview_widget"))), 0)
		if child != 0 {
			pcMoveWindow.Call(child, 0, 0, lp&0xFFFF, (lp>>16)&0xFFFF, 1)
		}
		return 0
	case 0x0010: // WM_CLOSE -> hide, stop video loop
		pcShowWindow.Call(hwnd, 0)
		pcStopMain(false)
		return 0
	case 0x0024: // WM_GETMINMAXINFO: min 240x150
		*(*int32)(unsafe.Pointer(lp + 24)) = 240
		*(*int32)(unsafe.Pointer(lp + 28)) = 150
		return 0
	}
	r, _, _ := pcDefWndProc.Call(hwnd, msg, wp, lp)
	return r
}

func pcStopMain(pause bool) {
	if pcMainWV == nil {
		return
	}
	pcMainWV.Dispatch(func() {
		js := "window.__sparrowPipStop&&window.__sparrowPipStop();"
		if pause {
			js = "window.__sparrowPipCmd&&window.__sparrowPipCmd('close',0);" + js
		}
		pcMainWV.Eval(js)
	})
}

// must run on the UI thread (inside Dispatch)
func pcEnsure() {
	if pcOuter != 0 {
		return
	}
	inst, _, _ := pcModuleHandle.Call(0)
	cls := syscall.StringToUTF16Ptr("SparrowVideoCard")
	cur, _, _ := pcLoadCursor.Call(0, 32512)
	bg, _, _ := pcStockObject.Call(4) // BLACK_BRUSH
	wc := pcWndClass{
		WndProc:    syscall.NewCallback(pcWndProc),
		Instance:   inst,
		Cursor:     cur,
		Background: bg,
		ClassName:  cls,
	}
	wc.Size = uint32(unsafe.Sizeof(wc))
	pcRegisterClass.Call(uintptr(unsafe.Pointer(&wc)))

	var wa [4]int32
	pcSysParams.Call(0x30, 0, uintptr(unsafe.Pointer(&wa[0])), 0) // SPI_GETWORKAREA
	cw, ch := int32(400), int32(260)
	x, y := wa[2]-cw-16, wa[3]-ch-16

	// WS_EX_TOPMOST|WS_EX_TOOLWINDOW ; WS_POPUP|WS_THICKFRAME|WS_CLIPCHILDREN
	pcOuter, _, _ = pcCreateWindow.Call(
		0x88, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Sparrow Video"))),
		0x80000000|0x40000|0x2000000,
		uintptr(x), uintptr(y), uintptr(cw), uintptr(ch), 0, 0, inst, 0)
	if pcOuter == 0 {
		return
	}

	pcCard = webview.NewWindow(false, unsafe.Pointer(pcOuter))
	pcCard.Bind("pipCmd", func(cmd string, val float64) {
		switch cmd {
		case "back":
			pcMainWV.Dispatch(func() {
				pcShowWindow.Call(pcOuter, 0)
				pcShowWindow.Call(pcMain, 9) // SW_RESTORE
				pcSetFG.Call(pcMain)
			})
			pcStopMain(false)
		case "close":
			pcMainWV.Dispatch(func() { pcShowWindow.Call(pcOuter, 0) })
			pcStopMain(true)
		default:
			pcMainWV.Dispatch(func() {
				pcMainWV.Eval(fmt.Sprintf("window.__sparrowPipCmd&&window.__sparrowPipCmd(%q,%v);", cmd, val))
			})
		}
	})
	pcCard.Bind("pipDrag", func() {
		pcMainWV.Dispatch(func() {
			pcReleaseCap.Call()
			pcSendMsg.Call(pcOuter, 0x00A1, 2, 0) // WM_NCLBUTTONDOWN, HTCAPTION
		})
	})
	pcCard.SetHtml(pcHTML)
}

func setupPipCard(w webview.WebView) {
	pcMainWV = w
	pcMain = uintptr(w.Window())

	w.Bind("pipOpen", func() {
		w.Dispatch(func() {
			pcEnsure()
			if pcOuter == 0 {
				return
			}
			pcSetWindowPos.Call(pcOuter, pcTopmost, 0, 0, 0, 0, 0x0001|0x0002|0x0010|0x0040) // NOSIZE|NOMOVE|NOACTIVATE|SHOWWINDOW
			pcShowWindow.Call(pcOuter, 4)                                                      // SW_SHOWNOACTIVATE
		})
	})
	w.Bind("pipClose", func() {
		w.Dispatch(func() {
			if pcOuter != 0 {
				pcShowWindow.Call(pcOuter, 0)
			}
		})
	})
	w.Bind("pipFrame", func(img string, state string) {
		if pcCard == nil || !atomic.CompareAndSwapInt32(&pcBusy, 0, 1) {
			return // drop frame while previous one is still painting
		}
		ib, _ := json.Marshal(img)
		sb, _ := json.Marshal(state)
		w.Dispatch(func() {
			pcCard.Eval("window.__f(" + string(ib) + "," + string(sb) + ");")
			atomic.StoreInt32(&pcBusy, 0)
		})
	})
}

const pcHTML = `<!doctype html><html><head><meta charset="utf-8"><style>
html,body{margin:0;height:100%;background:#000;overflow:hidden;font-family:Segoe UI,Arial,sans-serif;color:#fff;user-select:none}
#s{position:relative;width:100%;height:100%}
#f{width:100%;height:100%;object-fit:contain;background:#000}
.bar{position:absolute;left:0;right:0;opacity:0;transition:opacity .2s;display:flex;align-items:center;gap:6px;padding:8px;box-sizing:border-box}
#s:hover .bar{opacity:1}
#top{top:0;justify-content:space-between;background:linear-gradient(#000b,#0000)}
#bot{bottom:0;flex-wrap:wrap;background:linear-gradient(#0000,#000d)}
#drag{flex:1;height:30px;cursor:move}
button{background:#fff2;border:0;color:#fff;border-radius:6px;min-width:30px;height:30px;font-size:14px;cursor:pointer;padding:0 8px}
button:hover{background:#fff5}
input[type=range]{accent-color:#fff;cursor:pointer}
#seek{flex:1 1 100%}#vol{width:70px}
#time{font-size:12px;margin-left:auto;white-space:nowrap}
</style></head><body><div id="s">
<img id="f" draggable="false">
<div class="bar" id="top"><button onclick="pipCmd('back',0)">&#8599; Back to tab</button><div id="drag"></div><button onclick="pipCmd('close',0)">&#10005;</button></div>
<div class="bar" id="bot">
<input type="range" id="seek" min="0" max="1000" value="0">
<button onclick="pipCmd('skip',-10)">&#8634;10</button>
<button id="pp" onclick="pipCmd('toggle',0)">&#10074;&#10074;</button>
<button onclick="pipCmd('skip',10)">10&#8635;</button>
<button id="mu" onclick="pipCmd('mute',0)">&#128266;</button>
<input type="range" id="vol" min="0" max="1" step="0.05" value="1">
<button id="sp" onclick="spd()">1x</button>
<span id="time">0:00 / 0:00</span>
</div></div>
<script>
var $=function(i){return document.getElementById(i)},rates=[0.5,0.75,1,1.25,1.5,2],rate=1,drag=false;
function fmt(t){t=t||0;var h=Math.floor(t/3600),m=Math.floor(t%3600/60),s=Math.floor(t%60);return(h?h+':'+(m<10?'0':''):'')+m+':'+(s<10?'0':'')+s}
function spd(){pipCmd('rate',rates[(rates.indexOf(rate)+1)%rates.length])}
$('drag').onmousedown=function(){pipDrag()};
$('seek').onmousedown=function(){drag=true};
$('seek').onmouseup=function(){drag=false};
$('seek').oninput=function(){pipCmd('seek',this.value/1000)};
$('vol').oninput=function(){pipCmd('volume',parseFloat(this.value))};
window.__f=function(img,st){
  $('f').src=img;var s=JSON.parse(st);rate=s.r;
  $('pp').innerHTML=s.p?'&#9654;':'&#10074;&#10074;';
  $('mu').innerHTML=(s.m||s.v===0)?'&#128263;':'&#128266;';
  $('sp').textContent=s.r+'x';
  if(!drag&&s.d>0)$('seek').value=Math.round(s.t/s.d*1000);
  $('time').textContent=fmt(s.t)+' / '+fmt(s.d);
};
</script></body></html>`
