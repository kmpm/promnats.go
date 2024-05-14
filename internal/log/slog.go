package log

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

const (
	LevelFatal = slog.Level(12)
)

type slogAdapter struct {
	slog *slog.Logger
}

func NewSlogAdapter(l *slog.Logger) *slogAdapter {
	return &slogAdapter{slog: l}
}

func (l *slogAdapter) clone() *slogAdapter {
	c := *l
	return &c
}

func (l *slogAdapter) WithFields(fields map[string]string) Modular {
	tmp := l.slog
	for k, v := range fields {
		tmp = tmp.With(k, v)
	}
	c := l.clone()
	c.slog = tmp
	return c
}

func (l *slogAdapter) With(keyValues ...any) Modular {
	c := l.clone()
	c.slog = l.slog.With(keyValues...)
	return c
}

func (l *slogAdapter) Fatal(format string, v ...any) {
	l.slog.Log(context.Background(), LevelFatal, fmt.Sprintf(format, v...))
	// l.slog.Error(fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (l *slogAdapter) Error(format string, v ...any) {
	l.slog.Error(fmt.Sprintf(format, v...))
}

func (l *slogAdapter) Warn(format string, v ...any) {
	l.slog.Warn(fmt.Sprintf(format, v...))
}

func (l *slogAdapter) Info(format string, v ...any) {
	l.slog.Info(fmt.Sprintf(format, v...))
}

func (l *slogAdapter) Debug(format string, v ...any) {
	l.slog.Debug(fmt.Sprintf(format, v...))
}

func (l *slogAdapter) Fatalln(message string) {
	l.slog.Error(message)
	os.Exit(1)
}

func (l *slogAdapter) Errorln(message string) {
	l.slog.Error(message)
}

func (l *slogAdapter) Warnln(message string) {
	l.slog.Warn(message)
}

func (l *slogAdapter) Infoln(message string) {
	l.slog.Info(message)
}

func (l *slogAdapter) Debugln(message string) {
	l.slog.Debug(message)
}
