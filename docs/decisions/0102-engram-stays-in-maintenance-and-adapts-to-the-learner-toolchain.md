---
number: 0102
title: engram은 유지보수를 유지하고 수강생 도구 환경에만 맞춘다
date: 2026-10-11
status: accepted
---

# engram은 유지보수를 유지하고 수강생 도구 환경에만 맞춘다

## 맥락

engram의 목적이 다시 정해졌다. 사내 교육과정에서 llm-wiki 구축 과정을 가르칠 때 **위키를 처음 자리잡게 하는 초기 구축 도구**로 쓰고, 숙련자는 `eject`로 규칙을 소유해 넘어간다. 버전업은 계속 제공한다. 이 목적에 비추어 셋 중 하나를 정해야 했다. 유지보수 유지, 개발 가속, 컨셉 변경이다.

판단의 재료는 셋이다.

**engram 쪽.** 마지막 기능 커밋이 2026-08-23([0096](0096-docs-harness-checks-quantitative-claims.md))이고 이후는 lock 맞춤과 문서뿐이다. v1.0.0 이후 `cmd/`와 `internal/` 변경은 테스트 파일 하나다. `go test ./...`, parity, voice 빌드가 전부 통과한다. `docs/roadmap.md`의 1.1 트랙(용어 정규화 루프, 출력 색, 검색 재검토)은 착수 흔적이 없다.

**upstream 쪽.** llm-wiki는 2026-08-23 이후 266 커밋이 쌓였고 새 스크립트가 열넷이다. 목록을 읽어 성격을 가렸다. 세션 지식 회수(`detect-idle-sessions.py`, `update-session-capture-registry.sh`), 음성메모 감지와 로컬 전사(`detect-new-voice-memos.sh`, `voice_memo_local_stt.py` 등), 링크와 패치 제안 적용(`wiki_apply_links.py`, `wiki_propose_patch.py`, `wiki_apply_patch.py`), 유지보수 묶음(`wiki_maintenance.py`), 미러와 색인 동기화다. **전부 스케줄로 돌거나 에이전트를 직접 부르는 운영 자동화다.** upstream `AGENTS.md`도 이를 "스케줄 자동화는 `inbox/`까지만 쓴다"로 못 박는다. 규칙 명세 7종은 주간 harness가 추적하고 있고 이 기간의 명세 변화는 [0100](0100-upstream-sources-entry-and-agent-session-channel-need-no-engram-change.md)과 [0101](0101-upstream-inbox-exit-rule-is-recorded-as-unimplemented.md)이 무변경 또는 미이행으로 닫았다.

**수강생 쪽.** 공식 도구는 Copilot이고 Claude Enterprise도 ZTNA 안에서 된다. ChatGPT는 공식 경로가 아니다. 대부분 CLI 에이전트 사용자가 아니다. 저장소는 이미 공개 상태다.

## 판단 근거

### 개발 가속의 근거가 없다

llm-wiki의 신규 장치 열넷은 [0014](0014-llm-boundary-agent-drives-binary.md)의 경계 밖이다. engram은 LLM을 부르지 않고 스케줄도 갖지 않는다. 그 장치들을 옮기면 engram이 에이전트 실행기가 되고 "같은 입력에 같은 출력"이라는 게이트의 존재 조건이 흔들린다. 초기 구축 도구라는 목적에서 보면 그 장치들은 **위키가 자리잡은 뒤에 사용자가 자기 운영에 맞춰 붙이는 것**이며, 그 자리가 바로 `eject` 이후다([0013](0013-eject-redefined-seal-removed.md)). `eject` 산출물에 그것들이 없는 것은 결함이 아니라 경계다.

1.1 트랙도 같은 기준으로 본다. 용어 정규화 루프는 위키가 수백 문서로 자란 뒤에 가치가 나오고, 출력 색과 검색 재검토는 초기 구축 체험을 바꾸지 않는다. 지금 열면 출하된 1.0을 흔들 뿐이다.

### 컨셉 변경의 근거도 없다. 다만 전제 하나가 틀렸다

반대 논거는 이것이었다. 수강생 다수가 CLI를 쓰지 않는데 승급 승인은 CLI에만 있다. `init` 이후 손을 놓게 되면 자리잡아 주는 도구가 성립하지 않는다.

**승인이 CLI에만 있다는 전제가 틀렸다.** [0057](0057-approval-attaches-to-content-not-to-the-command.md)이 승인을 커맨드가 아니라 문서 내용에 붙였다. 에이전트가 정리한 본문과 승급 결과를 보여 주고 사람이 대화 안에서 승인하면 에이전트가 `promote`를 실행한다. 이 경로에 필요한 것은 터미널이 아니라 **셸 도구를 가진 에이전트**다. VS Code의 Copilot 에이전트 모드가 그것을 갖는다. 공식 문서(2026-10-07 기준)가 터미널 도구와 승인 단계를 명시하고, 개인 스킬 위치로 `~/.copilot/skills/`, `~/.claude/skills/`, `~/.agents/skills/` 셋을 읽는다고 적는다. 즉 비CLI 사용자도 VS Code 안에서 스킬 경로로 engram을 쓴다. MCP는 그 위에 얹는 선택지이고 조직 정책(기본 꺼짐)에 걸리면 빠져도 된다. [0043](0043-mcp-exposes-one-write-tool-and-omits-promote.md)대로 MCP에는 `promote`가 없으므로 MCP만으로는 어차피 승급이 안 되고, 셸 경로가 본선이다.

그래서 `serve`에 승인 버튼을 다는 식의 컨셉 변경은 기각한다. [0044](0044-serve-is-read-only-and-shows-only-vetted-knowledge.md)의 읽기 전용 계약과 충돌하고, 승인을 다시 커맨드 단위로 되돌리는 셈이며, 셸 경로가 이미 같은 일을 한다.

ChatGPT 보조 요청도 같은 자리에서 닫는다. 위키가 파일과 git이고 ChatGPT 웹은 로컬 파일을 만지지 못한다. 연결 방식(원격 MCP, 터널)의 문제가 아니라 전제의 문제라 engram이 메울 수 없다. 공식 경로가 아니므로 교재도 다루지 않는다.

### 그런데 실측에서 하나 걸렸다

Copilot만 쓰는 PC를 흉내내 확인했다. `~/.copilot/skills/`만 있는 빈 홈에서 `engram skills install --dry-run`이 **대상 없음으로 실패한다**. `internal/skills`의 후보가 `.claude/skills`와 `.agents/skills` 둘뿐이라서다. [0041](0041-skills-install-embeds-one-static-skill.md)이 "근거가 없는 경로는 넣지 않는다"고 했고 당시에는 Copilot이 스킬을 읽지 않았다. 지금은 공식 문서가 `~/.copilot/skills/`를 적으므로 근거가 생겼다. 이것은 기능 추가가 아니라 후보 목록 갱신이며, 0041이 "규약이 바뀌면 이 목록만 고친다"고 예정해 둔 자리다.

교재도 같은 어긋남이 있다. `hands-on.md`와 `agent-start.md`가 Claude Code 기준으로 쓰여 있고 Copilot 경로가 없다. 수강생 공식 도구가 Copilot이므로 그 경로가 먼저 보여야 한다.

## 결정

**유지보수를 유지한다.** 커맨드 범위는 1.0 그대로 닫아 두고 1.1 트랙은 열지 않는다. 주간 harness와 lock 맞춤은 지금처럼 돈다.

**수강생 도구 환경 적응 둘만 한다.** 둘 다 설계를 바꾸지 않는다.

1. `skills install`의 후보에 `.copilot/skills`를 더한다. 감지 규칙(실재하는 디렉토리만, 만들지 않음, 심볼릭 링크 합침)은 그대로다. 안내 메시지와 테스트를 함께 고친다.
2. 교재에 Copilot 경로를 병기한다. `agent-start.md`와 `hands-on.md` 4단계와 7단계에서 스킬 설치 출력과 MCP 등록 방법을 VS Code Copilot 기준으로 먼저 보이고 Claude Code를 뒤에 둔다. 출력은 실측값만 적는다.

**개발 가속과 컨셉 변경은 하지 않는다.** llm-wiki의 운영 자동화는 `eject` 이후 사용자 영역이다. 승급 승인은 0057의 경로로 비CLI 사용자에게도 닿는다.

## 결정을 뒤집을 조건

- 수강생이 VS Code 에이전트 모드를 쓸 수 없는 것으로 확인될 때. Copilot Business 좌석에서 에이전트 모드와 터미널 도구가 조직 정책으로 막혀 있으면 셸 경로가 사라지고 승인 경로가 실제로 없어진다. 그때는 컨셉 변경을 다시 본다.
- 실제 교육에서 `init`부터 첫 `promote`까지 수강생이 혼자 못 가는 것이 관찰될 때. 그 원인이 도구 경로라면 위와 같고, 규칙이 과하다면 프리셋 조정이며 이것은 유지보수 범위다.
- `eject`한 사용자가 운영 자동화를 붙이다가 engram 산출물(`meta/agent-contract.md`, 린터)과 충돌한다고 보고할 때. 그때 비로소 `eject` 산출물을 넓힐지 본다.

## 확인하지 못한 것

- 수강생 Copilot 좌석의 플랜과 "MCP servers in Copilot" 정책 값. 공식 문서상 기본값은 꺼짐이다.
- 수강생 좌석에서 VS Code 에이전트 모드와 터미널 도구가 허용되는지. 공식 문서의 기능 설명만 읽었고 조직 설정은 보지 못했다.
- Claude Enterprise 좌석에 Claude Code가 포함되는지.
- Copilot CLI가 `~/.copilot/skills/`를 읽는다는 것은 1.0.92 기준 확인값이고 VS Code 쪽은 문서만 읽었다. 실제 VS Code에서 심은 스킬이 잡히는지는 돌려 보지 않았다.

## 관련

- [0013](0013-eject-redefined-seal-removed.md) eject는 규칙 소유권 이양이다
- [0014](0014-llm-boundary-agent-drives-binary.md) engram은 LLM을 부르지 않는다
- [0041](0041-skills-install-embeds-one-static-skill.md) 스킬 설치 후보 목록
- [0043](0043-mcp-exposes-one-write-tool-and-omits-promote.md) MCP에 promote가 없다
- [0057](0057-approval-attaches-to-content-not-to-the-command.md) 승인은 문서 내용에 붙는다
