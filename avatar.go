package main

import webview "github.com/webview/webview_go"

func SetDeviceAvatar(path string) error {
	return globalConfig.Update(func(c *Config) { c.DeviceAvatar = path })
}

func GetDeviceAvatar() string {
	return globalConfig.Get().DeviceAvatar
}

// pickFilesNative opens a native file dialog to pick an image/file.
// If your project uses a different file picker implementation, ensure it matches this signature.
func pickFilesNative() ([]string, error) {
	// Native file picker implementation or placeholder returning paths
	return []string{}, nil
}

func bindAvatarAndGroupFunctions(w webview.WebView) {
	_ = w.Bind("goSetDeviceAvatar", func(path string) error {
		return SetDeviceAvatar(path)
	})
	_ = w.Bind("goGetDeviceAvatar", func() string {
		return GetDeviceAvatar()
	})
	_ = w.Bind("goPickAvatarImage", func() (string, error) {
		paths, err := pickFilesNative()
		if err != nil || len(paths) == 0 {
			return "", err
		}
		_ = SetDeviceAvatar(paths[0])
		return paths[0], nil
	})

	_ = w.Bind("goSendFileToDevices", func(filePath string, deviceIDs []string) ([]string, error) {
		ids, errs := SendFileToDevices(filePath, deviceIDs, state.turboMode)
		if len(errs) > 0 {
			return ids, errs[0]
		}
		return ids, nil
	})
	_ = w.Bind("goSendFolderToDevices", func(folderPath string, deviceIDs []string) ([]string, error) {
		ids, errs := SendFolderToDevices(folderPath, deviceIDs, state.turboMode)
		if len(errs) > 0 {
			return ids, errs[0]
		}
		return ids, nil
	})

	_ = w.Bind("goSendVoiceNote", func(base64Audio string, destIP string, destPort int) (string, error) {
		ts, err := SendVoiceNote(base64Audio, destIP, destPort, state.turboMode)
		if err != nil {
			return "", err
		}
		return ts.ID, nil
	})
	_ = w.Bind("goListVoiceNotes", func() []VoiceMessage {
		return globalVoiceInbox.List()
	})
}