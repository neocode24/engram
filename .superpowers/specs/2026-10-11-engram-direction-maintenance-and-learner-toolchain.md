# spec: engram 다음 방향. 유지보수 유지와 수강생 도구 환경 적응

결정의 근거는 ADR [0102](../../docs/decisions/0102-engram-stays-in-maintenance-and-adapts-to-the-learner-toolchain.md)에 있다. 이 문서는 그 결정을 구현 단위로 쪼갠 것이다.

## 판정

**유지보수 유지.** 커맨드 범위는 1.0 그대로. 1.1 트랙은 열지 않는다. llm-wiki의 신규 운영 자동화 열넷은 `eject` 이후 사용자 영역이라 옮기지 않는다.

예외로 **수강생 도구 환경 적응 둘**만 한다. 설계를 바꾸지 않는 범위다.

## 왜 개발 가속이 아닌가

- llm-wiki의 08-23 이후 신규 스크립트 열넷은 전부 스케줄 또는 에이전트 직접 호출형 운영 자동화다. engram은 LLM을 부르지 않는다(ADR 0014). 옮기면 결정론 경계가 깨진다.
- 규칙 명세 7종의 변화는 주간 harness가 추적하고 ADR 0100, 0101이 닫았다. 코드가 따라가야 할 명세 변화가 없다.
- 1.1 트랙(용어 정규화, 출력 색, 검색 재검토)은 초기 구축 체험을 바꾸지 않는다.

## 왜 컨셉 변경이 아닌가

- 반대 논거 "승인이 CLI에만 있다"는 전제가 틀렸다. ADR 0057이 승인을 문서 내용에 붙였고 에이전트가 승인 뒤 `promote`를 실행한다. 필요한 것은 터미널이 아니라 **셸 도구를 가진 에이전트**이며 VS Code Copilot 에이전트 모드가 그것을 갖는다.
- `serve`에 승인 UI를 다는 안은 ADR 0044(읽기 전용)와 충돌하고 승인을 커맨드 단위로 되돌린다. 기각.
- ChatGPT 보조는 위키가 파일과 git이라는 전제에서 막힌다. engram이 메울 자리가 아니다. 교재에서 다루지 않는다.

## 실측으로 확인한 결함 하나

```
$ T=$(mktemp -d); mkdir -p $T/.copilot/skills
$ HOME=$T engram skills install --dry-run
Error: 설치 대상이 될 에이전트 스킬 디렉토리를 찾지 못했습니다
찾아본 곳(홈 디렉토리 기준):
  .agents/skills (...)
  .claude/skills (...)
exit=1
```

Copilot만 쓰는 PC에서 스킬 설치가 실패한다. `internal/skills/skills.go`의 `candidates`가 둘뿐이다. VS Code 공식 문서(2026-10-07)는 개인 스킬 위치로 `~/.copilot/skills/`, `~/.claude/skills/`, `~/.agents/skills/` 셋을 적는다.

## 구현 목록

### 1. `skills install` 후보에 `.copilot/skills` 추가 (build)

- `internal/skills/skills.go` `candidates`에 `{".copilot/skills", "core.skills.note_copilot"}` 추가. 순서는 `.claude`, `.agents`, `.copilot` 또는 알파벳순. 순서가 결정론을 깨지 않는지 `Sources()`가 정렬하므로 안전하다.
- `internal/i18n/catalog_core.go` ko/en 두 벌에 `core.skills.note_copilot` 추가. 예: "GitHub Copilot 개인 스킬 디렉토리" / "GitHub Copilot personal skills directory".
- `internal/skills/skills_test.go`에 `.copilot/skills`만 있는 홈에서 1건 감지되는 케이스 추가. 셋이 전부 있을 때 셋 다 잡히는 케이스도.
- 완료 조건: 위 재현 커맨드가 성공하고 `go test ./...` 통과. `harness/docs`가 세는 수치에는 영향 없음(커맨드 수, 규칙 수 불변).
- 건드리지 않는 것: 감지 규칙(실재 디렉토리만, 만들지 않음, 심볼릭 링크 합침), 스킬 본문, MCP.

### 2. 교재에 Copilot 경로 병기 (scribe, 1 완료 뒤)

대상: `docs/course/agent-start.md`, `docs/course/hands-on.md` 4단계 2절과 7단계 1절, `docs/course/agent.html`과 `index.html`의 해당 장.

- 전제 문장 "셸이나 MCP를 쓰는 에이전트. Claude Code, Cursor 등"을 "VS Code의 Copilot 에이전트 모드, Claude Code 등"으로. Copilot을 먼저 쓴다.
- `skills install` 출력 예시를 `.copilot/skills`가 포함된 실측값으로 갱신. **손으로 지어내지 않는다.** 1이 머지된 바이너리로 실제 홈에서 돌린 출력을 쓴다.
- MCP 등록을 VS Code 기준(`.vscode/mcp.json` 또는 사용자 설정)으로 먼저 보이고 `claude mcp add`를 뒤에 둔다. 조직 정책으로 MCP가 꺼져 있을 수 있음을 한 줄 적고, **MCP가 없어도 스킬 경로로 승급까지 된다**는 점을 명시한다.
- 4단계 "에이전트에게 시킵니다" 이후의 승인 흐름(ADR 0057)은 도구와 무관하므로 바꾸지 않는다.
- 안내체 유지. 구현 사정과 "우리가 안 하는 것" 목록을 넣지 않는다(AGENTS.md 문체 규칙).
- 완료 조건: `go test ./harness/docs` 통과, `scripts/check-boundary.py` 통과, 인용 출력이 전부 실측.

### 3. `docs/roadmap.md` 현행화 (scribe, 2와 독립)

- "현재 상태"의 "남은 것은 저장소 공개 전환과 첫 릴리스"를 지운다. 둘 다 끝났다(저장소 public, v1.0.0 공개).
- "즉시 다음" 1번(공개 전환) 제거. 2, 3은 유지.
- "다음 트랙 (1.1)" 머리에 ADR 0102를 인용해 "열지 않는다. 열 조건은 0102의 뒤집을 조건"을 한 줄 적는다. 트랙 본문은 지우지 않는다. 설계 메모로 가치가 있다.
- 완료 조건: `go test ./harness/docs` 통과.

## 제약

- 커맨드를 늘리지 않는다. `engram --help` 목록과 AGENTS.md가 그대로다.
- lint 규칙과 임계값을 건드리지 않는다.
- MCP 도구 목록을 바꾸지 않는다(ADR 0043).
- 교재 인용 출력은 실측값만.
- 사내 식별자를 쓰지 않는다. 수강생 환경의 조직 맥락은 `private/`에만.

## 확인하지 못한 것

ADR 0102 "확인하지 못한 것" 절과 같다. 수강생 좌석의 에이전트 모드와 터미널 도구 허용 여부, MCP 정책 값, Claude Enterprise의 Claude Code 포함 여부, VS Code에서 심은 스킬이 실제로 잡히는지. 2의 scribe가 VS Code를 열 수 있으면 마지막 하나는 그 자리에서 확인한다.
