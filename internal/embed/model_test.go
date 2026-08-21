package embed

import (
	"slices"
	"strings"
	"testing"
)

func TestModelFilesTable(t *testing.T) {
	// 리비전 4de1325 에서 실제로 받아 계산한 기대값 그대로를 못 박는다.
	// 상수 하나가 바뀌면 이 시험이 잡는다.
	want := []ModelFile{
		{Remote: "onnx/sentence_transformers.onnx", Name: "sentence_transformers.onnx", Size: 724923, SHA256: "c53a8fe59f64ae6babb972b59b6679d8173e88b378637eba495ed0f7227f3dca"},
		{Remote: "onnx/model.onnx_data", Name: "model.onnx_data", Size: 2266820608, SHA256: "1eebfb28493f67bba03ce0ef64bfdc7fc5a3bd9d7493f818bb1d78cd798416b4"},
		{Remote: "tokenizer.json", Name: "tokenizer.json", Size: 17082821, SHA256: "6710678b12670bc442b99edc952c4d996ae309a7020c1fa0096dd245c2faf790"},
		{Remote: "config.json", Name: "config.json", Size: 770, SHA256: "734a79bf12d388c1467a4e3ab625f45de7f6906cffcfb93a1eca1787504bed95"},
		{Remote: "tokenizer_config.json", Name: "tokenizer_config.json", Size: 1173, SHA256: "7e4c1cc848840aeccdd763458c18dd525eb0f795c992e00ebe9c28554e7db2d4"},
		{Remote: "special_tokens_map.json", Name: "special_tokens_map.json", Size: 964, SHA256: "8c785abebea9ae3257b61681b4e6fd8365ceafde980c21970d001e834cf10835"},
	}
	files := ModelFiles()
	if !slices.Equal(files, want) {
		t.Fatalf("기대값 표가 다릅니다:\n got: %+v\nwant: %+v", files, want)
	}
	var total int64
	for _, f := range files {
		if strings.Contains(f.Name, "/") {
			t.Errorf("저장 이름이 평평해야 함: %q", f.Name)
		}
		total += f.Size
	}
	if want := int64(2_284_631_259); total != want {
		t.Errorf("여섯의 합계가 %d여야 함: %d", want, total)
	}
}
