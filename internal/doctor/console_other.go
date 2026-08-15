//go:build !windows

package doctor

// probeConsole는 Windows 외 플랫폼의 스텁이다. 콘솔 상태 진단은
// checkConsoleEncoding 이 Windows 일 때만 부르므로 여기 값은 쓰이지
// 않는다. 컴파일을 맞추기 위한 구현이다.
func probeConsole() consoleState {
	return consoleState{IsConsole: true, OutputCP: 65001, InputCP: 65001}
}
