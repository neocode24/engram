# 핸즈온 실습 진행서

> 이 문서는 실습 자료의 원본입니다. 단계별 HTML 자료는 여기서 확정한 내용으로 만듭니다.
> 결정하지 않은 단계는 비워 두고, 확정할 때마다 채웁니다. HTML을 먼저 만들지 않습니다.

강의 덱([index.html](index.html))이 끝난 뒤부터가 이 문서의 범위입니다. 수강생이 손으로 하는 전부를 여덟 단계로 나눕니다.

## 용어

같은 것을 여러 이름으로 부르면 실습장에서 바로 막힙니다. 아래 표의 왼쪽 말만 씁니다.

| 쓰는 말 | 뜻 | 쓰지 않는 말 |
|---|---|---|
| 실습 단계 | 이 문서의 1단계부터 8단계까지. 손으로 하는 순서 | 단위. 강의 단위와 헷갈립니다 |
| 강의 단위 | [curriculum.md](../curriculum.md)의 여덟 단위. 강의 구성 | 챕터, 세션 |
| 실습 위키 | 수강생이 만들어 끝까지 쓰는 위키. `~/engram-wiki` | 랩, lab, 내 위키 |
| 데모 위키 | 저장소의 `examples/education`. 6단계에서만 씁니다 | 씨앗 위키, seed, 예제 위키 |
| 실습 재료 | 저장소의 `examples/materials/`. 위키에 집어넣을 원재료 | 샘플, 교재, 데이터셋 |
| 저장소 | engram 소스 저장소 | 레포, 리포지토리 |

실습 단계와 강의 단위는 번호가 어긋납니다. 강의 1단위는 덱이라 손을 쓰지 않고, 실습 1단계는 설치입니다. 대응은 아래와 같습니다.

| 실습 단계 | 강의 단위 |
|---|---|
| (없음) | 1단위 오리엔테이션 |
| 1단계 설치와 준비, 2단계 첫 위키 | 2단위 설치와 첫 위키 |
| 3단계 | 3단위 넣기 |
| 4단계 | 4단위 올리기 |
| 5단계 | 5단위 꺼내 쓰기 |
| 6단계 | 6단위 다시 만나기 |
| 7단계 | 7단위 에이전트 연동 |
| 8단계 | 8단위 운영과 공유 |

## 진행 상태

| 실습 단계 | 주제 | 상태 |
|---|---|---|
| 1 | 설치와 준비 | 확정 |
| 2 | 첫 위키 | 미확정 |
| 3 | 넣기 | 미확정 |
| 4 | 올리기 | 미확정 |
| 5 | 꺼내 쓰기 | 미확정 |
| 6 | 다시 만나기 | 미확정 |
| 7 | 에이전트 연동 | 미확정 |
| 8 | 운영과 공유 | 미확정 |

## 전체를 관통하는 결정

단계마다 되풀이하지 않도록 여기 모읍니다.

**위키는 하나입니다.** 수강생은 1단계에서 `engram init`으로 `~/engram-wiki`를 만들고 8단계까지 그것만 씁니다. 실습 재료를 거기에 집어넣고, 승급하고, 검색하고, 반출합니다. 실습장을 여럿 만들지 않습니다.

**데모 위키는 6단계에서만 씁니다.** `resurface`와 `bridge`는 문서가 쌓이고 시간이 지나야 결과가 나옵니다. 수강생이 하루에 쓴 문서 예닐곱 개로는 아무것도 나오지 않습니다. 3단계부터 5단계까지는 빈 위키에서 시작하는 것이 오히려 맞습니다. 채우는 과정 자체가 그 단계의 내용이기 때문입니다.

**실습 재료는 저장소에서만 받습니다.** 바이너리에 넣지 않습니다. 위키 도구가 위키가 아닌 파일을 뿌리는 것은 역할에 맞지 않고, 바이너리만 커집니다. 그래서 바이너리를 내려받아 설치한 수강생도 저장소를 따로 받아야 합니다.

**실습 재료는 합성 자료입니다.** 조직 식별자가 들어가지 않습니다. 저장소가 공개 예정이기 때문입니다.

**문체는 경어체입니다.** 수강생에게 직접 말하는 자료입니다. 다만 문장을 늘리지 않습니다.

---

# 1단계. 설치와 준비

## 목표

engram이 어디서든 실행되는 상태를 만들고, 실습 재료를 손에 넣습니다.

## 갈래 둘

수강생이 처한 상황이 다르므로 두 갈래로 나눕니다. 어느 쪽이든 저장소가 필요합니다. 실습 재료가 거기 있기 때문입니다.

| 갈래 | 고르는 사람 | 얻는 것 |
|---|---|---|
| A. 소스 빌드 | Go를 이미 쓰거나 설치할 수 있는 사람 | 바이너리, 실습 재료, 데모 위키를 한 번에 |
| B. 바이너리 설치 | Go를 깔고 싶지 않은 사람 | 바이너리만. 실습 재료는 저장소를 따로 받습니다 |

강사는 A를 기본으로 안내하고, Go 설치에서 막히는 수강생에게 B로 넘어가게 합니다.

## 갈래 A. 소스 빌드

전제는 둘입니다. **Go 1.26 이상**과 **git**입니다.

```
git clone https://github.com/neocode24/engram.git
cd engram
```

빌드와 배치는 운영체제마다 다릅니다.

### macOS, Linux

```
go build -o engram ./cmd/engram
sudo mkdir -p /usr/local/bin
sudo mv engram /usr/local/bin/
```

`/usr/local/bin`에 두는 이유는 macOS의 `/etc/paths` 첫 줄이 그것이고 Linux 배포판 대부분도 기본 PATH에 넣기 때문입니다. `~/bin`은 macOS에서 PATH에 들어 있지 않고 디렉토리도 없습니다.

**`go build`를 sudo로 돌리지 않습니다.** 빌드 캐시가 root 소유로 만들어져서 다음부터 일반 사용자 빌드가 실패합니다. 빌드는 사용자 권한으로 하고 옮기는 것만 sudo로 합니다.

### Windows

PowerShell에서 진행합니다.

```powershell
go build -o engram.exe .\cmd\engram
New-Item -ItemType Directory -Force "$env:LOCALAPPDATA\Programs\engram"
Move-Item engram.exe "$env:LOCALAPPDATA\Programs\engram\"
```

PATH에 넣습니다. 사용자 범위로만 바꾸므로 관리자 권한이 필요 없습니다.

```powershell
$dir = "$env:LOCALAPPDATA\Programs\engram"
$cur = [Environment]::GetEnvironmentVariable("Path", "User")
[Environment]::SetEnvironmentVariable("Path", "$cur;$dir", "User")
```

**`setx PATH ...`를 쓰지 않습니다.** 1024자에서 잘리며 기존 PATH를 날릴 수 있습니다.

터미널을 닫았다 다시 엽니다. PATH 변경은 새 프로세스부터 적용됩니다.

### sudo나 PATH 편집을 피하고 싶은 경우 (전 운영체제)

```
go install ./cmd/engram
```

`go env GOPATH`가 가리키는 곳의 `bin`에 들어갑니다. 보통 `~/go/bin`이고 Windows는 `%USERPROFILE%\go\bin`입니다. **이 디렉토리는 PATH에 자동으로 들어가지 않습니다.** `go env GOPATH`로 위치를 확인하고 PATH에 직접 넣어야 합니다.

## 갈래 B. 바이너리 설치

### 내려받기

GitHub Releases에서 자기 플랫폼 아카이브를 받습니다. 이름 규칙은 `engram_<버전>_<운영체제>_<아키텍처>` 입니다.

| 운영체제 | 아키텍처 | 확장자 |
|---|---|---|
| darwin | amd64, arm64 | `.tar.gz` |
| linux | amd64, arm64 | `.tar.gz` |
| windows | amd64, arm64 | `.zip` |

Apple Silicon Mac은 `darwin_arm64`, Intel Mac은 `darwin_amd64`입니다. 헷갈리면 `uname -m`으로 확인합니다. `arm64`가 나오면 앞쪽입니다.

같은 릴리스의 `checksums.txt`로 검증합니다.

```
shasum -a 256 engram_1.0.0_darwin_arm64.tar.gz
```

출력이 `checksums.txt`의 해당 줄과 같아야 합니다.

### 배치

압축을 풀면 `engram`(Windows는 `engram.exe`)이 나옵니다. 갈래 A와 같은 자리에 둡니다.

- macOS, Linux: `sudo mv engram /usr/local/bin/`
- Windows: `%LOCALAPPDATA%\Programs\engram\`에 두고 사용자 PATH에 추가

macOS에서 처음 실행하면 서명되지 않은 바이너리라 Gatekeeper가 막습니다. 시스템 설정의 개인정보 보호 및 보안에서 한 번 허용하거나 아래를 돌립니다.

```
xattr -d com.apple.quarantine /usr/local/bin/engram
```

### Homebrew

tap이 켜지면 아래 한 줄이 됩니다. **지금은 꺼져 있습니다.** 저장소 공개 전환과 함께 켭니다.

```
brew install neocode24/tap/engram
```

### 실습 재료 받기

바이너리만으로는 실습을 시작할 수 없습니다. 저장소를 받습니다. 빌드는 하지 않습니다.

```
git clone https://github.com/neocode24/engram.git
```

## 확인

갈래와 운영체제에 상관없이 아래 셋이 통과해야 1단계가 끝납니다.

```
engram version
engram --help
engram doctor
```

- `version`이 버전과 빌드 정보를 냅니다. 갈래 A로 빌드하면 버전이 `dev`로 나오는 것이 정상입니다.
- `--help`가 커맨드 스물여덟을 냅니다. **이 목록이 진실원입니다.** 문서와 다르면 문서가 틀린 것입니다.
- `doctor`가 환경을 진단합니다. 아직 위키가 없으므로 위키 관련 항목은 건너뜁니다.

`command not found`가 나면 PATH 문제입니다. 터미널을 새로 열었는지부터 확인합니다.

## Windows 수강생에게 추가로 안내할 것

- **Windows Terminal을 권합니다.** engram은 콘솔 코드 페이지를 UTF-8로 바꿔서 한글을 냅니다([ADR 0026](../decisions/0026-windows-console-utf8.md)). 구형 conhost에서도 깨지지 않지만 글꼴에 따라 표가 어긋나 보일 수 있습니다.
- 위키 경로를 인자로 줄 때 역슬래시를 씁니다. `engram lint C:\Users\...\engram-wiki` 형태입니다.
- 파일명에 한글을 쓰므로 경로에 공백이 있으면 따옴표로 감쌉니다.

## 강사 노트

- 1단계는 15분을 넘기지 않습니다. Go 설치에서 막히는 수강생이 나오면 그 자리에서 갈래 B로 넘깁니다. 전원이 같은 갈래일 필요가 없습니다.
- 실습 재료 위치를 각자 적어 두게 합니다. 3단계에서 계속 참조합니다.
- `engram --help` 출력을 다 같이 한 번 훑습니다. 커맨드가 넣기, 올리기, 조회, 재발견, 관리 다섯 갈래로 나뉜다는 것만 짚고 넘어갑니다. 개별 커맨드는 각 단계에서 다룹니다.

## 이 단계에서 확정한 것

| 결정 | 내용 | 이유 |
|---|---|---|
| 설치 위치 | macOS와 Linux는 `/usr/local/bin`, Windows는 `%LOCALAPPDATA%\Programs\engram` | 기본 PATH에 들어 있거나 사용자 권한으로 넣을 수 있습니다. `~/bin`은 macOS에서 둘 다 아닙니다 |
| 빌드 권한 | `go build`는 사용자 권한, 배치만 sudo | root 소유 빌드 캐시가 생기면 이후 빌드가 깨집니다 |
| 설치 갈래 | 소스 빌드와 바이너리 설치 둘로 나눕니다 | Go 설치가 진입 장벽이 되면 안 됩니다 |
| 실습 재료 배포 | 저장소 클론으로만 얻습니다 | 바이너리에 임베드하면 위키 도구가 위키 아닌 파일을 뿌리게 됩니다 |
| Windows PATH | PowerShell의 `SetEnvironmentVariable`을 씁니다 | `setx`는 PATH를 자르거나 날립니다 |

---

# 2단계. 첫 위키

미확정입니다.

# 3단계. 넣기

미확정입니다. 실습 재료(`examples/materials/`)를 만들어야 진행할 수 있습니다.

# 4단계. 올리기

미확정입니다.

# 5단계. 꺼내 쓰기

미확정입니다.

# 6단계. 다시 만나기

미확정입니다. 데모 위키를 스무 개대로 키워야 진행할 수 있습니다.

# 7단계. 에이전트 연동

미확정입니다.

# 8단계. 운영과 공유

미확정입니다.

## 관련

- [curriculum.md](../curriculum.md) 강의 단위와 이수 기준
- [index.html](index.html) 1단위 오리엔테이션 덱
- [../journeys.md](../journeys.md) 여정 24개
- [../../examples/README.md](../../examples/README.md) 데모 위키
