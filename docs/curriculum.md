# 교육 커리큘럼

> 2026-08-17 재작성. 여정 24개([journeys.md](journeys.md))와 1.0 커맨드 스물여덟이 확정되어 강의 세션을 다시 매핑했다. 초판(2026-08-08)은 여정 다섯 개 시절의 뼈대였다. 강의 자료는 [course/](course/README.md)에 둔다.

## 대상과 형태

대상은 Claude Code 같은 CLI 에이전트를 이미 쓰는 개발자다. 마크다운과 git을 안다고 전제한다. RAG, frontmatter 같은 용어는 짧게 짚고 넘어간다.

세션은 여덟이다. 1세션은 60분 강의이고 나머지는 각 40분에서 60분의 핸즈온이다. 하루 워크숍이면 1세션부터 6세션까지, 이틀이면 8세션까지 간다. 각 세션은 앞 세션이 만든 위키를 이어서 쓴다. 그래서 순서를 바꾸지 않는다.

**강의 밖에서 혼자 시작하는 사람은 [course/agent-start.md](course/agent-start.md)를 본다.** 30분짜리이며 커맨드를 치지 않고 에이전트에게 시킨다. 여덟 세션을 대신하지 않고 한 바퀴만 돌린다.

교육용 위키는 수강생이 1세션 끝에 `engram init`으로 직접 만든다. `examples/personal`은 6세션 재발견 실습에서 쓴다. 실습 진행은 [course/hands-on.md](course/hands-on.md)가 진실원이다.

## 학습 목표

교육이 끝났을 때 수강생은 다음을 할 수 있다.

1. LLM wiki가 무엇이고 RAG와 어디서 갈라지는지, 왜 이 체계에 승급이 있는지 설명한다.
2. 자기 위키를 `init`으로 만들고 `doctor`로 환경을 점검한다.
3. 일상 입력을 `capture`로 inbox에 넣고 원본을 `source`로 보존한다.
4. `promote`와 `new`로 context에 올리며 게이트 거절을 읽고 고친다. 잘못 올린 것을 `demote`로 되돌리고 수명이 끝난 것을 `archive`로 보낸다.
5. `search`와 `recall`로 지식을 꺼내 쓰고, 둘의 차이를 안다.
6. `resurface`, `bridge`, `digest`로 잊힌 지식을 다시 만난다.
7. `skills install`과 `mcp`로 에이전트를 붙이되 에이전트가 쓸 수 있는 곳이 inbox까지인 이유를 안다.
8. `serve`와 `export`로 팀에 보이는 범위를 정하고, `eject`가 무엇을 넘기고 무엇을 남기는지 안다.

## 강의 세션

| 세션 | 주제 | 커맨드 | 여정 | 실습 산출물 | 자료 |
|---|---|---|---|---|---|
| 1 | 오리엔테이션. LLM wiki 원론, 이 체계의 특별한 점, 해결 방향, engram의 역할, 개인에서 조직으로 | 시연만 | 전체 조망 | 없음 | [course/index.html](course/index.html) |
| 2 | 설치와 첫 위키. 프리셋 셋, `engram.yaml`, 디렉토리가 성숙 단계인 이유 | `init`, `doctor`, `status`, `rules show` | 0 | 자기 위키와 `index.md` | [course/hands-on.md](course/hands-on.md) 2단계 |
| 3 | 넣기. 회의 중 캡처, 링크 드롭, 원본 보존. 날짜 세 필드 | `capture`, `source`, `lint` | 1, 3, 5 | inbox 문서 여럿, sources 문서 둘 | [course/hands-on.md](course/hands-on.md) 3단계 |
| 4 | 올리기. 게이트 거절과 교정, inbox 이동과 sources 파생의 차이, 되돌리기와 폐기 | `promote`, `new`, `demote`, `archive`, `mv`, `update` | 6, 18, 19, 20 | context 문서 셋 이상, 거절 로그 | [course/hands-on.md](course/hands-on.md) 4단계 |
| 5 | 꺼내 쓰기. 검색과 회수의 분리, 백링크, 검색 실패에서 재발견으로 | `reindex`, `search`, `recall`, `backlinks` | 7, 21, 22 | 자기 위키에서 뽑은 인용 조각 | [course/hands-on.md](course/hands-on.md) 5단계 |
| 6 | 다시 만나기. 시간, 관계, 기간 세 축. 상태가 두 곳에 나뉘는 이유 | `resurface`, `bridge`, `digest` | 9, 10, 11 | 기각 기록이 든 `engram-state.yaml` | [course/hands-on.md](course/hands-on.md) 6단계 |
| 7 | 에이전트 연동. 호출 방향, 스킬 문서, MCP 도구 열과 쓰기 하나 | `skills install`, `mcp` | 8, 23 | 에이전트로 capture하고 사람이 promote한 문서 | [course/hands-on.md](course/hands-on.md) 7단계 |
| 8 | 운영과 공유. 마이그레이션, 동기화, 읽기 전용 공유, 반출, 규칙 소유권 이양 | `migrate`, `sync`, `serve`, `export`, `eject` | 12, 13, 14, 15, 16, 17 | eject된 위키와 export 번들 | [course/hands-on.md](course/hands-on.md) 8단계 |

여정 4(주간 뉴스)는 바이너리 밖 동선이라 세션에 넣지 않는다.

여정 2(음성)는 **8세션 이후의 선택 세션**이다. 필수 경로에 넣지 않는다. 전사가 별도 모듈의 `engram-voice`이고 모델을 1.7GB 더 받아야 하므로, 여기서 이탈하는 수강생이 앞 여덟 세션의 이수에 영향을 받으면 안 된다([0079](decisions/0079-voice-is-a-separate-binary-in-a-separate-repository.md), [0080](decisions/0080-voice-is-a-nested-module-in-this-repository.md)). 3세션에서는 "전사 결과와 요약만 capture한다"는 한 줄로만 다룬다.

| 세션 | 주제 | 도구 | 여정 | 실습 산출물 | 자료 |
|---|---|---|---|---|---|
| 선택 | 음성. 전사와 화자 분할, 용어 사전 피드백 루프, 전사 결과를 위키로 | `engram-voice`, `capture`, `promote --to sources` | 2 | 자기 녹음에서 나온 sources 문서와 사전에 추가한 항목 | [course/hands-on.md](course/hands-on.md) 선택 단계 |

**수강생이 자기 목소리를 녹음한다.** 합성 음성을 준비해 주려 했으나 whisper 가 알아듣지 못해 실습이 성립하지 않았다([0085](decisions/0085-synthetic-speech-is-not-teaching-material.md)). 마이크가 없으면 짝을 지어 한 녹음을 같이 쓴다.

이 세션의 핵심은 도구 사용법이 아니라 **사전이 위키 자산이라는 것**이다. 사람이 고친 오탈자가 사전에 쌓이고 다음 전사가 개선된다. 그 루프를 한 바퀴 돌려 보는 것이 이 세션의 산출물이다.

각 세션의 마지막은 `engram lint`다. **도구가 만든 위키는 언제 lint를 돌려도 error가 0이어야 한다**는 불변식을 수강생 위키에서 매 세션 확인한다. 어긋나면 그 자리에서 무엇을 손으로 고쳤는지 찾는다.

## 세션별 핵심 문장

강사가 세션마다 한 번씩 소리 내어 말하는 문장이다. 슬라이드 제목이 된다.

| 세션 | 문장 |
|---|---|
| 1 | 판단은 사람이, 불변식은 코드가 |
| 2 | 디렉토리는 분류함이 아니라 성숙 단계다 |
| 3 | capture는 아무것도 검사하지 않는다. 관문은 올라갈 때 한 번이면 된다 |
| 4 | 거절 사유는 하나뿐이다. 거절 경험이 곧 교육이다 |
| 5 | search는 사람이 열어 볼 목록을, recall은 에이전트가 인용할 조각을 준다. 둘 다 요약하지 않는다 |
| 6 | 위키가 커질수록 가치가 나오고, 사람이 손으로는 못 하는 일이다. 축이 둘이고 하나는 낱말을 보고 하나는 뜻을 본다 |
| 7 | engram은 LLM을 부르지 않는다. 에이전트가 engram을 부른다 |
| 8 | engram은 다리다. 건너고 나면 각자 자기 것을 짓는다. 규칙은 당신 것이 되고 어려운 계산은 도구가 계속 해 준다 |
| 선택 | 잘못 들은 말을 고치면 그 교정이 위키에 남는다. 사전은 도구가 아니라 당신 것이다 |

## 이 과정이 무엇을 주지 않는가

**정답을 주지 않는다.** 여덟 세션이 가르치는 것은 하나의 방식이고 그것이 모두에게 맞을 리 없다. 사람마다 무엇을 적어야 안심이 되는지, 어떤 단위로 묶어야 나중에 찾아지는지가 다르다.

그런데도 하나의 방식을 끝까지 가르치는 이유가 있다. **처음부터 자기 방식을 지을 수 있는 사람은 없다.** 남의 방식을 한 번 완주해 봐야 무엇이 자기에게 안 맞는지가 보인다. 이 과정이 주는 것은 정답이 아니라 완결된 한 벌이다.

그래서 마지막 세션은 나가는 문을 보여 주고 끝낸다. `eject`가 규칙의 소유권을 넘기고 계산은 도구에 남긴다. **바꾸지 말라고 하지 않는다.** 바꿀 수 있게 만들어 두었다는 것을 알려 주는 것이 마지막 일이다.

## 평가

시험은 없다. 8세션이 끝났을 때 수강생 위키가 다음을 만족하면 이수다.

- `engram lint`가 error 0, reject 0이다.
- context 문서가 스무 개 이상이고 그중 하나는 sources에서 파생되었다.
- `engram-state.yaml`에 기각 쌍이 하나 이상 있다.
- `engram serve`로 띄웠을 때 inbox와 sources 문서가 목록에 없다.
- `eject`한 뒤 내보낸 Python 린터와 `engram lint --include-inbox`가 같은 판정을 낸다.
- **자기 위키에서 바꾸고 싶은 값 하나를 말할 수 있다.** 이것이 실제 이수 증거다.

## 미결정

- 강의 일정과 횟수. 하루 워크숍과 이틀 과정 중 무엇을 기본으로 둘지.
- 6세션의 재료 넣기(스물한 건)를 강의 중에 시킬지 결과 위키를 나눠 줄지. 20분이 걸린다.
- 2세션부터 8세션의 자료 형식. 1세션처럼 슬라이드와 읽기 모드를 겸한 단일 HTML로 갈지, 실습 문서는 마크다운으로 둘지.
- 7세션에서 쓸 에이전트. Claude Code를 기본으로 두되 다른 CLI 에이전트도 같은 스킬 문서로 붙는지 확인한다.

## 관련

- [course/README.md](course/README.md) 강의 자료 위치와 규칙
- [journeys.md](journeys.md) 여정 24개
- [design.md](design.md) 커맨드 체계와 마일스톤
