package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Zsh struct {
	Zinit   []string `json:"zinit"`
	History struct {
		Size          int  `json:"size"`
		ShareHistory  bool `json:"share_history"`
		IncAppend     bool `json:"inc_append"`
		IgnoreAllDups bool `json:"ignore_all_dups"`
		IgnoreSpace   bool `json:"ignore_space"`
	} `json:"history"`
	Variables   []KV   `json:"variables"`
	Aliases     []KV   `json:"aliases"`
	Functions   []KV   `json:"functions"`
	ExtraPrefix string `json:"extra_prefix"`
	ExtraSuffix string `json:"extra_suffix"`
}

type KV struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (z *Zsh) Ensure(ctx context.Context) error {
	if z == nil {
		return nil
	}

	if err := z.ensureZinit(ctx); err != nil {
		return fmt.Errorf("error ensuring zinit: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home dir: %w", err)
	}
	fmt.Println("writing .zshrc")
	if err := ioutil.WriteFile(filepath.Join(home, ".zshrc"), []byte(z.String()), 0644); err != nil {
		return fmt.Errorf("error writing .zshrc: %w", err)
	}
	return nil
}

const (
	zinitURL = "https://raw.githubusercontent.com/zdharma/zinit/master/zinit.zsh"
)

func (z *Zsh) ensureZinit(ctx context.Context) error {
	if len(z.Zinit) == 0 {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home dir: %w", err)
	}
	dstPath := filepath.Join(home, ".zinit", "bin", "zinit.zsh")

	_, err = os.Stat(dstPath)
	if err == nil {
		return nil // file already exists
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	fmt.Println("installing Zinit")
	cloneCmd := exec.CommandContext(ctx, "git", "clone", "https://github.com/zdharma/zinit.git", filepath.Join(home, ".zinit", "bin"))
	if output, err := cloneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error cloning zinit: %w\n%s", err, string(output))
	}
	return nil
}

func (z *Zsh) String() string {
	var sb strings.Builder

	// extra prefix
	sb.WriteString(z.ExtraPrefix)
	sb.WriteString("\n")

	// zinit
	for _, line := range z.Zinit {
		sb.WriteString(fmt.Sprintf("zinit %s\n", line))
	}
	sb.WriteString("\n")

	// history
	if z.History.Size != 0 {
		sb.WriteString(fmt.Sprintf("HISTSIZE=%d\n", z.History.Size))
		sb.WriteString(fmt.Sprintf("SAVEHIST=%d\n", z.History.Size))
	}
	if z.History.ShareHistory {
		sb.WriteString("setopt SHARE_HISTORY\n")
	}
	if z.History.IncAppend {
		sb.WriteString("setopt INC_APPEND_HISTORY\n")
	}
	if z.History.IgnoreAllDups {
		sb.WriteString("setopt HIST_IGNORE_ALL_DUPS\n")
	}
	if z.History.IgnoreSpace {
		sb.WriteString("setopt HIST_IGNORE_SPACE\n")
	}
	sb.WriteString("\n")

	// variables
	for _, kv := range z.Variables {
		sb.WriteString(fmt.Sprintf("export %s=%s\n", kv.Name, kv.Value))
	}
	sb.WriteString("\n")

	// aliases
	for _, kv := range z.Aliases {
		sb.WriteString(fmt.Sprintf("alias %s=\"%s\"\n", kv.Name, kv.Value))
	}
	sb.WriteString("\n")

	// functions
	for _, kv := range z.Functions {
		sb.WriteString(fmt.Sprintf("function %s() {\n", kv.Name))
		for _, l := range strings.Split(kv.Value, "\n") {
			sb.WriteString(fmt.Sprintf("\t%s\n", l))
		}
		sb.WriteString("}\n")
	}
	sb.WriteString("\n")

	// extra suffix
	sb.WriteString(z.ExtraSuffix)
	sb.WriteString("\n")
	return sb.String()
}
