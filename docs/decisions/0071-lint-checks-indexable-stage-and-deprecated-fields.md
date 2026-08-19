---
number: 0071
title: lint가 색인 자격과 폐기 필드를 검사한다
date: 2026-08-19
status: accepted
---

# lint가 색인 자격과 폐기 필드를 검사한다

## 배경

upstream 대조에서 나온 결손이 둘이다.

첫째, 색인 자격 검사가 없다. upstream `scripts/lint-frontmatter.sh:271-282`는 단계와 `indexable` 값의 정합을 삼단으로 본다. `inbox`는 `false`가 아니면 FAIL, `source`는 `false`가 아니면 WARN, `context`는 `true`가 아니면 WARN이다. engram lint에는 이 검사가 없어서 `inbox/idx.md`에 `indexable: true`를 넣어도 위반이 나오지 않았다. ADR 0063이 노출 판정에서 `indexable`을 읽게 되었으나 그것은 `serve`와 `export`의 판정이지 문서를 고치라고 알리는 검사가 아니었다.

둘째, 폐기 필드 차단이 없다. upstream은 `quality_level`과 `review_after`를 폐기하면서 lint에 재도입 차단을 넣었다. 폐기 이유가 각각 "두 이름이 같은 것을 가리켜 혼동을 만들었다"와 "예측이라 지킬 수 없고 지나도 아무 일이 일어나지 않는다"이므로, 차단이 없으면 폐기가 조용히 되돌아온다. engram에는 두 개념이 아예 없다. 그런데 upstream에서 마이그레이션한 위키가 그 키를 그대로 안고 들어온다. `migrate`는 모르는 키를 지적하지 않으므로 아무도 알려 주지 않는다.

## 판단 근거

### 색인 자격을 단계별로 잰다

규칙 `schema.indexable-stage`를 더한다. 판정은 upstream과 같다.

| 단계 | 조건 | 등급 |
|---|---|---|
| `inbox` | `indexable`이 `false`가 아니다 | error |
| `source` | `indexable`이 `false`가 아니다 | warn |
| `context` | `indexable`이 `true`가 아니다 | warn |
| `archive` | 검사하지 않는다 | |

`inbox`만 error인 이유는 해악의 방향이다. 색인되지 말아야 할 미검수 문서가 검수 문서와 같은 자격을 스스로 선언하는 것이므로 막는다. `source`와 `context`의 이탈은 승인 여부가 판단이라 경고만 한다.

`archive`를 빼는 근거는 둘이다. upstream에 `archive` 단계가 없어 대조할 규정이 없고, 폐기 문서의 색인 자격은 노출 판정이 `status`와 위치로 이미 정한다(ADR 0063).

`indexable` 축이 꺼진 위키에서는 판정하지 않는다. `cfg.Axes[config.AxisIndexable]`을 본다. 민감도 판정이 끈 축을 읽지 않는 방식과 같다.

`inbox` 판정은 lint의 기본 범위를 따른다(ADR 0070). `--include-inbox`를 줘야 돈다.

### 폐기 필드 목록은 설정에 둔다

upstream의 필드 이름을 engram에 박지 않는다. 폐기 필드 목록은 위키마다 다르다. upstream이 폐기한 두 키를 도구가 알면 남의 조직 이력을 engram이 짊어진다. `topics`와 `forms`를 위키별 설정에 둔 방식과 같은 자리다.

`engram.yaml`에 설정 키 `deprecated_fields`를 둔다. 문자열 목록이고 기본값은 빈 목록이다.

```yaml
deprecated_fields: [quality_level, review_after]
```

규칙 `schema.deprecated-field`는 이 목록에 있는 키가 프론트매터에 있으면 error로 잡는다. 값은 보지 않는다. 존재 자체가 폐기 회귀다.

기본값을 빈 목록으로 둔 근거는 새 위키에는 폐기 필드가 없다는 것이다. 값을 채우는 것은 마이그레이션하는 사람의 일이다. `init`이 만드는 `engram.yaml`에는 이 키를 주석으로만 보인다. 빈 목록을 파일에 적으면 설정 파일이 길어질 뿐이다.

### migrate가 지우지 않는다

`migrate`가 이 키를 지우게 하지 않는다. 지우면 정보가 사라진다. 폐기 필드의 값을 무엇으로 옮길지는 판단이다. lint가 알리고 사람이 정한다. `migrate`의 계약은 "기존 문서를 지금의 설정과 규칙에 맞춘다"(ADR 0038)이지만 그 계약은 규칙이 정하는 값을 채우는 것이지 판단을 대신하는 것이 아니다.

## 결과

- lint 규칙이 17종에서 19종이 된다. `docs/spec-map.md` 4.1의 대응 표에 둘 다 들어간다.
- `deprecated_fields`가 빈 목록인 위키에서는 아무 변화가 없다.
- upstream에서 마이그레이션한 위키는 폐기 키를 `engram.yaml`에 적는 순간 lint가 잡기 시작한다.

## 열린 항목

- 폐기 필드 값을 어디로 옮길지에 대한 안내를 위반 메시지에 더 넣을지는 두고 본다. 위키마다 옮길 곳이 다르므로 지금은 "정해 옮기라"로 족다.
