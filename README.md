# Éris

CLI pública para consultar o catálogo gerado pelo scraper, localizar instalações
de jogos e baixar releases diretamente para a pasta encontrada.

## Tecnologias

- Cobra para comandos e flags.
- Bubble Tea para a interface interativa.
- Lip Gloss para estilos e layout responsivo.

## Executar

```powershell
go run .
```

Controles da interface:

- `↑`/`↓` ou `j`/`k`: navegar.
- `s`: procurar a instalação do jogo.
- `Enter`: procurar, confirmar e baixar o arquivo para a pasta do jogo.
- `q`: sair.

## Comandos

```powershell
go run . list
go run . scan "crimson desert"
go run . download "crimson desert"
```

Use `--catalog` para outro JSON e `--scan-root` para priorizar um caminho ou
incluir uma unidade de rede:

```powershell
go run . --catalog .\games.json --scan-root D:\Games scan "crimson desert"
```

## Scanner

O scanner procura primeiro nas bibliotecas da Steam, incluindo as pastas
`steamapps/common` e `steamapps/downloading` de todas as bibliotecas registradas
em `libraryfolders.vdf`.

Quando o catálogo contém `exe`, a busca procura esse arquivo pelo nome exato em
qualquer profundidade. Se ele não estiver relacionado à Steam, o scanner usa
como fallback todos os discos locais fixos e removíveis. Se ainda assim não
encontrar, a CLI oferece abrir o `download_link` no navegador.

Quando a instalação é encontrada, o download é transmitido em streaming para um
arquivo temporário dentro da própria pasta do jogo. A TUI mostra bytes recebidos,
tamanho total e porcentagem. Só depois de concluir e sincronizar os dados o
arquivo recebe o nome definitivo; downloads interrompidos removem o `.part`.

Diretórios do sistema, caches de desenvolvimento, links simbólicos e locais sem
permissão são ignorados. Use `--scan-root` para priorizar um caminho conhecido
ou incluir caminhos externos, como compartilhamentos de rede. Assim que uma
instalação é localizada, as demais varreduras são encerradas.

Cada item do catálogo aceita `exe` para localizar a instalação pelo nome exato
do executável.

```json
{
  "game": "crimson desert",
  "download_link": "https://exemplo.com/release",
  "version": "1.17.00",
  "exe": "CrimsonDesert.exe"
}
```

## Segurança

O nome recebido em `Content-Disposition` é sanitizado, arquivos existentes não
são sobrescritos e respostas HTML são recusadas. A CLI não executa o conteúdo
baixado. O navegador só é oferecido quando nenhuma instalação é encontrada.
