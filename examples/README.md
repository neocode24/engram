# examples/

`engram init --preset education`이 생성하는 데모 위키. 회사 정보가 없는 깨끗한 예제다.

**이 디렉토리는 생성물이다.** 손으로 고치지 않는다. 내용을 바꾸려면 `init` 프리셋을 고치고 재생성한다. 재생성 결과가 커밋된 내용과 다르면 회귀로 간주한다. 즉 이 디렉토리는 예제인 동시에 테스트다.

검증용 골든 위키는 여기가 아니라 `harness/fixtures/`에 둔다. 그쪽은 upstream 스크립트와 Go 구현의 출력을 비교하기 위한 고정 입력이며 사람이 의도적으로 관리한다.

**상태: placeholder.** `init` 구현 시 채운다.

근거: [../docs/decisions/0011-repo-layout-and-module-name.md](../docs/decisions/0011-repo-layout-and-module-name.md)
