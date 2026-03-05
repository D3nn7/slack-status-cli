package main

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/slack-go/slack"
)

type model struct {
	client          *slack.Client
	status          statusInfo
	templates       []template
	templateList    list.Model
	durationList    list.Model
	pendingTemplate *template
	durationUnit    durationUnit
	templatesPath   string
	configPath      string
	state           viewState
	confirmDelete   bool
	cfg             config
	inputs          []textinput.Model
	focusIndex      int
	message         string
	err             error
	width           int
	height          int
	loading         bool
	// Calendar sync
	calSyncCfg     calSyncConfig
	calSyncEnabled bool
	calSync        calSyncState
	calSyncCfgPath string
}

func initialModel() model {
	cfgPath, cfgErr := resolvePath(configName)
	tmplPath, tmplErr := ensureTemplatesFile()

	var client *slack.Client
	var status statusInfo
	var loadErr error
	var cfg config

	if cfgErr == nil {
		loaded, err := loadConfig(cfgPath)
		if err != nil {
			loadErr = err
		} else {
			cfg = loaded
			client = slack.New(cfg.SlackToken)
		}
	} else {
		loadErr = cfgErr
		cfgPath = defaultConfigPath()
	}

	if tmplErr != nil {
		if loadErr != nil {
			loadErr = fmt.Errorf("%v; %w", loadErr, tmplErr)
		} else {
			loadErr = tmplErr
		}
		tmplPath = ""
	}

	delegate := newTemplateDelegate()
	ls := list.New([]list.Item{}, delegate, 42, 16)
	ls.Title = "Status Templates"
	ls.SetShowStatusBar(false)
	ls.SetShowHelp(false)
	ls.DisableQuitKeybindings()
	ls.SetFilteringEnabled(false)

	// Calendar sync: load config (optional)
	var calCfg calSyncConfig
	var calEnabled bool
	var calSyncCfgPath string
	var calSync calSyncState

	if p, err := resolvePath(calSyncConfigName); err == nil {
		calSyncCfgPath = p
		if loaded, err := loadCalSyncConfig(p); err == nil {
			calCfg = loaded
			calEnabled = loaded.Enabled && loaded.ICSUrl != ""
			if loaded.Debug {
				initCalDebugLog(loaded.DebugLogPath)
			}
		}
	}

	// Crash-recovery: if a state file exists from a previous run, restore active event info.
	if calEnabled && calCfg.StatePath != "" {
		if snap, err := loadSavedStatus(calCfg.StatePath); err == nil && snap.ActiveEventID != "" {
			calSync.ActiveEventID = snap.ActiveEventID
			if snap.ActiveEventEndUTC != "" {
				if t, err := time.Parse(time.RFC3339, snap.ActiveEventEndUTC); err == nil {
					calSync.ActiveEventEnd = t
				}
			}
			calSync.StatusSaved = true
			calSync.StatusSavedText = snap.Text
		}
	}

	return model{
		client:         client,
		status:         status,
		cfg:            cfg,
		confirmDelete:  effectiveConfirmDelete(cfg),
		templates:      []template{},
		templateList:   ls,
		durationList:   newDurationList(42, 16),
		templatesPath:  tmplPath,
		configPath:     cfgPath,
		state:          viewDashboard,
		message:        "Tab to switch, Enter to use, ? for help",
		err:            loadErr,
		calSyncCfg:     calCfg,
		calSyncEnabled: calEnabled,
		calSyncCfgPath: calSyncCfgPath,
		calSync:        calSync,
	}
}

func (m model) enterManualForm() model {
	m.state = viewManual
	m.message = "Set a custom status"
	m.inputs = buildStatusInputs("", "")
	return m
}

func (m model) enterEditForm() model {
	m.state = viewEditCurrent
	m.message = "Modify current status"
	m.inputs = buildStatusInputs(m.status.Text, m.status.Emoji)
	return m
}

func (m model) enterCreateTemplateForm() model {
	m.state = viewCreateTemplate
	m.message = "Create a reusable template"
	m.inputs = buildTemplateInputs()
	return m
}

func (m model) enterSettings() model {
	m.state = viewSettings
	m.message = "Update settings"
	m.inputs = buildSettingsInputs(m.cfg.SlackToken)
	m.focusIndex = 0
	return m
}

func (m model) enterDurationSelector(t template) model {
	w, h := panelSize(m.width, m.height)
	m.state = viewDurationSelector
	m.pendingTemplate = &t
	m.durationList = newDurationList(w, h)
	m.message = "Dauer waehlen"
	m.inputs = nil
	m.focusIndex = 0
	return m
}

func (m model) backToDurationSelector() model {
	m.state = viewDurationSelector
	m.inputs = nil
	m.focusIndex = 0
	m.message = "Dauer waehlen"
	return m
}

func (m model) cancelDurationSelector() model {
	m.state = viewDashboard
	m.inputs = nil
	m.focusIndex = 0
	m.pendingTemplate = nil
	m.message = "Auswahl abgebrochen"
	return m
}

func (m model) backToDashboard() model {
	m.state = viewDashboard
	m.inputs = nil
	m.focusIndex = 0
	m.message = "Back to dashboard"
	return m
}

func (m model) withError(err error) model {
	m.err = err
	return m
}

func (m model) selectedTemplate() *template {
	if len(m.templates) == 0 {
		return nil
	}
	item, ok := m.templateList.SelectedItem().(templateItem)
	if !ok {
		return nil
	}
	t := template(item)
	return &t
}

func messageCmd(text string) tea.Cmd {
	return func() tea.Msg {
		return setStatusMsg(text)
	}
}

func panelSize(width, height int) (int, int) {
	w := width/2 - 4
	h := height - 10
	if w < 30 {
		w = 30
	}
	if h < 10 {
		h = 10
	}
	return w, h
}
