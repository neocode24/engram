package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/neocode24/engram/internal/embed"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/spf13/cobra"
)

// model 커맨드의 플래그 이름이다.
const (
	flagFrom   = "from"
	flagVerify = "verify"
)

// newModelCmd는 임베딩 모델을 관리하는 model 커맨드 그룹을 반환한다.
// 위키가 아니라 사용자 전역 캐시를 다루므로 --wiki 를 받지 않는다.
func newModelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: i18n.T("cli.model.short"),
		Long:  i18n.T("cli.model.long"),
	}
	cmd.AddCommand(newModelPullCmd())
	cmd.AddCommand(newModelStatusCmd())
	return cmd
}

// newModelPullCmd는 모델을 내려받는 model pull 커맨드를 반환한다.
func newModelPullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: i18n.T("cli.model.pull.short"),
		Long:  i18n.T("cli.model.pull.long"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			from, err := stringFlag(cmd, flagFrom)
			if err != nil {
				return err
			}
			dir, err := embed.ModelDir()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if from != "" {
				imported, err := embed.Import(from, dir)
				if err != nil {
					return fmt.Errorf("%s: %w", i18n.T("cli.model.import_fail"), err)
				}
				fmt.Fprintln(out, i18n.T("cli.model.import_done", len(imported), dir))
				return nil
			}
			skipped, err := embed.Download(cmd.Context(), http.DefaultClient, embed.DownloadBase, dir, modelProgress(out))
			if err != nil {
				if errors.Is(err, embed.ErrChecksum) {
					return fmt.Errorf("%s: %w", i18n.T("cli.model.checksum_fail"), err)
				}
				return fmt.Errorf("%s: %w", i18n.T("cli.model.pull_fail"), err)
			}
			// 전부 이미 있으면 그 사실만 한 줄로 알린다.
			if len(skipped) == len(embed.ModelFiles()) {
				fmt.Fprintln(out, i18n.T("cli.model.already_present", dir))
				return nil
			}
			for _, name := range skipped {
				fmt.Fprintln(out, i18n.T("cli.model.skip_file", name))
			}
			fmt.Fprintln(out, i18n.T("cli.model.pull_done", dir))
			return nil
		},
	}
	cmd.Flags().String(flagFrom, "", i18n.T("cli.model.flag_from"))
	return cmd
}

// modelStatusJSON은 model status --json 이 내는 구조다.
type modelStatusJSON struct {
	Dir      string `json:"dir"`
	Revision string `json:"revision"`
	// Verified 는 검증 결과다. --verify 없이 부르면 nil 이고,
	// 그것은 검증에 실패한 것이 아니라 검사하지 않았다는 뜻이다.
	Verified      *bool              `json:"verified"`
	Complete      bool               `json:"complete"`
	Files         []embed.FileStatus `json:"files"`
	Present       int                `json:"present"`
	TotalBytes    int64              `json:"totalBytes"`
	ExpectedBytes int64              `json:"expectedBytes"`
}

// newModelStatusCmd는 모델 상태를 보는 model status 커맨드를 반환한다.
func newModelStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: i18n.T("cli.model.status.short"),
		Long:  i18n.T("cli.model.status.long"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			verify, err := boolFlag(cmd, flagVerify)
			if err != nil {
				return err
			}
			dir, err := embed.ModelDir()
			if err != nil {
				return err
			}
			files, err := embed.Inspect(dir, verify)
			if err != nil {
				return err
			}
			var total, expected int64
			present := 0
			for _, f := range files {
				total += f.Size
				expected += f.ExpectedSize
				if f.Exists {
					present++
				}
			}
			// 온전함은 존재와 크기에 더해 검증 결과를 본다. 크기가
			// 같고 내용이 틀린 훼손을 존재와 크기만으로는 못 잡는다.
			complete := present == len(files) && total == expected
			// verified 는 요청 여부가 아니라 결과다. 검사하지 않았으면
			// nil 이다. 값이 거짓인 것과 검사하지 않은 것을 소비자가
			// 구별할 수 있어야 한다. FileStatus.ChecksumMatches 와 같다.
			var verified *bool
			if verify {
				ok := true
				for _, f := range files {
					if f.ChecksumMatches == nil || !*f.ChecksumMatches {
						ok = false
						break
					}
				}
				verified = &ok
				complete = complete && ok
			}
			res := modelStatusJSON{
				Dir: dir, Revision: embed.Revision, Verified: verified,
				Complete: complete,
				Files:    files, Present: present, TotalBytes: total, ExpectedBytes: expected,
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(res); err != nil {
					return err
				}
			} else {
				printModelStatus(cmd.OutOrStdout(), res)
			}
			// 훼손은 실패로 낸다. 화면이 온전하지 않다고 말하는데
			// 종료코드가 0 이면 doctor 와 스크립트가 통과로 읽는다.
			// 모델이 아예 없는 것은 실패가 아니다. 아직 안 받았다는
			// 정보이며 그 자리에서 할 일은 model pull 이다.
			if verified != nil && !*verified && present > 0 {
				// 상태는 이미 보고했으므로 에러 문자열을 다시 인쇄하지
				// 않는다. 종료 코드 1만 내는 것이 목적이다. doctor 가
				// 같은 방식을 쓴다.
				cmd.SilenceErrors = true
				return errors.New(i18n.T("cli.model.status.corrupt"))
			}
			return nil
		},
	}
	cmd.Flags().Bool(flagVerify, false, i18n.T("cli.model.flag_verify"))
	return cmd
}

// printModelStatus는 사람용 보고를 인쇄한다.
func printModelStatus(w io.Writer, res modelStatusJSON) {
	fmt.Fprintln(w, i18n.T("cli.model.status.dir", res.Dir))
	fmt.Fprintln(w, i18n.T("cli.model.status.revision", res.Revision))
	for _, f := range res.Files {
		fmt.Fprintf(w, "  %s  %s  %s\n",
			padRight(f.Name, 23), padRight(modelSizeCell(f), 30), modelVerifyCell(f))
	}
	fmt.Fprintln(w, i18n.T("cli.model.status.total",
		humanBytes(res.TotalBytes), humanBytes(res.ExpectedBytes), res.Present, len(res.Files)))
	switch {
	case !res.Complete:
		fmt.Fprintln(w, i18n.T("cli.model.status.incomplete"))
	case res.Verified == nil:
		// 검사하지 않았으면 온전하다고 말하지 않는다. 존재와 크기만
		// 본 결론이 그 범위를 넘으면 안 된다.
		fmt.Fprintln(w, i18n.T("cli.model.status.present"))
	default:
		fmt.Fprintln(w, i18n.T("cli.model.status.complete"))
	}
}

// modelSizeCell은 크기 칸을 만든다. 없으면 없음이라고만 쓴다.
func modelSizeCell(f embed.FileStatus) string {
	if !f.Exists {
		return i18n.T("cli.model.cell_missing")
	}
	return i18n.T("cli.model.cell_size", f.Size, f.ExpectedSize)
}

// modelVerifyCell은 검증 칸을 만든다. --verify 를 주지 않았거나 크기가
// 이미 다르면 비운다.
func modelVerifyCell(f embed.FileStatus) string {
	if f.ChecksumMatches == nil {
		return ""
	}
	if *f.ChecksumMatches {
		return i18n.T("cli.model.cell_sum_ok")
	}
	return i18n.T("cli.model.cell_sum_bad")
}

// modelProgress는 진행률 출력기를 만든다. 터미널이면 같은 줄을 갱신하고
// 터미널이 아니면 드물게 줄을 바꿔 낸다. 어느 쪽이든 \r 로 도배하지
// 않는다.
func modelProgress(w io.Writer) embed.ProgressFn {
	// 진행률은 표준 출력의 성격을 따른다. 커맨드 출력이 파이프로
	// 흘러가도 사용자 눈에는 터미널이 보이는 경우가 있으므로 writer
	// 자체가 아니라 표준 출력을 본다.
	interactive := false
	if fi, err := os.Stdout.Stat(); err == nil {
		interactive = fi.Mode()&os.ModeCharDevice != 0
	}
	var last time.Time
	lastStep := int64(-1)
	return func(name string, received, total int64) {
		done := received >= total
		pct := int64(100)
		if total > 0 {
			pct = received * 100 / total
		}
		line := i18n.T("cli.model.progress", name, humanBytes(received), humanBytes(total), pct)
		if interactive {
			if !done && time.Since(last) < 200*time.Millisecond {
				return
			}
			last = time.Now()
			fmt.Fprintf(w, "\r%s", line)
			if done {
				fmt.Fprintln(w)
			}
			return
		}
		// 터미널이 아니면 10% 경계마다 한 줄씩 낸다.
		if step := pct / 10; done || step > lastStep {
			lastStep = step
			fmt.Fprintln(w, line)
		}
	}
}

// humanBytes는 진행률에 낼 사람이 읽는 크기다. status 의 정확한
// 숫자와 달리 눈대중용이다.
func humanBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
