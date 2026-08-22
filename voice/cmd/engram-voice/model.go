package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/modelfetch"

	"github.com/neocode24/engram/voice/internal/model"
)

// progressInterval은 진행률을 다시 그리는 간격이다. 파일 하나가
// 1GB 라 매 청크마다 그리면 출력이 진행률로 뒤덮인다.
const progressInterval = 200 * time.Millisecond

// runModel은 model 아래 커맨드를 가른다.
func runModel(ctx context.Context, client *http.Client, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(i18n.T("voice.model.need_sub"))
	}
	switch args[0] {
	case "pull":
		return runPull(ctx, client, args[1:], out)
	case "status":
		return runStatus(args[1:], out)
	default:
		return fmt.Errorf(i18n.T("voice.model.unknown_sub"), args[0])
	}
}

// sizeFlag는 --model 플래그를 붙이고 값을 판정한다.
func sizeFlag(fs *flag.FlagSet) *string {
	return fs.String("model", string(model.Default), i18n.T("voice.model.flag_size"))
}

// resolve는 크기와 그 디렉토리와 파일 목록을 한 번에 낸다.
func resolve(raw string) (model.Size, string, []model.ModelFile, error) {
	// 빈 값은 기본 크기로 친다. CLI 는 플래그 기본값이 채우지만 MCP 는
	// 인자를 생략하면 빈 문자열이 온다. 진입점마다 다르게 처리하면
	// 한쪽만 고쳐진다.
	if raw == "" {
		raw = string(model.Default)
	}
	size, err := model.ParseSize(raw)
	if err != nil {
		return "", "", nil, err
	}
	dir, err := model.Dir(size)
	if err != nil {
		return "", "", nil, err
	}
	files, err := model.Files(size)
	if err != nil {
		return "", "", nil, err
	}
	return size, dir, files, nil
}

// runPull은 모델을 내려받는다. --from 을 주면 그 경로에서 가져온다.
func runPull(ctx context.Context, client *http.Client, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("model pull", flag.ContinueOnError)
	raw := sizeFlag(fs)
	from := fs.String("from", "", i18n.T("voice.model.flag_from"))
	if err := fs.Parse(args); err != nil {
		return err
	}
	size, dir, files, err := resolve(*raw)
	if err != nil {
		return err
	}
	total, err := model.TotalSize(size)
	if err != nil {
		return err
	}

	if *from != "" {
		done, err := modelfetch.Import(*from, dir, files)
		if err != nil {
			return importError(err)
		}
		fmt.Fprintln(out, i18n.T("voice.model.imported", len(done), dir))
		return nil
	}

	fmt.Fprintln(out, i18n.T("voice.model.downloading", size, humanBytes(total)))
	fmt.Fprintln(out, i18n.T("voice.model.target", dir))

	// base 를 빈 문자열로 준다. 음성 모델은 파일마다 Base 를 갖고
	// 있으므로 기본 base 가 쓰일 자리가 없다. 표에 Base 가 빠진
	// 파일이 생기면 URL 이 "/경로" 가 되어 즉시 실패한다. 조용히
	// 엉뚱한 곳을 치는 것보다 낫다.
	skipped, err := modelfetch.Download(ctx, client, "", dir, files, progressTo(os.Stderr))
	if err != nil {
		return downloadError(err)
	}
	switch {
	case len(skipped) == len(files):
		fmt.Fprintln(out, i18n.T("voice.model.all_present"))
	case len(skipped) > 0:
		fmt.Fprintln(out, i18n.T("voice.model.got_some", len(files)-len(skipped), len(skipped)))
	default:
		fmt.Fprintln(out, i18n.T("voice.model.got_all", len(files)))
	}
	return nil
}

// runStatus는 받아 둔 모델의 상태를 낸다.
func runStatus(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("model status", flag.ContinueOnError)
	raw := sizeFlag(fs)
	verify := fs.Bool("verify", false, i18n.T("voice.model.flag_verify"))
	if err := fs.Parse(args); err != nil {
		return err
	}
	size, dir, files, err := resolve(*raw)
	if err != nil {
		return err
	}
	st, err := modelfetch.Inspect(dir, files, *verify)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "%s\n%s\n\n", size, dir)
	present, complete := 0, true
	var have, want int64
	for _, f := range st {
		want += f.ExpectedSize
		if f.Exists {
			present++
			have += f.Size
		}
		if !f.Exists || !f.SizeMatches {
			complete = false
		}
		fmt.Fprintf(out, "  %-24s %s\n", f.Name, fileLine(f, *verify))
	}
	fmt.Fprintln(out, i18n.T("voice.model.count", present, len(st), humanBytes(have), humanBytes(want)))

	// --verify 없이 부른 결과로 온전하다고 말하지 않는다. 크기만 본
	// 것이며 그것은 검증이 아니다.
	corrupt := false
	if *verify {
		for _, f := range st {
			if f.ChecksumMatches == nil || !*f.ChecksumMatches {
				corrupt = true
				break
			}
		}
	}
	switch {
	case !complete:
		fmt.Fprintln(out, i18n.T("voice.model.partial", size))
	case *verify && corrupt:
		fmt.Fprintln(out, i18n.T("voice.model.corrupt"))
		return errors.New(i18n.T("voice.model.corrupt_err"))
	case *verify:
		fmt.Fprintln(out, i18n.T("voice.model.intact"))
	default:
		fmt.Fprintln(out, i18n.T("voice.model.present_only"))
	}
	return nil
}

// fileLine은 파일 한 줄의 상태 문구를 만든다.
func fileLine(f model.FileStatus, verify bool) string {
	if !f.Exists {
		return i18n.T("voice.model.absent")
	}
	if !f.SizeMatches {
		return i18n.T("voice.model.size_mismatch", humanBytes(f.Size), humanBytes(f.ExpectedSize))
	}
	if verify {
		if f.ChecksumMatches != nil && *f.ChecksumMatches {
			return humanBytes(f.Size) + " " + i18n.T("voice.model.verified")
		}
		return humanBytes(f.Size) + " " + i18n.T("voice.model.checksum_bad")
	}
	return humanBytes(f.Size)
}

// progressTo는 진행률 출력기를 만든다. 터미널이면 같은 줄을 갱신하고
// 터미널이 아니면 드물게 줄을 바꿔 낸다. 어느 쪽이든 \r 로 도배하지
// 않는다. engram 본체의 modelProgress 와 같은 규율이다.
//
// 터미널 여부는 writer 가 아니라 표준 출력을 본다. 커맨드 출력이
// 파이프로 흘러가도 사용자 눈에는 터미널이 보이는 경우가 있다.
func progressTo(w io.Writer) modelfetch.ProgressFn {
	interactive := false
	if fi, err := os.Stdout.Stat(); err == nil {
		interactive = fi.Mode()&os.ModeCharDevice != 0
	}
	var last time.Time
	lastStep := int64(-1)
	lastName := ""
	return func(name string, received, total int64) {
		done := received >= total
		pct := int64(100)
		if total > 0 {
			pct = received * 100 / total
		}
		line := fmt.Sprintf("  %-24s %3d%%  %s / %s", name, pct, humanBytes(received), humanBytes(total))
		if interactive {
			if !done && name == lastName && time.Since(last) < progressInterval {
				return
			}
			last, lastName = time.Now(), name
			fmt.Fprintf(w, "\r%s        ", line)
			if done {
				fmt.Fprintln(w)
			}
			return
		}
		// 터미널이 아니면 파일마다 10% 경계에서 한 줄씩 낸다.
		if name != lastName {
			lastName, lastStep = name, -1
		}
		if step := pct / 10; done || step > lastStep {
			lastStep = step
			fmt.Fprintln(w, line)
		}
	}
}

// downloadError는 내려받기 실패를 사람이 읽을 문장으로 바꾼다.
func downloadError(err error) error {
	switch {
	case errors.Is(err, modelfetch.ErrChecksum):
		return fmt.Errorf(i18n.T("voice.model.err_checksum"), err)
	case errors.Is(err, modelfetch.ErrSize):
		return fmt.Errorf(i18n.T("voice.model.err_size"), err)
	default:
		return err
	}
}

// importError는 오프라인 반입 실패를 사람이 읽을 문장으로 바꾼다.
func importError(err error) error {
	if errors.Is(err, modelfetch.ErrMissingFile) {
		return fmt.Errorf(i18n.T("voice.model.err_missing"), err)
	}
	return err
}

// humanBytes는 바이트를 사람이 읽는 단위로 만든다.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(n)/float64(div), "KMGT"[exp])
}
