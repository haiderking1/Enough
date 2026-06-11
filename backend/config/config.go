package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/enough/enough/backend/enoughhome"
	"github.com/enough/enough/backend/secrets"
)

const (
	DefaultEndpoint = "https://opencode.ai/zen/go/v1"
	DefaultModel    = "deepseek-v4-flash"
)

type CompactionSettings struct {
	Enabled          bool `json:"enabled"`
	ReserveTokens    int  `json:"reserve_tokens"`
	KeepRecentTokens int  `json:"keep_recent_tokens"`
	ContextWindow    int  `json:"context_window,omitempty"`
}

// EvidenceConfig controls the v2 evidence runtime. Enabled=false restores
// v1 behavior (no ledger, no tool guard) for emergency rollback.
type EvidenceConfig struct {
	Enabled             bool `json:"enabled"`
	StrictVerifyReset   bool `json:"strict_verify_reset"`
	MaxCompletionRounds int  `json:"max_completion_rounds"`
	VerifierEnabled     bool `json:"verifier_enabled"`

	// ContinuityReads seeds read credit at turn start for agent-authored
	// files whose on-disk hash still matches the last recorded mutation.
	// Pointer so configs written before this field existed default to true.
	ContinuityReads *bool `json:"continuity_reads,omitempty"`
}

// ContinuityEnabled resolves the default-true tri-state.
func (e EvidenceConfig) ContinuityEnabled() bool {
	return e.ContinuityReads == nil || *e.ContinuityReads
}

func DefaultEvidence() EvidenceConfig {
	return EvidenceConfig{
		Enabled:             true,
		StrictVerifyReset:   true,
		MaxCompletionRounds: 12,
		VerifierEnabled:     true,
	}
}

type SkillsSettings struct {
	Enabled             bool     `json:"enabled"`
	EnableSkillCommands bool     `json:"enable_skill_commands"`
	Paths               []string `json:"paths"`
	Disabled            []string `json:"disabled"`
}

// Config holds non-secret settings persisted to disk.
type Config struct {
	Endpoint      string              `json:"endpoint"`
	Model         string              `json:"model"`
	ThinkingLevel string              `json:"thinking_level,omitempty"`
	HideThinking  bool                `json:"hide_thinking,omitempty"`
	Compaction    *CompactionSettings `json:"compaction,omitempty"`
	Evidence      *EvidenceConfig     `json:"evidence,omitempty"`
	Skills        *SkillsSettings     `json:"skills,omitempty"`

	// legacy field — migrated to secrets store on load, never written back
	apiKeyLegacy string `json:"-"`
}

// Runtime bundles config with the in-memory API key (never saved to config.json).
type Runtime struct {
	Endpoint      string
	Model         string
	APIKey        string
	ThinkingLevel string
	HideThinking  bool
	Compaction    CompactionSettings
	Evidence      EvidenceConfig
	Skills        SkillsSettings
}

func Default() Config {
	return Config{
		Endpoint: DefaultEndpoint,
		Model:    DefaultModel,
		Compaction: &CompactionSettings{
			Enabled:          true,
			ReserveTokens:    16384,
			KeepRecentTokens: 20000,
		},
		Skills: &SkillsSettings{
			Enabled:             true,
			EnableSkillCommands: true,
		},
	}
}

func Dir() (string, error) {
	return enoughhome.HomeDir(), nil
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

type fileConfig struct {
	Endpoint      string              `json:"endpoint"`
	Model         string              `json:"model"`
	ThinkingLevel string              `json:"thinking_level,omitempty"`
	HideThinking  bool                `json:"hide_thinking,omitempty"`
	APIKey        string              `json:"api_key,omitempty"`
	Compaction    *CompactionSettings `json:"compaction,omitempty"`
	Evidence      *EvidenceConfig     `json:"evidence,omitempty"`
	Skills        *SkillsSettings     `json:"skills,omitempty"`
}

func Load() (Config, error) {
	cfg := Default()

	path, err := Path()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		// Migration: check if old ~/.config/enough/config.json exists
		home, err := os.UserHomeDir()
		if err == nil {
			oldPath := filepath.Join(home, ".config", "enough", "config.json")
			oldData, err := os.ReadFile(oldPath)
			if err == nil {
				var raw fileConfig
				if err := json.Unmarshal(oldData, &raw); err == nil {
					cfg.Endpoint = raw.Endpoint
					cfg.Model = raw.Model
					cfg.ThinkingLevel = raw.ThinkingLevel
					cfg.HideThinking = raw.HideThinking
					cfg.apiKeyLegacy = raw.APIKey
					if raw.Compaction != nil {
						cfg.Compaction = raw.Compaction
					}
					if raw.Evidence != nil {
						cfg.Evidence = raw.Evidence
					}
					if raw.Skills != nil {
						cfg.Skills = raw.Skills
					}
					if cfg.Endpoint == "" {
						cfg.Endpoint = DefaultEndpoint
					}
					if cfg.Model == "" {
						cfg.Model = DefaultModel
					}
					if cfg.Compaction == nil {
						cfg.Compaction = &CompactionSettings{
							Enabled:          true,
							ReserveTokens:    16384,
							KeepRecentTokens: 20000,
						}
					}
					if cfg.Skills == nil {
						cfg.Skills = &SkillsSettings{
							Enabled:             true,
							EnableSkillCommands: true,
						}
					}
					_ = Save(cfg)
					return cfg, nil
				}
			}
		}
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}

	var raw fileConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	cfg.Endpoint = raw.Endpoint
	cfg.Model = raw.Model
	cfg.ThinkingLevel = raw.ThinkingLevel
	cfg.HideThinking = raw.HideThinking
	cfg.apiKeyLegacy = raw.APIKey
	if raw.Compaction != nil {
		cfg.Compaction = raw.Compaction
	}
	if raw.Evidence != nil {
		cfg.Evidence = raw.Evidence
	}
	if raw.Skills != nil {
		cfg.Skills = raw.Skills
	}

	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultEndpoint
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.Compaction == nil {
		cfg.Compaction = &CompactionSettings{
			Enabled:          true,
			ReserveTokens:    16384,
			KeepRecentTokens: 20000,
		}
	}
	if cfg.Skills == nil {
		cfg.Skills = &SkillsSettings{
			Enabled:             true,
			EnableSkillCommands: true,
		}
	}

	// one-time migration: move api key from config.json into secret store
	if raw.APIKey != "" && !secrets.HasAPIKey() {
		if err := secrets.SetAPIKey(raw.APIKey); err == nil {
			_ = Save(cfg)
		}
	}

	return cfg, nil
}

func Save(cfg Config) error {
	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultEndpoint
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.Compaction == nil {
		cfg.Compaction = &CompactionSettings{
			Enabled:          true,
			ReserveTokens:    16384,
			KeepRecentTokens: 20000,
		}
	}
	if cfg.Skills == nil {
		cfg.Skills = &SkillsSettings{
			Enabled:             true,
			EnableSkillCommands: true,
		}
	}

	dir, err := Dir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	path, err := Path()
	if err != nil {
		return err
	}

	raw := fileConfig{
		Endpoint:      cfg.Endpoint,
		Model:         cfg.Model,
		ThinkingLevel: cfg.ThinkingLevel,
		HideThinking:  cfg.HideThinking,
		Compaction:    cfg.Compaction,
		Evidence:      cfg.Evidence,
		Skills:        cfg.Skills,
	}

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

func LoadRuntime() (Runtime, error) {
	cfg, err := Load()
	if err != nil {
		return Runtime{}, err
	}

	key, err := secrets.GetAPIKey()
	if err != nil {
		return Runtime{}, err
	}

	var comp CompactionSettings
	if cfg.Compaction != nil {
		comp = *cfg.Compaction
	} else {
		comp = CompactionSettings{
			Enabled:          true,
			ReserveTokens:    16384,
			KeepRecentTokens: 20000,
		}
	}

	ev := DefaultEvidence()
	if cfg.Evidence != nil {
		ev = *cfg.Evidence
	}

	var sk SkillsSettings
	if cfg.Skills != nil {
		sk = *cfg.Skills
	} else {
		sk = SkillsSettings{
			Enabled:             true,
			EnableSkillCommands: true,
		}
	}

	return Runtime{
		Endpoint:      cfg.Endpoint,
		Model:         cfg.Model,
		APIKey:        key,
		ThinkingLevel: cfg.ThinkingLevel,
		HideThinking:  cfg.HideThinking,
		Compaction:    comp,
		Evidence:      ev,
		Skills:        sk,
	}, nil
}

func Connected() bool {
	return secrets.HasAPIKey()
}
