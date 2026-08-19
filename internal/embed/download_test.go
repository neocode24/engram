package embed

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// fakeBodies는 시험용 파일 내용 여섯을 만든다. onnx_data 는 쓰기 조각을
// 여러 번 돌게 하려고 64KB 보다 크게 잡는다.
func fakeBodies() map[string][]byte {
	return map[string][]byte{
		"onnx/model.onnx":         []byte("graph"),
		"onnx/model.onnx_data":    bytes.Repeat([]byte{0xA5}, 200_000),
		"tokenizer.json":          []byte(`{"tokens": []}`),
		"config.json":             []byte(`{"model_type": "bge-m3"}`),
		"tokenizer_config.json":   []byte(`{"do_lower_case": false}`),
		"special_tokens_map.json": []byte(`{"unk_token": "<unk>"}`),
	}
}

// fakeTable은 내용에서 크기와 sha256 을 계산해 시험용 기대값 여섯을
// 만든다.
func fakeTable(bodies map[string][]byte) []ModelFile {
	pairs := []struct{ remote, name string }{
		{"onnx/model.onnx", "model.onnx"},
		{"onnx/model.onnx_data", "model.onnx_data"},
		{"tokenizer.json", "tokenizer.json"},
		{"config.json", "config.json"},
		{"tokenizer_config.json", "tokenizer_config.json"},
		{"special_tokens_map.json", "special_tokens_map.json"},
	}
	files := make([]ModelFile, 0, len(pairs))
	for _, p := range pairs {
		b := bodies[p.remote]
		sum := sha256.Sum256(b)
		files = append(files, ModelFile{
			Remote: p.remote, Name: p.name,
			Size: int64(len(b)), SHA256: hex.EncodeToString(sum[:]),
		})
	}
	return files
}

// fileServer는 files 를 내주는 시험 서버다. Range 를 지원하며 noRange 를
// 참으로 주면 무시하고 전문으로 답한다. mutate 로 몸통을 바꿔 체크섬
// 불일치를 만든다.
type fileServer struct {
	*httptest.Server
	ranges map[string]string
	hits   map[string]int
}

func newFileServer(t *testing.T, files []ModelFile, bodies map[string][]byte, noRange bool, mutate func(name string, body []byte) []byte) *fileServer {
	t.Helper()
	fs := &fileServer{ranges: map[string]string{}, hits: map[string]int{}}
	mux := http.NewServeMux()
	for _, f := range files {
		body := bodies[f.Remote]
		if mutate != nil {
			body = mutate(f.Remote, body)
		}
		f := f
		mux.HandleFunc("/"+f.Remote, func(w http.ResponseWriter, r *http.Request) {
			fs.hits[f.Name]++
			rng := r.Header.Get("Range")
			if rng == "" || noRange {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(body)
				return
			}
			fs.ranges[f.Name] = rng
			v, ok := strings.CutPrefix(rng, "bytes=")
			if !ok {
				t.Errorf("Range 형태가 다릅니다: %q", rng)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			start, err := strconv.ParseInt(strings.TrimSuffix(v, "-"), 10, 64)
			if err != nil || start < 0 {
				t.Errorf("Range 시작점을 읽을 수 없습니다: %q", rng)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, int64(len(body))-1, int64(len(body))))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(body[start:])
		})
	}
	fs.Server = httptest.NewServer(mux)
	t.Cleanup(fs.Close)
	return fs
}

func TestDownloadModelFetchesEveryFile(t *testing.T) {
	bodies := fakeBodies()
	files := fakeTable(bodies)
	srv := newFileServer(t, files, bodies, false, nil)
	dir := t.TempDir()

	calls := 0
	skipped, err := downloadModel(context.Background(), srv.Client(), srv.URL, dir, files,
		func(name string, received, total int64) { calls++ })
	if err != nil {
		t.Fatalf("에러 없이 받아야 함: %v", err)
	}
	if len(skipped) != 0 {
		t.Errorf("건너뛴 파일이 없어야 함: %v", skipped)
	}
	for _, f := range files {
		got, err := os.ReadFile(filepath.Join(dir, f.Name))
		if err != nil {
			t.Fatalf("%s 를 받지 않았습니다: %v", f.Name, err)
		}
		if !bytes.Equal(got, bodies[f.Remote]) {
			t.Errorf("%s 내용이 다릅니다", f.Name)
		}
	}
	if calls == 0 {
		t.Error("진행률 콜백이 불리지 않았습니다")
	}
}

func TestDownloadModelResumesPartialFile(t *testing.T) {
	bodies := fakeBodies()
	files := fakeTable(bodies)
	srv := newFileServer(t, files, bodies, false, nil)
	dir := t.TempDir()
	target := filepath.Join(dir, "model.onnx_data")
	if err := os.WriteFile(target, bodies["onnx/model.onnx_data"][:100_000], 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := downloadModel(context.Background(), srv.Client(), srv.URL, dir, files, nil); err != nil {
		t.Fatalf("이어받기가 실패함: %v", err)
	}
	if got := srv.ranges["model.onnx_data"]; got != "bytes=100000-" {
		t.Errorf("보낸 Range 가 다릅니다: %q", got)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, bodies["onnx/model.onnx_data"]) {
		t.Error("이어받은 파일이 온전하지 않습니다")
	}
}

func TestDownloadModelRestartsFromScratch(t *testing.T) {
	bodies := fakeBodies()
	files := fakeTable(bodies)

	t.Run("서버가 Range 를 무시하면 처음부터 받습니다", func(t *testing.T) {
		srv := newFileServer(t, files, bodies, true, nil)
		dir := t.TempDir()
		target := filepath.Join(dir, "model.onnx_data")
		if err := os.WriteFile(target, bodies["onnx/model.onnx_data"][:100_000], 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := downloadModel(context.Background(), srv.Client(), srv.URL, dir, files, nil); err != nil {
			t.Fatalf("처음부터 받기가 실패함: %v", err)
		}
		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, bodies["onnx/model.onnx_data"]) {
			t.Error("전문을 다시 받지 않았습니다")
		}
	})

	t.Run("크기가 같지만 내용이 틀린 파일도 처음부터 받습니다", func(t *testing.T) {
		srv := newFileServer(t, files, bodies, false, nil)
		dir := t.TempDir()
		target := filepath.Join(dir, "model.onnx")
		wrong := bytes.Repeat([]byte("!"), len(bodies["onnx/model.onnx"]))
		if err := os.WriteFile(target, wrong, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := downloadModel(context.Background(), srv.Client(), srv.URL, dir, files, nil); err != nil {
			t.Fatalf("다시 받기가 실패함: %v", err)
		}
		if _, ok := srv.ranges["model.onnx"]; ok {
			t.Error("전체 크기짜리 파일에 Range 를 보냈습니다")
		}
		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, bodies["onnx/model.onnx"]) {
			t.Error("틀린 파일을 고쳐 받지 않았습니다")
		}
	})
}

func TestDownloadModelRejectsChecksumMismatch(t *testing.T) {
	bodies := fakeBodies()
	files := fakeTable(bodies)
	srv := newFileServer(t, files, bodies, false, func(name string, body []byte) []byte {
		if name != "tokenizer.json" {
			return body
		}
		tampered := slices.Clone(body)
		tampered[0] ^= 0xFF
		return tampered
	})
	_, err := downloadModel(context.Background(), srv.Client(), srv.URL, t.TempDir(), files, nil)
	if !errors.Is(err, ErrChecksum) {
		t.Fatalf("ErrChecksum 이어야 함: %v", err)
	}
}

func TestDownloadModelSkipsIntactFiles(t *testing.T) {
	bodies := fakeBodies()
	files := fakeTable(bodies)
	srv := newFileServer(t, files, bodies, false, nil)
	dir := t.TempDir()
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.Name), bodies[f.Remote], 0o644); err != nil {
			t.Fatal(err)
		}
	}

	skipped, err := downloadModel(context.Background(), srv.Client(), srv.URL, dir, files, nil)
	if err != nil {
		t.Fatalf("에러 없이 지나가야 함: %v", err)
	}
	if len(skipped) != len(files) {
		t.Fatalf("건너뛴 파일이 %d개여야 함: %v", len(files), skipped)
	}
	for _, f := range files {
		if !slices.Contains(skipped, f.Name) {
			t.Errorf("%s 가 건너뛰 목록에 없습니다", f.Name)
		}
	}
	for name, hits := range srv.hits {
		if hits != 0 {
			t.Errorf("%s 을 서버에 요청했습니다: %d회", name, hits)
		}
	}
}

func TestDownloadModelRejectsServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	files := fakeTable(fakeBodies())
	_, err := downloadModel(context.Background(), srv.Client(), srv.URL, t.TempDir(), files, nil)
	if err == nil {
		t.Fatal("서버 에러를 받아야 함")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("상태 코드가 에러에 없습니다: %v", err)
	}
}

func TestInspectModelReportsPresenceAndChecksum(t *testing.T) {
	bodies := fakeBodies()
	files := fakeTable(bodies)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "model.onnx"), bodies["onnx/model.onnx"], 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "model.onnx_data"), bodies["onnx/model.onnx_data"][:1_000], 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("기본은 존재와 크기만 봅니다", func(t *testing.T) {
		got, err := inspectModel(dir, files, false)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range got {
			if s.ChecksumMatches != nil {
				t.Errorf("%s 의 체크섬을 계산했습니다", s.Name)
			}
		}
		byName := map[string]FileStatus{}
		for _, s := range got {
			byName[s.Name] = s
		}
		if s := byName["model.onnx"]; !s.Exists || !s.SizeMatches {
			t.Errorf("model.onnx 상태가 다릅니다: %+v", s)
		}
		if s := byName["model.onnx_data"]; !s.Exists || s.SizeMatches {
			t.Errorf("일부만 받은 model.onnx_data 상태가 다릅니다: %+v", s)
		}
		if s := byName["tokenizer.json"]; s.Exists {
			t.Errorf("없는 tokenizer.json 이 있다고 나옵니다: %+v", s)
		}
	})

	t.Run("--verify 는 크기가 같은 파일만 해시합니다", func(t *testing.T) {
		// 크기는 같고 내용이 틀린 파일을 하나 섞는다.
		wrong := bytes.Repeat([]byte("!"), len(bodies["onnx/model.onnx"]))
		if err := os.WriteFile(filepath.Join(dir, "model.onnx"), wrong, 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := inspectModel(dir, files, true)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range got {
			switch s.Name {
			case "model.onnx":
				if s.ChecksumMatches == nil || *s.ChecksumMatches {
					t.Errorf("틀린 체크섬을 일치라고 합니다: %+v", s)
				}
			case "model.onnx_data":
				if s.ChecksumMatches != nil {
					t.Errorf("크기가 다른 파일의 해시를 계산했습니다: %+v", s)
				}
			default:
				if s.ChecksumMatches != nil {
					t.Errorf("없는 파일의 해시를 계산했습니다: %+v", s)
				}
			}
		}
	})
}

func TestImportModelFromDirectory(t *testing.T) {
	bodies := fakeBodies()
	files := fakeTable(bodies)

	t.Run("평평한 배치를 가져옵니다", func(t *testing.T) {
		src := t.TempDir()
		for _, f := range files {
			if err := os.WriteFile(filepath.Join(src, f.Name), bodies[f.Remote], 0o644); err != nil {
				t.Fatal(err)
			}
		}
		into := t.TempDir()
		done, err := importModel(src, into, files)
		if err != nil {
			t.Fatalf("가져오기 실패: %v", err)
		}
		if len(done) != len(files) {
			t.Fatalf("가져온 파일이 %d개여야 함: %v", len(files), done)
		}
		assertDirMatches(t, into, files, bodies)
	})

	t.Run("저장소 배치(onnx/ 아래)도 가져옵니다", func(t *testing.T) {
		src := t.TempDir()
		for _, f := range files {
			p := filepath.Join(src, filepath.FromSlash(f.Remote))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, bodies[f.Remote], 0o644); err != nil {
				t.Fatal(err)
			}
		}
		into := t.TempDir()
		if _, err := importModel(src, into, files); err != nil {
			t.Fatalf("가져오기 실패: %v", err)
		}
		assertDirMatches(t, into, files, bodies)
	})

	t.Run("파일이 모자라면 실패합니다", func(t *testing.T) {
		src := t.TempDir()
		if err := os.WriteFile(filepath.Join(src, "config.json"), bodies["config.json"], 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := importModel(src, t.TempDir(), files)
		if !errors.Is(err, ErrMissingFile) {
			t.Fatalf("ErrMissingFile 이어야 함: %v", err)
		}
	})

	t.Run("내용이 다르면 실패합니다", func(t *testing.T) {
		src := t.TempDir()
		for _, f := range files {
			body := bodies[f.Remote]
			if f.Name == "config.json" {
				// 같은 길이에 다른 내용을 넣어 크기가 아니라 체크섬이
				// 걸리게 한다.
				body = bytes.Repeat([]byte("!"), len(body))
			}
			if err := os.WriteFile(filepath.Join(src, f.Name), body, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		_, err := importModel(src, t.TempDir(), files)
		if !errors.Is(err, ErrChecksum) {
			t.Fatalf("ErrChecksum 이어야 함: %v", err)
		}
	})
}

// assertDirMatches는 dir 안 파일이 기대값과 같은지 본다.
func assertDirMatches(t *testing.T, dir string, files []ModelFile, bodies map[string][]byte) {
	t.Helper()
	for _, f := range files {
		got, err := os.ReadFile(filepath.Join(dir, f.Name))
		if err != nil {
			t.Fatalf("%s 가 없습니다: %v", f.Name, err)
		}
		if !bytes.Equal(got, bodies[f.Remote]) {
			t.Errorf("%s 내용이 다릅니다", f.Name)
		}
	}
}

func TestImportModelFromTar(t *testing.T) {
	bodies := fakeBodies()
	files := fakeTable(bodies)
	// 항목 이름은 저장소 배치로 섞고 잡동사니와 경계 밖 경로를 함께
	// 넣는다. 잡동사니는 건너뛰고 경계 밖은 풀리지 않아야 한다.
	entries := map[string][]byte{"../evil": []byte("x"), "./notes.txt": []byte("y")}
	for _, f := range files {
		entries[f.Remote] = bodies[f.Remote]
	}

	t.Run("묶음 그대로를 가져옵니다", func(t *testing.T) {
		src := buildTar(t, entries, false)
		into := t.TempDir()
		done, err := importModel(src, into, files)
		if err != nil {
			t.Fatalf("가져오기 실패: %v", err)
		}
		if len(done) != len(files) {
			t.Fatalf("가져온 파일이 %d개여야 함: %v", len(files), done)
		}
		assertDirMatches(t, into, files, bodies)
		got, err := os.ReadDir(into)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(files) {
			t.Errorf("대상 디렉토리에 파일이 %d개 있습니다: %d개여야 함", len(got), len(files))
		}
	})

	t.Run("gzip 묶음도 가져옵니다", func(t *testing.T) {
		src := buildTar(t, entries, true)
		into := t.TempDir()
		if _, err := importModel(src, into, files); err != nil {
			t.Fatalf("가져오기 실패: %v", err)
		}
		assertDirMatches(t, into, files, bodies)
	})

	t.Run("항목이 모자라면 실패합니다", func(t *testing.T) {
		partial := map[string][]byte{"config.json": bodies["config.json"]}
		src := buildTar(t, partial, false)
		_, err := importModel(src, t.TempDir(), files)
		if !errors.Is(err, ErrMissingFile) {
			t.Fatalf("ErrMissingFile 이어야 함: %v", err)
		}
	})

	t.Run("내용이 다르면 실패합니다", func(t *testing.T) {
		tampered := map[string][]byte{}
		for _, f := range files {
			tampered[f.Remote] = bodies[f.Remote]
		}
		wrong := bytes.Repeat([]byte("!"), len(bodies["config.json"]))
		tampered["config.json"] = wrong
		src := buildTar(t, tampered, false)
		_, err := importModel(src, t.TempDir(), files)
		if !errors.Is(err, ErrChecksum) {
			t.Fatalf("ErrChecksum 이어야 함: %v", err)
		}
	})
}

// buildTar은 항목을 묶어 tar 파일을 만들고 그 경로를 낸다. gz 가 참이면
// gzip 으로 한 번 더 싼다.
func buildTar(t *testing.T, entries map[string][]byte, gz bool) string {
	t.Helper()
	var buf bytes.Buffer
	var w io.Writer = &buf
	var zw *gzip.Writer
	if gz {
		zw = gzip.NewWriter(&buf)
		w = zw
	}
	tw := tar.NewWriter(w)
	for name, body := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if zw != nil {
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(t.TempDir(), "model.tar")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestModelFilesTable(t *testing.T) {
	// 리비전 4de1325 에서 실제로 받아 계산한 기대값 그대로를 못 박는다.
	// 상수 하나가 바뀌면 이 시험이 잡는다.
	want := []ModelFile{
		{"onnx/sentence_transformers.onnx", "sentence_transformers.onnx", 724923, "c53a8fe59f64ae6babb972b59b6679d8173e88b378637eba495ed0f7227f3dca"},
		{"onnx/model.onnx_data", "model.onnx_data", 2266820608, "1eebfb28493f67bba03ce0ef64bfdc7fc5a3bd9d7493f818bb1d78cd798416b4"},
		{"tokenizer.json", "tokenizer.json", 17082821, "6710678b12670bc442b99edc952c4d996ae309a7020c1fa0096dd245c2faf790"},
		{"config.json", "config.json", 770, "734a79bf12d388c1467a4e3ab625f45de7f6906cffcfb93a1eca1787504bed95"},
		{"tokenizer_config.json", "tokenizer_config.json", 1173, "7e4c1cc848840aeccdd763458c18dd525eb0f795c992e00ebe9c28554e7db2d4"},
		{"special_tokens_map.json", "special_tokens_map.json", 964, "8c785abebea9ae3257b61681b4e6fd8365ceafde980c21970d001e834cf10835"},
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
