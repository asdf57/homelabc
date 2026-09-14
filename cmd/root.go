/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	appconfig "github.com/asdf57/homelabc/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	settings = viper.NewWithOptions(viper.ExperimentalBindStruct())
)

const (
	generalImageKey           = "general.image"
	generalStigmergyAPIURLKey = "general.stigmergy_url"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "homelabc",
	Short: "homelab cli tool",
	Long:  `homelabc is a command line tool for managing homelab resources and configurations.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	flags := rootCmd.PersistentFlags()
	flags.StringVar(&cfgFile, "config", "", "config file (default is $HOME/.homelabc.yaml)")
	flags.String("image", "docker.io/library/homelab:latest", "homelab image")
	flags.String("stigmergy-api-url", "", "Stigmergy API URL")
	cobra.CheckErr(settings.BindPFlag(generalImageKey, flags.Lookup("image")))
	cobra.CheckErr(settings.BindPFlag(generalStigmergyAPIURLKey, flags.Lookup("stigmergy-api-url")))
}

func loadConfig() (appconfig.Config, error) {
	if cfgFile != "" {
		settings.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return appconfig.Config{}, fmt.Errorf("find home directory: %w", err)
		}

		settings.AddConfigPath(home)
		settings.SetConfigName(".homelabc")
		settings.SetConfigType("yaml")
	}

	settings.SetEnvPrefix("HOMELABC")
	settings.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	settings.AutomaticEnv()
	if err := settings.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !(cfgFile == "" && errors.As(err, &notFound)) {
			return appconfig.Config{}, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg appconfig.Config
	if err := settings.Unmarshal(&cfg); err != nil {
		return appconfig.Config{}, fmt.Errorf("decode config: %w", err)
	}
	cfg.ApplyDefaults()

	return cfg, nil
}
