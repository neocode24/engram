# Windows 실환경 검증 절차

CI 가 windows-latest(x64)와 windows-11-arm 러너에서 `go test ./...` 를
통과했다. 그러므로 기능 동작은 검증되었다. 이 절차는 CI 가 재현하지
못하는 환경 요소만 검증한다. 중복 노동을 하지 않는다.

## 무엇을 CI 가 이미 덮었는가

- 모든 패키지의 단위 테스트. 두 아키텍처에서 빌드와 테스트가 통과했다
- 경로 구분자와 줄바꿈 정규화 로직. 골든 스냅샷 비교가 포함된다

## 무엇을 이 절차가 검증하는가

CI 는 stdout 이 파이프로 연결되므로 실제 콘솔 코드페이지가 재현되지
않는다. 검증 대상은 셋이다.

1. 콘솔 인코딩. 레거시 cmd.exe 는 기본 코드페이지가 949 이고 그 환경에서
   한국어 출력이 깨질 수 있다
2. 경로 길이. Windows 는 전체 경로 260자 제한이 있다. 한글 슬러그를
   허용하므로(ADR 0020) 긴 제목이 긴 파일명이 된다
3. doctor 의 Windows 관련 점검 항목(파일시스템 대소문자, 콘솔 인코딩,
   core.autocrlf)이 실제로 옳게 판정하는지. 진단 도구가 틀린 진단을
   하면 존재 이유가 없다

## 바이너리 준비

개발 머신에서 교차 빌드해 실행 파일을 만든다. 두 조합 모두 필요하다.

```
GOOS=windows GOARCH=amd64 go build -o engram.exe ./cmd/engram
GOOS=windows GOARCH=arm64 go build -o engram-arm64.exe ./cmd/engram
```

x64 머신에서는 engram.exe 를, ARM 머신에서는 engram-arm64.exe 를
engram.exe 로 이름을 바꿔 쓴다. 실행 파일을 저장소의 `scripts` 디렉토리에
두면 스크립트가 인자 없이 찾는다.

## 스크립트 실행

```
powershell -ExecutionPolicy Bypass -File scripts\windows-verify.ps1
powershell -ExecutionPolicy Bypass -File scripts\windows-verify.ps1 C:\path\to\engram.exe
```

실행 정책이 스크립트를 막으면 위처럼 `-ExecutionPolicy Bypass` 로 우회한다.
PowerShell 7 를 쓴다면 `powershell` 대신 `pwsh` 로 같은 명령을 쓴다.
스크립트는 PowerShell 5.1 과 7 양쪽에서 동작한다.

임시 디렉토리는 스크립트가 만들고 끝나면 지운다. FAIL 이 하나라도 있으면
지우지 않고 경로를 알리므로 현장을 그대로 확인할 수 있다.

## 콘솔 두 종류에서 각각 돌린다

같은 스크립트를 아래 두 환경에서 각각 한 번씩 실행한다.

1. 레거시 cmd.exe. `chcp` 를 먼저 실행해 949 가 나오는지 확인한다.
   949 가 아니면 `chcp 949` 로 바꾼 뒤 돌린다
2. Windows Terminal. 코드페이지 65001 (UTF-8) 환경이다

둘의 결과가 다르면 그것이 발견이다. 특히 한글 파일명과 한글 출력
항목에서 차이가 나는지 본다.

## 사람이 눈으로 확인할 항목

스크립트가 자동 판정할 수 없는 항목은 확인필요 로 남는다.

- [ ] 한글 제목 capture 의 생성 파일명이 한글과 하이픈으로 온전히 보인다.
      깨졌거나 물음표로 바뀌면 발견이다
- [ ] status 출력의 현황, 적체 압력, 다음 행동 글자가 기대 문자열과
      같은 글자로 보인다. 스크립트가 기대 문자열을 함께 인쇄한다
- [ ] 한글 출력(콘솔 직결) 항목에서 한글이 온전히 보인다. 이것이
      ADR 0026 이 동작한다는 증거다. 자세한 것은 아래 캡처와 직결의
      차이 절을 본다
- [ ] 긴 제목 capture 가 실패했다면 에러 메시지가 무엇을 하라고
      알려주는지 이해할 수 있다
- [ ] 깊은 경로에서 실패했다면 마찬가지로 메시지를 이해할 수 있다
- [ ] doctor 의 env.fs-case 가 NTFS 에서 warn(대소문자 무시) 이다
- [ ] doctor 의 env.console-encoding 이 콘솔에서 ok 다. doctor 자신도
      콘솔 직결로 돌면 engram 이 출력 코드페이지를 65001 로 전환한
      뒤이므로 ok 가 정상이다. warn 이 나오면 전환 실패다.
      스크립트처럼 출력을 파이프로 받으면 stdout 이 콘솔이 아니라는
      안내와 함께 ok 가 나온다
- [ ] doctor 의 env.git-autocrlf 가 git 저장소가 아닌 위키에서 skip 이다

## 캡처와 콘솔 직결의 차이

Windows 콘솔에는 코드페이지가 둘이다. 출력(GetConsoleOutputCP)과
입력(GetConsoleCP)이다. engram 은 stdout 이 콘솔일 때 시작하면서
출력 코드페이지를 65001 (UTF-8) 로 바꾸고 끝나면 되돌린다(ADR 0026).

이 전환은 자식 프로세스의 stdout 이 콘솔에 직결되어 있을 때만 일어난다.
PowerShell 이 출력을 파이프로 캡처하면 자식의 stdout 은 파이프고
콘솔이 아니다. 그래서 engram 은 전환을 시도하지 않고 UTF-8 바이트를
그대로 흘린다. 깨지는지 여부는 받는 쪽 인코딩이 결정한다.

- 콘솔 직결(cmd.exe 에서 바로 실행): engram 이 코드페이지를 전환하므로
  949 콘솔에서도 한국어가 정상 출력된다. 2차 검증에서 확인했다
- PowerShell 파이프 캡처: PowerShell 이 [Console]::OutputEncoding
  기준으로 자식 출력을 해석한다. 949 로 되어 있으면 UTF-8 자식 출력을
  잘못 디코딩해 깨진다. 이것은 호스트 설정 문제다. 받기 전에 아래를
  실행한다.

  [Console]::OutputEncoding = [Text.Encoding]::UTF8

검증 스크립트의 한글 출력(파이프)과 한글 출력(콘솔 직결) 항목이 이
차이를 나란히 보여준다. 직결은 온전한데 파이프가 깨지면 정상 동작이다.

## 결과 보고

스크립트 마지막 요약(PASS, FAIL, 확인필요 건수)과 위 체크리스트의
판정을 이슈에 적는다. 항목별 지시문이 알려준 관측값(코드페이지, 경로
길이, 체크아웃 전후 위반 수)을 함께 남긴다. FAIL 이 있으면 스크립트가
알려준 작업 디렉토리 경로를 적고 지우지 않는다.

두 콘솔 환경의 결과가 다른 항목은 어느 환경의 결과인지 반드시 명시한다.
