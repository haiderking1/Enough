package tui

type composerMode int

const (
	modeTask composerMode = iota
	modeConnect
)

const (
	taskPlaceholder    = "describe what you want done..."
	connectPlaceholder = "paste api key..."
)

type slashCommand struct {
	name string
	desc string
}

var slashCommands = []slashCommand{
	{name: "connect", desc: "link your OpenCode API key"},
}
