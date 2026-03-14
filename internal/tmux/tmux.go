package tmux

import (
	"fmt"
	"math/rand"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Session represents a tmux session.
type Session struct {
	Name     string
	Windows  int
	Created  time.Time
	Attached bool
	Age      string // human-readable, e.g. "5m ago"
}

var adjectives = []string{
	"swift", "calm", "bright", "bold", "cool", "dark", "fast", "keen",
	"sharp", "warm", "wild", "free", "crisp", "clear", "deep", "gold",
	"iron", "jade", "nova", "pure", "ruby", "sage", "slim", "true",
	"vast", "wise",
}

var nouns = []string{
	"fox", "owl", "elk", "bay", "cove", "dawn", "dusk", "edge",
	"fern", "gate", "hawk", "isle", "knot", "lake", "mesa", "node",
	"oak", "pine", "reef", "sage", "tide", "vale", "wave", "wolf",
	"apex", "bolt", "core", "dome", "echo", "flux", "grid", "haze",
	"iris",
}

// ListSessions returns all tmux sessions, or nil if none.
func ListSessions() ([]Session, error) {
	out, err := exec.Command("tmux", "ls", "-F",
		"#{session_name}|#{session_windows}|#{session_created}|#{?session_attached,attached,}",
	).Output()
	if err != nil {
		return nil, nil // no server or no sessions
	}

	now := time.Now()
	var sessions []Session
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}

		name := stripANSI(parts[0])
		wins, _ := strconv.Atoi(parts[1])
		created, _ := strconv.ParseInt(parts[2], 10, 64)
		attached := parts[3] == "attached"

		createdTime := time.Unix(created, 0)
		age := formatAge(now.Sub(createdTime))

		sessions = append(sessions, Session{
			Name:     name,
			Windows:  wins,
			Created:  createdTime,
			Attached: attached,
			Age:      age,
		})
	}
	return sessions, nil
}

// NewSessionCmd builds an exec.Cmd for creating a new tmux session.
func NewSessionCmd(name, dir string, args ...string) *exec.Cmd {
	cmdArgs := []string{"new", "-s", name, "-c", dir}
	if len(args) > 0 {
		cmdArgs = append(cmdArgs, "--")
		cmdArgs = append(cmdArgs, args...)
	}
	return exec.Command("tmux", cmdArgs...)
}

// AttachSessionCmd builds an exec.Cmd for attaching to a tmux session.
func AttachSessionCmd(name string) *exec.Cmd {
	return exec.Command("tmux", "a", "-t", name)
}

// KillSession kills a tmux session by name.
func KillSession(name string) error {
	return exec.Command("tmux", "kill-session", "-t", name).Run()
}

// RenameSession renames a tmux session.
func RenameSession(oldName, newName string) error {
	return exec.Command("tmux", "rename-session", "-t", oldName, newName).Run()
}

// CapturePaneLast150 captures the last 150 lines from a session's active pane.
func CapturePaneLast150(name string) (string, error) {
	out, err := exec.Command("tmux", "capture-pane", "-t", name, "-p", "-S", "-150").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// RandomName generates a unique adjective-noun session name.
func RandomName() string {
	existing := make(map[string]bool)
	if sessions, _ := ListSessions(); sessions != nil {
		for _, s := range sessions {
			existing[s.Name] = true
		}
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		adj := adjectives[r.Intn(len(adjectives))]
		noun := nouns[r.Intn(len(nouns))]
		name := adj + "-" + noun
		if !existing[name] {
			return name
		}
	}
}

func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func stripANSI(s string) string {
	var result strings.Builder
	inEsc := false
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') {
				inEsc = false
			}
			continue
		}
		result.WriteByte(s[i])
	}
	return strings.TrimSpace(result.String())
}
