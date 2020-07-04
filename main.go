package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/danielmmetz/settle/pkg/brew"
	"github.com/danielmmetz/settle/pkg/files"
	"github.com/danielmmetz/settle/pkg/log"
	"github.com/danielmmetz/settle/pkg/nvim"
	"github.com/danielmmetz/settle/pkg/store"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v2"
)

func main() {
	fVerbose := flag.Bool("verbose", false, "enable verbose logging")
	fDumpConfig := flag.Bool("dump-config", false, "pretty print config then exit without applying changes")
	fSkipBrew := flag.Bool("skip-brew", false, "skip applying brew changes")
	flag.Parse()

	var logger log.Log
	if *fVerbose {
		logger.Level = log.LevelDebug
	}

	db, err := sql.Open("sqlite3", "inventory.db")
	if err != nil {
		logger.Fatal("error opening db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	inventory := store.New(db)
	if err := inventory.EnsureTable(ctx); err != nil {
		logger.Fatal("error ensuring table: %v", err)
	}

	config, err := loadConfig(*fSkipBrew)
	if err != nil {
		logger.Fatal("error loading config: %v", err)
	}
	if *fDumpConfig {
		logger.Info("%+v", config)
		os.Exit(0)
	}

	e := NewEnsurer(config)
	if err := e.Ensure(ctx, logger); err != nil {
		logger.Fatal("error applying config: %v", err)
	}
}

type config struct {
	Files files.Files
	Nvim  nvim.Nvim
	Brew  *brew.Brew
}

func (c config) String() string {
	pretty, _ := json.MarshalIndent(c, "", "  ")
	return string(pretty)
}

func loadConfig(skipBrew bool) (config, error) {
	bytes, err := ioutil.ReadFile("settle.yaml")
	if err != nil {
		return config{}, fmt.Errorf("error loading settle.yaml: %w", err)
	}
	var result config
	err = yaml.Unmarshal(bytes, &result)
	if skipBrew {
		result.Brew = nil
	}
	return result, err
}

type ensurer struct {
	log   log.Log
	files files.Files
	nvim  nvim.Nvim
	brew  *brew.Brew
}

func NewEnsurer(cfg config) ensurer {
	return ensurer{
		files: cfg.Files,
		nvim:  cfg.Nvim,
		brew:  cfg.Brew,
	}
}

func (e *ensurer) Ensure(ctx context.Context, logger log.Log) error {
	if err := e.files.Ensure(logger); err != nil {
		return err
	}
	if err := e.nvim.Ensure(ctx, logger); err != nil {
		return err
	}
	return e.brew.Ensure(ctx, logger)
}
