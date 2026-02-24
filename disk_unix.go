//go:build !windows

package main

import "syscall"

func getFreeSpaceMBOS(folder string) float64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(folder, &stat); err != nil {
		return 0
	}
	return float64(stat.Bavail) * float64(stat.Bsize) / 1024 / 1024
}
