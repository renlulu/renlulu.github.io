package skillpack

import (
	"embed"
	"fmt"
	"strings"

	"github.com/renlulu/hearx/backend/internal/model"
	"gopkg.in/yaml.v3"
)

type Loader struct {
	packs map[string]*model.SkillPack
}

func NewLoader(fs embed.FS) (*Loader, error) {
	l := &Loader{packs: make(map[string]*model.SkillPack)}
	entries, err := fs.ReadDir("skills")
	if err != nil {
		return nil, fmt.Errorf("reading embedded skills: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := fs.ReadFile("skills/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", e.Name(), err)
		}
		var sp model.SkillPack
		if err := yaml.Unmarshal(data, &sp); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", e.Name(), err)
		}
		l.packs[sp.Name] = &sp
	}
	return l, nil
}

func (l *Loader) Get(name string) (*model.SkillPack, bool) {
	sp, ok := l.packs[name]
	return sp, ok
}

func (l *Loader) List() []*model.SkillPack {
	result := make([]*model.SkillPack, 0, len(l.packs))
	for _, sp := range l.packs {
		result = append(result, sp)
	}
	return result
}

func (l *Loader) Merge(names []string) (*MergedPack, error) {
	merged := &MergedPack{
		Glossary: make(map[string]string),
	}

	// Always include general pack
	if gen, ok := l.packs["general"]; ok {
		mergePack(merged, gen)
	}

	for _, name := range names {
		if name == "general" {
			continue
		}
		sp, ok := l.packs[name]
		if !ok {
			return nil, fmt.Errorf("skill pack %q not found", name)
		}
		mergePack(merged, sp)
	}
	return merged, nil
}

type MergedPack struct {
	Glossary map[string]string
	Prompts  []string
	Entities []model.EntityRule
}

func mergePack(merged *MergedPack, sp *model.SkillPack) {
	for k, v := range sp.Glossary {
		merged.Glossary[k] = v
	}
	if sp.Prompt != "" {
		merged.Prompts = append(merged.Prompts, sp.Prompt)
	}
	merged.Entities = append(merged.Entities, sp.Entities...)
}
