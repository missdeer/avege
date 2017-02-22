#!/bin/bash

function try () {
"$@" #|| exit -1
}

[ -z "$ANDROID_NDK_HOME" ] && ANDROID_NDK_HOME=$HOME/android-ndk-r13b

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
DEPS=$DIR/.deps
ANDROID_ARM_TOOLCHAIN=$DEPS/android-toolchain-16-arm
ANDROID_X86_TOOLCHAIN=$DEPS/android-toolchain-16-386

ANDROID_ARM_CC=$ANDROID_ARM_TOOLCHAIN/bin/arm-linux-androideabi-gcc
ANDROID_ARM_STRIP=$ANDROID_ARM_TOOLCHAIN/bin/arm-linux-androideabi-strip

ANDROID_X86_CC=$ANDROID_X86_TOOLCHAIN/bin/i686-linux-android-gcc
ANDROID_X86_STRIP=$ANDROID_X86_TOOLCHAIN/bin/i686-linux-android-strip


if [ ! -d "$DEPS" ]; then
    mkdir -p $DEPS 
fi

if [ ! -d "$ANDROID_ARM_TOOLCHAIN" ]; then
    echo "Make standalone toolchain for ARM arch"
    $ANDROID_NDK_HOME/build/tools/make_standalone_toolchain.py --arch arm \
        --api 16 --install-dir $ANDROID_ARM_TOOLCHAIN
fi

if [ ! -d "$ANDROID_X86_TOOLCHAIN" ]; then
    echo "Make standalone toolchain for X86 arch"
    $ANDROID_NDK_HOME/build/tools/make_standalone_toolchain.py --arch 386 \
        --api 16 --install-dir $ANDROID_X86_TOOLCHAIN
fi

export GOPATH=$GOPATH:$PWD/../..

basename=${PWD##*/}

echo "Cross compile $basename for Android armv7"
try env CGO_ENABLED=1 CC=$ANDROID_ARM_CC GOOS=android GOARCH=arm GOARM=7 go build -ldflags="-s -w" && tar czvf $basename-android-armv7.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.android.armv7

echo "Cross compile $basename for Android 386"
try env CGO_ENABLED=1 CC=$ANDROID_X86_CC GOOS=android GOARCH=386 go build -ldflags="-s -w"  && tar czvf $basename-android-386.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.android.386

echo "Cross compile $basename for Linux (Netgear R6300v2 with Merlin firmware) armv5"
try env GOOS=linux GOARCH=arm GOARM=5 go build -ldflags="-s -w"  && tar czvf $basename-linux-armv5.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.rpi1

echo "Cross compile $basename for Linux (Raspberry Pi 1) armv6"
try env GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w"  && tar czvf $basename-linux-armv6.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.rpi1

echo "Cross compile $basename for Linux (Raspberry Pi 2/3) armv7"
try env GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w"  && tar czvf $basename-linux-armv7.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.rpi2

echo "Cross compile $basename for Linux AMD64"
try env GOOS=linux GOARCH=amd64 go build -ldflags="-s -w"  && tar czvf $basename-linux-amd64.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.linux.amd64

echo "Cross compile $basename for Linux ARM64"
try env GOOS=linux GOARCH=arm64 go build -ldflags="-s -w"  && tar czvf $basename-linux-arm64.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.linux.arm64

echo "Cross compile $basename for Linux 386"
try env GOOS=linux GOARCH=386 go build -ldflags="-s -w"  && tar czvf $basename-linux-386.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.linux.386

echo "Cross compile $basename for Linux PPC64"
try env GOOS=linux GOARCH=ppc64 go build -ldflags="-s -w"  && tar czvf $basename-linux-ppc64.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.linux.ppc64

echo "Cross compile $basename for Linux PPC64LE"
try env GOOS=linux GOARCH=ppc64le go build -ldflags="-s -w"  && tar czvf $basename-linux-ppc64le.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.linux.ppc64le

echo "Cross compile $basename for Linux MIPS64"
try env GOOS=linux GOARCH=mips64 go build -ldflags="-s -w"  && tar czvf $basename-linux-mips64.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.linux.mips64

echo "Cross compile $basename for Linux MIPS64LE"
try env GOOS=linux GOARCH=mips64le go build -ldflags="-s -w"  && tar czvf $basename-linux-mips64le.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.linux.mips64le

echo "Cross compile $basename for Windows AMD64"
try env GOOS=windows GOARCH=amd64 go build -ldflags="-s -w"  && tar czvf $basename-win-amd64.tar.gz $basename.exe conf resources templates config-sample.json && mv $basename.exe $basename.win.amd64.exe

echo "Cross compile $basename for Windows 386"
try env GOOS=windows GOARCH=386 go build -ldflags="-s -w"  && tar czvf $basename-win-386.tar.gz $basename.exe conf resources templates config-sample.json && mv $basename.exe $basename.win.386.exe

echo "Cross compile $basename for Darwin AMD64"
try env GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w"  && tar czvf $basename-darwin-amd64.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.darwin.amd64

echo "Cross compile $basename for FreeBSD AMD64"
try env GOOS=freebsd GOARCH=amd64 go build -ldflags="-s -w"  && tar czvf $basename-freebsd-amd64.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.freebsd.amd64

echo "Cross compile $basename for FreeBSD 386"
try env GOOS=freebsd GOARCH=386 go build -ldflags="-s -w"  && tar czvf $basename-freebsd-386.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.freebsd.386

echo "Cross compile $basename for FreeBSD ARM"
try env GOOS=freebsd GOARCH=arm go build -ldflags="-s -w"  && tar czvf $basename-freebsd-arm.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.freebsd.arm

echo "Cross compile $basename for NetBSD AMD64"
try env GOOS=netbsd GOARCH=amd64 go build -ldflags="-s -w"  && tar czvf $basename-netbsd-amd64.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.netbsd.amd64

echo "Cross compile $basename for NetBSD 386"
try env GOOS=netbsd GOARCH=386 go build -ldflags="-s -w"  && tar czvf $basename-netbsd-386.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.netbsd.386

echo "Cross compile $basename for NetBSD ARM"
try env GOOS=netbsd GOARCH=arm go build -ldflags="-s -w"  && tar czvf $basename-netbsd-arm.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.netbsd.arm

echo "Cross compile $basename for OpenBSD AMD64"
try env GOOS=openbsd GOARCH=amd64 go build -ldflags="-s -w"  && tar czvf $basename-openbsd-amd64.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.openbsd.amd64

echo "Cross compile $basename for OpenBSD 386"
try env GOOS=openbsd GOARCH=386 go build -ldflags="-s -w"  && tar czvf $basename-openbsd-386.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.openbsd.386

echo "Cross compile $basename for OpenBSD ARM"
try env GOOS=openbsd GOARCH=arm go build -ldflags="-s -w"  && tar czvf $basename-openbsd-arm.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.openbsd.arm

echo "Cross compile $basename for DragonflyBSD AMD64"
try env GOOS=dragonfly GOARCH=amd64 go build -ldflags="-s -w"  && tar czvf $basename-dragonflybsd-amd64.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.dragonflybsd

echo "Cross compile $basename for Solaris AMD64"
try env GOOS=solaris GOARCH=amd64 go build -ldflags="-s -w"  && tar czvf $basename-solaris.tar.gz $basename conf resources templates config-sample.json && mv $basename $basename.solaris

