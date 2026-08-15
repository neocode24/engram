//go:build windows

package cli

import (
	"syscall"
	"unsafe"
)

// utf8CodePage는 Windows 콘솔의 UTF-8 출력 코드페이지다.
const utf8CodePage = 65001

// setupConsole는 stdout 이 콘솔일 때 출력 코드페이지를 UTF-8 로 바꾸고
// 종료 시 되돌리는 함수를 반환한다. Go 는 UTF-8 바이트를 쓰는데 레거시
// 콘솔의 기본 코드페이지(949 같은 ANSI 코드페이지)는 그것을 다르게
// 해석해 한국어 출력이 깨진다. ADR 0026.
//
// 판정과 복구 전 과정에서 API 호출이 실패해도 죽지 않는다. 출력이 깨지는
// 것이 최악이지 죽는 것보다는 낫다. stdout 이 파이프나 파일이면
// 아무것도 하지 않는다. 표준 syscall 만 쓴다.
func setupConsole() func() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")
	getConsoleOutputCP := kernel32.NewProc("GetConsoleOutputCP")
	setConsoleOutputCP := kernel32.NewProc("SetConsoleOutputCP")

	// stdout 이 콘솔인지는 GetConsoleMode 가 성공하는지로 판정한다.
	var mode uint32
	r, _, _ := getConsoleMode.Call(uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		return func() {}
	}
	prev, _, _ := getConsoleOutputCP.Call()
	if prev == utf8CodePage {
		return func() {}
	}
	ok, _, _ := setConsoleOutputCP.Call(utf8CodePage)
	if ok == 0 {
		return func() {}
	}
	return func() {
		// 되돌리기 실패는 무시한다.
		setConsoleOutputCP.Call(prev)
	}
}
