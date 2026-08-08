# Changelog

As mudanças relevantes do Éris são registradas neste arquivo. Cada release usa
a seção correspondente à sua tag como descrição publicada no Gitea.

## [v0.0.1] - 2026-08-08

### Adicionado

- Interface interativa no terminal para navegar pelo catálogo de hypervisors.
- Comandos `list`, `scan` e `download` para uso não interativo.
- Descoberta das bibliotecas Steam pelo Registro do Windows e pelo arquivo
  `libraryfolders.vdf`.
- Busca por executável exato na Steam, em discos locais e em caminhos informados
  com `--scan-root`.
- Download em streaming para a pasta encontrada, com progresso, cancelamento e
  arquivo temporário `.part`.
- Abertura opcional do link no navegador quando a instalação não é localizada.
- Executável `eris.exe`, comando global `eris` e informação de versão com
  `eris --version`.
- Pipeline do Gitea Actions para testes, build, empacotamento, checksum e
  publicação automática de releases versionadas.

### Alterado

- Lista principal renomeada para **HYPERVISORS DISPONÍVEIS**.
- Painel da lista ampliado e responsivo para acomodar nomes longos.
- Nomes extensos são truncados com reticências sem quebrar caracteres Unicode.
- Versões aparecem no painel de detalhes, mantendo a lista principal mais limpa.
- Documentação reorganizada para usuários que instalam o binário sem Go.

### Segurança

- Validação de URLs HTTP/HTTPS e rejeição de links com credenciais.
- Sanitização do nome do arquivo recebido pelo servidor.
- Prevenção de sobrescrita de arquivos existentes e de saída da pasta de destino.
- Rejeição de respostas HTML e limite de redirecionamentos HTTP.
- O conteúdo baixado nunca é executado automaticamente.
