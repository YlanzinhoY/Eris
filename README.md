# Éris

Éris é uma aplicação de terminal para consultar releases, localizar instalações
de jogos e baixar arquivos diretamente para a pasta encontrada. A interface
interativa funciona no próprio terminal e não exige Go instalado.

## Instalação no Windows

1. Abra a página **Releases** do repositório.
2. Baixe o arquivo `eris-vX.Y.Z-windows-amd64.zip` da versão mais recente.
3. Extraia o ZIP para uma pasta permanente, por exemplo `C:\Tools\Eris`.
4. Mantenha `eris.exe` e `games.json` juntos nessa pasta.
5. Adicione `C:\Tools\Eris` ao `PATH` do seu usuário.
6. Feche e abra o terminal novamente.

Para adicionar a pasta ao `PATH` pela interface do Windows:

1. Pesquise por **Editar as variáveis de ambiente da sua conta**.
2. Selecione `Path` em **Variáveis de usuário** e clique em **Editar**.
3. Clique em **Novo**, informe `C:\Tools\Eris` e confirme as janelas.

Valide a instalação em um terminal novo:

```powershell
eris --version
eris
```

> O usuário final não precisa instalar Go. O pacote da release já contém o
> executável e o catálogo necessários.

## Atualização

Baixe o ZIP da nova release e substitua `eris.exe` e `games.json` na pasta de
instalação. Consulte o `CHANGELOG.md` ou as notas da release antes de atualizar.

## Interface interativa

Execute `eris` sem argumentos para abrir a interface:

```powershell
eris
```

Controles:

- `↑`/`↓` ou `j`/`k`: navegar entre os hypervisors disponíveis.
- `s`: procurar a instalação selecionada.
- `Enter`: procurar, confirmar e baixar para a pasta encontrada.
- `y`: confirmar uma ação.
- `n` ou `Esc`: cancelar uma confirmação.
- `q` ou `Ctrl+C`: sair; durante um download, solicita o cancelamento.

## Comandos

```powershell
eris list
eris scan "crimson desert"
eris download "crimson desert"
eris download --yes "crimson desert"
```

Use outro catálogo com `--catalog`:

```powershell
eris --catalog C:\caminho\games.json list
```

Priorize uma pasta conhecida ou inclua uma unidade de rede com `--scan-root`:

```powershell
eris --scan-root D:\Games scan "crimson desert"
eris --scan-root \\servidor\jogos scan "crimson desert"
```

## Como o scanner funciona

O scanner procura primeiro nas bibliotecas da Steam, incluindo
`steamapps/common` e `steamapps/downloading` de todas as bibliotecas registradas
em `libraryfolders.vdf`.

Quando o catálogo informa `exe`, a busca usa o nome exato do executável em
qualquer profundidade. Se a instalação não estiver na Steam, o scanner usa como
fallback os discos locais fixos e removíveis. Pastas passadas por `--scan-root`
também entram nessa busca.

Diretórios do sistema, caches de desenvolvimento, links simbólicos e locais sem
permissão são ignorados. Quando uma instalação é encontrada, as outras
varreduras são canceladas.

## Download e segurança

O download é transmitido para um arquivo temporário `.part` dentro da pasta do
jogo. O nome definitivo só é aplicado depois que todo o conteúdo foi recebido e
sincronizado.

- Nomes recebidos em `Content-Disposition` são sanitizados.
- Arquivos existentes não são sobrescritos.
- Respostas HTML são recusadas.
- Downloads cancelados removem o arquivo temporário quando o encerramento é
  concluído normalmente.
- Éris baixa o arquivo, mas não executa o conteúdo.

Baixe o Éris somente pela página de Releases do repositório e confira o arquivo
`.sha256` publicado junto ao ZIP.

## Formato do catálogo

Cada item de `games.json` aceita:

```json
{
  "game": "crimson desert",
  "download_link": "https://exemplo.com/release",
  "version": "1.17.00",
  "exe": "CrimsonDesert.exe"
}
```

`game`, `download_link` e `version` são obrigatórios. `exe` é recomendado para
tornar a localização da instalação mais precisa.

## Desenvolvimento

Esta seção é destinada a colaboradores. Para trabalhar no código-fonte, instale
a versão de Go indicada em `go.mod`.

```powershell
go test ./...
go vet ./...
go run .
```

Para gerar o executável local:

```powershell
go build -o eris.exe .
.\eris.exe --version
```

O projeto usa Cobra para os comandos, Bubble Tea para a interface interativa e
Lip Gloss para estilos e layout responsivo.

## Publicação de versões

As releases do GitHub são montadas localmente pelo Gitea Actions quando uma tag
SemVer `vX.Y.Z` é enviada ao Gitea. Antes de criar a tag, adicione uma seção
correspondente no `CHANGELOG.md`.

```powershell
git push origin main
git push gitea main
git tag -a v0.0.1 -m "Release v0.0.1"
git push origin v0.0.1
git push gitea v0.0.1
```

O pipeline valida o código, compila `eris.exe`, inclui `games.json`, gera o ZIP
e o checksum SHA-256, usa a seção da versão no changelog como notas e publica os
artefatos na Release correspondente do GitHub.

O repositório no GitHub é a fonte principal. O Gitea recebe a mesma branch e tag
para iniciar um Gitea Runner no Windows, registrado com o rótulo `windows:host`.
O host precisa ter Git, Node.js, Go, Windows PowerShell e GitHub CLI disponíveis.
O token usado na publicação deve ser armazenado no secret `GH_RELEASE_TOKEN` do
repositório no Gitea.
