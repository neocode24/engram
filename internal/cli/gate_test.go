package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/lint"
	"github.com/neocode24/engram/internal/walk"
)

// TestGateTargetAgreement는 게이트 대상 집계가 세 곳(lint, status, cli)에서
// 같은 값을 쓰는지 본다. 각자 세면 곧 갈라진다.
func TestGateTargetAgreement(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"engram.yaml": "preset: personal\n",
		// 색인. root_files 로 대상에 남는다.
		"index.md": "---\ntype: system\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
			"source_channel: manual\nderived_context: []\n---\n\n# 색인\n",
		// 링크 1개짜리 context 문서. 대상에 포함된다.
		"context/a.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\nrelated:\n  - \"[[index]]\"\n" +
			"source_channel: manual\nderived_context: []\n---\n\n본문\n",
		// sources 문서. promote 가 옮기지 않으므로 대상에 포함된다.
		"sources/s1.md": "---\ntype: source-summary\nartifact_stage: source\nstatus: sourced\n" +
			"indexable: false\nsource_refs: []\nderived_from: []\nderived_context: []\n" +
			"source_channel: web\ncreated: 2026-01-01\nsourced_at: 2026-01-02\n---\n\n원본\n",
		// inbox 문서. promote 되면 슬러그가 바뀌므로 대상에서 빠진다.
		"inbox/2026-03-05-memo.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
			"indexable: false\nsource_channel: manual\ncreated: 2026-03-05\n---\n\n메모\n",
	}
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	walked, err := walk.Files(root, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// cli 의 집계와 lint 의 집계는 같은 수를 낸다. 대상은 색인, context, sources 셋.
	_, cliTargets := gateInputs(map[string]bool{}, walked, "", "")
	if cliTargets != 3 {
		t.Errorf("cli 대상 수 = %d, want 3", cliTargets)
	}
	if got := lint.LinkableTargets(walked, ""); got != cliTargets {
		t.Errorf("lint 대상 수 %d 와 cli 대상 수 %d 가 다릅니다", got, cliTargets)
	}

	// 게이트는 실제로 이어지는 링크만 센다. 없는 슬러그 둘로는 못 넘는다.
	ghosts := map[string]bool{"없는문서하나": true, "없는문서둘": true}
	if resolved, _ := gateInputs(ghosts, walked, "", ""); resolved != 0 {
		t.Errorf("존재하지 않는 슬러그가 %d개 세어졌습니다. 0이어야 합니다", resolved)
	}
	// inbox 문서를 가리키는 링크도 세지 않는다. 승급하면 슬러그가 바뀐다.
	toInbox := map[string]bool{"2026-03-05-memo": true, "index": true}
	if resolved, _ := gateInputs(toInbox, walked, "", ""); resolved != 1 {
		t.Errorf("이어지는 링크 = %d, want 1 (index 만)", resolved)
	}
	// 판정 대상 문서 자신은 대상 수에서 빠진다.
	if got := lint.LinkableTargets(walked, "context/a.md"); got != 2 {
		t.Errorf("자신을 뺀 대상 수 = %d, want 2", got)
	}
	// 같은 집계로 잰 게이트는 context 문서 1개(링크 1개)를 유예 없이 거절한다.
	if g := lint.EvaluateGate(1, lint.LinkableTargets(walked, "context/a.md"), cfg.Thresholds.MinWikilinks); g.Passed {
		t.Errorf("대상 2개, 링크 1개면 거절이어야 함: %+v", g)
	}
}
