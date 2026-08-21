#!/usr/bin/env bash
# 화자 분할을 견줄 정답 자료를 만든다.
#
# 화자 분할의 파편 필터 하한과 군집 임계값은 근거 없이 정한 값이고,
# 근거를 만들려면 누가 언제 말했는지의 정답이 있어야 한다. 실제 회의
# 녹음은 공개 경계 밖이라 저장소에 둘 수 없다(ADR 0024).
#
# 대본이 곧 정답이다. 각 줄의 길이를 재서 화자 정답 tsv 를 함께 낸다.
#
# **산출물을 커밋하지 않는다.** 이것은 재는 자리이지 교재가 아니다.
# 합성 음성은 whisper 가 알아듣기 어려워 교재로 못 쓴다.
#
# # 목소리를 먼저 받아야 한다
#
# say 는 한국어 목소리를 아홉 개 열거하지만 기본으로 설치된 것은 Yuna
# 하나다. 나머지는 이름만 나오고 소리는 0.016초짜리 빈 파일이며 오류도
# 내지 않는다. 시스템 설정의 손쉬운 사용에서 사람이 받아야 한다.
#
# 한 목소리의 음높이를 옮겨 화자 셋을 흉내 내 봤으나 화자 분할이 셋을
# 둘로 합쳤다. 같은 성대에서 나온 소리라 임베딩이 안 갈린다. 그 길은
# 막혔으므로 여기서는 진짜로 다른 목소리 셋을 요구한다.
#
# **만들어진 녹음을 교재에 쓰지 않는다.** 합성 음성은 whisper 가
# 알아듣기 어렵다([0085](../../../docs/decisions/0085-synthetic-speech-is-not-teaching-material.md)).
# 이 스크립트는 화자 분할 임계값을 잴 정답 자료를 만드는 자리다.
set -euo pipefail

cd "$(dirname "$0")"

SCRIPT=알림-정리-회의.txt
OUT=알림-정리-회의.m4a
TRUTH=알림-정리-회의.화자.tsv

RATE=145
SR=22050
GAP=0.45

command -v say >/dev/null || { echo "say 가 없습니다. macOS 에서 실행하세요" >&2; exit 1; }
command -v ffmpeg >/dev/null || { echo "ffmpeg 가 필요합니다" >&2; exit 1; }

# 화자마다 다른 목소리를 준다.
voice_for() {
	case "$1" in
		A) echo Yuna ;;
		B) echo Eddy ;;
		C) echo Sandy ;;
		*) echo Yuna ;;
	esac
}

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# 줄 사이에 짧은 침묵을 넣는다. 붙여 놓으면 구간을 나눌 자리가 없다.
ffmpeg -f lavfi -i "anullsrc=r=$SR:cl=mono" -t "$GAP" -c:a pcm_s16le "$WORK/gap.wav" -y -loglevel error

n=0
at=0
: > "$WORK/list.txt"
# 화자 정답을 함께 낸다. 대본과 각 줄의 길이를 아니까 누가 언제
# 말했는지가 계산으로 나온다. 화자 분할을 재려면 이것이 있어야 한다.
printf '시작\t끝\t화자\t대사\n' > "$TRUTH"

while IFS= read -r line; do
	# 대괄호로 시작하는 머리말과 빈 줄은 읽지 않는다.
	case "$line" in
		\[*|"") continue ;;
	esac
	who="${line%%:*}"
	text="${line#*: }"
	n=$((n + 1))
	f="$WORK/$(printf '%03d' "$n").wav"
	v="$(voice_for "$who")"

	# -r 로 속도를 낮춘다. 기본값은 회의 말투보다 빠르다.
	say -v "$v" -r "$RATE" -o "$f" --data-format="LEI16@$SR" "$text"

	# 설치되지 않은 목소리는 오류 없이 빈 파일을 낸다. 그것을 그대로
	# 이어붙이면 대본에는 있는데 소리에는 없는 줄이 생긴다. 실제로
	# 겪은 실패라 여기서 막는다.
	dur0="$(ffprobe -v error -show_entries format=duration -of default=nw=1:nk=1 "$f")"
	if awk -v d="$dur0" 'BEGIN{exit !(d < 0.3)}'; then
		echo "목소리 $v 가 소리를 못 냈습니다(${dur0}초)." >&2
		echo "시스템 설정 > 손쉬운 사용 > 음성 콘텐츠에서 받으세요" >&2
		exit 1
	fi

	printf "file '%s'\nfile '%s'\n" "$f" "$WORK/gap.wav" >> "$WORK/list.txt"

	dur="$(ffprobe -v error -show_entries format=duration -of default=nw=1:nk=1 "$f")"
	read -r start end at <<-EOF
	$(awk -v a="$at" -v d="$dur" -v g="$GAP" 'BEGIN{printf "%.2f %.2f %.2f", a, a+d, a+d+g}')
	EOF
	printf '%s\t%s\t%s\t%s\n' "$start" "$end" "$who" "$text" >> "$TRUTH"
done < "$SCRIPT"

echo "$n 줄을 읽었습니다"
ffmpeg -f concat -safe 0 -i "$WORK/list.txt" -c:a aac -b:a 64k -ar 44100 -ac 1 "$OUT" -y -loglevel error
ls -l "$OUT"
ffprobe -v error -show_entries format=duration -of default=nw=1:nk=1 "$OUT" | awk '{printf "길이 %.1f초\n", $1}'
awk -F'\t' 'NR>1{n[$3]++; t[$3]+=$2-$1} END{for (w in n) printf "화자 %s  %d줄  %.1f초\n", w, n[w], t[w]}' "$TRUTH" | sort
