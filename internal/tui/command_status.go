package tui

func (m model) commandLaneEnabled() bool {
	if m.currentView == viewUsers && m.userFormOpen {
		return false
	}
	return true
}

func (m *model) setCommandStatus(level, message string) {
	m.commandStatusLevel = level
	m.commandStatus = message
}

func (m *model) setCommandInfo(message string) {
	m.setCommandStatus("info", message)
}

func (m *model) setCommandOK(message string) {
	m.setCommandStatus("ok", message)
}

func (m *model) setCommandWarn(message string) {
	m.setCommandStatus("warn", message)
}

func (m *model) setCommandError(message string) {
	m.setCommandStatus("error", message)
}

func (m model) commandHelpText() string {
	switch m.currentView {
	case viewServices:
		return "services help: svc <list|enable|start|stop|disable|install|uninstall|restart|next|prev|select> [service|service1,service2|all], or <service> <action>"
	case viewUsers:
		return "users help: user <list|add|rename|set|role|perms|delete|next|prev|select> [name|name1,name2]"
	case viewLogs:
		return "logs help: log <list|next|prev|select> [service|service1,service2], log service <list|next|prev|select> [service|service1,service2], log source <list|next|prev|select> [source|source1,source2]"
	case viewNetwork:
		return "network help: net <list|next|prev|select> [iface|iface1,iface2], nav <view>, system <refresh|ftp-complete <path|dir>>"
	case viewDiagnostics:
		return "diagnostics help: diag <summary|verification|ownership|config|actor [target]|focus <target>|next|prev|select>, list actors, nav <view>, system <refresh|ftp-complete <path|dir>>"
	case viewStatus:
		return "status help: status <system|services|platform|next|prev>, nav <view>, system <refresh|ftp-complete <path|dir>>"
	case viewSetup:
		return "primer help: type hostname then admin, enter to apply | bootstrap <admin> | system refresh (primer stays locked until checks pass)"
	default:
		return "grammar: command mod params... | commands: help list nav status diag svc user log net bootstrap system back | system: refresh | dos2unix [config|services|all] | ftp-complete <path|dir> | service-as-command: <service> <action> | chaining: cmd1; cmd2 or cmd1 && cmd2"
	}
}
