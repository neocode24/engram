// Package vi18n는 engram-voice 의 사용자 대면 문자열을 루트 i18n 에
// 등록한다.
//
// **루트의 카탈로그와 같은 저장소를 쓴다.** 언어 결정과 조회는 루트가
// 하고 여기는 항목만 더한다. ID 를 `voice.` 로 시작해 본체와 겹치지
// 않게 한다. 겹치면 루트 i18n 이 등록 시점에 멈춘다.
//
// 대상은 `cmd/engram-voice` 의 출력이다. `internal/` 의 저수준 오류
// 문자열은 그대로 두는데, 본체도 같은 자리를 그렇게 두고 있어 기준을
// 맞춘 것이다. 그 경계를 옮기려면 본체와 함께 옮긴다.
package vi18n

import "github.com/neocode24/engram/internal/i18n"

func init() {
	i18n.Register(i18n.LangKO, map[string]string{
		// 커맨드 뼈대
		"voice.cmd.need_command":    "커맨드가 필요합니다",
		"voice.cmd.unknown":         "모르는 커맨드: %s",
		"voice.cmd.interrupted":     "\n중단했습니다. 다시 실행하면 이어받습니다",
		"voice.cmd.bad_lang":        "언어 값이 허용값이 아님: %q (허용값: ko, en)",
		"voice.usage":               voiceUsageKO,
		"voice.mcp.starting":        "engram-voice MCP 서버를 stdio 로 엽니다",
		"voice.mcp.desc":            "오디오를 전사 텍스트로 바꿉니다. 위키에 쓰지 않으며 결과를 돌려줄 뿐입니다.",
		"voice.mcp.tool_status":     "전사 모델이 준비되었는지 봅니다. 파일 목록과 준비 여부를 냅니다.",
		"voice.mcp.tool_transcribe": "오디오를 전사합니다. 화자 번호가 붙은 줄과 용어 교정 목록을 냅니다. 녹음 길이의 절반에서 팔 할이 걸립니다.",
		"voice.mcp.instructions":    voiceInstructionsKO,
		"voice.mcp.pull_hint":       "engram-voice model pull --model %s 를 실행하세요. %s 를 내려받습니다",

		// 전사
		"voice.tr.need_audio":     "오디오 파일 하나가 필요합니다",
		"voice.tr.audio_model":    "오디오 %s, 모델 %s",
		"voice.tr.diarized":       "화자 분할 %s, 화자 %d명",
		"voice.tr.guessed":        "경고: 화자 수를 추정했습니다. 이 값은 믿을 수 없습니다. 아는 값이 있으면 --speakers 로 주세요",
		"voice.tr.done":           "전사 %s, 줄 %d개",
		"voice.tr.progress":       "  전사 %3.0f%%  %d/%d 구간",
		"voice.tr.remaining":      "  남은 시간 %s",
		"voice.tr.wav_unreadable": "wav 를 읽을 수 없음: %w",
		"voice.tr.bad_rate":       "%dHz 가 아닙니다: %d",
		"voice.tr.diarize_failed": "화자 분할 실패: %w",
		"voice.tr.segment_failed": "구간 나누기 실패: %w",
		"voice.tr.audio_open":     "오디오를 열 수 없음: %w",
		"voice.tr.need_converter": "%w\nwav 가 아닌 파일은 변환기가 있어야 합니다",
		"voice.tr.raw_wav":        "안내: 변환기가 없어 준 wav 를 그대로 씁니다",
		"voice.tr.converting":     "%s 로 변환 중",

		// 산출물
		"voice.out.heading":       "# 전사: %s",
		"voice.out.length":        "- 길이: %s",
		"voice.out.model":         "- 모델: whisper %s",
		"voice.out.speakers":      "- 화자: %d명 (사람이 지정)",
		"voice.out.speakers_est":  "- 화자: %d명 (추정. **이 값은 믿을 수 없습니다**)",
		"voice.out.unknown_lines": "- 화자를 붙이지 못한 줄: %d개",
		"voice.out.no_diar":       "- 화자: 나누지 않음",
		"voice.out.name_note":     "- 이름은 도구가 붙이지 않습니다. 번호를 사람 이름으로 바꾸세요",
		"voice.out.body":          "\n## 본문\n\n",
		"voice.out.corr_heading":  "\n### 용어 교정 %d건\n\n",
		"voice.out.corr_note":     "사전이 바꾼 것입니다. 틀린 것이 있으면 사전을 고치세요.\n\n",
		"voice.out.speaker":       "화자 %d",
		"voice.out.speaker_none":  "화자 미상",
		"voice.dur.sec":           "%.1f초",
		"voice.dur.min":           "%d분 %d초",
		"voice.dur.hour":          "%d시간 %d분",

		// 용어 사전
		"voice.gloss.missing": "안내: %s 에 용어 사전이 없어 교정하지 않았습니다",
		"voice.gloss.failed":  "경고: 용어 사전을 읽지 못해 교정하지 않았습니다: %v",
		"voice.gloss.report":  "용어 사전 %s: 규칙 %d개 읽음, %d개가 맞아 %d건 교정, 검토 대상 %d개는 건드리지 않음",

		// 모델
		"voice.model.need_sub":      "model 아래 커맨드가 필요합니다: pull, status",
		"voice.model.unknown_sub":   "모르는 커맨드: model %s",
		"voice.model.flag_size":     "모델 크기(large-v3, medium, small)",
		"voice.model.flag_from":     "오프라인 자료 경로. 디렉토리 또는 tar 아카이브",
		"voice.model.flag_verify":   "체크섬까지 확인합니다. 기가바이트를 읽으므로 시간이 걸립니다",
		"voice.model.imported":      "가져왔습니다. 파일 %d개 -> %s",
		"voice.model.target":        "받는 곳: %s",
		"voice.model.all_present":   "이미 받아 둔 모델이 온전합니다. 받을 것이 없습니다",
		"voice.model.got_some":      "받았습니다. 파일 %d개, 이미 있던 것 %d개",
		"voice.model.got_all":       "받았습니다. 파일 %d개",
		"voice.model.partial":       "받다 만 상태입니다. engram-voice model pull --model %s 로 이어받으세요",
		"voice.model.corrupt":       "파일이 손상됐습니다. 지우고 다시 받으세요",
		"voice.model.corrupt_err":   "체크섬 불일치",
		"voice.model.present_only":  "파일이 다 있습니다. 내용까지 보려면 --verify 를 주세요",
		"voice.model.checksum_bad":  "체크섬 불일치",
		"voice.model.err_checksum":  "%w\n받은 파일이 기대값과 다릅니다. 그 파일을 지우고 다시 받으세요",
		"voice.model.err_size":      "%w\n받다 끊겼습니다. 다시 실행하면 이어받습니다",
		"voice.model.err_missing":   "%w\n--from 경로에 필요한 파일이 없습니다",
		"voice.model.downloading":   "%s 를 내려받습니다. %s",
		"voice.model.done":          "받았습니다: %s",
		"voice.model.verified":      "검증됨",
		"voice.model.absent":        "없음",
		"voice.model.count":         "\n파일 %d/%d, %s / %s",
		"voice.model.intact":        "모델이 온전합니다",
		"voice.model.incomplete":    "모델이 온전하지 않습니다. engram-voice model pull 을 실행하세요",
		"voice.model.size_mismatch": "크기 다름 %s / %s",
		"voice.model.missing_files": "모델 파일 %d개가 없거나 온전하지 않습니다\nengram-voice model pull --model %s 를 먼저 실행하세요",
	})

	i18n.Register(i18n.LangEN, map[string]string{
		"voice.cmd.need_command":    "A command is required",
		"voice.cmd.unknown":         "Unknown command: %s",
		"voice.cmd.interrupted":     "\nInterrupted. Run it again to resume",
		"voice.cmd.bad_lang":        "Language is not an allowed value: %q (allowed: ko, en)",
		"voice.usage":               voiceUsageEN,
		"voice.mcp.starting":        "Opening the engram-voice MCP server on stdio",
		"voice.mcp.desc":            "Turns audio into transcript text. It never writes to the wiki; it only returns the result.",
		"voice.mcp.tool_status":     "Check whether the transcription model is ready. Returns the file list and readiness.",
		"voice.mcp.tool_transcribe": "Transcribe audio. Returns lines tagged with speaker numbers and the list of glossary corrections. It takes half to eight tenths of the recording length.",
		"voice.mcp.instructions":    voiceInstructionsEN,
		"voice.mcp.pull_hint":       "Run engram-voice model pull --model %s. It downloads %s",

		"voice.tr.need_audio":     "Exactly one audio file is required",
		"voice.tr.audio_model":    "Audio %s, model %s",
		"voice.tr.diarized":       "Diarization %s, %d speakers",
		"voice.tr.guessed":        "Warning: the speaker count was guessed. Do not trust it. Pass --speakers if you know the number",
		"voice.tr.done":           "Transcription %s, %d lines",
		"voice.tr.progress":       "  transcribing %3.0f%%  segment %d/%d",
		"voice.tr.remaining":      "  %s left",
		"voice.tr.wav_unreadable": "cannot read the wav: %w",
		"voice.tr.bad_rate":       "not %dHz: %d",
		"voice.tr.diarize_failed": "diarization failed: %w",
		"voice.tr.segment_failed": "segmentation failed: %w",
		"voice.tr.audio_open":     "cannot open the audio: %w",
		"voice.tr.need_converter": "%w\nA converter is required for anything that is not a wav",
		"voice.tr.raw_wav":        "Note: no converter found, using the given wav as is",
		"voice.tr.converting":     "Converting with %s",

		"voice.out.heading":       "# Transcript: %s",
		"voice.out.length":        "- Length: %s",
		"voice.out.model":         "- Model: whisper %s",
		"voice.out.speakers":      "- Speakers: %d (given by a person)",
		"voice.out.speakers_est":  "- Speakers: %d (guessed. **Do not trust this number**)",
		"voice.out.unknown_lines": "- Lines with no speaker: %d",
		"voice.out.no_diar":       "- Speakers: not separated",
		"voice.out.name_note":     "- The tool does not assign names. Replace the numbers with real names",
		"voice.out.body":          "\n## Body\n\n",
		"voice.out.corr_heading":  "\n### %d glossary corrections\n\n",
		"voice.out.corr_note":     "The glossary changed these. If one is wrong, fix the glossary.\n\n",
		"voice.out.speaker":       "Speaker %d",
		"voice.out.speaker_none":  "Unknown speaker",
		"voice.dur.sec":           "%.1fs",
		"voice.dur.min":           "%dm %ds",
		"voice.dur.hour":          "%dh %dm",

		"voice.gloss.missing": "Note: no glossary in %s, nothing was corrected",
		"voice.gloss.failed":  "Warning: could not read the glossary, nothing was corrected: %v",
		"voice.gloss.report":  "Glossary %s: %d rules read, %d matched, %d corrections, %d review entries left alone",

		"voice.model.need_sub":      "A command under model is required: pull, status",
		"voice.model.unknown_sub":   "Unknown command: model %s",
		"voice.model.flag_size":     "Model size (large-v3, medium, small)",
		"voice.model.flag_from":     "Offline source path. A directory or a tar archive",
		"voice.model.flag_verify":   "Also check the checksums. It reads gigabytes, so it takes a while",
		"voice.model.imported":      "Imported %d files -> %s",
		"voice.model.target":        "Into: %s",
		"voice.model.all_present":   "The model you already have is intact. Nothing to download",
		"voice.model.got_some":      "Downloaded %d files, %d were already there",
		"voice.model.got_all":       "Downloaded %d files",
		"voice.model.partial":       "The download is incomplete. Resume with engram-voice model pull --model %s",
		"voice.model.corrupt":       "The files are damaged. Delete them and download again",
		"voice.model.corrupt_err":   "checksum mismatch",
		"voice.model.present_only":  "All files are present. Pass --verify to check their contents",
		"voice.model.checksum_bad":  "checksum mismatch",
		"voice.model.err_checksum":  "%w\nA downloaded file does not match the expected value. Delete it and download again",
		"voice.model.err_size":      "%w\nThe download was cut off. Run it again to resume",
		"voice.model.err_missing":   "%w\nThe --from path is missing required files",
		"voice.model.downloading":   "Downloading %s. %s",
		"voice.model.done":          "Downloaded: %s",
		"voice.model.verified":      "verified",
		"voice.model.absent":        "missing",
		"voice.model.count":         "\nFiles %d/%d, %s / %s",
		"voice.model.intact":        "The model is intact",
		"voice.model.incomplete":    "The model is not intact. Run engram-voice model pull",
		"voice.model.size_mismatch": "size differs %s / %s",
		"voice.model.missing_files": "%d model files are missing or incomplete\nRun engram-voice model pull --model %s first",
	})
}

const voiceInstructionsKO = `이 서버는 오디오를 전사할 뿐이고 위키에 쓰지 않는다.

- 전사 전에 model_status 로 모델을 확인한다. 준비 안 됐으면 사용자에게
  engram-voice model pull 을 실행하라고 알리고 멈춘다. 1.7GB 라 네가
  대신 받지 않는다.
- **화자가 몇 명인지 사용자에게 먼저 묻는다.** 추정값은 긴 녹음에서
  무너진다. 혼자 말한 녹음이면 noSpeakers 를 준다.
- 결과의 화자는 번호다. **약한 근거로 이름을 지어내지 마라.** 대화에서
  이름이 분명히 드러나지 않으면 번호를 그대로 둔다.
- 전사를 위키에 넣는 것은 engram 의 capture 도구가 한다. 사용자 검토
  없이 sources 나 context 로 올리지 않는다.

회의록으로 정리하는 구조와 나머지 규약은 engram 스킬 문서에 있다.
engram MCP 서버가 그것을 instructions 로 낸다.
`

const voiceUsageKO = `engram-voice는 오디오를 전사 텍스트로 바꿉니다.

위키에 쓰지 않습니다. 전사 결과를 표준 출력으로 내며 위키에 넣는 것은
engram capture 가 합니다.

사용법:
  engram-voice transcribe <오디오> [--speakers N] [--wiki <위키>] [--json]
  engram-voice model pull [--model <크기>] [--from <경로>]
  engram-voice model status [--model <크기>] [--verify]
  engram-voice mcp
  engram-voice version

크기는 large-v3(기본), medium, small 입니다.

화자 수를 아는 값이 있으면 --speakers 로 주세요. 생략하면 추정하는데
그 값은 믿을 수 없습니다.

--wiki 를 주면 그 위키의 meta/terminology.md 를 읽어 전사 뒤에 용어를
교정합니다. 사전은 위키가 소유하고 사람이 채웁니다.

에이전트로 쓰려면 mcp 로 띄웁니다. 도구가 둘이며 transcribe 가
전사를, model_status 가 모델 준비 상태를 냅니다.

출력 언어는 --lang ko 또는 --lang en 으로 고릅니다. ENGRAM_LANG 도
같은 일을 합니다.

전사 결과는 표준 출력으로 나가고 진행률은 표준 오류로 나갑니다.
그대로 위키에 넣으려면 이렇게 씁니다.

  engram-voice transcribe 회의.m4a --speakers 3 | engram capture --title "회의"
`

const voiceInstructionsEN = `This server only transcribes audio. It never writes to the wiki.

- Call model_status before transcribing. If the model is not ready, tell
  the person to run engram-voice model pull and stop. It is 1.7GB; do not
  download it on their behalf.
- **Ask the person how many speakers there are.** Guessing falls apart on
  long recordings. Pass noSpeakers for a solo recording.
- Speakers come back as numbers. **Do not invent names from weak
  evidence.** If a name is not clearly stated, leave the number.
- engram's capture tool is what puts a transcript in the wiki. Never
  promote to sources or context without the person reviewing it.

The meeting-note structure and the rest of the contract are in engram's
skill document, which the engram MCP server sends as instructions.
`

const voiceUsageEN = `engram-voice turns audio into transcript text.

It never writes to the wiki. The transcript goes to standard output and
engram capture is what puts it in the wiki.

Usage:
  engram-voice transcribe <audio> [--speakers N] [--wiki <wiki>] [--json]
  engram-voice model pull [--model <size>] [--from <path>]
  engram-voice model status [--model <size>] [--verify]
  engram-voice mcp
  engram-voice version

Sizes are large-v3 (default), medium, and small.

Pass --speakers if you know how many people spoke. Without it the count
is guessed, and the guess is not trustworthy.

With --wiki the tool reads meta/terminology.md from that wiki and
corrects terminology after transcription. The wiki owns the glossary and
a person fills it in.

For agents, start it with mcp. Two tools: transcribe does the work and
model_status reports whether the model is ready.

Output language comes from --lang ko or --lang en. ENGRAM_LANG does the
same thing.

The transcript goes to standard output and progress goes to standard
error, so you can pipe it straight into the wiki.

  engram-voice transcribe meeting.m4a --speakers 3 | engram capture --title "Meeting"
`
