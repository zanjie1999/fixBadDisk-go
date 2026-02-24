//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

func getFreeSpaceMBOS(folder string) float64 {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")
	var freeBytes int64
	ptr, _ := syscall.UTF16PtrFromString(folder)
	getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(ptr)),
		uintptr(unsafe.Pointer(&freeBytes)),
		0, 0,
	)
	return float64(freeBytes) / 1024 / 1024
}
