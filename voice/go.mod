module github.com/neocode24/engram/voice

go 1.26.5

require (
	// 루트 모듈을 정식 태그로 가져온다. replace 를 두지 않는 이유는
	// go install ...@latest 가 replace 있는 모듈을 거절하기 때문이다.
	// 개발 중에는 루트의 go.work 가 이 버전을 덮어 작업 사본을 쓴다.
	// ADR 0080.
	github.com/k2-fsa/sherpa-onnx-go v1.13.6
	github.com/neocode24/engram v1.0.0
)

require (
	github.com/k2-fsa/sherpa-onnx-go-linux v1.13.6 // indirect
	github.com/k2-fsa/sherpa-onnx-go-macos v1.13.6 // indirect
	github.com/k2-fsa/sherpa-onnx-go-windows v1.13.6 // indirect
)
