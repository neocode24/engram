package cli

import (
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// version은 릴리스 태그다. 릴리스 빌드에서 ldflags로 주입하며 기본값은 dev다.
var version = "dev"

// versionInfo는 version 커맨드가 출력하는 빌드 정보다.
type versionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Dirty     bool   `json:"dirty"`
	CommitAt  string `json:"commitAt"`
	GoVersion string `json:"goVersion"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
}

// collectVersionInfo는 현재 바이너리의 빌드 정보를 모은다.
// 커밋과 dirty 여부는 런타임에 debug.ReadBuildInfo의 vcs 설정에서 읽는다.
func collectVersionInfo() versionInfo {
	info := versionInfo{
		Version:   version,
		GoVersion: runtime.Version(),
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				info.Commit = s.Value
			case "vcs.modified":
				info.Dirty = s.Value == "true"
			case "vcs.time":
				info.CommitAt = s.Value
			}
		}
	}
	return info
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "버전과 빌드 정보를 출력한다",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			info := collectVersionInfo()
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}
			w := cmd.OutOrStdout()
			commit := info.Commit
			if len(commit) > 12 {
				commit = commit[:12]
			}
			fmt.Fprintf(w, "version: %s\n", info.Version)
			fmt.Fprintf(w, "commit: %s", commit)
			if info.Dirty {
				fmt.Fprint(w, " (dirty)")
			}
			fmt.Fprintln(w)
			fmt.Fprintf(w, "commit 시각: %s\n", info.CommitAt)
			fmt.Fprintf(w, "go: %s\n", info.GoVersion)
			fmt.Fprintf(w, "platform: %s/%s\n", info.GOOS, info.GOARCH)
			return nil
		},
	}
}
