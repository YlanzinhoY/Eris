package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ylanzinhoy/eris/internal/catalog"
	"github.com/ylanzinhoy/eris/internal/scanner"
	"github.com/ylanzinhoy/eris/internal/ui"
)

var (
	catalogPath string
	extraRoots  []string
)

// Version is replaced at build time for tagged releases.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:           "eris",
	Short:         "Encontre seus jogos e acesse releases disponíveis",
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		games, err := catalog.Load(catalogPath)
		if err != nil {
			return err
		}

		gameScanner := scanner.New(extraRoots)
		return ui.Run(games, gameScanner)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "erro: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// A aplicação possui uma TUI própria e pode ser iniciada pelo Explorer.
	cobra.MousetrapHelpText = ""

	rootCmd.PersistentFlags().StringVarP(
		&catalogPath,
		"catalog",
		"c",
		"games.json",
		"caminho do catálogo JSON",
	)
	rootCmd.PersistentFlags().StringSliceVar(
		&extraRoots,
		"scan-root",
		nil,
		"pasta prioritária ou externa para procurar primeiro (pode repetir)",
	)

	rootCmd.AddCommand(newListCommand())
	rootCmd.AddCommand(newScanCommand())
	rootCmd.AddCommand(newDownloadCommand())
}
