package embed

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/neocode24/engram/internal/index"
)

// CacheSchemaVersion는 벡터 캐시 JSON의 스키마 버전이다. 모델이나
// 풀링이나 Chars 가 바뀌면 벡터 값 자체가 달라지므로 올리고, 버전이 다른
// 캐시는 통째로 버린다.
//
// 1은 bge-m3 fp32 를 평균 풀링으로 읽던 형식이다. 2는 CLS 풀링으로
// 바꾼 것이다. 같은 모델이지만 벡터가 전혀 다르다. 1로 만든 캐시를
// 2에서 쓰면 코사인이 뭉개진 값 그대로 남는다(ADR 0074, 0075).
const CacheSchemaVersion = 2

// VectorsFileName은 위키 루트 .engram 아래의 벡터 캐시 파일 이름이다.
// .engram 은 gitignore 대상이므로 커밋되지 않는다.
const VectorsFileName = "vectors.json"

// cacheFile은 벡터 캐시의 디스크 형식이다. 사람이 열어 볼 수 있는
// JSON이다. 문서 81개에 1.7MB 규모라 이진 형식을 쓸 이유가 없다(ADR 0075).
type cacheFile struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Vectors       map[string][]float32 `json:"vectors"`
}

// LoadCache는 위키 루트의 .engram/vectors.json 을 읽는다. 파일이 없거나
// 깨졌거나 스키마 버전이 다르면 빈 캐시를 반환한다. 낡은 캐시는 에러가
// 아니라 다시 계산할 대상이므로 죽지 않는다. index.Load 와 같은 규약이다.
func LoadCache(wikiRoot string) map[string][]float32 {
	data, err := os.ReadFile(filepath.Join(wikiRoot, index.IndexDirName, VectorsFileName))
	if err != nil {
		return map[string][]float32{}
	}
	var cf cacheFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return map[string][]float32{}
	}
	if cf.SchemaVersion != CacheSchemaVersion || cf.Vectors == nil {
		return map[string][]float32{}
	}
	return cf.Vectors
}

// SaveCache는 벡터 캐시를 .engram/vectors.json 에 쓴다. keep 는 이번
// 저장 대상 키의 집합이며 keep 에 없는 항목은 버린다. 지우거나 이름을
// 바꾼 문서의 벡터가 남으면 캐시가 무한히 자란다(ADR 0075). upstream 이
// 같은 것을 한다.
func SaveCache(wikiRoot string, vectors map[string][]float32, keep map[string]bool) error {
	if err := os.MkdirAll(filepath.Join(wikiRoot, index.IndexDirName), 0o755); err != nil {
		return err
	}
	trimmed := make(map[string][]float32, len(keep))
	for k, v := range vectors {
		if keep[k] {
			trimmed[k] = v
		}
	}
	data, err := json.MarshalIndent(cacheFile{SchemaVersion: CacheSchemaVersion, Vectors: trimmed}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(wikiRoot, index.IndexDirName, VectorsFileName), data, 0o644)
}
