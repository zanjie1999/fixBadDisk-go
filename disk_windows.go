//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

func init() {
	// 启用 Windows 控制台 ANSI 转义码支持
	setConsoleMode := kernel32.NewProc("SetConsoleMode")
	getStdHandle := kernel32.NewProc("GetStdHandle")
	const stdOutputHandle = ^uintptr(10) // -11
	handle, _, _ := getStdHandle.Call(stdOutputHandle)
	if handle != 0 {
		const enableVirtualTerminalProcessing = 0x0004
		var mode uint32
		getConsoleMode := kernel32.NewProc("GetConsoleMode")
		getConsoleMode.Call(handle, uintptr(unsafe.Pointer(&mode)))
		setConsoleMode.Call(handle, uintptr(mode|enableVirtualTerminalProcessing))
	}
}

func getFreeSpaceMBOS(folder string) float64 {
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
