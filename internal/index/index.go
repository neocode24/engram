// Package index는 위키 검색 색인을 만들고 질의에 BM25 순위로 답한다.
// 랭킹을 소유해야 upstream parity 비교에 검색 순위를 넣을 수 있으므로
// 외부 검색 라이브러리를 쓰지 않는다. ADR 0010.
package index

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/neocode24/engram/internal/walk"
)

// SchemaVersion는 색인 JSON의 스키마 버전이다. 다르면 색인을 무시하고
// 낡은 것으로 취급한다.
const SchemaVersion = 1

// IndexDirName은 색인이 놓이는 위키 루트의 캐시 디렉토리다. gitignore 대상이다.
const IndexDirName = ".engram"

// IndexFileName은 색인 파일 이름이다.
const IndexFileName = "index.json"

// BM25 상수다. 튜닝 대상이 아니라 표준값으로 고정한다. k1=1.2, b=0.75.
const (
	bm25K1 = 1.2
	bm25B  = 0.75
)

// FieldWeights는 색인 대상 필드별 가중치다. 기본값은 전부 1.0이고
// 튜닝은 나중에 한다.
type FieldWeights struct {
	Title  float64
	Body   float64
	Topics float64
	Tags   float64
}

// DefaultWeights는 전부 1.0인 기본 가중치다.
func DefaultWeights() FieldWeights {
	return FieldWeights{Title: 1, Body: 1, Topics: 1, Tags: 1}
}

// DocEntry는 문서 하나의 색인 항목이다. 크기와 수정 시각은 신선도
// 판정용으로 실제 파일과 대조할 때 쓴다.
type DocEntry struct {
	Path    string             `json:"path"`
	Slug    string             `json:"slug"`
	Title   string             `json:"title"`
	TF      map[string]float64 `json:"tf"`
	Length  float64            `json:"length"`
	Size    int64              `json:"size"`
	ModTime int64              `json:"modTime"`
}

// Index는 위키 전체의 색인이다. 문서 수가 수백에서 2000개 규모이므로
// 사람이 열어 볼 수 있는 JSON으로 저장한다.
type Index struct {
	SchemaVersion int            `json:"schemaVersion"`
	Docs          []DocEntry     `json:"docs"`
	AvgLength     float64        `json:"avgLength"`
	DF            map[string]int `json:"df"`
}

// Build는 순회 결과로 색인을 만든다. 파싱에 실패한 문서는 건너뛴다.
// 수정 시각은 파일에서 읽는 값이며 시계를 읽지 않는다.
func Build(wikiRoot string, walked []walk.Doc, w FieldWeights) (*Index, error) {
	ix := &Index{SchemaVersion: SchemaVersion, DF: map[string]int{}}
	var total float64
	for _, wd := range walked {
		if wd.Err != nil {
			continue
		}
		tf := map[string]float64{}
		title := docTitle(wd)
		addTokens(tf, Tokenize(title), w.Title)
		addTokens(tf, Tokenize(wd.Parsed.Body), w.Body)
		for _, f := range wd.Parsed.Fields {
			switch f.Key {
			case "topics":
				addTokens(tf, Tokenize(strings.Join(f.List, " ")), w.Topics)
			case "tags":
				addTokens(tf, Tokenize(strings.Join(f.List, " ")), w.Tags)
			}
		}
		// 프론트매터의 다른 속성은 색인하지 않는다. 필터링용이지 검색어가 아니다.
		length := 0.0
		for _, n := range tf {
			length += n
		}
		fi, err := os.Stat(filepath.Join(wikiRoot, filepath.FromSlash(wd.Rel)))
		if err != nil {
			return nil, err
		}
		slug := strings.TrimSuffix(filepath.Base(wd.Rel), ".md")
		ix.Docs = append(ix.Docs, DocEntry{
			Path: wd.Rel, Slug: slug, Title: title, TF: tf, Length: length,
			Size: fi.Size(), ModTime: fi.ModTime().UnixNano(),
		})
		total += length
	}
	sort.Slice(ix.Docs, func(i, j int) bool { return ix.Docs[i].Path < ix.Docs[j].Path })
	if len(ix.Docs) > 0 {
		ix.AvgLength = total / float64(len(ix.Docs))
	}
	for _, e := range ix.Docs {
		for t := range e.TF {
			ix.DF[t]++
		}
	}
	return ix, nil
}

// addTokens는 토큰 빈도에 가중치를 곱해 더한다.
func addTokens(tf map[string]float64, tokens []string, weight float64) {
	if weight == 0 {
		return
	}
	for _, t := range tokens {
		tf[t] += weight
	}
}

// docTitle는 문서 제목을 찾는다. 본문 첫 헤딩을 우선하고 없으면 슬러그에서
// 파생한다.
func docTitle(wd walk.Doc) string {
	for _, line := range strings.Split(wd.Parsed.Body, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "#") {
			heading := strings.TrimSpace(strings.TrimLeft(t, "#"))
			heading = strings.TrimSpace(heading)
			if heading != "" {
				return heading
			}
		}
	}
	slug := strings.TrimSuffix(filepath.Base(wd.Rel), ".md")
	return strings.ReplaceAll(slug, "-", " ")
}

// SearchResult는 질의에 대한 문서 하나의 결과다.
type SearchResult struct {
	Path  string  `json:"path"`
	Slug  string  `json:"slug"`
	Title string  `json:"title"`
	Score float64 `json:"score"`
}

// Search는 질의를 BM25 순위로 검색해 상위 결과를 반환한다.
// 동점인 문서의 순위는 슬러그 순으로 고정한다. 검색 순위는 ADR 0005의
// parity 비교 축이므로 이 순서가 흔들리면 안 된다.
func (ix *Index) Search(query string, limit int) []SearchResult {
	if len(ix.Docs) == 0 || ix.AvgLength == 0 {
		return nil
	}
	// 질의 토큰은 중복을 제거해 한 번씩만 더한다.
	seen := map[string]bool{}
	var qtokens []string
	for _, t := range Tokenize(query) {
		if !seen[t] {
			seen[t] = true
			qtokens = append(qtokens, t)
		}
	}
	var results []SearchResult
	for _, e := range ix.Docs {
		score := 0.0
		for _, t := range qtokens {
			tf, ok := e.TF[t]
			if !ok {
				continue
			}
			idf := math.Log(1 + (float64(len(ix.Docs))-float64(ix.DF[t])+0.5)/(float64(ix.DF[t])+0.5))
			denom := tf + bm25K1*(1-bm25B+bm25B*e.Length/ix.AvgLength)
			score += idf * tf * (bm25K1 + 1) / denom
		}
		if score > 0 {
			results = append(results, SearchResult{Path: e.Path, Slug: e.Slug, Title: e.Title, Score: score})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Slug < results[j].Slug
	})
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// Save는 색인을 위키 루트의 .engram/index.json 에 쓴다.
// 맵 키는 encoding/json 이 정렬해서 쓰고 문서는 경로순이므로 같은 색인의
// 저장은 항상 바이트까지 같다.
func (ix *Index) Save(wikiRoot string) error {
	if err := os.MkdirAll(filepath.Join(wikiRoot, IndexDirName), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ix, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(wikiRoot, IndexDirName, IndexFileName), data, 0o644)
}

// Load는 위키 루트의 색인을 읽는다. 파일이 없거나 파싱에 실패하거나
// 스키마 버전이 다르면 nil을 반환한다. 낡은 색인은 에러가 아니라 다시
// 만들 대상이므로 죽지 않는다.
func Load(wikiRoot string) *Index {
	data, err := os.ReadFile(filepath.Join(wikiRoot, IndexDirName, IndexFileName))
	if err != nil {
		return nil
	}
	var ix Index
	if err := json.Unmarshal(data, &ix); err != nil {
		return nil
	}
	if ix.SchemaVersion != SchemaVersion {
		return nil
	}
	return &ix
}

// Fresh는 색인이 현재 문서 파일과 일치하는지 판정한다. 파싱에 실패한
// 문서는 Build가 건너뛰므로 여기서도 제외한다. 경로 집합이 다르거나
// 크기나 수정 시각이 다르면 낡았다.
func (ix *Index) Fresh(walked []walk.Doc, wikiRoot string) bool {
	current := map[string]DocEntry{}
	for _, wd := range walked {
		if wd.Err != nil {
			continue
		}
		fi, err := os.Stat(filepath.Join(wikiRoot, filepath.FromSlash(wd.Rel)))
		if err != nil {
			return false
		}
		current[wd.Rel] = DocEntry{Size: fi.Size(), ModTime: fi.ModTime().UnixNano()}
	}
	if len(current) != len(ix.Docs) {
		return false
	}
	for _, e := range ix.Docs {
		c, ok := current[e.Path]
		if !ok || c.Size != e.Size || c.ModTime != e.ModTime {
			return false
		}
	}
	return true
}
