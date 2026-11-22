package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/pin"
	"github.com/spf13/cobra"
)

// flagPin은 문서를 넣는 세 명령(capture, source, promote --to sources)이 받는 선언 플래그 이름이다.
const flagPin = "pin"

// addPinFlag는 선언 플래그를 단다. 문서를 넣는 세 명령에서 필수다.
func addPinFlag(cmd *cobra.Command) {
	cmd.Flags().String(flagPin, "", i18n.T("cli.pin.flag_pin"))
}

// pinPlan은 문서를 넣는 명령 하나가 Pin 에 맡기는 일이다.
type pinPlan struct {
	wiki string
	decl *pin.Declaration
}

// planPin은 --pin 인자를 읽는다. pin/ 이 없는 위키는 아직 갱신하지 않았으므로 아무것도 쓰지 않고 init 을
// 다시 실행하라고 안내하며 끝낸다. 인자가 없을 때도 아무것도 쓰지 않는다. 에러 문구가 pin list 를 보고
// 선언을 만들라고 안내한다.
func planPin(cmd *cobra.Command, root string) (*pinPlan, error) {
	if !pin.Present(root) {
		return nil, errors.New(i18n.T("cli.pin.not_updated"))
	}
	raw, err := stringFlag(cmd, flagPin)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, errors.New(i18n.T("cli.pin.required", cmd.CommandPath()))
	}
	text := raw
	if raw != "none" {
		text, err = readPinSource(cmd, raw)
		if err != nil {
			return nil, err
		}
	}
	decl, err := pin.ParseDeclaration(text)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("cli.pin.decl_invalid"), err)
	}
	return &pinPlan{wiki: root, decl: decl}, nil
}

// readPinSource는 선언 파일을 읽는다. `-` 는 표준 입력이다.
func readPinSource(cmd *cobra.Command, name string) (string, error) {
	var data []byte
	var err error
	if name == "-" {
		data, err = io.ReadAll(cmd.InOrStdin())
	} else {
		data, err = os.ReadFile(name)
	}
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T("cli.pin.decl_read_fail", name), err)
	}
	return string(data), nil
}

// pinDoc은 지금 쓰는 문서를 선언의 근거로 쓸 수 있게 한 것이다. path 는 위키 루트 상대 슬래시 경로다.
func pinDoc(root, path, stage, date, text string) pin.Doc {
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		rel = path
	}
	if len(date) == 7 { // 연월 정밀도의 날짜는 근거의 날짜로 쓸 수 없다. 그 달 1일로 둔다
		date += "-01"
	}
	return pin.Doc{Path: filepath.ToSlash(rel), Stage: stage, Date: date, Text: text}
}

// pinOutcome은 --json 출력에 더하는 Pin 결과다.
type pinOutcome struct {
	None      bool     `json:"none,omitempty"`
	Items     []string `json:"items,omitempty"`
	Proposals []string `json:"proposals,omitempty"`
}

func pinOutcomeOf(rep *pin.Report) *pinOutcome {
	if rep == nil || rep.DryRun {
		return nil
	}
	return &pinOutcome{None: rep.None, Items: rep.Items, Proposals: rep.Proposals}
}

// printPinReport는 선언의 결과를 사람이 읽는 글로 낸다.
func printPinReport(w io.Writer, rep *pin.Report) {
	if rep == nil || rep.DryRun {
		return
	}
	switch {
	case rep.None:
		fmt.Fprintln(w, i18n.T("cli.pin.done_none"))
	case len(rep.Items) > 0:
		fmt.Fprintln(w, i18n.T("cli.pin.done_items", strings.Join(rep.Items, ", "), strings.Join(rep.Proposals, ", ")))
	default:
		fmt.Fprintln(w, i18n.T("cli.pin.done_updated"))
	}
}

// printAffected는 옮겨진 문서를 근거로 둔 열린 일을 알린다. 닫을지는 사람이 화면에서 정한다.
func printAffected(w io.Writer, moved string, affected []pin.Affected) {
	if len(affected) == 0 {
		return
	}
	fmt.Fprintln(w, i18n.T("cli.pin.affected_head", moved, len(affected)))
	for _, a := range affected {
		fmt.Fprintln(w, i18n.T("cli.pin.affected_item", a.ID, a.Title, a.Status))
	}
	fmt.Fprintln(w, i18n.T("cli.pin.affected_tail"))
}

// pinFollow는 문서 이동 하나를 근거 경로 따라가기에 맡긴다. 아직 갱신하지 않은 위키(pin/ 이 없음)에서는
// 따라갈 근거가 없으므로 아무것도 하지 않는다. 선언이 없는 기계적 변경이라 rev 를 올리지 않는다.
func pinFollow(cmd *cobra.Command, root string, moves []pin.Move) error {
	if len(moves) == 0 || !pin.Present(root) {
		return nil
	}
	affected, err := pin.Follow(root, moves, Now(cmd))
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cli.pin.follow_fail"), err)
	}
	for _, m := range moves {
		printAffected(cmd.ErrOrStderr(), m.From, affected)
	}
	return nil
}

// ---------------------------------------------------------------------------
// engram pin
// ---------------------------------------------------------------------------

func newPinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pin",
		Short: i18n.T("cli.pin.short"),
		Long:  i18n.T("cli.pin.long"),
	}
	cmd.AddCommand(newPinListCmd(), newPinApplyCmd())
	return cmd
}

func newPinListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: i18n.T("cli.pin.list.short"),
		Long:  i18n.T("cli.pin.list.long"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			l, err := pin.List(root)
			if err != nil {
				return err
			}
			if jsonOutput(cmd) {
				return writeJSON(cmd.OutOrStdout(), l)
			}
			printPinList(cmd.OutOrStdout(), l)
			return nil
		},
	}
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.ingest.flag_wiki"))
	return cmd
}

func printPinList(w io.Writer, l *pin.Listing) {
	fmt.Fprintln(w, i18n.T("cli.pin.list.items", len(l.Items)))
	for _, it := range l.Items {
		mark := ""
		if it.Proposed {
			mark = i18n.T("cli.pin.list.proposed")
		}
		fmt.Fprintln(w, i18n.T("cli.pin.list.item", it.ID, it.Title, it.Status, it.Owner, it.Work, mark))
	}
	fmt.Fprintln(w, i18n.T("cli.pin.list.works", len(l.Works)))
	for _, wk := range l.Works {
		fmt.Fprintln(w, i18n.T("cli.pin.list.work", wk.ID, wk.Name))
	}
	if len(l.Rejected) > 0 {
		fmt.Fprintln(w, i18n.T("cli.pin.list.rejected", len(l.Rejected)))
		for _, r := range l.Rejected {
			fmt.Fprintln(w, i18n.T("cli.pin.list.rejected_item", r))
		}
	}
	if l.StaleEvidence > 0 {
		fmt.Fprintln(w, i18n.T("cli.pin.list.stale", l.StaleEvidence))
	}
}

func newPinApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: i18n.T("cli.pin.apply.short"),
		Long:  i18n.T("cli.pin.apply.long"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			file, err := stringFlag(cmd, "requests")
			if err != nil {
				return err
			}
			if file == "" {
				return errors.New(i18n.T("cli.pin.apply.no_requests"))
			}
			text, err := readPinSource(cmd, file)
			if err != nil {
				return err
			}
			reqs, err := pin.ReadRequests([]byte(text))
			if err != nil {
				return err
			}
			sum, err := pin.Apply(root, reqs, time.Now())
			if err != nil {
				return err
			}
			if jsonOutput(cmd) {
				if err := writeJSON(cmd.OutOrStdout(), sum); err != nil {
					return err
				}
			} else {
				printApplySummary(cmd.OutOrStdout(), sum)
			}
			if sum.Rejected > 0 {
				return errors.New(i18n.T("cli.pin.apply.rejected", sum.Rejected))
			}
			return nil
		},
	}
	cmd.Flags().String("requests", "", i18n.T("cli.pin.apply.flag_requests"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.ingest.flag_wiki"))
	return cmd
}

func printApplySummary(w io.Writer, s *pin.Summary) {
	for _, r := range s.Results {
		fmt.Fprintln(w, i18n.T("cli.pin.apply.line", r.Result, r.ID, r.Code, r.Message))
	}
	fmt.Fprintln(w, i18n.T("cli.pin.apply.summary", s.Applied, s.Noop, s.Rejected, s.Skipped, len(s.Archived)))
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
