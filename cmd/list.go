package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ylanzinhoy/eris/internal/catalog"
)

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lista os jogos disponíveis no catálogo",
		RunE: func(cmd *cobra.Command, _ []string) error {
			games, err := catalog.Load(catalogPath)
			if err != nil {
				return err
			}

			for _, game := range games {
				fmt.Fprintf(cmd.OutOrStdout(), "%-32s  v%s\n", game.Name, game.Version)
			}
			return nil
		},
	}
}
