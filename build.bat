@echo off
cd /d %~dp0
if exist build rmdir /s /q build
mkdir build

set CGO_ENABLED=0
set NAME=fixBadDisk

set GOOS=windows
set GOARCH=amd64
go build -ldflags="-w -s" -o build\%NAME%.exe

set GOARCH=386
go build -ldflags="-w -s" -o build\%NAME%-i386.exe

set GOOS=linux
set GOARCH=386
go build -ldflags="-w -s" -o build\%NAME%-linux-i386

set GOARCH=amd64
go build -ldflags="-w -s" -o build\%NAME%-linux

set GOARCH=arm
go build -ldflags="-w -s" -o build\%NAME%-linux-arm

set GOARCH=arm64
go build -ldflags="-w -s" -o build\%NAME%-linux-arm64

set GOARCH=mips
go build -ldflags="-w -s" -o build\%NAME%-linux-mips

set GOOS=darwin
set GOARCH=arm64
go build -ldflags="-w -s" -o build\%NAME%-darwin-arm64

set GOARCH=amd64
go build -ldflags="-w -s" -o build\%NAME%-darwin

echo Done
dir build
