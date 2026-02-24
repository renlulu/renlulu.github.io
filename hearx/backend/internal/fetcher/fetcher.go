package fetcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Fetcher struct{}

func New() *Fetcher {
	return &Fetcher{}
}

// Fetch downloads audio from url using yt-dlp, converts to 16kHz mono WAV via ffmpeg.
// Returns the path to the WAV file and a cleanup function.
func (f *Fetcher) Fetch(ctx context.Context, url string) (wavPath string, cleanup func(), err error) {
	tmpDir, err := os.MkdirTemp("", "hearx-*")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp dir: %w", err)
	}
	cleanup = func() { os.RemoveAll(tmpDir) }

	rawPath := filepath.Join(tmpDir, "audio.%(ext)s")
	downloadedPath := filepath.Join(tmpDir, "audio") // yt-dlp will add extension

	// Download audio with yt-dlp
	ytdlp := exec.CommandContext(ctx, "yt-dlp",
		"-x",                   // extract audio
		"--audio-format", "wav", // convert to wav
		"-o", rawPath,
		"--no-playlist",
		url,
	)
	ytdlp.Stderr = os.Stderr
	if err := ytdlp.Run(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("yt-dlp failed: %w", err)
	}

	// Find the downloaded file
	matches, _ := filepath.Glob(downloadedPath + ".*")
	if len(matches) == 0 {
		cleanup()
		return "", nil, fmt.Errorf("no downloaded file found in %s", tmpDir)
	}
	srcFile := matches[0]

	// Convert to 16kHz mono WAV with ffmpeg
	wavPath = filepath.Join(tmpDir, "audio_16k.wav")
	ffmpeg := exec.CommandContext(ctx, "ffmpeg",
		"-i", srcFile,
		"-ar", "16000",
		"-ac", "1",
		"-y",
		wavPath,
	)
	ffmpeg.Stderr = os.Stderr
	if err := ffmpeg.Run(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("ffmpeg failed: %w", err)
	}

	return wavPath, cleanup, nil
}
