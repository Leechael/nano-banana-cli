package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type OutputMode string

const (
	ModeHuman OutputMode = "human"
	ModePlain OutputMode = "plain"
	ModeJSON  OutputMode = "json"
)

type Result struct {
	OK       bool      `json:"ok"`
	Model    string    `json:"model,omitempty"`
	Prompt   string    `json:"prompt,omitempty"`
	Size     string    `json:"size,omitempty"`
	Aspect   string    `json:"aspect,omitempty"`
	Files    []string  `json:"files,omitempty"`
	Cost     float64   `json:"cost,omitempty"`
	Error    string    `json:"error,omitempty"`
	Warnings []string  `json:"warnings,omitempty"`
	At       time.Time `json:"at"`
}

func LogLine(mode OutputMode, level, format string, a ...any) {
	switch mode {
	case ModeHuman:
		fmt.Fprintf(os.Stderr, "[imagen] "+format+"\n", a...)
	case ModePlain:
		fmt.Fprintf(os.Stderr, format+"\n", a...)
	}
}

func ExitError(err error, mode OutputMode) {
	if mode == ModeJSON {
		_ = json.NewEncoder(os.Stdout).Encode(Result{OK: false, Error: err.Error(), At: time.Now().UTC()})
	} else {
		fmt.Fprintln(os.Stderr, "Error:", err.Error())
	}
	os.Exit(1)
}
