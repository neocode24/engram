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
    Write-Output ("실행 파일이 없다: " + $EngramPath)
    Write-Output "교차 빌드로 만들어 다시 실행한다. 명령은 docs/windows-verification.md 에 있다."
    exit 2
}
$EngramPath = (Resolve-Path $EngramPath).Path

# 결과를 모은다. 판정은 PASS, FAIL, 확인필요 셋이다.
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
    $codeNote = $codeNote + " (레거시 cmd.exe 기본값. engram 이 실행 중에 콘솔을 UTF-8 로 전환하므로 아래 한글 출력 항목에서 한글이 온전히 보여야 한다)"
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
    Add-Result "한글 제목 capture" "확인필요" ("기대 파일명은 <날짜>-" + $hangulSlug + ".md 이다. 실제 파일명이 한글과 하이픈으로 온전히 보이는지 확인한다. 깨졌거나 물음표로 바뀌면 발견이다. 파일명: " + $hangulFile.Name)
}

# 5. 한글 본문 출력. 화면에 기대 문자열을 함께 찍어 대조한다.
$code = Invoke-Engram @("status", $WikiA)
if ($code -ne 0) {
    $KeepWork = $true
    Add-Result "한글 출력" "FAIL" ("status 종료 코드 " + $code)
} else {
    Write-Output "기대 문자열 (스크립트가 직접 출력): 현황 / 적체 압력 / 다음 행동"
    Show-Output
    Add-Result "한글 출력" "확인필요" "위의 기대 문자열과 실행 출력의 한글이 같은 글자로 보이는지 눈으로 비교한다. 깨짐이 있으면 코드페이지와 콘솔 종류를 결과에 적는다."
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
    Add-Result "긴 제목 capture" "FAIL" ("종료 코드 0 인데 파일이 보이지 않는다")
    Show-Output
} else {
    Write-Output "실패 시 출력. 에러 메시지를 이해할 수 있는지 눈으로 판단한다."
    Show-Output
    Add-Result "긴 제목 capture" "확인필요" "실패했다. 메시지가 무엇을 하라고 알려주는지 사람이 판단해 결과에 적는다."
}

# 7. 깊은 경로. 중첩 디렉토리로 전체 경로를 260자에 근접하게 만든다.
$DeepRoot = $WikiA
while (($DeepRoot.Length + 60) -lt 250) {
    $DeepRoot = Join-Path $DeepRoot "level-dir-aaaaaaaaaaaaaaaaaaaaaaaaaa"
}
[void](New-Item -ItemType Directory -Force -Path $DeepRoot)
$code = Invoke-Engram @("init", $DeepRoot)
if ($code -ne 0) {
    Write-Output "실패 시 출력. 에러 메시지를 이해할 수 있는지 눈으로 판단한다."
    Show-Output
    Add-Result "깊은 경로 init" "확인필요" ("실패했다. 경로 길이 " + $DeepRoot.Length + "자. 메시지 판독성을 결과에 적는다.")
} else {
    $code2 = Invoke-Engram @("capture", "--wiki", $DeepRoot, "--title", "깊은 경로 문서", "--slug", "deep-note", "깊은 경로 동작 확인용.")
    $deepFile = Get-ChildItem -Path (Join-Path $DeepRoot "inbox") -Filter "*deep-note.md" -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($code2 -eq 0 -and $deepFile -ne $null) {
        Add-Result "깊은 경로" "PASS" ("생성됨. 전체 경로 길이 " + $deepFile.FullName.Length + "자")
    } else {
        Write-Output "실패 시 출력. 에러 메시지를 이해할 수 있는지 눈으로 판단한다."
        Show-Output
        Add-Result "깊은 경로" "확인필요" ("capture 실패. init 경로 길이 " + $DeepRoot.Length + "자. 메시지 판독성을 결과에 적는다.")
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
    Add-Result "promote 전 동선" "FAIL" "capture 산물 문서를 찾지 못했다"
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

# 9. doctor. 항목별 판정 출력. 판정이 옳은지는 사람이 확인한다.
$code = Invoke-Engram @("doctor", $WikiA)
if ($code -eq 0) {
    Show-Output
    Add-Result "doctor 판정" "확인필요" "기대: env.fs-case 는 NTFS 기준 warn(대소문자 무시), env.console-encoding 은 코드페이지 65001 이면 ok, 949 면 warn, env.git-autocrlf 는 위키가 git 저장소가 아니므로 skip. 출력과 기대가 다르면 발견이다."
} else {
    Show-Output
    Add-Result "doctor 판정" "FAIL" ("종료 코드 " + $code + ". fail 항목이 있으면 그 판정이 옳은지도 결과에 적는다.")
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
    Add-Result "autocrlf lint 안정성" "SKIP" "git 이 없어 이 항목은 검사하지 않는다. 현재 커맨드는 git 없이 동작한다."
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
        Add-Result "autocrlf lint 안정성" "확인필요" "git 명령이 실패해 검증을 마치지 못했다. git 설치 상태를 결과에 적는다."
    } else {
    # 작업 복사본을 지우고 다시 내려받아 autocrlf true 로 줄바꿈이 바뀌게 한다.
    Get-ChildItem -Path $WikiC -Recurse -File | Where-Object { $_.FullName -notmatch "\\\.git\\" } | Remove-Item -Force
    & git -C $WikiC checkout -- . 2>&1 | Out-Null
    $lintCode2 = Invoke-Engram @("lint", $WikiC)
    $after = ($script:LastOutput -split "`n" | Where-Object { $_ -match "\[(error|warn|reject)\]" }).Count
    if ($before -eq $after) {
        Add-Result "autocrlf lint 안정성" "PASS" ("체크아웃 전 위반 " + $before + "건, 후 " + $after + "건으로 같다")
    } else {
        $KeepWork = $true
        Add-Result "autocrlf lint 안정성" "FAIL" ("체크아웃 전 위반 " + $before + "건, 후 " + $after + "건으로 다르다. 프론트매터 판정이 흔들렸다")
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
    Write-Output ("작업 디렉토리를 남긴다: " + $WorkRoot)
} else {
    Remove-Item -Recurse -Force $WorkRoot
    Write-Output "작업 디렉토리를 지웠다."
}
if ($fail -gt 0) { exit 1 }
exit 0
