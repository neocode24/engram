# harness

ADR 0005가 정한 upstream llm-wiki 규칙 명세와 Go 구현의 출력 비교 장치다.

## 무엇을 보장하는가

`lint_golden_test.go`는 고정 입력 위키 `fixtures/golden-wiki`를 커맨드 계층으로
검사한 결과가 `golden/lint.txt` 와 `golden/lint.json` 스냅샷과 바이트까지
같은지 본다. 러너는 바이너리를 exec 하지 않고 패키지를 직접 호출한다.
exec 는 빌드 산출물 위치에 의존해서 CI 에서 깨지기 때문이다.

출력의 줄바꿈은 LF 로, 경로 구분자는 슬래시로 비교 직전에 정규화한다.
정규화 지점은 `lint_golden_test.go` 의 `normalize` 함수 하나뿐이다.
git 이 스냅샷을 CRLF 로 풀지 않도록 `golden/.gitattributes` 가 줄바꿈을 고정한다.

테스트는 결정론적이다. lint 출력에 시각이나 난수가 들어가지 않는 한
같은 커밋에서는 항상 같은 결과가 나온다.

## 무엇을 보장하지 않는가

- 비교 축 넷 중 지금 덮는 것은 lint 위반 목록 하나뿐이다. resurface 선정 순위,
  프론트매터 정규화 결과, eject 산출물 diff 셋은 아직 러너가 없다.
- upstream 스크립트와의 실제 동등성 검증은 `parity/` 가 맡는다. `lint_golden_test.go`
  는 Go 구현 자신의 출력이 언제 바뀌는지를 지키는 회귀망이다.
- 스냅샷이 곧 정답은 아니다. 스냅샷은 "지금 구현이 내는 출력"의 기록이므로
  버그를 포착하는 것과 버그를 고정하는 것을 갈리는 판단은 아래 기준이 한다.

## 스냅샷 갱신

```
go test ./harness -update
```

## 갱신이 정당한 경우와 회귀인 경우

실패 원인을 먼저 분류하고, 어느 쪽인지 확정되지 않으면 갱신하지 않는다.

갱신이 정당한 경우는 출력 형식이나 규칙 자체가 의도적으로 바뀐 경우다.

- 위반 메시지 문구를 손보거나 규칙 ID를 추가, 제거한 경우
- lint 출력 형식(텍스트 포맷, JSON 구조)을 바꾼 경우
- 골든 위키 픽스처 자체를 고친 경우
- 규칙의 판정 기준을 바꾸기로 결정한 경우(관련 ADR 이 있는 경우)

회귀인 경우는 구현이 의도와 다르게 움직인 경우다. 이때는 갱신하지 않고
원인을 고친다.

- 파서, 스키마 검사, 링크 추출의 실수로 위반 목록이 달라진 경우
- 정렬 순서가 바뀌어 같은 위키의 출력이 흔들리는 경우
- 코드 펜스 안 링크를 세는 등 이미 결정된 판정 규칙이 어긋난 경우

판단이 서지 않으면 diff 를 근거로 판단자에게 물은 뒤 결정을 커밋 메시지에
남긴다. 실패할 때마다 갱신하면 이 harness 는 아무것도 지키지 못한다.

## 디렉토리

| 경로 | 역할 |
|---|---|
| `fixtures/golden-wiki/` | 고정 입력 위키. 손으로 관리한다. 사례표는 안쪽 README |
| `golden/` | lint 출력 스냅샷. `go test ./harness -update` 로 재생성 |
| `lint_golden_test.go` | 골든 비교 러너 |
| `upstream/` | 사본으로 고정한 upstream 규칙 명세. **손으로 고치지 않는다** |
| `upstream.lock` | vendoring 시점의 upstream 커밋 해시와 파일 목록 |
| `parity/` | upstream 스크립트와의 출력 비교 러너 |

## upstream 명세 동기화

```
python3 scripts/upstream-sync.py --upstream ~/Git/llm-wiki --check
python3 scripts/upstream-sync.py --upstream ~/Git/llm-wiki
```

명세 목록은 upstream `AGENTS.md` 가 선언한 것을 매번 읽는다. upstream 은 자기
문서에서 이것을 "계약 파일" 이라 부르며 `upstream-sync.py` 가 그 문자열을 그대로
파싱하므로 그쪽 표기는 바뀌지 않는다(ADR 0034). 이쪽에
목록을 복제하지 않는다(ADR 0029). `terminology-normalization.md` 는 사전
전체가 조직 어휘 목록이라 제외한다.

복사한 파일은 `private/vendor-replacements.txt` 로 익명화한 뒤
`scripts/check-boundary.py` 를 통과해야 한다. **걸리면 sync 가 실패하고
vendored 파일이 지워진다.** 사전에 항목을 추가하고 다시 돌린다.

명세 변경분은 `private/deltas/` 에 남는다. upstream CHANGELOG 원문을
인용하므로 공개하지 않는다(ADR 0030).
