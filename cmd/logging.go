package cmd

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

func InitLogging(cmd *cobra.Command) error {

	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return err
	}

	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return err
	}

	silent, err := cmd.Flags().GetBool("silent")
	if err != nil {
		return err
	}

	json, err := cmd.Flags().GetBool("json-output")
	if err != nil {
		return err
	}

	if silent {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
		return nil
	}

	handler := log.NewWithOptions(os.Stdout, log.Options{
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
	})

	var logger *slog.Logger
	if json {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		logger = slog.New(handler)
	}

	slog.SetDefault(logger)

	if debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		handler.SetLevel(log.DebugLevel)
	}

	if quiet {
		slog.SetLogLoggerLevel(slog.LevelWarn)
		handler.SetLevel(log.WarnLevel)
	}

	return nil

}

func LogFatal(err error) {
	slog.Error(err.Error())
	os.Exit(1)
}

func LogFatalMsg(msg string, err error) {
	slog.Error(msg, "error", err.Error()+"\n")
	os.Exit(1)
}

func LogFatalMultiple(errs []error) {
	for _, e := range errs {
		slog.Error(e.Error())
	}
	os.Exit(1)
}
