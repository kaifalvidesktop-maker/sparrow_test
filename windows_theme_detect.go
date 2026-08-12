package main

import (
	"syscall"
	"unsafe"
)

var (
	advapi32             = syscall.NewLazyDLL("advapi32.dll")
	procRegOpenKeyExW    = advapi32.NewProc("RegOpenKeyExW")
	procRegCloseKey      = advapi32.NewProc("RegCloseKey")
	procRegQueryValueExW = advapi32.NewProc("RegQueryValueExW")
)

const (
	hkeyCurrentUser = 0x80000001
	keyRead         = 0x20019
)

const themeRegPath = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`
const themeRegValue = "AppsUseLightTheme"

func GetWindowsThemePreference() string {
	var hKey syscall.Handle
	pathPtr, err := syscall.UTF16PtrFromString(themeRegPath)
	if err != nil {
		return "dark"
	}

	ret, _, _ := procRegOpenKeyExW.Call(
		uintptr(hkeyCurrentUser),
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		uintptr(keyRead),
		uintptr(unsafe.Pointer(&hKey)),
	)
	if ret != 0 {
		return "dark"
	}
	defer procRegCloseKey.Call(uintptr(hKey))

	valueNamePtr, err := syscall.UTF16PtrFromString(themeRegValue)
	if err != nil {
		return "dark"
	}

	var dataType uint32
	var data uint32
	var dataLen uint32 = 4

	ret, _, _ = procRegQueryValueExW.Call(
		uintptr(hKey),
		uintptr(unsafe.Pointer(valueNamePtr)),
		0,
		uintptr(unsafe.Pointer(&dataType)),
		uintptr(unsafe.Pointer(&data)),
		uintptr(unsafe.Pointer(&dataLen)),
	)
	if ret != 0 {
		return "dark"
	}

	if data == 1 {
		return "light"
	}
	return "dark"
}