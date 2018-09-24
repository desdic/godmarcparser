package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"dmarc/pkg/cfg"
	"dmarc/pkg/dmarc"
	"dmarc/pkg/storage"
	"dmarc/pkg/version"

	log "github.com/sirupsen/logrus"
)

var (
	s storage.Storage

	errors chan error
	queue  chan dmarc.Content
)

func run(ctx context.Context, cancel context.CancelFunc, cfg cfg.HTTPCfg) error {

	if err := s.Initialize(ctx); err != nil {
		return fmt.Errorf("Unable to initialize storage: %v", err)
	}

	srv, c := httpStart(ctx, cancel, cfg)

	// Gracefull shutdown via ctrl+c or if something fails during startup
	log.Debug("Web service started")
	select {
	case <-ctx.Done():
		return fmt.Errorf("Stopping due to errors")
	case <-c:
		log.Infof("Shutting down")
	}
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("Shutdown failed: %v", err)
	}

	return nil
}

func main() {

	var (
		showVersion bool
		cfgfile     string
	)
	// Parse flags
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.StringVar(&cfgfile, "cfgfile", "config.json", "Path to config file")
	flag.Parse()

	if showVersion {
		version.Show()
		os.Exit(0)
	}

	c, err := cfg.ReadConfig(cfgfile)
	if err != nil {
		log.Fatal(err)
	}

	// Support for multiple drivers
	switch c.Storage.Type {
	case "postgresql":
		s = &storage.Postgresql{
			URL: c.Storage.URL,
		}
	default:
		log.Fatalf("Unsupported driver %s", c.Storage.Type)
	}

	switch c.Log.Level {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	errors = make(chan error)
	queue = make(chan dmarc.Content)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for e := range errors {
			log.Errorf("Got error %#v\n", e)
		}
	}()

	go func() {
		for q := range queue {
			log.Debugf("Reading %s", q.From)
			f, err := dmarc.Read(q.Data.Bytes())
			if err != nil {
				errors <- fmt.Errorf("Unable to parse %s: %v", q.Name, err)
				continue
			}
			f.FromFile = q.From
			if err = s.Write(ctx, f); err != nil {
				errors <- fmt.Errorf("Unable to store feedback: %v", err)
			}
		}
	}()

	go func(ctx context.Context, c cfg.ScanDirectory) {
		if c.Path == "" {
			return
		}
	DONE:
		for {
			log.Debugf("Scanning directory %s", c.Path)
			ScanDirectory(ctx, queue, errors, c.Path)

			select {
			case <-ctx.Done():
				break DONE
			default:
			}

			time.Sleep(time.Duration(c.Interval) * time.Second)
		}
	}(ctx, c.Directory)

	if err := run(ctx, cancel, c.HTTP); err != nil {
		log.Errorf("Stopping server: %v", err)
	}
}
