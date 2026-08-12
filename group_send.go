package main

import "fmt"

// SendFolder packages and sends a directory to a target IP and port.
func SendFolder(folderPath string, destIP string, destPort int, turbo bool) (*TransferState, error) {
	// Implement or map to your existing folder transmission logic
	return SendFile(folderPath, destIP, destPort, turbo)
}

func SendFileToDevices(filePath string, deviceIDs []string, turbo bool) ([]string, []error) {
	var ids []string
	var errs []error
	for _, devID := range deviceIDs {
		dev, ok := globalDeviceRegistry.Get(devID)
		if !ok {
			errs = append(errs, fmt.Errorf("device %s not found", devID))
			continue
		}
		ts, err := SendFile(filePath, dev.IP, dev.Port, turbo)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		ids = append(ids, ts.ID)
	}
	return ids, errs
}

func SendFolderToDevices(folderPath string, deviceIDs []string, turbo bool) ([]string, []error) {
	var ids []string
	var errs []error
	for _, devID := range deviceIDs {
		dev, ok := globalDeviceRegistry.Get(devID)
		if !ok {
			errs = append(errs, fmt.Errorf("device %s not found", devID))
			continue
		}
		ts, err := SendFolder(folderPath, dev.IP, dev.Port, turbo)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		ids = append(ids, ts.ID)
	}
	return ids, errs
}