package cmd

import (
	"fmt"

	"github.com/enzom/hv-game-cli/internal/catalog"
	"github.com/enzom/hv-game-cli/internal/downloader"
	"github.com/enzom/hv-game-cli/internal/launcher"
	"github.com/enzom/hv-game-cli/internal/scanner"
	"github.com/spf13/cobra"
)

func newDownloadCommand() *cobra.Command {
	var assumeYes bool

	command := &cobra.Command{
		Use:   "download <jogo>",
		Short: "Localiza o jogo e baixa o arquivo diretamente para sua pasta",
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
				fmt.Fprintln(cmd.OutOrStdout(), "Instalação não encontrada na Steam nem nos discos locais.")
				if assumeYes {
					return launcher.Open(game.DownloadLink)
				}
				return askToOpenDownload(cmd, game)
			}

			destination := results[0].Path
			fmt.Fprintf(cmd.OutOrStdout(), "Instalação encontrada: %s [%s]\n", destination, results[0].Source)
			if !assumeYes {
				confirmed, confirmErr := confirmAction(
					cmd,
					fmt.Sprintf("Baixar o arquivo diretamente para %s?", destination),
				)
				if confirmErr != nil {
					return confirmErr
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Cancelado.")
					return nil
				}
			}

			progressPrinted := false
			result, err := downloader.Download(cmd.Context(), game.DownloadLink, destination, func(progress downloader.Progress) {
				progressPrinted = true
				fmt.Fprintf(cmd.OutOrStdout(), "\r%-78s", commandProgressLabel(progress))
			})
			if progressPrinted {
				fmt.Fprintln(cmd.OutOrStdout())
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"Download concluído: %s (%s)\n",
				result.Path,
				commandFormatBytes(result.Bytes),
			)
			return nil
		},
	}

	command.Flags().BoolVarP(&assumeYes, "yes", "y", false, "não pedir confirmação")
	return command
}

func commandProgressLabel(progress downloader.Progress) string {
	if progress.Total <= 0 {
		return fmt.Sprintf("Baixando... %s recebidos", commandFormatBytes(progress.Downloaded))
	}
	percentage := float64(progress.Downloaded) / float64(progress.Total) * 100
	return fmt.Sprintf(
		"Baixando... %6.2f%%  %s / %s",
		percentage,
		commandFormatBytes(progress.Downloaded),
		commandFormatBytes(progress.Total),
	)
}

func commandFormatBytes(value int64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	divisor := int64(unit)
	exponent := 0
	for quotient := value / unit; quotient >= unit && exponent < 4; quotient /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(value)/float64(divisor), "KMGTPE"[exponent])
}
