<!-- upstream llm-wiki meta/ingest-rules.md 에서 가져왔다.
     원본 커밋 05f7279. 치환 없음.
     손으로 고치지 않는다. scripts/upstream-sync.py 가 다시 만든다. -->
# Ingest Rules

## Default Ingest Steps

1. Identify source channel.
2. Save or reference original evidence under `sources/`.
3. Create a short processing note under `inbox/<channel>/`.
4. Extract durable facts, decisions, procedures, and unresolved questions.
5. Check existing `context/` for duplicates and conflicts.
6. Propose promotion target and title.

## Channel Notes

### Teams

Prefer meeting title, date, participants, and transcript source.

### Slack

Prefer workspace, channel or DM context, thread timestamp, participants, and permalink if available.

### Confluence

Prefer cloud/site, page ID, title, version, and URL.

### Jira

Prefer issue key, status, linked PRs, comments, and relevant dates.

### Voice Memos

Prefer recording date, device/source, transcript tool, and summary confidence.

Do not commit raw audio files by default.

Keep Apple Voice Memos audio in its original synced location and store only:

- transcript Markdown
- summary Markdown
- source manifest with original file path, duration, recording date, and transcription method

If a raw audio file must be preserved for legal or audit reasons, stage it under `sources/raw-private/` first and decide explicitly whether it is safe to commit.

Default local source path:

```text
~/Library/Group Containers/group.com.apple.VoiceMemos.shared/Recordings/
```
