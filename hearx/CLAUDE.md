# HearX

English audio/video → Chinese translation engine.

## Architecture

- **Backend**: Go (deployed to Railway/Fly.io)
- **Frontend**: Next.js (deployed to Vercel) — not yet implemented

## Backend Structure

```
hearx/backend/
├── cmd/server/main.go          # Entry point, DI, HTTP server
├── internal/
│   ├── config/                 # Environment variable config
│   ├── model/                  # Job, TranscriptSegment, SkillPack types
│   ├── skillpack/              # YAML loader + merge + embed.FS
│   ├── fetcher/                # yt-dlp + ffmpeg audio download
│   ├── asr/                    # ASR providers (Whisper, Deepgram)
│   ├── llm/                    # LLM providers (OpenAI, Anthropic)
│   ├── translator/             # Chunker + concurrent translation
│   ├── renderer/               # Markdown, SRT, Summary output
│   ├── pipeline/               # Full pipeline orchestration
│   ├── jobstore/               # In-memory sync.Map store
│   └── api/                    # HTTP handlers + chi router
├── skills/                     # YAML Skill Pack definitions
└── skills.go                   # embed.FS for skill YAML files
```

## Key Commands

```bash
cd hearx/backend
make build    # Build binary
make run      # Run server
make test     # Run tests
```

## API Endpoints

- `POST /api/jobs` — Create translation job (202 + async)
- `GET /api/jobs/{id}` — Get job status/result
- `GET /api/skills` — List available Skill Packs

## Environment Variables

See `hearx/backend/.env.example` for required API keys.

## External Dependencies

Requires `yt-dlp` and `ffmpeg` installed on the host.
