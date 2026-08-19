package embed

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neocode24/engram/internal/index"
)

func TestCacheRoundTrip(t *testing.T) {
	root := t.TempDir()
	vecs := map[string][]float32{
		Key("문서 하나"): {1, 0, 0},
		Key("문서 둘"):  {0, 1, 0},
	}
	if err := SaveCache(root, vecs, map[string]bool{Key("문서 하나"): true, Key("문서 둘"): true}); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}
	got := LoadCache(root)
	if len(got) != 2 {
		t.Fatalf("왕복하면 벡터 둘: %d", len(got))
	}
	if v, ok := got[Key("문서 하나")]; !ok || len(v) != 3 || v[0] != 1 {
		t.Errorf("벡터 값: %+v", got[Key("문서 하나")])
	}
}

func TestCacheSaveDropsStaleKeys(t *testing.T) {
	// keep 에 없는 키는 저장할 때 버린다. 지우거나 이름을 바꾼 문서의
	// 벡터가 남으면 캐시가 무한히 자란다.
	root := t.TempDir()
	vecs := map[string][]float32{
		Key("남을 문서"):  {1},
		Key("지워진 문서"): {2},
	}
	keep := map[string]bool{Key("남을 문서"): true}
	if err := SaveCache(root, vecs, keep); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}
	got := LoadCache(root)
	if len(got) != 1 {
		t.Fatalf("keep 밖의 키는 버려야 합니다: %+v", got)
	}
	if _, ok := got[Key("지워진 문서")]; ok {
		t.Error("지워진 문서의 벡터가 남았습니다")
	}
}

func TestCacheLoadReturnsEmptyOnProblems(t *testing.T) {
	root := t.TempDir()
	// 파일이 없으면 빈 캐시.
	if got := LoadCache(root); len(got) != 0 {
		t.Errorf("파일이 없으면 빈 캐시: %+v", got)
	}
	dir := filepath.Join(root, index.IndexDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// 깨진 JSON 이어도 빈 캐시. 에러로 죽지 않는다.
	if err := os.WriteFile(filepath.Join(dir, VectorsFileName), []byte("{깨짐"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := LoadCache(root); len(got) != 0 {
		t.Errorf("깨진 파일이면 빈 캐시: %+v", got)
	}
	// 스키마 버전이 다르면 통째로 버린다.
	if err := os.WriteFile(filepath.Join(dir, VectorsFileName),
		[]byte(`{"schemaVersion":9999,"vectors":{"k":[1]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := LoadCache(root); len(got) != 0 {
		t.Errorf("버전이 다르면 빈 캐시: %+v", got)
	}
}
