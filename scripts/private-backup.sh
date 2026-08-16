#!/bin/sh
# private/ 백업 (ADR 0033).
#
# private/ 는 gitignore 대상이라 이 저장소에 커밋되지 않는다. 그 안의
# 치환 사전과 금지 패턴 목록은 잃으면 손으로 다시 만들어야 하고, 빠뜨린
# 항목이 곧 공개 저장소로 나가는 식별자가 된다. 백업 위치는 upstream
# llm-wiki 의 meta/engram/ 이다. upstream 은 원격에 push 되는 비공개
# 저장소이므로 별도 백업 수단을 만들지 않는다.
#
# private/deltas/ 는 upstream meta/CHANGELOG.md 에서 파생한 산물이라
# 제외한다. 파생물을 원본 옆에 두지 않는다.
#
# 사용법:
#   ENGRAM_UPSTREAM=~/Git/llm-wiki sh scripts/private-backup.sh            백업
#   ENGRAM_UPSTREAM=~/Git/llm-wiki sh scripts/private-backup.sh --restore  복구
#
# upstream 저장소에 커밋을 남기는 행위이므로 자동화하지 않는다. private/ 를
# 고친 뒤 사람이 돌린다. 커밋과 push 는 하지 않는다. 사본만 맞추고 안내한다.
set -e

ROOT=$(git rev-parse --show-toplevel)
LOCAL="$ROOT/private"
NAME="meta/engram"

if [ -z "$ENGRAM_UPSTREAM" ]; then
    echo "ENGRAM_UPSTREAM 이 없다. upstream llm-wiki 경로를 준다" >&2
    echo "예: ENGRAM_UPSTREAM=~/Git/llm-wiki sh scripts/private-backup.sh" >&2
    exit 1
fi

# ~ 는 셸이 변수 대입에서 펼치지 않는 경우가 있어 직접 펼친다.
case "$ENGRAM_UPSTREAM" in
    "~"|"~/"*) ENGRAM_UPSTREAM="$HOME${ENGRAM_UPSTREAM#\~}" ;;
esac
REMOTE="$ENGRAM_UPSTREAM/$NAME"

if [ ! -d "$ENGRAM_UPSTREAM/.git" ]; then
    echo "upstream 이 git 저장소가 아니다: $ENGRAM_UPSTREAM" >&2
    exit 1
fi

if [ "$1" = "--restore" ]; then
    if [ ! -d "$REMOTE" ]; then
        echo "백업이 없다: $REMOTE" >&2
        exit 1
    fi
    mkdir -p "$LOCAL"
    rsync -a "$REMOTE/" "$LOCAL/"
    echo "복구했다: $REMOTE 에서 $LOCAL 로"
    exit 0
fi

if [ ! -d "$LOCAL" ]; then
    echo "백업할 것이 없다: $LOCAL 이 없다" >&2
    exit 1
fi

mkdir -p "$REMOTE"
rsync -a --delete --exclude 'deltas/' "$LOCAL/" "$REMOTE/"
echo "백업했다: $LOCAL 에서 $REMOTE 로"

if git -C "$ENGRAM_UPSTREAM" status --porcelain -- "$NAME" | grep -q .; then
    echo "upstream 에 변경이 남았다. 직접 커밋하고 push 한다:"
    echo "  git -C $ENGRAM_UPSTREAM add $NAME && git -C $ENGRAM_UPSTREAM commit -m 'chore: engram private 백업 갱신' && git -C $ENGRAM_UPSTREAM push"
else
    echo "upstream 사본이 이미 같다. 커밋할 것이 없다"
fi
