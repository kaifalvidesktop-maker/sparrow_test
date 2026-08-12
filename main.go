package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	webview "github.com/webview/webview_go"
)

type appState struct {
	deviceName       string
	appVersion       string
	turboMode        bool
	themeMode        string
	locked           bool
	pinCode          string
	downloadDir      string
	deviceAvatar     string
	deviceID         string
	discovery        *DiscoveryService
	transferReceiver *TransferReceiver
}

var state *appState

func main() {
	runtime.LockOSThread()

	InitConfig()
	InitBackground()
	InitLaunchArgs()
	InitFavorites()
	InitNicknames()
	InitBandwidthStats()

	state = &appState{
		deviceName:  defaultDeviceName(),
		appVersion:  "0.1.0",
		turboMode:   true,
		themeMode:   GetWindowsThemePreference(),
		locked:      false,
		pinCode:     "",
		downloadDir: defaultDownloadDir(),
	}

	state.deviceID = generateDeviceID()

	state.transferReceiver = NewTransferReceiver(globalTransferManager)
	state.transferReceiver.requirePin = func() bool {
		return globalConfig.Get().RequirePinReceive
	}
	if err := state.transferReceiver.Start(); err != nil {
		log.Fatalf("Sparrow failed to start transfer receiver: %v", err)
	}
	defer state.transferReceiver.Stop()

	state.discovery = NewDiscoveryService(globalDeviceRegistry, state.deviceID)
	if err := state.discovery.Start(); err != nil {
		log.Fatalf("Sparrow failed to start discovery: %v", err)
	}
	defer state.discovery.Stop()

	log.Println("Sparrow starting...")
	log.Println("Device name:", state.deviceName)
	log.Println("Download dir:", state.downloadDir)

	uiSrv, err := StartUIServer()
	if err != nil {
		log.Fatalf("Sparrow failed to start UI server: %v", err)
	}
	defer uiSrv.Shutdown()

	log.Println("UI server running at:", uiSrv.URL())

	lanSrv, err := StartLANServer()
	if err != nil {
		log.Fatalf("Sparrow failed to start LAN server: %v", err)
	}
	defer lanSrv.Shutdown()
	log.Println("LAN server running at:", fmt.Sprintf("http://%s:%d", getLocalIP(), LANPort))

	w := webview.New(true)
	defer w.Destroy()

	w.SetTitle("Sparrow")
	w.SetSize(1180, 760, webview.HintNone)
	w.SetSize(900, 600, webview.HintMin)

	bindBackendFunctions(w)
	bindTransferFunctions(w)
	bindAuthFunctions(w)
	bindExtraFunctions(w)
	bindAvatarAndGroupFunctions(w)
	bindDiagnosticsFunctions(w)
	bindStatsFunctions(w)
	bindFinalFunctions(w)
	
	StartVoiceNoteWatcher()
	StartTransferNotifications()
	StartSoundAlerts()
	StartBandwidthTracking()

	w.Navigate(uiSrv.URL())
	w.Run()

	log.Println("Sparrow closed.")
}

func bindBackendFunctions(w webview.WebView) {
	_ = w.Bind("goGetDeviceName", func() string {
		return state.deviceName
	})

	_ = w.Bind("goSetDeviceName", func(name string) bool {
		if name == "" {
			return false
		}
		state.deviceName = name
		log.Println("Device name changed to:", name)
		return true
	})

	_ = w.Bind("goGetAppVersion", func() string {
		return state.appVersion
	})

	_ = w.Bind("goGetTurboMode", func() bool {
		return state.turboMode
	})

	_ = w.Bind("goSetTurboMode", func(on bool) bool {
		state.turboMode = on
		log.Println("Turbo mode set to:", on)
		return true
	})

	_ = w.Bind("goGetThemeMode", func() string {
		return state.themeMode
	})

	_ = w.Bind("goSetThemeMode", func(mode string) bool {
		if mode != "dark" && mode != "light" {
			return false
		}
		state.themeMode = mode
		return true
	})

	_ = w.Bind("goGetDownloadDir", func() string {
		return state.downloadDir
	})

	_ = w.Bind("goIsLocked", func() bool {
		return state.locked
	})

	_ = w.Bind("goLockApp", func() bool {
		state.locked = true
		return true
	})

	_ = w.Bind("goUnlockApp", func(pin string) bool {
		if state.pinCode == "" || pin == state.pinCode {
			state.locked = false
			return true
		}
		return false
	})

	_ = w.Bind("goSetPin", func(pin string) bool {
		state.pinCode = pin
		return true
	})

	_ = w.Bind("goGetLocalIP", func() string {
		return getLocalIP()
	})

	_ = w.Bind("goGetLanURL", func() string {
		return fmt.Sprintf("http://%s:%d", getLocalIP(), LANPort)
	})

	_ = w.Bind("goGetQRCodeBase64", func() (string, error) {
		url := fmt.Sprintf("http://%s:%d", getLocalIP(), LANPort)
		return generateQRCodeBase64(url, 260)
	})

	_ = w.Bind("goListDevices", func() []DeviceInfo {
		return globalDeviceRegistry.List()
	})
}

func bindTransferFunctions(w webview.WebView) {
	_ = w.Bind("goSendFile", func(filePath string, destIP string, destPort int) (string, error) {
		ts, err := SendFile(filePath, destIP, destPort, state.turboMode)
		if err != nil {
			return "", err
		}
		SetTransferPath(ts.ID, filePath)
		return ts.ID, nil
	})

	_ = w.Bind("goListTransfers", func() []TransferSnapshot {
		return globalTransferManager.List()
	})

	_ = w.Bind("goPauseTransfer", func(id string) error {
		return globalTransferManager.Pause(id)
	})

	_ = w.Bind("goResumeTransfer", func(id string) error {
		return globalTransferManager.Resume(id)
	})

	_ = w.Bind("goCancelTransfer", func(id string) error {
		return globalTransferManager.Cancel(id)
	})
}

func bindExtraFunctions(w webview.WebView) {
	// we can bind additional functions here as needed
}

func defaultDeviceName() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "Sparrow Device"
	}
	return host
}

func defaultDownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	dir := home + string(os.PathSeparator) + "Downloads" + string(os.PathSeparator) + "Sparrow"
	_ = os.MkdirAll(dir, 0755)
	return dir
}