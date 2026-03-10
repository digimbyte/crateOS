package users

// ShellAccessRequest represents a request to enter break-glass shell.
type ShellAccessRequest struct {
	User      string `json:"user"`
	Reason    string `json:"reason,omitempty"`
	SourceIP  string `json:"source_ip,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// ShellAccessLog tracks shell access for audit.
type ShellAccessLog struct {
	Timestamp   string `json:"timestamp"`
	User        string `json:"user"`
	Result      string `json:"result"` // allowed, denied
	Reason      string `json:"reason,omitempty"`
	SourceIP    string `json:"source_ip,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	ExitCode    int    `json:"exit_code,omitempty"`
	Duration    string `json:"duration,omitempty"`
}
