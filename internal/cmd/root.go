package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"

	"github.com/KatelynHaworth/notarization-helper/v2/config"
	. "github.com/KatelynHaworth/notarization-helper/v2/internal/cmd/globals"
	"github.com/KatelynHaworth/notarization-helper/v2/internal/cmd/notary"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	rootCmd = cobra.Command{
		Use:               "notarization-helper",
		Version:           "devel",
		Short:             "Flexible, simple, cross-platform macOS code signing (soon™️) and notarizing",
		PersistentPreRunE: preRun,
		RunE:              run,
	}

	verbose    *bool
	targetFile *string

	legacyUsername *string
	legacyPassword *string
	legacyTeamId   *string
	legacyStaple   *bool
)

func init() {
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		rootCmd.Version = buildInfo.Main.Version
	}

	verbose = rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enables logging of debug level logs by the utility")
	targetFile = rootCmd.PersistentFlags().StringP("file", "f", "notarization.yaml", "Specifies either the file to process or utility configuration (if the file extension is .json, .yaml, or .yml)")

	legacyUsername = rootCmd.Flags().String("username", "", "(Legacy) Specifies the Apple Developer account username for notarization")
	legacyPassword = rootCmd.Flags().String("password", "", "(Legacy) Specifies the password for the Apple Developer Account")
	legacyTeamId = rootCmd.Flags().String("team-id", "", "(Legacy) Optionally specifies the Team ID associated with the Apple Developer account")
	_ = rootCmd.Flags().String("bundle-id", "", "(Legacy) Specifies the primary bundle ID of the package to be notarized (ignored, not required anymore)")
	legacyStaple = rootCmd.Flags().Bool("staple", false, "(Legacy) Optionally specifies that the notarization ticket should be staple to the package on completion (for supported file types)")

	rootCmd.AddCommand(notary.NotarizeCmd)
}

func preRun(_ *cobra.Command, _ []string) error {
	if *verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	Logger.Info().Msg("Loading utility configuration")
	var err error

	if len(*legacyUsername) != 0 && len(*legacyPassword) != 0 {
		Logger.Warn().Msg("Detected usage of legacy config flags, using legacy configuration mode")

		legacyCfg := &config.ConfigurationV1{
			Username: *legacyUsername,
			Password: *legacyPassword,
			TeamID:   *legacyTeamId,
			Packages: []config.Package{{
				File:   *targetFile,
				Staple: *legacyStaple,
			}},
		}

		Config, err = legacyCfg.ToV2()
		if err != nil {
			return fmt.Errorf("convert V1 config to V2: %w", err)
		}
	} else {
		format := config.ConfigFormatJSON
		if ext := filepath.Ext(*targetFile); ext == ".yaml" || ext == ".yml" {
			format = config.ConfigFormatYAML
		}

		Config, err = config.LoadConfigurationFromFile(*targetFile, format)
		if err != nil {
			return fmt.Errorf("load config from file: %w", err)
		}
	}

	return nil
}

func run(cmd *cobra.Command, args []string) error {
	Logger.Warn().Msg("No sub-command supplied, operating in legacy mode and defaulting to the `notarize` sub-command")

	return notary.NotarizeCmd.RunE(cmd, args)
}

func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		Logger.Fatal().Err(err).Msg("Utility encountered a fatal error")
	}
}
