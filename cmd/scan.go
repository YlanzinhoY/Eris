package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ylanzinhoy/eris/internal/catalog"
	"github.com/ylanzinhoy/eris/internal/launcher"
	"github.com/ylanzinhoy/eris/internal/scanner"
)

func newScanCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scan <jogo>",
		Short: "Procura a instalação de um jogo no computador",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			games, err := catalog.Load(catalogPath)
			if err != nil {
				return err
			}

			game, err := catalog.Find(games, args[0])
			if err != nil {
				return err
			}

			results, err := scanner.New(extraRoots).Scan(cmd.Context(), game.Name, game.Executable)
			if err != nil {
				return err
			}
			if len(results) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "%s não foi encontrado na Steam nem nos discos locais.\n", game.Name)
				return askToOpenDownload(cmd, game)
			}

			for _, result := range results {
				fmt.Fprintf(cmd.OutOrStdout(), "%s  [%s]\n", result.Path, result.Source)
			}
			return nil
		},
	}
}

func askToOpenDownload(cmd *cobra.Command, game catalog.Game) error {
	confirmed, err := confirmAction(cmd, fmt.Sprintf("Abrir o link de download de %s no navegador?", game.Name))
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(cmd.OutOrStdout(), "Cancelado.")
		return nil
	}
	return launcher.Open(game.DownloadLink)
}
