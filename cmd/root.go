/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/asdf57/homelabc/schemas"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "homelabc",
	Short: "homelab cli tool",
	Long:  `homelabc is a command line tool for managing homelab resources and configurations.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.homelabc.yaml)")
}

var config schemas.Config

func initConfig() error {
	v := viper.GetViper()
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("find home directory: %w", err)
		}

		v.AddConfigPath(home)
		v.SetConfigName(".homelabc")
		v.SetConfigType("yaml")
	}

	v.SetEnvPrefix("HOMELABC")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.SetDefault(gitPrivKeyMountPathKey, "/etc/ssh/git_provisioning_key")
	for _, key := range []string{
		bootstrapContainerEnvFileKey,
		bootstrapMountPathKey,
		bootstrapDockerSocketPathKey,
		bootstrapHostDataPathKey,
		bootstrapImageKey,
		gitPrivKeyPathKey,
		gitPrivKeyMountPathKey,
	} {
		if err := v.BindEnv(key); err != nil {
			return fmt.Errorf("bind environment variable for %s: %w", key, err)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !(cfgFile == "" && errors.As(err, &notFound)) {
			return fmt.Errorf("read config: %w", err)
		}
	}

	if err := v.Unmarshal(&config); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}

	return nil
}
