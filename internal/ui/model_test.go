package ui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/ylanzinhoy/eris/internal/catalog"
)

func TestFormatGameListRowTruncatesLongName(t *testing.T) {
	game := catalog.Game{
		Name:    "monster hunter stories 3: twisted reflection",
		Version: "1.1.00",
	}

	row := formatGameListRow(game, true, 43)

	if got := lipgloss.Width(row); got != 43 {
		t.Fatalf("largura da linha = %d, esperava 43: %q", got, row)
	}
	if strings.Contains(row, game.Name) {
		t.Fatalf("nome longo não foi truncado: %q", row)
	}
	if !strings.Contains(row, "…") {
		t.Fatalf("linha truncada não contém reticências: %q", row)
	}
	if strings.Contains(row, game.Version) {
		t.Fatalf("versão foi exibida na lista: %q", row)
	}
}

func TestFormatGameListRowOmitsVersion(t *testing.T) {
	game := catalog.Game{Name: "crimson desert", Version: "1.17.00"}

	row := formatGameListRow(game, false, 43)

	want := "  crimson desert"
	if row != want {
		t.Fatalf("linha = %q, esperava %q", row, want)
	}
}

func TestFormatGameListRowPreservesUTF8(t *testing.T) {
	game := catalog.Game{
		Name:    "ação épica com dragões 🎮 e exploração",
		Version: "2.0.0",
	}

	row := formatGameListRow(game, false, 24)

	if !utf8.ValidString(row) {
		t.Fatalf("truncamento produziu UTF-8 inválido: %q", row)
	}
	if got := lipgloss.Width(row); got != 24 {
		t.Fatalf("largura da linha = %d, esperava 24: %q", got, row)
	}
}

func TestRenderListLongNameDoesNotExpandPanel(t *testing.T) {
	shortList := model{
		games: []catalog.Game{{Name: "jogo curto", Version: "1.0.0"}},
		width: 80,
	}.renderList()
	longList := model{
		games: []catalog.Game{{
			Name:    "monster hunter stories 3: twisted reflection",
			Version: "1.1.00",
		}},
		width: 80,
	}.renderList()

	if got, want := lipgloss.Width(longList), lipgloss.Width(shortList); got != want {
		t.Fatalf("nome longo expandiu o painel: largura = %d, esperava %d", got, want)
	}
}

func TestWideGameListFitsLongTitle(t *testing.T) {
	game := catalog.Game{Name: "monster hunter stories 3: twisted reflection"}

	row := formatGameListRow(game, true, gameListPanelWidth)

	if !strings.Contains(row, game.Name) {
		t.Fatalf("painel largo truncou o título: %q", row)
	}
	if strings.Contains(row, "…") {
		t.Fatalf("painel largo exibiu reticências desnecessárias: %q", row)
	}
}

func TestRenderListAddsSpaceBelowTitle(t *testing.T) {
	game := catalog.Game{Name: "crimson desert"}
	rendered := ansi.Strip(model{
		games: []catalog.Game{game},
		width: wideLayoutMinWidth,
	}.renderList())
	lines := strings.Split(rendered, "\n")

	for index, line := range lines {
		if !strings.Contains(line, "HYPERVISORS DISPONÍVEIS") {
			continue
		}
		if index+2 >= len(lines) {
			t.Fatalf("não há espaço suficiente após o título: %q", rendered)
		}
		if strings.Contains(lines[index+1], game.Name) {
			t.Fatalf("jogo ficou colado ao título: %q", rendered)
		}
		for _, followingLine := range lines[index+2:] {
			if strings.Contains(followingLine, game.Name) {
				return
			}
		}
		t.Fatalf("jogo não apareceu depois da margem: %q", rendered)
	}

	t.Fatalf("título não encontrado: %q", rendered)
}
