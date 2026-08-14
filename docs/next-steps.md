# 다음 스텝과 미결정 사항

> 2026-08-08 기준. 오늘은 계획 수립까지만 마무리.

## 즉시 다음

1. ~~GitHub remote 생성~~ — 완료. `neocode24/engram` (ssh-over-443: `ssh://git@ssh.github.com:443/neocode24/engram.git`), 기본 브랜치 `main`.
2. ~~하위 repo 구성 결정~~ — 완료. 단일 저장소 유지, 저장소 루트가 Go 모듈 루트. [ADR 0011](decisions/0011-repo-layout-and-module-name.md).
3. **Go 프로젝트 init** — 루트에 `go.mod`(모듈명 `github.com/neocode24/engram`), `cmd/engram/`, `internal/`.

## 설계 미결정

- 스키마 보편화 범위 → [design.md](design.md) 스키마 매핑
- 교육용 데모 위키 내용 → `examples/`
- 강의 일정/대상/평가 → [curriculum.md](curriculum.md)
- 사내 Enterprise 레포 명/위치 → [0004](decisions/0004-ip-and-distribution-path.md)

## 마일스톤 0.1 정의

- `init` / `capture` / `source` / `promote` / `lint` / `status` 동작
- 최소 스키마 파일
- 자체 점검(`go test` 최소 1개)

## 참고

- 모든 의사결정: [decisions/](decisions/)
- 설계: [design.md](design.md)
- 커리큘럼: [curriculum.md](curriculum.md)
