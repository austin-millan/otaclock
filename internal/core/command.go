package core

import (
	"strconv"
	"strings"
)

// CommandKind identifies a parsed slash-command.
type CommandKind int

const (
	CommandNotACommand CommandKind = iota
	CommandHelp
	CommandLogin
	CommandConfigure
	CommandPomodoro
	CommandUnknown
)

// ParsedCommand is the structured result of parsing a line of input.
type ParsedCommand struct {
	Kind CommandKind
	// Key/Value carry the configure arguments.
	Key   string
	Value string
	// State carries the pomodoro sub-command (start/stop/restart).
	State string
	// Unknown carries the offending verb for CommandUnknown.
	Unknown string
}

// CommandEffect signals a side effect the session must perform.
type CommandEffect int

const (
	EffectNone CommandEffect = iota
	// EffectRestartMGXBoot replays the MGX DATAFLOW boot sequence in the chat panel.
	EffectRestartMGXBoot
)

// CommandResult is the outcome of executing a command.
type CommandResult struct {
	Success bool
	Lines   []string
	Effect  CommandEffect
}

func ok(effect CommandEffect, lines ...string) CommandResult {
	return CommandResult{Success: true, Lines: lines, Effect: effect}
}

func fail(lines ...string) CommandResult {
	return CommandResult{Success: false, Lines: lines, Effect: EffectNone}
}

var commandVerbs = map[string]bool{
	"help": true, "h": true, "?": true,
	"login":     true,
	"configure": true, "config": true, "cfg": true,
	"pomodoro": true, "pomo": true,
}

// ParseCommand classifies a line as a known command or plain chat.
func ParseCommand(line string) ParsedCommand {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ParsedCommand{Kind: CommandNotACommand}
	}

	body := trimmed
	if strings.HasPrefix(body, "/") {
		body = strings.TrimSpace(strings.TrimPrefix(body, "/"))
	}
	if body == "" {
		return ParsedCommand{Kind: CommandNotACommand}
	}

	parts := splitCommandLine(body)
	if len(parts) == 0 {
		return ParsedCommand{Kind: CommandNotACommand}
	}
	verb := strings.ToLower(parts[0])
	if !commandVerbs[verb] {
		return ParsedCommand{Kind: CommandNotACommand}
	}
	parts = parts[1:]

	switch verb {
	case "help", "h", "?":
		return ParsedCommand{Kind: CommandHelp}
	case "login":
		return ParsedCommand{Kind: CommandLogin}
	case "configure", "config", "cfg":
		if len(parts) < 2 {
			return ParsedCommand{Kind: CommandUnknown, Unknown: "configure (missing key and value)"}
		}
		key := strings.ToLower(parts[0])
		value := strings.Join(parts[1:], " ")
		return ParsedCommand{Kind: CommandConfigure, Key: key, Value: value}
	case "pomodoro", "pomo":
		state := ""
		if len(parts) > 0 {
			state = strings.ToLower(parts[0])
		}
		return ParsedCommand{Kind: CommandPomodoro, State: state}
	default:
		return ParsedCommand{Kind: CommandUnknown, Unknown: verb}
	}
}

// splitCommandLine splits on whitespace, honoring simple double-quoted segments.
func splitCommandLine(input string) []string {
	var tokens []string
	var current strings.Builder
	inQuotes := false

	for _, ch := range input {
		switch {
		case ch == '"':
			inQuotes = !inQuotes
		case (ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r') && !inQuotes:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// CommandContext is the mutable state a command may read or modify.
type CommandContext struct {
	Settings Settings
	Pomodoro PomodoroTimer
	NowMs    int
}

// ExecuteCommand parses and runs a line against the context.
func ExecuteCommand(line string, ctx *CommandContext) CommandResult {
	parsed := ParseCommand(line)
	if parsed.Kind == CommandNotACommand {
		return fail("Internal error: execute called on non-command input.")
	}
	return ExecuteParsed(parsed, ctx)
}

// ExecuteParsed runs an already-parsed command against the context.
func ExecuteParsed(parsed ParsedCommand, ctx *CommandContext) CommandResult {
	switch parsed.Kind {
	case CommandHelp:
		return ok(EffectNone, HelpText())
	case CommandLogin:
		return ok(EffectRestartMGXBoot, "MGX boot sequence restarted.")
	case CommandConfigure:
		return configure(parsed.Key, parsed.Value, ctx)
	case CommandPomodoro:
		return controlPomodoro(parsed.State, ctx)
	case CommandUnknown:
		return fail("Unknown command: "+parsed.Unknown, "Type help for available commands.")
	default:
		return fail("Internal error: execute called on non-command input.")
	}
}

func configure(key, value string, ctx *CommandContext) CommandResult {
	switch key {
	case "ai.enable":
		enabled, okBool := parseBool(value)
		if !okBool {
			return fail("Expected boolean: true | false")
		}
		ctx.Settings.AIEnabled = enabled
		return ok(EffectNone, "ai.enabled = "+strconv.FormatBool(enabled))

	case "ai.provider":
		provider, valid := ProviderFromAlias(value)
		if !valid {
			return fail("Provider must be: claude | openai")
		}
		ctx.Settings.Provider = provider
		return ok(EffectNone, "ai.provider = "+string(provider))

	case "ai.api_key":
		ctx.Settings.APIKey = value
		return ok(EffectNone, "ai.api_key updated.")

	case "ai.host":
		ctx.Settings.Host = value
		return ok(EffectNone, "ai.host = "+emptyAsDefault(value))

	case "ai.model":
		ctx.Settings.Model = value
		return ok(EffectNone, "ai.model = "+emptyAsDefault(value))

	case "pomodoro.short":
		minutes, err := strconv.Atoi(value)
		if err != nil || minutes <= 0 {
			return fail("Expected positive integer (minutes).")
		}
		ctx.Pomodoro.Config.ShortBreakMinutes = minutes
		return ok(EffectNone, "pomodoro.short = "+strconv.Itoa(minutes)+" min")

	case "pomodoro.long":
		minutes, err := strconv.Atoi(value)
		if err != nil || minutes <= 0 {
			return fail("Expected positive integer (minutes).")
		}
		ctx.Pomodoro.Config.LongBreakMinutes = minutes
		return ok(EffectNone, "pomodoro.long = "+strconv.Itoa(minutes)+" min")

	default:
		return fail("Unknown configure key: "+key, configureKeysHelp())
	}
}

func controlPomodoro(state string, ctx *CommandContext) CommandResult {
	switch state {
	case "start":
		ctx.Pomodoro.StartOrResume(ctx.NowMs)
		return ok(EffectNone, "Pomodoro started.", ctx.Pomodoro.StatusText())
	case "stop":
		ctx.Pomodoro.Stop()
		return ok(EffectNone, "Pomodoro stopped.", ctx.Pomodoro.StatusText())
	case "restart":
		ctx.Pomodoro.Restart(ctx.NowMs)
		return ok(EffectNone, "Pomodoro restarted.", ctx.Pomodoro.StatusText())
	case "":
		return fail("Usage: pomodoro <start|stop|restart>")
	default:
		return fail("Unknown state: "+state, "Usage: pomodoro <start|stop|restart>")
	}
}

// HelpText is the full command reference shown by `help`.
func HelpText() string {
	return `OtaClock commands
─────────────────
help
    Show this help.

login
    Replay the MGX DATAFLOW boot sequence.

configure <key> <value>
    ai.enable <bool>       true | false
    ai.provider <string>   claude | openai
    ai.api_key <string>    API key for active provider
    ai.host <string>       API base URL (empty = default)
    ai.model <string>      Model id (empty = default)
    pomodoro.short <int>   Short break length (minutes)
    pomodoro.long <int>    Long break length (minutes)

pomodoro <state>
    start | stop | restart

Anything else is sent to Otacon via the codec.`
}

func configureKeysHelp() string {
	return "Keys: ai.enable, ai.provider, ai.api_key, ai.host, ai.model, pomodoro.short, pomodoro.long"
}

func parseBool(raw string) (bool, bool) {
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}
