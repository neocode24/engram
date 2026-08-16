---
name: engram
description: engram 위키에 자료를 넣거나, 검색하거나, 문서 상태와 승급 파이프라인을 조회할 때 쓴다. engram은 판단하지 않고 재료만 낸다. 요약, 판단, 문장화가 필요한 작업이면 이 스킬을 따라 engram을 직접 호출해 재료를 모은 뒤 당신이 처리한다.
---

# engram 다루기

## 호출 방향

engram은 LLM을 부르지 않는다. 네가 engram을 부른다. engram에게 요약이나
판단을 시키려 하지 마라. engram은 결정론적 연산과 규칙 검사만 한다.
문장을 만들고 결정을 제안하는 것은 네 몫이다.

## 쓰기 경계

네가 쓸 수 있는 곳은 inbox까지다. 새 자료는 capture로 inbox에 넣는다.
context로 올리는 일은 승급 게이트를 지나야 하고 확정은 사람이 한다.
promote를 제안할 수 있지만 실행은 사람에게 맡겨라. context에 직접
파일을 만들지 않는다. 단계 이동이 필요하면 promote, demote, archive를
알려 주고 사람이 결정하게 한다.

## 조회는 --json이 주 경로다

조회 커맨드는 완성된 산문이 아니라 재료를 낸다. search, recall, status
같은 조회는 --json으로 구조화된 재료를 받아 네가 문장을 만든다.
resurface와 digest도 재료를 낼 뿐 다이제스트 글을 쓰지 않는다.

## --now로 기준 시각을 고정한다

전역 플래그 --now에 RFC3339 시각을 주면 그 시각을 기준으로 판정한다.
같은 결과를 다시 얻어야 하거나 결과를 검증해야 할 때 쓴다.

## 규칙은 rules show에게 물어라

임계값과 허용값을 지어내지 마라. 위키마다 값이 다르다. 그 위키에 지금
적용되는 규칙 전부는 아래 커맨드가 낸다.

    engram rules show --json --wiki <위키 경로>

lint의 위반 목록도 규칙 ID와 함께 나오므로 같이 읽는다.

## 커맨드 갈래

- 넣기: capture, source
- 올리기: promote, demote
- 조회: search, status
- 재발견: resurface, digest
- 관리: init, update

커맨드 전체 목록의 진실원은 아래 출력이다. 여기 적은 것은 대표뿐이다.

    engram --help

각 커맨드의 플래그와 인자는 그 커맨드의 --help로 확인한다.
