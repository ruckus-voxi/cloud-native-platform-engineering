package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
)

const (
	cols         = "\n%s%-10s%10s%s%s"
	errbar       = "[error]"
	debugbar     = "[debug]"
	warnbar      = "[warning]"
	infobar      = "[info]"
	inputbar     = "[input]"
	inputPrefix  = "input:"
	linePrefix   = "line:"
	headerPrefix = "header:"
	timeFormat   = "[15:04:05.000]"
	Blue         = "\033[34;1m"
	DarkGrey     = "\033[90;1m"
	Green        = "\033[32;1m"
	Grey         = "\033[37;1m"
	Magenta      = "\033[35;1m"
	Yellow       = "\033[33;1m"
	Red          = "\033[31;1m"
	Reset        = "\033[0m"
)

var logger *slog.Logger

// slog handler inspired by: https://dusted.codes/creating-a-pretty-console-logger-using-gos-slog-package

type Handler struct {
	h slog.Handler
	b *bytes.Buffer
	m *sync.Mutex
}

func NewHandler(opts *slog.HandlerOptions) *Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		}
	}

	b := &bytes.Buffer{}

	return &Handler{
		b: b,
		h: slog.NewJSONHandler(b, &slog.HandlerOptions{
			Level:       opts.Level,
			AddSource:   opts.AddSource,
			ReplaceAttr: suppressDefaults(opts.ReplaceAttr),
		}),
		m: &sync.Mutex{},
	}
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		h: h.h.WithAttrs(attrs),
		b: h.b,
		m: h.m,
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		h: h.h.WithGroup(name),
		b: h.b,
		m: h.m,
	}
}

type Logmsg struct {
	color   string
	logbar  string
	message string
	JSON    string
	time    string
}

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	lm := Logmsg{
		message: record.Message,
		time:    record.Time.Format(timeFormat),
	}

	switch record.Level {
	case slog.LevelDebug:
		lm.logbar = debugbar
		lm.color = Magenta
	case slog.LevelInfo:
		lm.logbar = infobar
		lm.color = Green
	case slog.LevelWarn:
		lm.logbar = warnbar
		lm.color = Yellow
	case slog.LevelError:
		lm.logbar = errbar
		lm.color = Red
	}

	attrs, err := h.computeAttrs(ctx, record)
	if err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(attrs, "", "  ")
	if err != nil {
		return fmt.Errorf("error when marshaling attrs: %w", err)
	}

	lm.JSON = string(bytes)
	colorize(&lm)

	return nil
}

//nolint:forbidigo
func colorize(l *Logmsg) {
	msg := fmt.Sprintf(cols, l.color, l.logbar, Grey, l.message, Reset)

	switch {
	case l.logbar == debugbar:
		// print timestamp above msg with no new line
		tstamp := fmt.Sprintf(cols, l.color, l.logbar, Grey, l.time, Reset)
		fmt.Print(tstamp)
		fmt.Println(msg)
	case strings.HasPrefix(l.message, inputPrefix):
		// change logbar to [input] and print without newline
		i := customLogLine(l.message, inputPrefix)
		msg := fmt.Sprintf(cols, l.color, i[2], Grey, i[3], Reset)
		fmt.Print(msg)
	case strings.HasPrefix(l.message, linePrefix):
		// log line without a logbar, and without newline
		i := customLogLine(l.message, linePrefix)
		msg := fmt.Sprintf(cols, i[1], "", "", i[3], Reset)
		fmt.Print(msg)
	case strings.HasPrefix(l.message, headerPrefix):
		// colorized header for subsequent log lines
		i := customLogLine(l.message, headerPrefix)
		msg := fmt.Sprintf(cols, l.color, "", "", i[3], Reset)
		fmt.Printf("\n%s", msg)
	default:
		fmt.Println(msg)
	}
}

func customLogLine(msg, prefix string) map[int]string {
	attrs := make(map[int]string, 3)
	if prefix == inputPrefix {
		attrs[2] = inputbar
	}

	attrs[1] = Grey
	attrs[3] = strings.TrimPrefix(msg, prefix)

	return attrs
}

func (h *Handler) computeAttrs(ctx context.Context, record slog.Record) (map[string]any, error) {
	h.m.Lock()

	defer func() {
		h.b.Reset()
		h.m.Unlock()
	}()

	if err := h.h.Handle(ctx, record); err != nil {
		return nil, fmt.Errorf("error when calling inner handler's Handle: %w", err)
	}

	var attrs map[string]any

	err := json.Unmarshal(h.b.Bytes(), &attrs)
	if err != nil {
		return nil, fmt.Errorf("error when unmarshaling inner handler's Handle result: %w", err)
	}

	return attrs, nil
}

func suppressDefaults(
	next func([]string, slog.Attr) slog.Attr,
) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, attr slog.Attr) slog.Attr {
		if attr.Key == slog.TimeKey ||
			attr.Key == slog.LevelKey ||
			attr.Key == slog.MessageKey {
			return slog.Attr{}
		}

		if next == nil {
			return attr
		}

		return next(groups, attr)
	}
}

func InputPrompt(log string, cond string, promptStr string) bool {
	var input string

	prompt := fmt.Sprintf("%s%s: ", inputPrefix, promptStr)

	switch log {
	case "debug":
		logger.Debug(prompt)
	case "info":
		logger.Info(prompt)
	case "warn":
		logger.Warn(prompt)
	default:
		logger.Info(prompt)
	}

	r := bufio.NewReader(os.Stdin)

	val, err := r.ReadString('\n')
	if err != nil {
		errStr := fmt.Sprintf("%s input error: %v", strings.ToLower(promptStr), err)
		logger.Error(errStr)
	}

	input = strings.TrimSpace(val)

	return input == cond
}

func init() {
	logger = slog.New(NewHandler(nil))
}

func NewLogger() *slog.Logger {
	return slog.New(NewHandler(nil))
}
