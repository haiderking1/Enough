package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/enough/enough/backend/auth"
	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/core"
	"github.com/enough/enough/backend/skills"
)

func (a *App) handleSlash(input string) {
	name, arg, _ := strings.Cut(strings.TrimPrefix(input, "/"), " ")
	name = strings.ToLower(strings.TrimSpace(name))
	arg = strings.TrimSpace(arg)

	switch name {
	case "connect":
		if arg != "" {
			a.saveAPIKey(arg)
			return
		}
		a.mode = modeConnect
		a.editor = NewEditor(1024)
		endpoint, model, err := auth.Settings()
		if err != nil {
			a.appendMessage("error", err.Error())
			a.mode = modeTask
			a.editor = NewEditor(512)
			return
		}
		a.appendMessage("system", fmt.Sprintf("connect — %s · %s\npaste your api key below", endpoint, model))
	case "sessions":
		a.showSessionsList()
	case "resume":
		a.openSessionPicker()
	case "new":
		a.startNewSession()
	case "compact":
		a.startCompact(arg)
	case "auto-compact":
		cfg, err := config.Load()
		if err != nil {
			a.appendMessage("error", err.Error())
			return
		}
		if cfg.Compaction == nil {
			cfg.Compaction = &config.CompactionSettings{
				Enabled:          true,
				ReserveTokens:    16384,
				KeepRecentTokens: 20000,
			}
		}

		val := strings.ToLower(arg)
		if val == "on" {
			cfg.Compaction.Enabled = true
			a.appendMessage("system", "Auto-compaction enabled")
		} else if val == "off" {
			cfg.Compaction.Enabled = false
			a.appendMessage("system", "Auto-compaction disabled")
		} else {
			a.appendMessage("error", "Usage: /auto-compact on|off")
			return
		}

		if err := config.Save(cfg); err != nil {
			a.appendMessage("error", err.Error())
			return
		}

		if runCfg, err := config.LoadRuntime(); err == nil {
			a.mu.Lock()
			if a.agent != nil {
				a.agent.UpdateConfig(runCfg)
			}
			a.mu.Unlock()
		}
		a.requestRender()
	case "tree":
		if a.session == nil {
			a.appendMessage("error", "no active session")
			return
		}
		if a.running {
			a.appendMessage("error", "wait for the agent to finish")
			return
		}

		roots := a.session.GetTree()
		if len(roots) == 0 {
			a.appendMessage("system", "No entries in session tree")
			return
		}

		a.treePickerNodes = a.buildFlatTreeNodes(roots, 0, a.session.LeafID())
		a.treePickerCursor = 0
		a.treePickerConfirm = 0
		a.treePickerChoice = 0
		a.treePickerTarget = ""
		a.mode = modeTreePicker
		a.editor.SetValue("")
		a.requestRender()
	case "skills":
		cfg, err := config.LoadRuntime()
		if err != nil {
			a.appendMessage("error", err.Error())
			return
		}
		ag := a.ensureAgent(cfg)
		discovered, _ := skills.DiscoverAllSkills(ag.WorkDir(), cfg.Skills.Paths, cfg.Skills.Disabled)
		if len(discovered) == 0 {
			a.appendMessage("system", "No skills discovered. Skills live in ~/.enough/skills/")
			return
		}
		var lines []string
		lines = append(lines, fmt.Sprintf("Discovered %d skills:", len(discovered)))
		for _, sk := range discovered {
			cat := sk.Category
			if cat == "" {
				cat = "general"
			}
			lines = append(lines, fmt.Sprintf("- [%s] %s: %s", cat, sk.Name, sk.Description))
		}
		a.appendMessage("system", strings.Join(lines, "\n"))
	case "skills-toggle":
		cfg, err := config.Load()
		if err != nil {
			a.appendMessage("error", err.Error())
			return
		}
		val := strings.ToLower(arg)
		if val == "on" {
			cfg.Skills.Enabled = true
			a.appendMessage("system", "Skills system enabled")
		} else if val == "off" {
			cfg.Skills.Enabled = false
			a.appendMessage("system", "Skills system disabled")
		} else {
			a.appendMessage("error", "Usage: /skills-toggle on|off")
			return
		}
		if err := config.Save(cfg); err != nil {
			a.appendMessage("error", err.Error())
			return
		}
		if runCfg, err := config.LoadRuntime(); err == nil {
			a.mu.Lock()
			if a.agent != nil {
				a.agent.UpdateConfig(runCfg)
			}
			a.mu.Unlock()
		}
		a.requestRender()
	case "skill-commands":
		cfg, err := config.Load()
		if err != nil {
			a.appendMessage("error", err.Error())
			return
		}
		val := strings.ToLower(arg)
		if val == "on" {
			cfg.Skills.EnableSkillCommands = true
			a.appendMessage("system", "Skill commands enabled")
		} else if val == "off" {
			cfg.Skills.EnableSkillCommands = false
			a.appendMessage("system", "Skill commands disabled")
		} else {
			a.appendMessage("error", "Usage: /skill-commands on|off")
			return
		}
		if err := config.Save(cfg); err != nil {
			a.appendMessage("error", err.Error())
			return
		}
		if runCfg, err := config.LoadRuntime(); err == nil {
			a.mu.Lock()
			if a.agent != nil {
				a.agent.UpdateConfig(runCfg)
			}
			a.mu.Unlock()
		}
		a.requestRender()
	case "skill-archive":
		if arg == "" {
			a.appendMessage("error", "Usage: /skill-archive <name>")
			return
		}
		ok, msg := skills.ArchiveSkill(arg)
		if ok {
			a.appendMessage("system", msg)
			a.bumpChat()
		} else {
			a.appendMessage("error", msg)
		}
		a.requestRender()
	case "skill-restore":
		if arg == "" {
			a.appendMessage("error", "Usage: /skill-restore <name>")
			return
		}
		ok, msg := skills.RestoreSkill(arg)
		if ok {
			a.appendMessage("system", msg)
			a.bumpChat()
		} else {
			a.appendMessage("error", msg)
		}
		a.requestRender()
	default:
		if strings.HasPrefix(name, "skill:") {
			skillName := strings.TrimPrefix(name, "skill:")
			if !auth.Connected() {
				a.appendMessage("error", "not connected — type / and pick connect")
				return
			}
			if a.running {
				a.appendMessage("error", "wait for the agent to finish")
				return
			}
			cfg, err := config.LoadRuntime()
			if err != nil {
				a.appendMessage("error", err.Error())
				return
			}
			ag := a.ensureAgent(cfg)
			sessionId := ""
			if a.session != nil {
				sessionId = a.session.SessionID()
			}
			expandedPrompt, cleanBody, err := skills.ExpandSkillSlashCommand(skillName, arg, ag.WorkDir(), cfg, sessionId)
			if err != nil {
				a.appendMessage("error", fmt.Sprintf("failed to expand skill: %v", err))
				return
			}

			if a.compacting {
				a.compactionQueuedMessages = append(a.compactionQueuedMessages, expandedPrompt)
				a.messages = append(a.messages, chatMsg{
					role:     "skillSummary",
					toolName: skillName,
					toolArgs: arg,
					text:     cleanBody,
				})
				a.bumpChat()
				a.requestRender()
				return
			}

			a.messages = append(a.messages, chatMsg{
				role:     "skillSummary",
				toolName: skillName,
				toolArgs: arg,
				text:     cleanBody,
			})
			a.bumpChat()

			a.startAgent(expandedPrompt)
			a.requestRender()
			return
		}
		a.appendMessage("error", "unknown command: /"+name)
	}
}

func (a *App) saveAPIKey(key string) {
	a.mode = modeTask
	a.editor = NewEditor(512)

	if err := auth.SaveAPIKey(key); err != nil {
		a.appendMessage("error", err.Error())
		return
	}

	a.appendMessage("assistant", "Done — connected. api key saved securely.")
	if a.agent != nil {
		_ = a.agent.Reset()
		a.agent = nil
	}
	if a.session != nil {
		_ = a.session.NewSession()
		a.messages = nil
		a.bumpChat()
	}
}

func (a *App) cancelConnect() {
	if a.mode == modeConnect {
		a.mode = modeTask
		a.editor = NewEditor(512)
		a.editor.SetValue("")
		a.appendMessage("system", "connect cancelled")
	}
}

func (a *App) startCompact(customInstructions string) {
	if !auth.Connected() {
		a.appendMessage("error", "not connected — type / and pick connect")
		return
	}
	if a.session == nil {
		a.appendMessage("error", "no active session")
		return
	}
	cfg, err := config.LoadRuntime()
	if err != nil {
		a.appendMessage("error", err.Error())
		return
	}

	cmdLine := "/compact"
	if strings.TrimSpace(customInstructions) != "" {
		cmdLine += " " + strings.TrimSpace(customInstructions)
	}
	a.appendMessage("user", cmdLine)
	a.setCompacting(true, "Compacting context...")
	a.requestRender()

	a.runAgentTask(func(emit func(core.Event)) {
		a.mu.Lock()
		ag := a.ensureAgent(cfg)
		ag.SetEmit(emit)
		a.mu.Unlock()

		_, _ = ag.Compact(context.Background(), customInstructions)
	})
}

func (a *App) runAgentTask(task func(emit func(core.Event))) {
	a.mu.Lock()
	a.running = true
	ch := make(chan core.Event, 64)
	a.agentCh = ch
	a.mu.Unlock()

	go func() {
		defer close(ch)
		emit := func(e core.Event) {
			ch <- e
			a.requestRender()
		}
		task(emit)
	}()
	a.requestRender()
}
