package client

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// setupModel represents the first-time setup flow
type setupModel struct {
	configManager     *ConfigManager
	config            *ClientConfig
	step              int // 0: server URL, 1: login/register choice, 2: username, 3: password, 4: confirm password (register only)
	authMode          string // "login" or "register"
	serverURL         string
	username          string
	password          string
	confirmPassword   string
	errorMsg          string
	width             int
	height            int
}

type setupCompleteMsg struct {
	config *ClientConfig
}

type authErrorMsg struct {
	error string
}

type authSuccessMsg struct {
	token    string
	username string
}

// newSetupModel creates a new setup model
func newSetupModel(cm *ConfigManager) setupModel {
	config := GetDefaultConfig()
	return setupModel{
		configManager: cm,
		config:        config,
		step:          0,
		serverURL:     config.ServerURL,
		width:         80,
		height:        24,
	}
}

func (m setupModel) Init() tea.Cmd {
	return nil
}

func (m setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleSetupKeys(msg)

	case authSuccessMsg:
		// Save configuration
		m.config.Username = msg.username
		m.config.AuthToken = msg.token
		m.config.ServerURL = m.serverURL
		
		if err := m.configManager.Save(m.config); err != nil {
			m.errorMsg = fmt.Sprintf("Failed to save config: %v", err)
			return m, nil
		}
		
		return m, func() tea.Msg {
			return setupCompleteMsg{config: m.config}
		}

	case authErrorMsg:
		m.errorMsg = msg.error
		return m, nil

	case setupCompleteMsg:
		// Setup is complete, quit the setup program
		return m, tea.Quit
	}

	return m, nil
}

func (m setupModel) View() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("Welcome to TaskTime"))
	s.WriteString("\n\n")

	if m.errorMsg != "" {
		s.WriteString(errorStyle.Render("Error: " + m.errorMsg))
		s.WriteString("\n\n")
	}

	switch m.step {
	case 0:
		// Server URL input
		s.WriteString("Enter server URL:\n\n")
		s.WriteString(selectedStyle.Render(m.serverURL + "_"))
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("Enter: continue • Esc: use default"))

	case 1:
		// Login or Register choice
		s.WriteString(fmt.Sprintf("Server: %s\n\n", m.serverURL))
		s.WriteString("Do you have an account?\n\n")
		
		if m.authMode == "" || m.authMode == "login" {
			s.WriteString(selectedStyle.Render("▶ Login"))
			s.WriteString("\n")
			s.WriteString(normalStyle.Render("  Register"))
		} else {
			s.WriteString(normalStyle.Render("  Login"))
			s.WriteString("\n")
			s.WriteString(selectedStyle.Render("▶ Register"))
		}
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("↑/↓: select • Enter: continue"))

	case 2:
		// Username input
		s.WriteString(fmt.Sprintf("Server: %s\n", m.serverURL))
		s.WriteString(fmt.Sprintf("Mode: %s\n\n", strings.Title(m.authMode)))
		s.WriteString("Username:\n\n")
		s.WriteString(selectedStyle.Render(m.username + "_"))
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("Enter: continue • Esc: go back"))

	case 3:
		// Password input
		s.WriteString(fmt.Sprintf("Server: %s\n", m.serverURL))
		s.WriteString(fmt.Sprintf("Mode: %s\n", strings.Title(m.authMode)))
		s.WriteString(fmt.Sprintf("Username: %s\n\n", m.username))
		s.WriteString("Password:\n\n")
		s.WriteString(selectedStyle.Render(strings.Repeat("*", len(m.password)) + "_"))
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("Enter: continue • Esc: go back"))

	case 4:
		// Confirm password input (only for registration)
		s.WriteString(fmt.Sprintf("Server: %s\n", m.serverURL))
		s.WriteString(fmt.Sprintf("Mode: %s\n", strings.Title(m.authMode)))
		s.WriteString(fmt.Sprintf("Username: %s\n\n", m.username))
		s.WriteString("Confirm Password:\n\n")
		s.WriteString(selectedStyle.Render(strings.Repeat("*", len(m.confirmPassword)) + "_"))
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("Enter: submit • Esc: go back"))
	}

	return s.String()
}

func (m setupModel) handleSetupKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.errorMsg = "" // Clear error on new input

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		if m.step > 0 {
			m.step--
			if m.step == 2 {
				m.password = ""
			}
			if m.step == 3 {
				m.confirmPassword = ""
			}
		} else {
			// Use default server URL
			m.serverURL = m.config.ServerURL
			m.step = 1
			m.authMode = "login"
		}

	case "enter":
		switch m.step {
		case 0:
			// Validate server URL
			if m.serverURL == "" {
				m.serverURL = m.config.ServerURL
			}
			m.step = 1
			m.authMode = "login"

		case 1:
			// Move to username input
			if m.authMode == "" {
				m.authMode = "login"
			}
			m.step = 2

		case 2:
			// Validate username and move to password
			if m.username == "" {
				m.errorMsg = "Username cannot be empty"
				return m, nil
			}
			if len(m.username) < 3 || len(m.username) > 50 {
				m.errorMsg = "Username must be 3-50 characters"
				return m, nil
			}
			m.step = 3

		case 3:
			// Validate password and move to next step
			if m.password == "" {
				m.errorMsg = "Password cannot be empty"
				return m, nil
			}
			if len(m.password) < 8 {
				m.errorMsg = "Password must be at least 8 characters"
				return m, nil
			}
			
			// For login, authenticate directly. For register, go to confirm password
			if m.authMode == "login" {
				return m, m.authenticate()
			} else {
				m.step = 4 // Go to confirm password
			}

		case 4:
			// Confirm password and attempt registration
			if m.confirmPassword == "" {
				m.errorMsg = "Please confirm your password"
				return m, nil
			}
			if m.password != m.confirmPassword {
				m.errorMsg = "Passwords do not match"
				return m, nil
			}
			return m, m.authenticate()
		}

	case "up", "k":
		if m.step == 1 {
			if m.authMode == "register" {
				m.authMode = "login"
			}
		}

	case "down", "j":
		if m.step == 1 {
			if m.authMode == "login" {
				m.authMode = "register"
			}
		}

	case "backspace":
		switch m.step {
		case 0:
			if len(m.serverURL) > 0 {
				m.serverURL = m.serverURL[:len(m.serverURL)-1]
			}
		case 2:
			if len(m.username) > 0 {
				m.username = m.username[:len(m.username)-1]
			}
		case 3:
			if len(m.password) > 0 {
				m.password = m.password[:len(m.password)-1]
			}
		case 4:
			if len(m.confirmPassword) > 0 {
				m.confirmPassword = m.confirmPassword[:len(m.confirmPassword)-1]
			}
		}

	default:
		// Handle text input
		if len(msg.String()) == 1 {
			switch m.step {
			case 0:
				m.serverURL += msg.String()
			case 2:
				m.username += msg.String()
			case 3:
				m.password += msg.String()
			case 4:
				m.confirmPassword += msg.String()
			}
		}
	}

	return m, nil
}

func (m setupModel) authenticate() tea.Cmd {
	return func() tea.Msg {
		var authResp *AuthResponse
		var err error

		if m.authMode == "register" {
			authResp, err = registerUser(m.serverURL, m.username, m.password)
		} else {
			authResp, err = loginUser(m.serverURL, m.username, m.password)
		}

		if err != nil {
			return authErrorMsg{error: err.Error()}
		}

		return authSuccessMsg{
			token:    authResp.Token,
			username: authResp.Username,
		}
	}
}
