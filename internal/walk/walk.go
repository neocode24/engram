// Package walk는 위키 문서 순회를 담는다. lint, status, 이어지는 promote 와
// new 커맨드가 같은 순회를 쓴다. 순회 규칙(page_dirs 아래 .md 문서와
// root_files, 숨김 디렉토리 제외, 경로 정렬)이 두 벌이 되면 한쪽만 고치는
// 사고가 반드시 나므로 이곳이 단일 진실원이다.
package walk

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/i18n"
)

// ErrUnclosed 는 프론트매터가 닫는 구분자 없이 끝났음을 나타낸다.
// 위반 규칙 분류를 에러 문자열 매칭 없이 하기 위해 센티널로 둔다.
var ErrUnclosed = errors.New("프론트매터가 닫는 --- 구분자 없이 끝났다")

// Doc은 문서 하나의 순회 결과다. 경로, 원문, 파싱 결과와 파싱 오류가
// 함께 실리므로 호출자는 파일을 다시 읽거나 파싱하지 않는다.
type Doc struct {
	Rel     string  // 위키 루트 상대 슬래시 경로
	Root    bool    // root_files(색인) 문서인가
	Content string  // BOM 과 CRLF 를 정규화한 원문
	Parsed  doc.Doc // 파싱 결과. Err 이 nil 일 때만 유효하다
	Err     error   // 프론트매터 파싱 실패. ErrUnclosed 인지 errors.Is 로 구분한다
}

// Files는 위키의 문서를 경로순으로 순회해 파싱까지 마친 결과를 반환한다.
// 파싱에 실패한 문서도 목록에 남되 Err 을 싣는다. 검사한 파일 수를 셀 때
// 파싱 실패 문서가 숨지 않게 하기 위해서다.
func Files(wikiRoot string, cfg config.Config) ([]Doc, error) {
	rels, err := listFiles(wikiRoot, cfg)
	if err != nil {
		return nil, err
	}
	docs := make([]Doc, 0, len(rels))
	for _, rel := range rels {
		d, err := loadDoc(wikiRoot, rel, cfg)
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, nil
}

// listFiles는 검사 대상 .md 문서의 상대 경로를 정렬해 반환한다.
func listFiles(wikiRoot string, cfg config.Config) ([]string, error) {
	seen := map[string]bool{}
	var rels []string
	add := func(rel string) {
		if !seen[rel] {
			seen[rel] = true
			rels = append(rels, rel)
		}
	}
	dirs := append([]string{}, cfg.PageDirs...)
	sort.Strings(dirs)
	for _, d := range dirs {
		if strings.HasPrefix(d, ".") {
			continue
		}
		err := filepath.WalkDir(filepath.Join(wikiRoot, d), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			if entry.IsDir() {
				if strings.HasPrefix(entry.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(entry.Name(), ".md") && !ignoredFile(entry.Name(), cfg) {
				rel, err := filepath.Rel(wikiRoot, path)
				if err != nil {
					return err
				}
				add(filepath.ToSlash(rel))
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	for _, f := range cfg.RootFiles {
		if !strings.HasSuffix(f, ".md") {
			continue
		}
		if _, err := os.Stat(filepath.Join(wikiRoot, f)); err == nil {
			add(filepath.ToSlash(f))
		}
	}
	sort.Strings(rels)
	return rels, nil
}

// ignoredFile은 파일명이 ignore_files 목록과 완전 일치하는지를 반환한다.
// page_dirs 아래 모든 깊이에 적용한다. 디렉토리를 설명하는 README 같은
// 비문서 마크다운을 순회에서 빼는 것이 목적이다(ADR 0036). 순회를 부르는
// 모든 커맨드가 같은 답을 내도록 적용 지점은 여기 한 곳이다.
func ignoredFile(name string, cfg config.Config) bool {
	for _, f := range cfg.IgnoreFiles {
		if f == name {
			return true
		}
	}
	return false
}

// loadDoc는 문서 하나를 읽어 정규화하고 파싱한다.
func loadDoc(wikiRoot, rel string, cfg config.Config) (Doc, error) {
	rootFile := false
	for _, f := range cfg.RootFiles {
		if f == rel {
			rootFile = true
		}
	}
	raw, err := os.ReadFile(filepath.Join(wikiRoot, filepath.FromSlash(rel)))
	if err != nil {
		return Doc{}, fmt.Errorf("%s: %w", i18n.T("core.walk.read_fail", rel), err)
	}
	// BOM 과 CRLF 는 여기서 한 번만 정규화한다. doc.Parse 도 스스로
	// 정규화하지만 닫는 구분자 검사가 원문을 보기 전에 끝나야 한다.
	text := strings.TrimPrefix(string(raw), "\xEF\xBB\xBF")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	d := Doc{Rel: rel, Root: rootFile, Content: text}

	// 닫는 구분자 검사를 doc.Parse 앞에 둔다. 파싱 오류의 종류를
	// 에러 문자열 매칭 없이 확정하기 위해서다.
	lines := strings.Split(text, "\n")
	if len(lines) > 0 && strings.TrimRight(lines[0], " \t") == "---" {
		closed := false
		for _, l := range lines[1:] {
			if strings.TrimRight(l, " \t") == "---" {
				closed = true
				break
			}
		}
		if !closed {
			d.Err = ErrUnclosed
			return d, nil
		}
	}

	parsed, err := doc.Parse(rel, []byte(text))
	if err != nil {
		d.Err = err
		return d, nil
	}
	d.Parsed = parsed
	return d, nil
}
