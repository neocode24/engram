package cli

import "testing"

// TestSetupConsoleNoop는 콘솔 준비가 부작용 없이 동작하는지 본다.
// macOS 와 리눅스에서는 no-op 이고 복원 함수도 아무것도 하지 않는다.
// Windows 의 실제 동작은 사용자가 VM 에서 확인한다.
func TestSetupConsoleNoop(t *testing.T) {
	restore := setupConsole()
	if restore == nil {
		t.Fatal("복원 함수가 nil 이면 안 됨")
	}
	restore()
	restore()
}
