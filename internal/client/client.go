package client

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
	"github.com/ifrunruhin12/tasktime/internal/models"
	"github.com/ifrunruhin12/tasktime/internal/storage"
)

type Client struct {
	serverURL string
}

func New(serverURL string) *Client {
	return &Client{
		serverURL: serverURL,
	}
}

func (c *Client) Start() error {
	p := tea.NewProgram(c.initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (c *Client) initialModel() model {
	localStore, _ := storage.NewLocalStore()
	return model{
		client:         c,
		personalTasks:  []models.Task{},
		teamTasks:      []models.Task{},
		cursor:         0,
		showInput:      false,
		width:          80,
		height:         24,
		currentSection: "personal", // Start with personal tasks
		localStore:     localStore,
	}
}

type model struct {
	client         *Client
	personalTasks  []models.Task
	teamTasks      []models.Task
	cursor         int
	showInput      bool
	inputTitle     string
	inputProject   string
	inputMode      int
	ws             *websocket.Conn
	width          int
	height         int
	currentSection string // "personal" or "team"
	localStore     *storage.LocalStore
}

type personalTasksLoadedMsg []models.Task
type teamTasksLoadedMsg []models.Task
type wsConnectedMsg *websocket.Conn
type tickMsg time.Time
type taskCreationFailedMsg struct{}
type wsDisconnectedMsg struct{}
type wsConnectionFailedMsg struct{}
type wsRetryMsg struct{}
type taskOperationFailedMsg struct{}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.loadPersonalTasks(),
		m.loadTeamTasks(),
		m.connectWebSocket(),
		m.tick(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.showInput {
			return m.handleInputKeys(msg)
		}
		return m.handleNormalKeys(msg)

	case personalTasksLoadedMsg:
		m.personalTasks = []models.Task(msg)
		return m, nil

	case teamTasksLoadedMsg:
		m.teamTasks = []models.Task(msg)
		return m, nil

	case wsConnectedMsg:
		m.ws = msg
		return m, m.listenWebSocket()

	case tickMsg:
		return m, m.tick()

	case models.WSMessage:
		return m.handleWebSocketMessage(msg)

	case taskCreationFailedMsg:
		// Reload tasks as fallback when creation fails
		if m.currentSection == "personal" {
			return m, m.loadPersonalTasks()
		}
		return m, m.loadTeamTasks()

	case wsDisconnectedMsg:
		// WebSocket disconnected, try to reconnect
		m.ws = nil
		return m, m.connectWebSocket()

	case wsConnectionFailedMsg:
		// WebSocket connection failed, try again after a delay
		m.ws = nil
		return m, tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
			return wsRetryMsg{}
		})

	case wsRetryMsg:
		// Retry WebSocket connection
		return m, m.connectWebSocket()

	case taskOperationFailedMsg:
		// Task operation failed, reload tasks to get current state
		if m.currentSection == "personal" {
			return m, m.loadPersonalTasks()
		}
		return m, m.loadTeamTasks()
	}

	return m, nil
}

func (m model) View() string {
	if m.showInput {
		return m.renderInputMode()
	}

	var s strings.Builder

	// Title with WebSocket status
	title := "TaskTime - Dual Task Manager"
	if m.ws != nil {
		title += " [LIVE]"
	} else {
		title += " [OFFLINE]"
	}
	s.WriteString(titleStyle.Render(title))
	s.WriteString("\n\n")

	// Section tabs
	personalTab := "Personal Tasks"
	teamTab := "Team Tasks"
	
	if m.currentSection == "personal" {
		personalTab = "▶ " + personalTab + " ◀"
		teamTab = "  " + teamTab + "  "
	} else {
		personalTab = "  " + personalTab + "  "
		teamTab = "▶ " + teamTab + " ◀"
	}

	s.WriteString(selectedStyle.Render(personalTab))
	s.WriteString("   ")
	s.WriteString(normalStyle.Render(teamTab))
	s.WriteString("\n\n")

	// Get current tasks based on section
	currentTasks := m.personalTasks
	if m.currentSection == "team" {
		currentTasks = m.teamTasks
	}

	if len(currentTasks) == 0 {
		s.WriteString("No tasks yet. Press 'n' to create one!\n\n")
	} else {
		for i, task := range currentTasks {
			line := m.renderTaskLine(i, task)
			if m.cursor == i {
				s.WriteString(selectedStyle.Render(line))
			} else {
				s.WriteString(normalStyle.Render(line))
			}
			s.WriteString("\n")
		}
		s.WriteString("\n")
	}

	s.WriteString(helpStyle.Render("tab: switch • n: new • d: done • s: timer • x: delete • r: refresh • q: quit"))

	return s.String()
}
