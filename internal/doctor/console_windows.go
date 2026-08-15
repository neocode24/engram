//go:build windows

package doctor

import (
	"syscall"
	"unsafe"
)

// probeConsole는 현재 프로세스의 콘솔 상태를 읽는다. 출력 코드페이지는
// GetConsoleOutputCP, 입력 코드페이지는 GetConsoleCP 에서 온다. 둘은
// 다르므로 섞어 쓰면 진단이 자기 출력과 모순된다. ADR 0026.
// stdout 이 콘솔인지는 GetConsoleMode 가 성공하는지로 판정한다.
// API 호출이 실패해도 죽지 않고 0 값을 반환해 진단이 조치를 안내하게 둔다.
func probeConsole() consoleState {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")
	getConsoleOutputCP := kernel32.NewProc("GetConsoleOutputCP")
	getConsoleCP := kernel32.NewProc("GetConsoleCP")

	var mode uint32
	r, _, _ := getConsoleMode.Call(uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&mode)))
	out, _, _ := getConsoleOutputCP.Call()
	in, _, _ := getConsoleCP.Call()
	return consoleState{
		IsConsole: r != 0,
		OutputCP:  uint32(out),
		InputCP:   uint32(in),
	}
}
