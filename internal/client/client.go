package client

import (
	"strings"
	"time"

	"github.com/ifrunruhin12/tasktime-mvp/internal/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
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
	return model{
		client:    c,
		tasks:     []models.Task{},
		cursor:    0,
		showInput: false,
		width:     80,
		height:    24,
	}
}

type model struct {
	client       *Client
	tasks        []models.Task
	cursor       int
	showInput    bool
	inputTitle   string
	inputProject string
	inputMode    int
	ws           *websocket.Conn
	width        int
	height       int
}

type tasksLoadedMsg []models.Task
type wsConnectedMsg *websocket.Conn
type tickMsg time.Time
type taskCreationFailedMsg struct{}
type wsDisconnectedMsg struct{}
type wsConnectionFailedMsg struct{}
type wsRetryMsg struct{}
type taskOperationFailedMsg struct{}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.loadTasks(),
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

	case tasksLoadedMsg:
		m.tasks = []models.Task(msg)
		return m, nil

	case wsConnectedMsg:
		m.ws = msg
		return m, m.listenWebSocket()

	case tickMsg:
		return m, m.tick()

	case models.WSMessage:
		return m.handleWebSocketMessage(msg)



	case taskCreationFailedMsg:
		// Reload all tasks as fallback when creation fails
		return m, m.loadTasks()

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
		return m, m.loadTasks()
	}

	return m, nil
}

func (m model) View() string {
	if m.showInput {
		return m.renderInputMode()
	}

	var s strings.Builder
	
	// Title with WebSocket status
	title := "TaskTime - Team Task Manager"
	if m.ws != nil {
		title += " [LIVE]"
	} else {
		title += " [OFFLINE]"
	}
	s.WriteString(titleStyle.Render(title))
	s.WriteString("\n\n")

	if len(m.tasks) == 0 {
		s.WriteString("No tasks yet. Press 'n' to create one!\n\n")
	} else {
		for i, task := range m.tasks {
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

	s.WriteString(helpStyle.Render("n: new • d: done • s: timer • x: delete • r: refresh • q: quit"))
	
	return s.String()
}