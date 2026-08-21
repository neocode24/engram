#!/usr/bin/env bash
# engram-voice 배포 아카이브를 만든다. ADR 0081.
#
# 이 바이너리는 단일 바이너리가 아니다. sherpa-onnx Go 바인딩이 동적
# 라이브러리로 링크되고 모듈에 정적 라이브러리가 없다. 그래서 아카이브에
# 바이너리와 lib/ 둘이 들어간다. engram 본체와 다른 점이다.
#
# 그냥 빌드하면 rpath 에 모듈 캐시 절대 경로가 박힌다. 빌드한 기계에서는
# 돌지만 배포하면 그 경로가 없고, 경로에 빌더의 홈 디렉토리가 들어 있어
# 공개 경계 문제이기도 하다. 그래서 상대 rpath 를 더하고 포장할 때 모듈
# 캐시 경로를 지운다.
#
# 사용법
#   voice/scripts/package.sh <버전> [출력 디렉토리]
#
# 대상은 호스트다. CGO 는 교차 컴파일이 안 되므로 대상마다 그 플랫폼의
# 러너가 필요하다. 릴리스 워크플로가 러너를 나누는 이유다.
set -euo pipefail

VERSION="${1:?버전을 주세요. 예: 1.0.0}"
OUTDIR="${2:-dist}"

cd "$(dirname "$0")/.."

GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"

# 교차 빌드를 막는다. CGO 라 링크 단계에서 어차피 실패하는데 그 오류가
# 원인을 가리키지 않는다.
if [ "$GOOS" != "$(GOOS= go env GOHOSTOS)" ] || [ "$GOARCH" != "$(GOARCH= go env GOHOSTARCH)" ]; then
	echo "교차 빌드는 되지 않습니다. 대상 플랫폼의 기계에서 실행하세요" >&2
	exit 1
fi

# sherpa-onnx 가 라이브러리를 두는 대상 트리플이다. 모듈 안의 디렉토리
# 이름이며 우리가 정하는 값이 아니다.
case "$GOOS/$GOARCH" in
	darwin/arm64)  PLATFORM=macos   TRIPLE=aarch64-apple-darwin ;;
	darwin/amd64)  PLATFORM=macos   TRIPLE=x86_64-apple-darwin ;;
	linux/amd64)   PLATFORM=linux   TRIPLE=x86_64-unknown-linux-gnu ;;
	linux/arm64)   PLATFORM=linux   TRIPLE=aarch64-unknown-linux-gnu ;;
	windows/amd64) PLATFORM=windows TRIPLE=x86_64-pc-windows-gnu ;;
	*) echo "지원하지 않는 대상입니다: $GOOS/$GOARCH" >&2; exit 1 ;;
esac

# 모듈을 먼저 받는다. go list -m 은 내려받지 않으므로 캐시에 없으면
# Dir 이 빈 문자열이 되고 LIBSRC 가 /lib/<트리플> 이 된다. CI 에서는
# 앞선 빌드가 받아 둬서 안 드러났고 릴리스에서 처음 터졌다.
go mod download "github.com/k2-fsa/sherpa-onnx-go-$PLATFORM"

MODDIR="$(go list -m -f '{{.Dir}}' "github.com/k2-fsa/sherpa-onnx-go-$PLATFORM")"
[ -n "$MODDIR" ] || { echo "모듈 디렉토리를 찾을 수 없습니다" >&2; exit 1; }
LIBSRC="$MODDIR/lib/$TRIPLE"
[ -d "$LIBSRC" ] || { echo "라이브러리를 찾을 수 없습니다: $LIBSRC" >&2; exit 1; }

NAME="engram-voice_${VERSION}_${GOOS}_${GOARCH}"
STAGE="$OUTDIR/$NAME"
rm -rf "$STAGE"
mkdir -p "$STAGE/lib"

BIN="$STAGE/engram-voice"
[ "$GOOS" = windows ] && BIN="$STAGE/engram-voice.exe"

# 상대 rpath 를 더한다. windows 는 rpath 개념이 없고 exe 옆의 DLL 을
# 먼저 보므로 아무것도 더하지 않는다.
EXTLD=""
case "$GOOS" in
	darwin) EXTLD="-Wl,-rpath,@executable_path/lib" ;;
	linux)  EXTLD='-Wl,-rpath,$ORIGIN/lib' ;;
esac

echo "빌드 $GOOS/$GOARCH"
CGO_ENABLED=1 go build -trimpath \
	-ldflags "-X main.version=$VERSION ${EXTLD:+-extldflags '$EXTLD'}" \
	-o "$BIN" ./cmd/engram-voice

# windows 는 DLL 을 exe 옆에 둔다. lib/ 를 봐 주지 않는다.
if [ "$GOOS" = windows ]; then
	rmdir "$STAGE/lib"
	cp "$LIBSRC"/*.dll "$STAGE/"
else
	cp "$LIBSRC"/*.dylib "$STAGE/lib/" 2>/dev/null || cp "$LIBSRC"/*.so* "$STAGE/lib/"
fi

# 모듈 캐시 경로를 지운다. 남기면 빌더의 홈 경로가 공개 산출물에 들어간다.
case "$GOOS" in
	darwin)
		install_name_tool -delete_rpath "$LIBSRC" "$BIN"
		# install_name_tool 이 서명을 깨뜨리므로 다시 서명한다. 깨진
		# 서명이면 macOS 가 SIGKILL 로 죽인다.
		codesign --force --sign - "$BIN"
		;;
	linux)
		# patchelf 가 없으면 모듈 캐시 경로가 남는다. 기능은 돌지만
		# 빌더 경로가 새므로 조용히 넘어가지 않는다.
		command -v patchelf >/dev/null || { echo "patchelf 가 필요합니다" >&2; exit 1; }
		patchelf --set-rpath '$ORIGIN/lib' "$BIN"
		;;
esac

cp ../README.md ../LICENSE "$STAGE/"
chmod 644 "$STAGE"/README.md "$STAGE"/LICENSE

echo "포장 $NAME"
if [ "$GOOS" = windows ]; then
	# zip 이 Windows 에 늘 있지는 않다. 7z 를 거쳐 PowerShell 로 내려간다.
	#
	# PowerShell 에 넘길 때는 경로를 Windows 형식으로 바꿔야 한다. bash
	# 쪽 경로가 /d/a/... 인데 PowerShell 이 그것을 \d\a\... 로 읽고
	# 못 찾는다. 실측으로 릴리스가 여기서 터졌다.
	if command -v zip >/dev/null; then
		(cd "$OUTDIR" && zip -qr "$NAME.zip" "$NAME")
	elif command -v 7z >/dev/null; then
		(cd "$OUTDIR" && 7z a -tzip -bso0 -bsp0 "$NAME.zip" "$NAME")
	else
		src="$OUTDIR/$NAME"; dst="$OUTDIR/$NAME.zip"
		if command -v cygpath >/dev/null; then
			src="$(cygpath -w "$src")"; dst="$(cygpath -w "$dst")"
		fi
		powershell -NoProfile -Command \
			"Compress-Archive -Path '$src' -DestinationPath '$dst' -Force"
	fi
else
	tar -C "$OUTDIR" -czf "$OUTDIR/$NAME.tar.gz" "$NAME"
fi
rm -rf "$STAGE"

# 체크섬을 아카이브 옆에 둔다. 본체는 goreleaser 가 checksums.txt 하나로
# 모으지만 이 아카이브는 그 바깥에서 만들어지므로 여기서 낸다.
ARCHIVE="$OUTDIR/$NAME.tar.gz"
[ "$GOOS" = windows ] && ARCHIVE="$OUTDIR/$NAME.zip"
if command -v sha256sum >/dev/null; then
	(cd "$OUTDIR" && sha256sum "$(basename "$ARCHIVE")" > "$(basename "$ARCHIVE").sha256")
else
	(cd "$OUTDIR" && shasum -a 256 "$(basename "$ARCHIVE")" > "$(basename "$ARCHIVE").sha256")
fi
ls -l "$ARCHIVE" "$ARCHIVE.sha256"
