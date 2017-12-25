package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gorilla/sessions"
	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"

	"github.com/umputun/remark/app/migrator"
	"github.com/umputun/remark/app/rest"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/store"
)

var opts struct {
	DBFile  string   `long:"db" env:"BOLTDB_FILE" default:"/tmp/remark.db" description:"bolt file name"`
	SiteURL string   `long:"site-url" env:"REMARK_URL" default:"http://remark.umputun.com:8080" description:"url to remark site"`
	Admins  []string `long:"admin" env:"ADMIN" default:"umputun@gmail.com" description:"admin(s) names" env-delim:","`
	DevMode bool     `long:"dev" env:"DEV" description:"development mode, no auth enforced"`
	Dbg     bool     `long:"dbg" env:"DEBUG" description:"debug mode"`

	ServerCommand struct {
		SessionStore string `long:"session" env:"SESSION_STORE" default:"/tmp" description:"path to session store directory"`
		StoreKey     string `long:"store-key" env:"STORE_KEY" default:"secure-store-key" description:"store key"`

		GoogleCID  string `long:"google-cid" env:"REMARK_GOOGLE_CID" description:"Google OAuth client ID"`
		GoogleCSEC string `long:"google-csec" env:"REMARK_GOOGLE_CSEC" description:"Google OAuth client secret"`
		GithubCID  string `long:"github-cid" env:"REMARK_GITHUB_CID" description:"Github OAuth client ID"`
		GithubCSEC string `long:"github-csec" env:"REMARK_GITHUB_CSEC" description:"Github OAuth client secret"`
	} `command:"server" description:"run server"`

	ImportCommand struct {
		Provider  string `long:"provider" default:"disqus" description:"provider type"`
		SiteID    string `long:"site" default:"site" description:"site ID"`
		InputFile string `long:"file" default:"disqus.xml" description:"input file"`
	} `command:"import" description:"import comments from external sources"`
}

var revision = "unknown"

func main() {
	fmt.Printf("remark %s\n", revision)
	p := flags.NewParser(&opts, flags.Default)
	if _, e := p.ParseArgs(os.Args[1:]); e != nil {
		os.Exit(1)
	}

	setupLog(opts.Dbg)
	log.Print("[INFO] started remark")

	dataStore, err := store.NewBoltDB(opts.DBFile)
	if err != nil {
		log.Fatalf("[ERROR] can't initialize data store, %+v", err)
	}

	if p.Active != nil && p.Command.Find("import") == p.Active {
		if err := importComments(dataStore); err != nil {
			log.Fatalf("[ERROR] failed to import, %+v", err)
		}
		return
	}

	sessionStore := sessions.NewFilesystemStore(opts.ServerCommand.SessionStore, []byte(opts.ServerCommand.StoreKey))

	srv := rest.Server{
		Version:      revision,
		Store:        dataStore,
		SessionStore: sessionStore,
		Admins:       opts.Admins,
		DevMode:      opts.DevMode,
		AuthGoogle: auth.NewGoogle(auth.Params{
			Cid:          opts.ServerCommand.GoogleCID,
			Csecret:      opts.ServerCommand.GoogleCSEC,
			SessionStore: sessionStore,
			SiteURL:      opts.SiteURL,
		}),
		AuthGithub: auth.NewGithub(auth.Params{
			Cid:          opts.ServerCommand.GithubCID,
			Csecret:      opts.ServerCommand.GithubCSEC,
			SessionStore: sessionStore,
			SiteURL:      opts.SiteURL,
		}),
	}

	if opts.DevMode {
		log.Printf("[WARN] running in dev mode, no auth!")
	}

	srv.Run()
}

func importComments(dataStore store.Interface) error {
	log.Printf("[INFO] import from %s (%s) to %s",
		opts.ImportCommand.InputFile, opts.ImportCommand.Provider, opts.ImportCommand.SiteID)
	importer := migrator.Disqus{DataStore: dataStore}

	fh, err := os.Open(opts.ImportCommand.InputFile)
	if err != nil {
		return errors.Wrapf(err, "can't open import file %s", opts.ImportCommand.InputFile)
	}

	defer func() {
		if err = fh.Close(); err != nil {
			log.Printf("[WARN] can't close %s, %s", opts.ImportCommand.InputFile, err)
		}
	}()

	return importer.Import(fh, opts.ImportCommand.SiteID)
}

func setupLog(dbg bool) {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel("INFO"),
		Writer:   os.Stdout,
	}

	log.SetFlags(log.Ldate | log.Ltime)

	if dbg {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
		filter.MinLevel = logutils.LogLevel("DEBUG")
	}
	log.SetOutput(filter)
}
