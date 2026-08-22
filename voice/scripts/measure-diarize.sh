#!/usr/bin/env bash
# 화자 분할의 군집 임계값과 파편 필터 하한을 실제 사람 녹음으로 잰다.
#
# 정답이 있어야 재는 자리다. 합성 음성으로 정답을 만들려 했으나 그 길은
# 막혔다. macOS say 의 목소리는 서로 달라도 화자 임베딩이 갈리지 않는다
# ([0092](../../docs/decisions/0092-diarization-thresholds-are-measured-against-real-recordings.md)).
#
# 대신 sherpa-onnx 가 화자 분할 예제로 배포하는 실제 사람 녹음 넷을 쓴다.
# 파일 이름에 화자 수가 들어 있어 그것이 정답이다. 넷을 이어 붙이면
# 서로 겹치지 않는 화자 열 명짜리 표본이 하나 더 나온다.
#
# **산출물을 커밋하지 않는다.** 재는 자리이지 교재가 아니다.
set -euo pipefail

cd "$(dirname "$0")/.."

WORK="${DIARIZE_WORK:-$(mktemp -d)}"
BASE=https://github.com/k2-fsa/sherpa-onnx/releases/download/speaker-segmentation-models
FILES=(0-four-speakers-zh.wav 1-two-speakers-en.wav 2-two-speakers-en.wav 3-two-speakers-en.wav)
TRUTH=(4 2 2 2)

command -v ffmpeg >/dev/null || { echo "ffmpeg 가 필요합니다" >&2; exit 1; }

MODEL_DIR="${ENGRAM_VOICE_MODEL_DIR:-}"
if [ -z "$MODEL_DIR" ]; then
	MODEL_DIR="$HOME/Library/Caches/engram/models/voice/large-v3"
	[ -d "$MODEL_DIR" ] || MODEL_DIR="$HOME/.cache/engram/models/voice/large-v3"
fi
for m in segmentation.onnx speaker-embedding.onnx; do
	[ -f "$MODEL_DIR/$m" ] || { echo "$MODEL_DIR/$m 가 없습니다. engram-voice model pull 을 먼저 돌리세요" >&2; exit 1; }
done

echo "작업 디렉토리: $WORK"
for f in "${FILES[@]}"; do
	[ -f "$WORK/$f" ] || curl -sSL -o "$WORK/$f" "$BASE/$f"
done

# 넷을 이어 붙여 화자 열 명짜리 표본을 만든다. 서로 다른 녹음이라
# 화자가 겹치지 않으므로 4+2+2+2 가 정답이다.
: > "$WORK/cat.txt"
for f in "${FILES[@]}"; do printf "file '%s'\n" "$WORK/$f" >> "$WORK/cat.txt"; done
ffmpeg -f concat -safe 0 -i "$WORK/cat.txt" -c:a pcm_s16le -ar 16000 -ac 1 "$WORK/ten-speakers.wav" -y -loglevel error

go build -o "$WORK/measure" ./cmd/measure

speakers() { # wav, 플래그들
	local wav="$1"; shift
	"$WORK/measure" --diarize --wav "$wav" --model-dir "$MODEL_DIR" --out "$WORK/o.json" "$@" >/dev/null 2>&1
	python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["speakers"])' "$WORK/o.json"
}

sweep() { # 플래그이름, 값들...
	local flag="$1"; shift
	printf '\n=== %s ===\n' "$flag"
	printf '%8s | %6s %6s %6s %6s %8s | %s\n' 값 '4명' '2명a' '2명b' '2명c' '10명' '맞음/총오차'
	for v in "$@"; do
		local hit=0 err=0 row=""
		for i in "${!FILES[@]}"; do
			local n; n="$(speakers "$WORK/${FILES[$i]}" "--$flag" "$v")"
			row="$row$(printf '%7s' "$n")"
			[ "$n" -eq "${TRUTH[$i]}" ] && hit=$((hit + 1))
			err=$((err + (n > TRUTH[i] ? n - TRUTH[i] : TRUTH[i] - n)))
		done
		local n; n="$(speakers "$WORK/ten-speakers.wav" "--$flag" "$v")"
		row="$row$(printf '%9s' "$n")"
		[ "$n" -eq 10 ] && hit=$((hit + 1))
		err=$((err + (n > 10 ? n - 10 : 10 - n)))
		printf '%8s |%s | %d/5  %d\n' "$v" "$row" "$hit" "$err"
	done
}

sweep threshold 0.5 0.6 0.65 0.68 0.70 0.72 0.75 0.78 0.80 0.85
sweep min-speech-ratio 0.005 0.01 0.02 0.05 0.10 0.15 0.20

echo
echo "정답: 4, 2, 2, 2, 10"
[ -n "${DIARIZE_WORK:-}" ] || echo "임시 디렉토리를 지우려면: rm -rf $WORK"
