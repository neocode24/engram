// Package modelfetch은 크기와 체크섬을 고정한 모델 파일을 받고 검증한다.
//
// 표준 라이브러리만 쓴다. 그것이 이 패키지가 따로 있는 이유다.
// internal/embed 에 두면 그 패키지의 추론 의존(hugot, gomlx)이 함께
// 링크되어, 임베딩을 쓰지 않는 바이너리가 추론 스택을 통째로 지고
// 간다. voice 모듈이 그 경우다(ADR 0080, 0081).
//
// 어떤 모델인지는 이 패키지가 모른다. 호출자가 ModelFile 목록을 준다.
package modelfetch

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// 내려받기 실패의 종류다. 호출자가 문구를 고르게 보내는 값이다.
var (
	// ErrChecksum은 받은 파일의 sha256 이 기대값과 다를 때의 오류다.
	ErrChecksum = errors.New("체크섬 불일치")
	// ErrSize은 받은 파일의 크기가 기대값과 다를 때의 오류다.
	ErrSize = errors.New("크기 불일치")
	// ErrMissingFile은 오프라인 자료에 기대한 파일이 없을 때의 오류이다.
	ErrMissingFile = errors.New("파일 없음")
)

// ModelFile은 모델을 이루는 파일 하나의 기대값이다.
type ModelFile struct {
	// Base는 이 파일만의 내려받기 URL 앞부분이다. 비면 호출자가 준
	// 기본 base 를 쓴다.
	//
	// 모델 하나가 호스트 하나에서 오지 않는 경우가 있다. 음성 모델이
	// 그렇다. whisper 는 HuggingFace 에, 화자 임베딩은 GitHub 릴리스에
	// 있다(ADR 0081). 그때 파일마다 호스트를 준다.
	Base string
	// Remote는 base 뒤에 붙는 경로다.
	Remote string
	// Name은 모델 디렉토리 안에 놓이는 파일 이름이다. Remote 와 달리
	// 평평하다. 원격의 디렉토리 계층을 만들지 않는다.
	Name string
	// Size는 바이트 크기다.
	Size int64
	// SHA256은 기대 체크섬이다.
	SHA256 string
}

// ProgressFn은 내려받기 진행률을 받는 콜백이다. received 가 total 에
// 이르면 그 파일이 끝난 것이다. 이미 있어 건너뛴 파일에는 부르지
// 않는다. 건너뜀은 Download 의 반환값으로 안다.
type ProgressFn func(name string, received, total int64)

// Download는 files 를 dir 로 받는다. 이미 있고 체크섬이 맞은 파일은
// 받지 않고 그 이름을 돌려준다. 파일에 Base 가 있으면 그것을 쓰고
// 없으면 인자로 받은 base 를 쓴다.
//
// 이어받기는 Range 로 한다. 부분 파일이 있으면 그 지점부터 받고 서버가
// 206 을 주지 않으면 처음부터 받는다. Ctrl-C 로 죽어도 다음 실행이 이어
// 받을 수 있도록 임시 파일을 쓰지 않고 대상 파일에 직접 쓰며 검증은
// 다 받은 뒤에 한다. hugot 의 DownloadModel 을 쓰지 않는 이유는
// 체크섬 고정도 이어받기도 진행률도 호출자 손에 없기 때문이다(ADR 0074).
func Download(ctx context.Context, client *http.Client, base, dir string, files []ModelFile, prog ProgressFn) ([]string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var skipped []string
	for _, f := range files {
		skip, err := downloadFile(ctx, client, base, dir, f, prog)
		if err != nil {
			return skipped, fmt.Errorf("%s: %w", f.Name, err)
		}
		if skip {
			skipped = append(skipped, f.Name)
		}
	}
	return skipped, nil
}

// downloadFile은 파일 하나를 받는다. 이미 기대값과 같으면 참을 낸다.
func downloadFile(ctx context.Context, client *http.Client, base, dir string, f ModelFile, prog ProgressFn) (bool, error) {
	dst := filepath.Join(dir, f.Name)
	ok, err := fileMatches(dst, f)
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}
	offset, err := resumeOffset(dst, f)
	if err != nil {
		return false, err
	}
	if f.Base != "" {
		base = f.Base
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/"+f.Remote, nil)
	if err != nil {
		return false, err
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusPartialContent:
		// 이어받는다.
	case http.StatusOK:
		// 서버가 Range 를 무시했다. 처음부터 받는다.
		offset = 0
	default:
		return false, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	received, err := writeBody(resp.Body, dst, offset, f, prog)
	if err != nil {
		return false, err
	}
	if received != f.Size {
		return false, fmt.Errorf("%w: %d/%d 바이트", ErrSize, received, f.Size)
	}
	if err := verifyDownloaded(dst, f); err != nil {
		return false, err
	}
	return false, nil
}

// resumeOffset은 이어받기를 시작할 위치를 정한다. 부분 파일만 이어
// 받는다. 크기가 이미 같거나 더 큰 파일은 체크섬이 틀렸다는 뜻이므로
// 처음부터 다시 받는다.
func resumeOffset(path string, f ModelFile) (int64, error) {
	fi, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if fi.Size() >= f.Size {
		return 0, nil
	}
	return fi.Size(), nil
}

// writeBody는 응답 몸통을 대상 파일에 쓴다. 이어받기면 offset 으로
// 가고 처음부터 받으면 비운다. 임시 파일을 쓰지 않고 직접 쓰는 이유는
// Download 의 주석에 있다.
func writeBody(body io.Reader, dst string, offset int64, f ModelFile, prog ProgressFn) (int64, error) {
	out, err := openForResume(dst, offset)
	if err != nil {
		return 0, err
	}
	buf := make([]byte, 64*1024)
	received := offset
	for {
		n, rerr := body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				_ = out.Close()
				return 0, werr
			}
			received += int64(n)
			if prog != nil {
				prog(f.Name, received, f.Size)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			_ = out.Close()
			return 0, rerr
		}
	}
	if err := out.Close(); err != nil {
		return 0, err
	}
	return received, nil
}

// openForResume은 대상 파일을 받기 위해 연다. 이어받기면 그 지점으로
// 가고 처음부터 받으면 비운다.
func openForResume(dst string, offset int64) (*os.File, error) {
	if offset == 0 {
		return os.Create(dst)
	}
	out, err := os.OpenFile(dst, os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	if _, err := out.Seek(offset, io.SeekStart); err != nil {
		_ = out.Close()
		return nil, err
	}
	return out, nil
}

// verifyDownloaded은 다 받은 파일이 기대값과 같은지 본다. 크기를 먼저
// 보고 같을 때만 해시를 읽는다.
func verifyDownloaded(path string, f ModelFile) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.Size() != f.Size {
		return fmt.Errorf("%w: %d/%d 바이트", ErrSize, fi.Size(), f.Size)
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return err
	}
	if sum != f.SHA256 {
		return ErrChecksum
	}
	return nil
}

// fileMatches은 파일이 기대값과 같은지 본다. 크기가 다르면 해시를
// 읽지 않고 바로 가른다. 2.3GB 를 읽는 일은 크기가 같을 때만 한다.
func fileMatches(path string, f ModelFile) (bool, error) {
	fi, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if fi.Size() != f.Size {
		return false, nil
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return false, err
	}
	return sum == f.SHA256, nil
}

// fileSHA256은 파일의 sha256 을 흐르게 계산한다.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// FileStatus는 model status 가 낼 파일 하나의 상태다.
type FileStatus struct {
	Name            string `json:"name"`
	Exists          bool   `json:"exists"`
	Size            int64  `json:"size"`
	ExpectedSize    int64  `json:"expectedSize"`
	SizeMatches     bool   `json:"sizeMatches"`
	ExpectedSHA256  string `json:"expectedSha256"`
	ChecksumMatches *bool  `json:"checksumMatches,omitempty"`
}

// Inspect는 dir 안 모델 파일의 상태를 낸다. verify 가 참일 때만
// sha256 을 계산한다. 기가바이트를 읽는 비용을 기본으로 물지
// 않게 하기 위해서다.
func Inspect(dir string, files []ModelFile, verify bool) ([]FileStatus, error) {
	out := make([]FileStatus, 0, len(files))
	for _, f := range files {
		s := FileStatus{Name: f.Name, ExpectedSize: f.Size, ExpectedSHA256: f.SHA256}
		path := filepath.Join(dir, f.Name)
		fi, err := os.Stat(path)
		switch {
		case err == nil:
			s.Exists = true
			s.Size = fi.Size()
			s.SizeMatches = fi.Size() == f.Size
		case errors.Is(err, os.ErrNotExist):
			// 없음. 아래 칸을 비운 채로 낸다.
		default:
			return nil, err
		}
		// 크기가 다른 파일은 해시를 읽지 않는다. 크기란이 이미
		// 사실을 말하고 있어 검증 칸을 채울 새 정보가 없다.
		if verify && s.SizeMatches {
			ok, err := fileMatches(path, f)
			if err != nil {
				return nil, err
			}
			s.ChecksumMatches = &ok
		}
		out = append(out, s)
	}
	return out, nil
}

// Import는 오프라인 자료에서 files 를 가져와 dir 에 놓는다. src 는
// 디렉토리 또는 tar 아카이브다. tar 는 묶음 그대로와 gzip 으로 묶은
// 것 둘 다 받는다. 항목 이름은 평평한 파일 이름이거나 저장소처럼
// onnx/ 아래에 있는 경로다. 어떤 배치로 와도 ModelDir 에는 평평하게
// 놓는다. 가져온 파일은 내려받기와 같은 검증을 지나므로 내용이 기대값과
// 다르면 실패한다. 가져다 놓은 파일 이름 목록을 돌려준다.
func Import(src, dir string, files []ModelFile) ([]string, error) {
	fi, err := os.Stat(src)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return importFromDir(src, dir, files)
	}
	return importFromTar(src, dir, files)
}

// importFromDir는 디렉토리에서 가져온다. 파일마다 평평한 배치와
// 저장소 배치(onnx/ 아래) 둘을 차례로 본다.
func importFromDir(from, into string, files []ModelFile) ([]string, error) {
	var done []string
	for _, f := range files {
		src, err := importSourcePath(from, f)
		if err != nil {
			return nil, err
		}
		if src == "" {
			return nil, fmt.Errorf("%w: %s", ErrMissingFile, f.Name)
		}
		if err := copyVerified(src, filepath.Join(into, f.Name), f); err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
		done = append(done, f.Name)
	}
	return done, nil
}

// importSourcePath는 from 안에서 f 를 찾은 경로를 낸다. 없으면 빈
// 문자열이다.
func importSourcePath(from string, f ModelFile) (string, error) {
	for _, rel := range []string{f.Name, f.Remote} {
		p := filepath.Join(from, filepath.FromSlash(rel))
		fi, err := os.Stat(p)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if !fi.IsDir() {
			return p, nil
		}
	}
	return "", nil
}

// copyVerified는 src 를 dst 로 복사하고 기대값을 검증한다.
func copyVerified(src, dst string, f ModelFile) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	return writeVerified(in, dst, f)
}

// writeVerified는 r 을 dst 에 쓰고 기대값을 검증한다.
func writeVerified(r io.Reader, dst string, f ModelFile) error {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, r); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return verifyDownloaded(dst, f)
}

// importFromTar는 tar 아카이브에서 가져온다. 기대한 목록과 이름이
// 맞는 항목만 풀고 나머지는 건너뛴다. 경계 밖 경로는 애초에 목록과
// 이름이 같지 않으므로 풀리지 않는다.
func importFromTar(src, into string, files []ModelFile) ([]string, error) {
	archive, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer archive.Close()
	// gzip 묶음인지는 파일 머리 두 바이트로 가린다.
	br := bufio.NewReader(archive)
	var r io.Reader = br
	if magic, _ := br.Peek(2); len(magic) == 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		zr, err := gzip.NewReader(br)
		if err != nil {
			return nil, err
		}
		defer zr.Close()
		r = zr
	}
	tr := tar.NewReader(r)
	seen := map[string]bool{}
	var done []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		mf, ok := matchTarEntry(hdr.Name, files)
		if !ok || seen[mf.Name] {
			continue
		}
		seen[mf.Name] = true
		if err := writeVerified(tr, filepath.Join(into, mf.Name), mf); err != nil {
			return nil, fmt.Errorf("%s: %w", mf.Name, err)
		}
		done = append(done, mf.Name)
	}
	for _, f := range files {
		if !seen[f.Name] {
			return nil, fmt.Errorf("%w: %s", ErrMissingFile, f.Name)
		}
	}
	return done, nil
}

// matchTarEntry는 아카이브 항목 이름이 기대한 파일 하나와 맞는지 본다.
// 평평한 이름과 저장소 경로 둘 다 받는다.
func matchTarEntry(name string, files []ModelFile) (ModelFile, bool) {
	clean := path.Clean(name)
	if path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return ModelFile{}, false
	}
	for _, f := range files {
		if clean == f.Name || clean == f.Remote {
			return f, true
		}
	}
	return ModelFile{}, false
}
