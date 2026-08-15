# 골든 위키

ADR 0005의 parity 비교 중 lint 위반 목록 축에 쓰는 고정 입력 위키다.
ADR 0011이 정한 대로 사람이 의도적으로 관리한다. `engram init`의 재생성물인
`examples/`와는 달리 손으로 고친다.

## 구성

`engram.yaml`은 education 프리셋을 기준으로 하고 topics 개방 집합과
forms 폐쇄 집합을 명시했다. 나머지 키는 internal/config 기본값을 쓴다.
프론트매터 필드와 허용값은 internal/config와 upstream 계약을 따랐다.
taxonomy의 문서 필드는 설정의 복수 키에 맞춰 `form` 하나와 `topics` 목록으로 썼다.

## 사례 표

각 문서는 lint 가 판정해야 할 사례를 하나씩 대표한다.
기대 판정이 사례 하나에 정확히 대응하도록 문서 하나에는 이상이 하나만 있다.
예외는 깨진 링크로, go-table-driven-tests.md 가 함께 담는다.

| 파일 | 대표하는 사례 | 기대 판정 |
|---|---|---|
| context/testing-pyramid.md | 완전히 정상. 위키링크 2개, 필수 축 전부 채움 | 정상 |
| inbox/voice-memo-raw.md | 프론트매터가 아예 없음 | 오류 |
| inbox/unclosed-frontmatter.md | 프론트매터 닫는 구분자 누락 | 오류 |
| context/go-table-driven-tests.md | 위키링크 1개뿐. min_wikilinks 2 게이트 거절. 링크 대상이 없어 깨진 링크이기도 함 | 거결 |
| inbox/orphan-note.md | 위키링크 0개인 고아 문서 | 경고 |
| context/cli-flag-conventions.md | forms 폐쇄 집합에 없는 값 (cheatsheet) | 오류 |
| context/sqlite-foreign-keys.md | topics 개방 집합에 없는 값 (sqlite). 문서는 통과 | 경고 |
| inbox/wrong-stage.md | artifact_stage 허용값 아닌 값 (draft) | 오류 |
| context/markdown-link-syntax.md | 코드 펜스와 인라인 코드 안에 가짜 링크. 실제 링크 0개. 펜스 안을 세면 판정이 틀어진다 | 거결 |
| inbox/crlf-meeting-note.md | CRLF 줄바꿈. 링크 줄 번호는 LF 문서와 같게 잡혀야 함 | 정상 |
| sources/tech-talk-summary.md | sources 계층. created 와 sourced_at 있음, updated 없음 | 정상 |
| context/legacy-import-notes.md | created 가 연월 형식 (2019-11) | 정상 |

루트 index.md 는 사례가 아니라 위키 구조의 일부다. archive 는 비워 둔다.

## 관리 규칙

- 날짜는 전부 고정값이다. 픽스처는 결정론적이어야 하므로 오늘 날짜를 쓰지 않는다.
- CRLF 문서를 제외한 모든 파일은 LF 다. CRLF 가 git 설정으로 바뀌지 않도록
  `harness/fixtures/.gitattributes` 에서 해당 파일을 `-text` 로 지정했다.
- 사례를 추가할 때는 이 표에 함께 적는다. 표가 없으면 픽스처의 의도를 아무도 알 수 없다.
