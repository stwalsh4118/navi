package tui

import (
	"maps"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/pm"
	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

const pmPollInterval = 30 * time.Second

func pmTickCmd() tea.Cmd {
	return tea.Tick(pmPollInterval, func(t time.Time) tea.Msg {
		return pmTickMsg(t)
	})
}

func pmRunCmd(engine *pm.Engine, sessions []session.Info, taskResults map[string]*task.ProviderResult) tea.Cmd {
	sessionCopy := append([]session.Info(nil), sessions...)
	resultsCopy := maps.Clone(taskResults)

	return func() tea.Msg {
		output, err := engine.Run(sessionCopy, resultsCopy)
		return pmOutputMsg{output: output, err: err}
	}
}
