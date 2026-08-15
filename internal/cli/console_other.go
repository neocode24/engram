//go:build !windows

package cli

// setupConsole는 Windows 외 플랫폼에서는 아무것도 하지 않는다.
// Execute 가 같은 코드로 콘솔 준비를 부르게 하기 위한 no-op 이다.
func setupConsole() func() {
	return func() {}
}
