---
number: 0101
title: upstream이 inbox 퇴장 규칙을 신설해도 engram은 코드를 바꾸지 않고 미이행으로 기록한다
date: 2026-10-04
status: accepted
---

# upstream이 inbox 퇴장 규칙을 신설해도 engram은 코드를 바꾸지 않고 미이행으로 기록한다

## 맥락

upstream 규칙 명세 사본이 lock bf401e4에서 07f8737 사이에 셋 바뀌었다.

첫째, `promotion-rules.md`가 `Inbox Exit` 절을 신설했다(2026-09-13). 초안이 `inbox/`를 떠나는 사건을 셋으로 정한다. 승급 커밋에서 초안을 `archive/<channel>/`로 함께 옮기는 것, 사람이 `exports/docs/`나 맞는 채널로 재배치하는 것, 사람이 승급하지 않기로 하고 지우는 것이다. 나이 기한은 두지 않는다. `sources/`와 `context/` 문서의 `source_refs`와 `derived_from`이 아직 존재하는 `inbox/` 경로를 가리키면 lint가 실패해야 한다고도 적었다. 같은 날 upstream `scripts/lint-frontmatter.sh`에 그 검사가 들어갔다. 승급 대상이 `context`인데 초안 본문을 담은 `sources` 문서가 없으면 `archive/`로 옮기기 전에 source summary를 먼저 만들라는 문장도 있다. 그렇지 않으면 context는 증류본인데 원문이 어디에도 남지 않는다는 이유다.

둘째, `security-rules.md`의 Mirror Handling이 바뀌었다(2026-09-25). 미러 제외 기준을 "private 또는 raw source 폴더"에서 "Git에 올라가지 않는 경로"로 바꾸고, 미러는 `sensitivity` 값으로 거르지 않는다고 적었다.

셋째, `terminology-format.md`의 자동 교정 칸 어휘에 `n/a`가 늘었다. 용어 사전에 인명 절이 생기면서, 전사에 이름이 한 번도 안 나와 치환 대상이 아닌 인물 한 명을 `n/a`로 적었기 때문이다.

## 결정

**engram 코드는 고치지 않는다.** 다만 둘을 남긴다.

`Inbox Exit`의 잔류 참조 검사를 `docs/spec-map.md` 4.3절과 6절에 **미이행**으로 적는다. 이유와 결정할 시점도 같이 적는다.

용어 사전 파서가 `n/a`를 치환하지 않는다는 것을 시험으로 고정한다. 지금까지 이 동작은 `yes`로 시작하는 행만 고르는 조건에서 우연히 따라 나왔을 뿐 어디에도 명시돼 있지 않았다. upstream이 어휘를 넷에서 다섯으로 늘린 것이 이번에 처음 드러난 사례이므로 계약으로 못 박는다.

## 근거

임시 위키를 만들어 같은 입력을 engram과 upstream 스크립트에 돌렸다.

**engram의 `promote`는 upstream이 막으려는 상태를 스스로 만들지 않는다.** `capture`로 넣은 초안을 `promote --to sources`로 올리면 `inbox/`에서 파일이 사라지고 `sources/`에 문서가 생긴다. 거기서 `context`로 파생하면 `inbox/` 경로를 가리키는 참조가 어느 문서에도 남지 않는다. 초안이 옮겨지므로([ADR 0022](0022-promote-moves-inbox-derives-sources.md), [0058](0058-promote-to-sources-moves-evidence.md)) 잔류할 수 없다.

**upstream이 걱정하는 "원문이 어디에도 남지 않는" 상태는 이미 경고로 드러난다.** `inbox` 초안을 곧장 `context`로 올리면 `source_refs: []`로 나오고 `graph.empty-provenance`가 warn을 낸다. 고치는 법 안내가 `promote --to sources`를 가리킨다([0073](0073-provenance-must-not-be-empty.md)).

**차이가 실제로 나는 자리는 하나다.** 사용자가 `source --ref inbox/<초안>`처럼 존재하는 `inbox/` 경로를 직접 적은 경우다. 이때 upstream 스크립트는 `source_refs points at ... which still exists`로 FAIL하고 초안을 `archive/`로 옮기면 통과한다. engram `lint --include-inbox`는 그 `sources` 문서에 위반을 하나도 내지 않는다. 에이전트 스킬과 핸즈온은 `--ref`에 출처 설명이나 URL을 쓰라고 안내하며 `inbox/` 경로를 적게 하지 않는다.

**넣으려면 고칠 길이 먼저 필요하다.** 걸린 사용자가 쓸 커맨드가 없다. `archive`는 `context` 문서만 받고 `inbox` 문서를 거절한다. `update`는 `--force` 없이 `sources` 문서의 `source_refs`를 못 고친다([0064](0064-update-refuses-to-change-sources.md)). upstream이 요구하는 해법인 초안을 `archive/<channel>/`로 옮기는 일은 손으로 파일을 옮기는 수밖에 없다. 그렇게 옮기면 `artifact_stage: inbox`인 문서가 `archive/` 아래 있게 되어 `location.stage-agreement`가 warn을 낸다. 선언이 위치보다 낮은 방향이라 막지 않고 알리기만 하는 것이 [0035](0035-stage-mismatch-severity-by-direction.md)의 판단이다. 고칠 길 없이 막는 등급을 두면 사용자가 도구를 버린다는 같은 논거가 여기서도 선다.

**`n/a`는 치환 대상이 아니다.** 이전 사전과 새 사전을 engram 파서로 각각 읽었다. 자동 교정 규칙이 283개에서 290개로, 사람이 볼 항목이 29개에서 30개로 늘었다. 규칙 증가분 7은 새로 등록된 인명 네 건의 변형 수(둘, 둘, 둘, 하나)와 같다. `n/a` 행은 규칙이 되지 않고 검토 항목으로만 센다.

**미러 개정은 engram이 다루지 않는 영역이다.** [spec-map 4.9절](../spec-map.md)이 `mirror/`를 범위 밖으로 적어 두었다. `HiddenSensitivities`는 웹 뷰어와 반출의 노출 판정이고 iCloud 미러와 목적지가 다르다.

## 확인

- `go test ./...`와 `ENGRAM_UPSTREAM=~/Git/llm-wiki go test ./harness/parity/ -v` 모두 통과한다.
- parity의 lint 축과 resurface 축 출력은 이전 lock(bf401e4)의 upstream과 이번 lock(07f8737)의 upstream에서 소요 시간 줄을 빼면 바이트 단위로 같다. **그러나 이것은 새 규칙이 동등하다는 증거가 아니다.** `harness/fixtures/golden-wiki`에 `inbox/` 잔류 참조 사례가 없어 그 규칙을 아무도 비교하지 않는다. 위 임시 위키 실측이 실제 근거다.
- 골든 스냅샷을 재생성할 필요가 없었다. 코드가 바뀌지 않았다.
- `docs/spec-map.md` 6절 절차의 3번(lint 규칙과 임계값)과 4번(프리셋 기본값)에 고칠 것이 없다.

## 대안

**잔류 참조 검사를 warn으로 지금 넣는다** — 검사 자체는 결정론적이다. 그러나 engram이 만든 위키에서는 그 상태가 생기지 않고, 생겼을 때 쓸 고치는 법이 없다. 등급을 낮추면 안내가 "손으로 옮기세요"가 되는데 그 결과가 곧바로 `location.stage-agreement` warn이므로 해소되지 않는 경고 두 개를 번갈아 만든다. 규칙이 스물에서 스물하나가 되어 문서 수치 여럿이 함께 움직이는 비용도 있다.

**`promote`가 `inbox` 초안을 지우지 않고 `archive/<channel>/`로 옮기게 바꾼다** — upstream과 같은 동작이 된다. 그러나 engram의 모델은 `sources` 문서가 증거를 담고 `inbox` 원본은 옮겨져 사라지는 것이다. 원본이 남으면 같은 내용이 두 벌이 된다는 0022의 근거가 그대로 서 있다. 바꾸려면 0022와 0058을 개정하는 새 결정이 필요하고, 한 번에 `artifact_stage` 불일치 처리와 채널 디렉토리 도입까지 걸린다. harness 에이전트가 한 판단으로 정할 크기가 아니다.

## 열린 항목

- `rules show`와 `engram-voice`의 "사람이 볼 항목" 수에 `n/a` 행이 들어간다. 코드 주석의 정의(자동 교정 대상이 아닌 항목 수)로는 맞지만 문구는 한 건에서 부정확하다. 지금은 한 건이라 두고, 같은 어휘가 여러 건으로 늘면 그때 센다.
- 결정할 시점은 `inbox` 초안을 `archive`로 보존하는 커맨드를 만들 때, 또는 위 잔류 상태가 실제 위키에서 걸려 나올 때다. 그때 픽스처에 사례를 더해 parity가 이 규칙을 비교하게 한다.
- `docs/course/hands-on.md`가 인용한 upstream `inbox/README.md` 문구는 개정 전 것이다. 교재가 가르치는 세 길(삭제, `--to sources`, `promote`)은 지금도 맞으므로 인용을 갱신할지는 교재 쪽에서 정한다.
