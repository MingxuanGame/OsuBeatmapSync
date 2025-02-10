package utils

import (
	"os"
	"path/filepath"
	"runtime"
)

func XDGHome() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE")
	}
	return os.Getenv("HOME")
}

func XDGDataHome(app string) string {
	if runtime.GOOS == "windows" {
		roaming := os.Getenv("APPDATA")
		if roaming == "" {
			roaming = filepath.Join(XDGHome(), "AppData", "Roaming")
		}
		return filepath.Join(roaming, app)
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(XDGHome(), ".local", "share")
	}
	return filepath.Join(dataHome, app)
}
