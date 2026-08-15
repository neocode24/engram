# Windows 실환경 검증 스크립트. CI 가 못 잡는 것만 검증한다.
# 대상: 콘솔 인코딩, 경로 길이, doctor 의 Windows 판정, autocrlf.
# 사용법: powershell -ExecutionPolicy Bypass -File scripts\windows-verify.ps1 [실행 파일 경로]
# 인자를 주지 않으면 스크립트와 같은 디렉토리의 engram.exe 를 쓴다.
param(
    [string]$EngramPath = ""
)

$ErrorActionPreference = "Continue"

# 실행 파일 경로를 정한다.
if ($EngramPath -eq "") {
    $EngramPath = Join-Path $PSScriptRoot "engram.exe"
}
if (-not (Test-Path $EngramPath)) {
    Write-Output ("실행 파일이 없습니다: " + $EngramPath)
    Write-Output "교차 빌드로 만들어 다시 실행하세요. 명령은 docs/windows-verification.md 에 있습니다."
    exit 2
}
$EngramPath = (Resolve-Path $EngramPath).Path

# 결과를 모은다. 판정은 PASS, FAIL, 확인필요, SKIP 넷이다.
$Results = New-Object System.Collections.ArrayList
$WorkRoot = Join-Path $env:TEMP ("engram-verify-" + $PID)
$KeepWork = $false

function Add-Result {
    param([string]$Name, [string]$Verdict, [string]$Note)
    [void]$Results.Add(@{ Name = $Name; Verdict = $Verdict; Note = $Note })
    Write-Output ("[" + $Verdict + "] " + $Name)
    if ($Note -ne "") {
        Write-Output ("      " + $Note)
    }
    Write-Output ""
}

# Invoke-Engram 은 engram 을 실행해 출력을 $script:LastOutput 에 남기고
# 종료 코드를 반환한다. 출력은 화면에도 찍어 눈으로 보게 한다.
function Invoke-Engram {
    param([string[]]$Arguments)
    $script:LastOutput = (& $script:EngramPath @Arguments 2>&1 | Out-String)
    return $LASTEXITCODE
}

# 콘솔 직결 실행은 함수로 감싸지 않는다. PowerShell 에서 함수 호출 결과를
# 변수에 대입하면 그 함수 안의 네이티브 명령 stdout 까지 함께 캡처되어
# 자식 stdout 이 콘솔이 아니게 된다. 그러면 ADR 0026 의 코드페이지 전환
# 조건 자체가 성립하지 않는다. 호출부에서 & 로 직접 부르고
# $LASTEXITCODE 만 따로 읽는다.

function Show-Output {
    Write-Output "----- 실행 출력 -----"
    Write-Output $script:LastOutput
    Write-Output "---------------------"
}

# 작업 디렉토리를 만든다.
[void](New-Item -ItemType Directory -Force -Path $WorkRoot)

# 1. 현재 콘솔 코드페이지
$chcp = (cmd /c chcp 2>&1 | Out-String).Trim()
$codeNum = ""
if ($chcp -match "(\d+)") { $codeNum = $Matches[1] }
$codeNote = "chcp 결과: " + $chcp
if ($codeNum -eq "949") {
    $codeNote = $codeNote + " (레거시 cmd.exe 기본값입니다. engram 이 콘솔에 직접 쓸 때만 UTF-8 로 전환하므로 5-1 항목의 한글이 온전해야 합니다)"
} elseif ($codeNum -eq "65001") {
    $codeNote = $codeNote + " (UTF-8)"
}
Add-Result "콘솔 코드페이지" "PASS" $codeNote

# 2. engram version
$code = Invoke-Engram @("version")
if ($code -eq 0) {
    Add-Result "engram version" "PASS" ("종료 코드 " + $code)
} else {
    Add-Result "engram version" "FAIL" ("종료 코드 " + $code)
    Show-Output
    exit 1
}

# 3. 임시 디렉토리에 init. 생성물 확인.
$WikiA = Join-Path $WorkRoot "wiki-a"
$code = Invoke-Engram @("init", $WikiA)
$missing = @()
foreach ($item in @("engram.yaml", "index.md", ".gitignore", "inbox", "sources", "context", "archive")) {
    if (-not (Test-Path (Join-Path $WikiA $item))) { $missing += $item }
}
if ($code -eq 0 -and $missing.Count -eq 0) {
    Add-Result "init 생성물" "PASS" ("생성물 7종 모두 있음. 위치: " + $WikiA)
} else {
    $KeepWork = $true
    Add-Result "init 생성물" "FAIL" ("종료 코드 " + $code + ", 없는 생성물: " + ($missing -join ", "))
    Show-Output
}

# 4. 한글 제목 capture. 슬러그를 따로 주지 않아 제목이 파일명이 되게 한다.
$hangulTitle = "한글 제목 검증 문서"
$hangulSlug = "한글-제목-검증-문서"
$code = Invoke-Engram @("capture", "--wiki", $WikiA, "--title", $hangulTitle, "한글 제목이 파일명으로 정상 내려가는지 본다.")
$hangulFile = Get-ChildItem -Path (Join-Path $WikiA "inbox") -Filter ("*" + $hangulSlug + ".md") -ErrorAction SilentlyContinue | Select-Object -First 1
if ($code -ne 0 -or $hangulFile -eq $null) {
    $KeepWork = $true
    Add-Result "한글 제목 capture" "FAIL" ("종료 코드 " + $code + ". 기대 파일명: <날짜>-" + $hangulSlug + ".md")
    Show-Output
} else {
    Add-Result "한글 제목 capture" "확인필요" ("기대 파일명은 <날짜>-" + $hangulSlug + ".md 입니다. 실제 파일명이 한글과 하이픈으로 온전히 보이는지 확인하세요. 깨졌거나 물음표로 바뀌면 발견입니다. 파일명: " + $hangulFile.Name)
}

# 5. 한글 본문 출력. 화면에 기대 문자열을 함께 찍어 대조한다.
# 이 항목은 파이프로 캡처한 출력이다. 자식 stdout 이 콘솔이 아니므로
# 콘솔 코드페이지 전환 대상이 아니다.
$code = Invoke-Engram @("status", $WikiA)
if ($code -ne 0) {
    $KeepWork = $true
    Add-Result "한글 출력(파이프, 호스트 기본 인코딩)" "FAIL" ("status 종료 코드 " + $code)
} else {
    Write-Output "기대 문자열 (스크립트가 직접 출력): 현황 / 적체 압력 / 다음 행동"
    Show-Output
    Add-Result "한글 출력(파이프, 호스트 기본 인코딩)" "확인필요" "여기서 깨지는 것이 정상입니다. engram 은 UTF-8 바이트를 그대로 내고 PowerShell 5.1 호스트는 그것을 CP949 로 읽습니다. engram 결함이 아니라 받는 쪽 설정 문제이며 5-1 과 5-2 로 갈라 봅니다."
}

# 5-1. 같은 커맨드를 콘솔에 직접 출력한다. 파이프도 리다이렉트도 걸지 않는다.
# 여기서만 자식 stdout 이 콘솔이므로 ADR 0026 의 코드페이지 전환이 걸린다.
Write-Output "----- 콘솔 직결 출력 -----"
& $EngramPath "status" $WikiA
$codeDirect = $LASTEXITCODE
Write-Output "--------------------------"
if ($codeDirect -ne 0) {
    $KeepWork = $true
    Add-Result "한글 출력(콘솔 직결)" "FAIL" ("status 종료 코드 " + $codeDirect)
} else {
    Add-Result "한글 출력(콘솔 직결)" "확인필요" "기대: 현황, 적체 압력, 다음 행동 문장이 한글로 온전히 보입니다. 이 항목이 ADR 0026 의 본 검증입니다. 여기가 깨지면 발견입니다."
}

# 5-2. 파이프로 캡처하되 호스트 출력 인코딩을 UTF-8 로 맞춘다.
# 5번이 engram 결함인지 호스트 설정 문제인지 이 항목이 갈라 준다.
$prevEnc = [Console]::OutputEncoding
try {
    # BOM 없는 UTF-8 로 준다. [Text.Encoding]::UTF8 은 preamble 을 내보내
    # 캡처 문자열 앞에 보이지 않는 문자가 붙는다.
    [Console]::OutputEncoding = (New-Object System.Text.UTF8Encoding $false)
    $codeUtf = Invoke-Engram @("status", $WikiA)
} finally {
    [Console]::OutputEncoding = $prevEnc
}
if ($codeUtf -ne 0) {
    $KeepWork = $true
    Add-Result "한글 출력(파이프, UTF-8 호스트)" "FAIL" ("status 종료 코드 " + $codeUtf)
} else {
    Show-Output
    Add-Result "한글 출력(파이프, UTF-8 호스트)" "확인필요" "기대: 한글이 온전히 보입니다. 5번은 깨지는데 여기가 온전하면 engram 은 정상이고 호스트 인코딩만 문제였다는 뜻입니다. 여기도 깨지면 engram 이 잘못된 바이트를 내는 것이므로 발견입니다."
}

# 6. 긴 제목 capture. 한글 120자 이상. 슬러그를 따로 주지 않아 제목이
# 그대로 긴 파일명이 되게 한다.
$longTitle = ("긴제목검증" * 26)
$code = Invoke-Engram @("capture", "--wiki", $WikiA, "--title", $longTitle, "경로 길이 한계 동작 확인용 문서.")
$longFile = Get-ChildItem -Path (Join-Path $WikiA "inbox") -Filter "*.md" -ErrorAction SilentlyContinue | Where-Object { $_.BaseName.Length -gt 60 } | Select-Object -First 1
if ($longFile -ne $null) {
    Add-Result "긴 제목 capture" "PASS" ("생성됨. 전체 경로 길이 " + $longFile.FullName.Length + "자")
} elseif ($code -eq 0) {
    $KeepWork = $true
    Add-Result "긴 제목 capture" "FAIL" ("종료 코드 0 인데 파일이 보이지 않습니다")
    Show-Output
} else {
    Write-Output "실패 시 출력입니다. 에러 메시지를 이해할 수 있는지 눈으로 판단하세요."
    Show-Output
    Add-Result "긴 제목 capture" "확인필요" "실패했습니다. 메시지가 무엇을 하라고 알려주는지 사람이 판단해 결과에 적으세요."
}

# 7. 깊은 경로. 중첩 디렉토리로 전체 경로를 260자에 근접하게 만든다.
$DeepRoot = $WikiA
while (($DeepRoot.Length + 60) -lt 250) {
    $DeepRoot = Join-Path $DeepRoot "level-dir-aaaaaaaaaaaaaaaaaaaaaaaaaa"
}
[void](New-Item -ItemType Directory -Force -Path $DeepRoot)
$code = Invoke-Engram @("init", $DeepRoot)
if ($code -ne 0) {
    Write-Output "실패 시 출력입니다. 에러 메시지를 이해할 수 있는지 눈으로 판단하세요."
    Show-Output
    Add-Result "깊은 경로 init" "확인필요" ("실패했습니다. 경로 길이 " + $DeepRoot.Length + "자. 메시지 판독성을 결과에 적으세요.")
} else {
    $code2 = Invoke-Engram @("capture", "--wiki", $DeepRoot, "--title", "깊은 경로 문서", "--slug", "deep-note", "깊은 경로 동작 확인용.")
    $deepFile = Get-ChildItem -Path (Join-Path $DeepRoot "inbox") -Filter "*deep-note.md" -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($code2 -eq 0 -and $deepFile -ne $null) {
        Add-Result "깊은 경로" "PASS" ("생성됨. 전체 경로 길이 " + $deepFile.FullName.Length + "자")
    } else {
        Write-Output "실패 시 출력입니다. 에러 메시지를 이해할 수 있는지 눈으로 판단하세요."
        Show-Output
        Add-Result "깊은 경로" "확인필요" ("capture 가 실패했습니다. init 경로 길이 " + $DeepRoot.Length + "자. 메시지 판독성을 결과에 적으세요.")
    }
}

# 8. promote 로 전 동선. 깨끗한 위키에서 init, capture 2건, promote 2건, lint 위반 0.
$WikiB = Join-Path $WorkRoot "wiki-b"
[void](Invoke-Engram @("init", $WikiB))
[void](Invoke-Engram @("capture", "--wiki", $WikiB, "--title", "첫 문서", "--slug", "first", "승급 동선 검증용 첫 문서."))
[void](Invoke-Engram @("capture", "--wiki", $WikiB, "--title", "둘째 문서", "--slug", "second", "승급 동선 검증용 둘째 문서."))
$firstFile = Get-ChildItem -Path (Join-Path $WikiB "inbox") -Filter "*first.md" | Select-Object -First 1
$secondFile = Get-ChildItem -Path (Join-Path $WikiB "inbox") -Filter "*second.md" | Select-Object -First 1
if ($firstFile -eq $null -or $secondFile -eq $null) {
    $KeepWork = $true
    Add-Result "promote 전 동선" "FAIL" "capture 산물 문서를 찾지 못했습니다"
    Show-Output
} else {
    $code = Invoke-Engram @("promote", $firstFile.FullName, "--wiki", $WikiB, "--related", "index", "--related", "second")
    $code2 = Invoke-Engram @("promote", $secondFile.FullName, "--wiki", $WikiB, "--related", "index", "--related", "first")
    $lintCode = Invoke-Engram @("lint", $WikiB)
    if ($code -eq 0 -and $code2 -eq 0 -and $lintCode -eq 0) {
        Add-Result "promote 전 동선" "PASS" "promote 2건과 lint 위반 0 확인"
    } else {
        $KeepWork = $true
        Add-Result "promote 전 동선" "FAIL" ("promote 종료 코드 " + $code + ", " + $code2 + ", lint 종료 코드 " + $lintCode)
        Show-Output
    }
}

# 9. doctor. 콘솔 직결로 돌린다. 파이프로 돌리면 env.console-encoding 이
# 항상 파이프 분기로 답해 콘솔 분기를 검증할 수 없다.
Write-Output "----- doctor 출력 (콘솔 직결) -----"
& $EngramPath "doctor" $WikiA
$code = $LASTEXITCODE
Write-Output "-----------------------------------"
if ($code -eq 0) {
    Add-Result "doctor 판정" "확인필요" "기대: env.fs-case 는 NTFS 기준 warn(대소문자 무시), env.console-encoding 은 콘솔 직결이므로 ok 이며 UTF-8 로 전환했다고 말합니다, env.git-autocrlf 는 위키가 git 저장소가 아니거나 git 이 없어 skip. 출력과 기대가 다르면 발견입니다."
} else {
    Add-Result "doctor 판정" "FAIL" ("종료 코드 " + $code + ". fail 항목이 있으면 그 판정이 옳은지도 결과에 적으세요.")
}

# 10. autocrlf true 상태에서 프론트매터 판정이 흔들리는지.
# git 존재를 먼저 확인한다. git 이 없으면 PowerShell 이
# CommandNotFoundException 을 던지고 LASTEXITCODE 는 이전 값에 묶여
# 거짓 성공이 나므로 파일을 지우는 단계에 진입하지 않는다.
$WikiC = Join-Path $WorkRoot "wiki-c"
[void](Invoke-Engram @("init", $WikiC))
[void](Invoke-Engram @("capture", "--wiki", $WikiC, "--title", "줄바꿈 검증", "--slug", "crlf-note", "autocrlf 검증용 문서."))
[void](Invoke-Engram @("lint", $WikiC))
$before = ($script:LastOutput -split "`n" | Where-Object { $_ -match "\[(error|warn|reject)\]" }).Count
$gitCommand = Get-Command git -ErrorAction SilentlyContinue
if ($gitCommand -eq $null) {
    Add-Result "autocrlf lint 안정성" "SKIP" "git 이 없어 이 항목은 검사하지 않습니다. 현재 커맨드는 git 없이 동작합니다. 이 항목까지 덮으려면 Git for Windows 를 설치하고 다시 돌리세요."
} else {
    $gitOk = $true
    foreach ($gitArgs in @(
            @("init", "-q", $WikiC),
            @("-C", $WikiC, "config", "user.name", "verify"),
            @("-C", $WikiC, "config", "user.email", "verify@example.com"),
            @("-C", $WikiC, "config", "core.autocrlf", "true"),
            @("-C", $WikiC, "add", "-A"),
            @("-C", $WikiC, "commit", "-q", "-m", "verify")
        )) {
        [void](& git @gitArgs 2>&1 | Out-String)
        if ($LASTEXITCODE -ne 0) { $gitOk = $false }
    }
    if (-not $gitOk) {
        Add-Result "autocrlf lint 안정성" "확인필요" "git 명령이 실패해 검증을 마치지 못했습니다. git 설치 상태를 결과에 적으세요."
    } else {
    # 작업 복사본을 지우고 다시 내려받아 autocrlf true 로 줄바꿈이 바뀌게 한다.
    Get-ChildItem -Path $WikiC -Recurse -File | Where-Object { $_.FullName -notmatch "\\\.git\\" } | Remove-Item -Force
    & git -C $WikiC checkout -- . 2>&1 | Out-Null
    $lintCode2 = Invoke-Engram @("lint", $WikiC)
    $after = ($script:LastOutput -split "`n" | Where-Object { $_ -match "\[(error|warn|reject)\]" }).Count
    if ($before -eq $after) {
        Add-Result "autocrlf lint 안정성" "PASS" ("체크아웃 전 위반 " + $before + "건, 후 " + $after + "건으로 같습니다")
    } else {
        $KeepWork = $true
        Add-Result "autocrlf lint 안정성" "FAIL" ("체크아웃 전 위반 " + $before + "건, 후 " + $after + "건으로 다릅니다. 프론트매터 판정이 흔들렸습니다")
        Show-Output
    }
    }
}

# 요약. 판정은 PASS, FAIL, 확인필요, SKIP 넷이다.
$pass = @($Results | Where-Object { $_.Verdict -eq "PASS" }).Count
$fail = @($Results | Where-Object { $_.Verdict -eq "FAIL" }).Count
$check = @($Results | Where-Object { $_.Verdict -eq "확인필요" }).Count
$skip = @($Results | Where-Object { $_.Verdict -eq "SKIP" }).Count
Write-Output "===== 요약 ====="
Write-Output ("PASS " + $pass + "건, FAIL " + $fail + "건, 확인필요 " + $check + "건, SKIP " + $skip + "건")

if ($fail -gt 0) {
    $KeepWork = $true
}
if ($KeepWork) {
    Write-Output ("작업 디렉토리를 남깁니다: " + $WorkRoot)
} else {
    Remove-Item -Recurse -Force $WorkRoot
    Write-Output "작업 디렉토리를 지웠습니다."
}
if ($fail -gt 0) { exit 1 }
exit 0
