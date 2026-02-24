package main

import (
	"log"
	"net/http"

	backend "github.com/renlulu/hearx/backend"
	"github.com/renlulu/hearx/backend/internal/api"
	"github.com/renlulu/hearx/backend/internal/asr"
	"github.com/renlulu/hearx/backend/internal/config"
	"github.com/renlulu/hearx/backend/internal/fetcher"
	"github.com/renlulu/hearx/backend/internal/jobstore"
	"github.com/renlulu/hearx/backend/internal/llm"
	"github.com/renlulu/hearx/backend/internal/pipeline"
	"github.com/renlulu/hearx/backend/internal/renderer"
	"github.com/renlulu/hearx/backend/internal/skillpack"
	"github.com/renlulu/hearx/backend/internal/translator"
)

func main() {
	cfg := config.Load()

	// Load skill packs
	spLoader, err := skillpack.NewLoader(backend.SkillsFS)
	if err != nil {
		log.Fatalf("loading skill packs: %v", err)
	}
	log.Printf("loaded %d skill packs", len(spLoader.List()))

	// ASR provider
	var asrProvider asr.Provider
	switch cfg.ASRProvider {
	case "deepgram":
		asrProvider = asr.NewDeepgram(cfg.DeepgramKey)
		log.Println("ASR provider: Deepgram Nova-2")
	default:
		asrProvider = asr.NewWhisper(cfg.OpenAIKey)
		log.Println("ASR provider: OpenAI Whisper")
	}

	// LLM provider
	var llmProvider llm.Provider
	switch cfg.LLMProvider {
	case "anthropic":
		llmProvider = llm.NewAnthropic(cfg.AnthropicKey)
		log.Println("LLM provider: Anthropic Claude")
	default:
		llmProvider = llm.NewOpenAI(cfg.OpenAIKey)
		log.Println("LLM provider: OpenAI GPT-4o-mini")
	}

	// Wire dependencies
	store := jobstore.New()
	f := fetcher.New()
	t := translator.New(llmProvider, spLoader)
	r := renderer.New(llmProvider)
	pipe := pipeline.New(f, asrProvider, t, r, store)

	handler := api.NewHandler(store, pipe, spLoader)
	router := api.NewRouter(handler)

	addr := ":" + cfg.Port
	log.Printf("HearX backend listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
