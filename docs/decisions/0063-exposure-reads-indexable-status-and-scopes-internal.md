---
number: 0063
title: 노출 판정이 indexable과 status를 읽고 internal은 반출에서 뺀다
date: 2026-08-19
status: accepted
---

# 선언한 노출 경계와 실제로 나가는 것이 달랐다

## 배경

upstream 대조 감사에서 나온 세 항목이다. 셋 다 "덜 만들었다"가 아니라 "선언과 동작이 어긋난다"이며 무엇을 더 만들지와 무관하게 닫아야 한다.

**첫째, `indexable` 값을 읽는 코드가 없다.** `lint`가 `indexable`을 모든 문서의 필수 필드로 요구하고 `internal/cli/promote.go:467`이 승급하는 문서마다 `indexable: true`를 무조건 박는다. 그런데 그 값을 판정에 쓰는 코드가 한 줄도 없었다. 사용자가 `indexable: false`를 적으면 아무 일도 일어나지 않고 그 사실을 알려 주는 경고도 없다. upstream은 색인 자격 판정에 이 값을 쓴다(`docs/upstream-gap.md` G4).

**둘째, 노출 판정이 `status`를 보지 않는다.** [0044](0044-serve-is-read-only-and-shows-only-vetted-knowledge.md)가 `serve`는 검수된 지식만 보여준다고 정했다. `archived`와 `inbox`는 위치로 걸린다. `superseded`는 스키마 허용값이지만 어디서도 걸리지 않아, `context/`에 남은 채 대체된 문서가 `serve`와 `export`로 그대로 나갔다(G5).

**셋째, 승급한 문서는 전부 반출 가능 상태로 시작한다.** `internal/wiki/wiki.go:311`의 `stageSensitivity`가 inbox 를 뺀 모든 단계에 `internal`을 기본값으로 준다. **`internal/cli/promote.go`에는 `sensitivity` 라는 문자열이 한 번도 나오지 않는다.** 승급이 그 값을 손대지 않으므로 승급 문서의 민감도는 언제나 `internal`이다. 그런데 `expose.HiddenSensitivities`는 `private-local-only`와 `restricted` 둘만 뺀다. 사람이 손으로 값을 올리지 않는 한, 승급한 문서 전부가 아무 경고 없이 반출된다(G8).

## 판단 근거

### 반대 논거: internal 을 빼면 export 가 사실상 아무것도 안 내보낸다

기본 민감도가 `internal`이므로, 이 값을 반출에서 빼면 사용자가 문서마다 손으로 값을 올리기 전에는 반출 결과가 비어 있다. 커맨드를 처음 써 본 사람은 빈 디렉토리를 받는다. 이 논거는 사실이다.

그런데도 기각한다. [0044](0044-serve-is-read-only-and-shows-only-vetted-knowledge.md)가 이미 같은 자리에서 답을 냈다. **넓히는 쪽만 있고 좁히는 쪽은 기본값이다. 기본이 전체 노출이면 사고가 나기 때문이다.** 반출은 파일이 기계 밖으로 나가는 행위이고 되돌릴 수 없다. 되돌릴 수 없는 쪽에서 기본값을 넓게 두는 것은 이 저장소가 일관되게 거절해 온 선택이다.

빈 결과는 실패가 아니다. **사용자가 무엇을 공개할지 아직 정하지 않았다는 뜻이고, 그것을 알려 주는 것이 옳다.** 그래서 빠진 문서 수를 반드시 출력에 낸다. 무엇이 왜 안 나갔는지 모른 채 빈 디렉토리를 보는 일은 없다.

### serve 와 export 의 기본값을 가른 근거

[0046](0046-pack-exports-files-and-anonymizes-by-user-dictionary.md)과 [0047](0047-export-not-pack.md)이 판정을 두 벌 두지 말라고 경고했다. **판정을 두 벌 두면 `serve`가 감추는 문서를 `export`가 내보내고 민감도 선언이 무의미해진다.**

이 결정은 그 경고가 가리키는 방향이 아니다. 판정 함수는 하나이며 `internal/expose`의 `Exclude` 하나뿐이다. 달라지는 것은 그 함수에 넘기는 옵션이고, **`export`가 `serve`보다 더 좁다.** 위험한 방향은 반대쪽, `export`가 더 넓은 쪽이며 그 방향은 만들지 않았다.

| 커맨드 | `IncludeInternal` 기본값 | 근거 |
|---|---|---|
| `serve` | 참 | 로컬호스트에서 자기 위키를 자기가 보는 것이다. 자기 문서를 감추는 것은 방해다 |
| `export` | 거짓 | 파일이 기계 밖으로 나간다. 기본이 전체 노출이면 사고가 난다 |

`serve`에는 좁히는 플래그를 두지 않는다. 기본이 참이고 바꿀 수단이 없다. 로컬 조회에 좁히는 선택지를 두면 쓸 이유가 없는 옵션이 하나 는다.

### internal 은 뒤집을 수 없는 제외가 아니다

`HiddenSensitivities`는 그대로 둔다. 그 목록은 **뒤집을 수 없는 제외의 진실원**이며 [0044](0044-serve-is-read-only-and-shows-only-vetted-knowledge.md)가 "값을 붙인 사람의 판단을 도구가 덮지 않는다"로 정한 것이다. `internal`은 플래그로 열리므로 성격이 다르고, 목록에 넣으면 그 목록의 뜻이 흐려진다.

같은 이유로 제외 사유 상수도 나눈다. `ReasonSensitivity`가 아니라 `ReasonInternal`을 쓰고 집계도 따로 센다. 사용자가 받는 안내가 "문서의 값을 고치세요"와 "플래그로 엽니다"로 갈리기 때문이다.

### indexable 축이 꺼진 위키

`indexable`은 축이다. **축이 꺼진 위키에서는 판정하지 않는다.** 민감도 판정이 `cfg.Axes[config.AxisSensitivity]`를 보는 방식과 같다. 사용자가 그 구분을 쓰지 않기로 한 상태에서 값을 읽어 거르면, 끄는 선택이 아무 의미가 없어진다.

값이 없는 문서도 거르지 않는다. 거짓이라고 적힌 문서만 뺀다. 안 적은 문서를 감추면 `lint`가 필수 필드를 요구하기 전에 만든 문서가 통째로 사라진다.

`status`는 축이 아니라 필수 필드이므로 축 검사가 없다. 값이 없으면 거르지 않는다. `archived`와 `inbox`는 이미 위치로 걸리므로 `superseded` 하나만 본다. 같은 판정을 두 자리에 두면 어느 쪽이 걸렸는지 집계가 갈린다.

### 로컬 조회는 거르지 않는다

이 결정은 `search`, `recall`, `mcp`를 건드리지 않는다. 셋은 노출 판정을 거치지 않으며 그 경계는 `docs/architecture.md` 11절에 이미 있다. 밖으로 나가는 세 경로 중 판정을 거치는 것은 `serve`와 `export` 둘이고 `mcp`는 같은 기계의 에이전트에게 나가므로 거치지 않는다.

그 경계를 바꾸지 않는 이유가 있다. 스킬 경로의 에이전트는 승급을 준비하며 `inbox`와 `sources`를 훑어야 한다([0052](0052-agent-prepares-the-promotion-and-the-human-decides-it.md)). 노출 판정을 로컬 조회에 걸면 승급 준비가 자기 재료를 못 본다.

## 결정

**노출 판정이 `indexable`과 `status`를 읽는다. `internal`은 반출에서 빼고 플래그로 연다.**

| 조건 | 사유 상수 | 축 검사 |
|---|---|---|
| `sensitivity`가 `private-local-only` 또는 `restricted` | `ReasonSensitivity` | `sensitivity` 축 |
| `sensitivity`가 `internal`이고 `IncludeInternal`이 거짓 | `ReasonInternal` | `sensitivity` 축 |
| `indexable`이 거짓 | `ReasonNotIndexable` | `indexable` 축 |
| `status`가 `superseded` | `ReasonSuperseded` | 없다. 필수 필드다 |

- `expose.Options`에 `IncludeInternal`을 더한다. `IncludeArchive`와 같은 성격이다.
- `serve`는 `IncludeInternal`을 참으로 고정한다. 플래그를 두지 않는다.
- `export`는 거짓이 기본이고 `--include-internal`로 연다.
- `Exposure`에 `ExcludedInternal`, `ExcludedNotIndexable`, `ExcludedSuperseded`, `IncludedInternal`을 더한다.
- `export`의 요약 출력이 빠진 것을 말한다. `internal` 때문에 빠진 수는 0이면 줄을 내지 않는다. `--include-internal`을 쓴 실행에서는 포함한 수를 낸다. **넓히는 선택을 했다는 사실이 출력에 남는다.**
- `serve`의 안내 줄에는 `internal` 관련 문구를 넣지 않는다. 기본이 참이라 알릴 것이 없다.

## 결과

- `lint`가 요구하는 `indexable`과 `status`가 처음으로 동작에 쓰인다. 값을 적는 일이 결과를 바꾼다.
- 승급 문서의 기본 민감도가 `internal`이므로, `sensitivity` 축이 켜진 위키에서 `export`는 사용자가 값을 올린 문서만 내보낸다. 기본 프리셋인 `personal`은 축이 꺼져 있어 이 판정이 걸리지 않는다.
- `context/`에 남은 `superseded` 문서가 `serve`와 `export`에서 사라진다. 폐기된 결정이 팀에 보이지 않는다.
- 제외 사유가 여섯에서 아홉이 된다. 커맨드가 제외된 문서를 지목당했을 때 사유별로 다른 문장을 낸다.
- 회귀 시험을 `internal/expose`, `internal/export`, `internal/cli`, `internal/serve`에 더했다. **변이 시험으로 검증했다.** 판정을 하나씩 지우면 해당 시험이 실패한다.

## 열린 항목

- `serve`의 시작 안내가 `superseded`와 `indexable: false` 때문에 빠진 수를 내지 않는다. 자기 문서가 왜 안 보이는지 묻는 상황이 생길 수 있다. 안내 줄을 늘리는 문제이므로 따로 정한다.
- `scope: work`는 여전히 반출 판정에 쓰이지 않는다(G8의 나머지 절반).
- 민감 자료가 `context/`로 올라가는 것 자체는 아무도 막지 않는다. 게이트가 민감도를 보지 않기 때문이다(G7). 이 결정은 나가는 쪽만 닫았다.

## 관련

- [0044 serve는 읽기 전용이고 검수된 지식만 보여준다](0044-serve-is-read-only-and-shows-only-vetted-knowledge.md) 노출 판정의 진실원. 이 ADR이 판정 셋을 더한다
- [0046 pack은 파일을 반출하고 사용자 사전으로 익명화한다](0046-pack-exports-files-and-anonymizes-by-user-dictionary.md) 판정을 두 벌 두지 말라는 경고
- [0047 커맨드 이름은 export다](0047-export-not-pack.md) 같은 경고의 재확인
- [0052 에이전트가 승급을 준비하고 사람이 결정한다](0052-agent-prepares-the-promotion-and-the-human-decides-it.md) 로컬 조회를 거르지 않는 이유
- [0009 스키마 프리셋과 게이트 임계값](0009-schema-presets-and-thresholds.md) `indexable`과 `sensitivity` 축, 프리셋별 구성
