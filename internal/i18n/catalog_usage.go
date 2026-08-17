// Package i18n의 usage 카탈로그. cobra Use 필드의 인자 자리표시자다.
// 커맨드 이름은 첫 토큰이라 번역 대상이 아니고 뒤의 자리표시자만 담는다.
// ADR 0049.
package i18n

func init() {
	Register(LangKO, map[string]string{
		"usage.args.path":        "<경로>",
		"usage.args.path_opt":    "[경로]",
		"usage.args.slug":        "<슬러그>",
		"usage.args.slugs_opt":   "[슬러그...]",
		"usage.args.content":     "[내용]",
		"usage.args.query":       "<질의>",
		"usage.args.title":       "<제목>",
		"usage.args.slug_rename": "<옛슬러그> <새슬러그>",
	})
	Register(LangEN, map[string]string{
		"usage.args.path":        "<path>",
		"usage.args.path_opt":    "[path]",
		"usage.args.slug":        "<slug>",
		"usage.args.slugs_opt":   "[slug...]",
		"usage.args.content":     "[content]",
		"usage.args.query":       "<query>",
		"usage.args.title":       "<title>",
		"usage.args.slug_rename": "<old-slug> <new-slug>",
	})
}
