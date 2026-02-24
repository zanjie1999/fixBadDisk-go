cd `dirname $0`
rm -rf build
mkdir build

export CGO_ENABLED=0
NAME=fixBadDisk

export GOOS=windows
export GOARCH=amd64
go build -ldflags="-w -s" -o build/$NAME.exe

export GOARCH=386
go build -ldflags="-w -s" -o build/$NAME-i386.exe

export GOOS=linux
export GOARCH=386
go build -ldflags="-w -s" -o build/$NAME-linux-i386

export GOARCH=amd64
go build -ldflags="-w -s" -o build/$NAME-linux

export GOARCH=arm
go build -ldflags="-w -s" -o build/$NAME-linux-arm

export GOARCH=arm64
go build -ldflags="-w -s" -o build/$NAME-linux-arm64

export GOARCH=mips
go build -ldflags="-w -s" -o build/$NAME-linux-mips

export GOOS=darwin
export GOARCH=arm64
go build -ldflags="-w -s" -o build/$NAME-darwin-arm64

export GOARCH=amd64
go build -ldflags="-w -s" -o build/$NAME-darwin

echo "Done"
ls -lh build/
