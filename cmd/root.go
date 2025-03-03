package cmd

import (
	"fmt"
	"os"

	"github.com/alecthomas/chroma/v2"
	"github.com/charmbracelet/bubbletea"
	"github.com/noahgorstein/jqp/tui/bubbles/jqplayground"
	"github.com/noahgorstein/jqp/tui/theme"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Version:      "0.6.0",
	Use:          "jqp",
	Short:        "jqp is a TUI to explore jq",
	Long:         `jqp is a TUI to explore the jq command line utility`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		configTheme := viper.GetString(configKeysName.themeName)
		if !cmd.Flags().Changed(flagsName.theme) {
			flags.theme = configTheme
		}
		themeOverrides := viper.GetStringMapString(configKeysName.themeOverrides)

		styleOverrides := viper.GetStringMapString(configKeysName.styleOverrides)
		jqtheme, defaultTheme := theme.GetTheme(flags.theme, styleOverrides)

		// If not using the default theme,
		// and if theme specified is the same as in the config,
		// which happens if the theme flag was used,
		// apply chroma style overrides.
		if !defaultTheme && configTheme == flags.theme && len(themeOverrides) > 0 {
			// Reverse chroma.StandardTypes to be keyed by short string
			chromaTypes := make(map[string]chroma.TokenType)
			for tokenType, short := range chroma.StandardTypes {
				chromaTypes[short] = tokenType
			}

			builder := jqtheme.ChromaStyle.Builder()
			for k, v := range themeOverrides {
				builder.Add(chromaTypes[k], v)
			}
			style, err := builder.Build()
			if err == nil {
				jqtheme.ChromaStyle = style
			}
		}

		if isInputFromPipe() {
			stdin := streamToBytes(os.Stdin)

			_, isJsonLines, err := isValidInput(stdin)
			if err != nil {
				return err
			}

			bubble := jqplayground.New(stdin, "STDIN", jqtheme, isJsonLines)
			p := tea.NewProgram(bubble, tea.WithAltScreen())
			if err := p.Start(); err != nil {
				return err
			}
			return nil
		}

		// get the file
		file, e := getFile()
		if e != nil {
			return e
		}
		defer file.Close()

		// read the file
		data, err := os.ReadFile(flags.filepath)
		if err != nil {
			return err
		}

		_, isJsonLines, err := isValidInput(data)
		if err != nil {
			return err
		}

		// get file info so we can get the filename
		fi, err := os.Stat(flags.filepath)
		if err != nil {
			return err
		}

		bubble := jqplayground.New(data, fi.Name(), jqtheme, isJsonLines)
		p := tea.NewProgram(bubble, tea.WithAltScreen())

		if err := p.Start(); err != nil {
			return err
		}
		return nil

	},
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Config file %s was unable to be read: %v\n", viper.ConfigFileUsed(), err)
		}
		return
	}
	// Find home directory.
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	// Search config in home directory
	viper.AddConfigPath(home)

	// register the config file
	viper.SetConfigName(".jqp")

	//only read from yaml files
	viper.SetConfigType("yaml")

	// Try to read the default config file
	if err := viper.ReadInConfig(); err != nil {
		// Check if the error is due to the file not existing
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// For errors other than file not found, print the error message
			fmt.Fprintf(os.Stderr, "Default config file %s was unable to be read: %v\n", viper.ConfigFileUsed(), err)
		}
	}
}

var flags struct {
	filepath, theme string
}

var flagsName = struct {
	file, fileShort, theme, themeShort string
}{
	file:       "file",
	fileShort:  "f",
	theme:      "theme",
	themeShort: "t",
}

var configKeysName = struct {
	themeName      string
	themeOverrides string
	styleOverrides string
}{
	themeName:      "theme.name",
	themeOverrides: "theme.chromaStyleOverrides",
	styleOverrides: "theme.styleOverrides",
}

var cfgFile string

func Execute() {

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "path to config file (default is $HOME/.jqp.yaml)")

	rootCmd.Flags().StringVarP(
		&flags.filepath,
		flagsName.file,
		flagsName.fileShort,
		"", "path to the input JSON file")

	rootCmd.Flags().StringVarP(
		&flags.theme,
		flagsName.theme,
		flagsName.themeShort,
		"", "jqp theme")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
