# NEON/MODS — ZZZ Mod Desk v3.3.1

Gerenciador de mods do Zenless Zone Zero para Windows, com suporte à pasta `Mods` do ZZMI / XXMI.

## Retratos corrigidos (v3.3.1)

Os retratos da **Belle, Wise, Remielle, Sigrid, Claret, Roxy, Phoenix e Severian** agora são incluídos diretamente no executável Windows. Eles aparecem sem conexão com a internet; os demais retratos continuam vindo da CDN oficial, quando disponível. As fotos personalizadas que você já definiu continuam tendo prioridade.

Os arquivos de arte são baixados e normalizados na etapa de compilação `scripts/download_portraits.py`; o GitHub Actions usa Pillow para converter formatos retornados pelas fontes. Os seis agentes mais recentes usam imagens da [HoYoverse](https://zenless.hoyoverse.com/en-us/main?catchSpider=1); Belle e Wise usam os retratos do jogo catalogados pela [Zenless Zone Zero Wiki](https://zenless-zone-zero.fandom.com/wiki/Category:Agent_Portraits). As artes pertencem a HoYoverse / miHoYo.

Para compilar localmente, execute `python -m pip install pillow` e `python scripts/download_portraits.py` antes de `go test ./...` e `go build .`.

## Instalação

1. Feche a versão antiga do aplicativo (inclusive qualquer processo do gerenciador que ainda esteja aberto).
2. Execute `ZZZ-Mod-Desk.exe` no Windows 10/11 64-bit. Ele abre uma janela do Microsoft Edge no modo aplicativo; se não encontrar o Edge, abre o navegador padrão.
3. Em **Configurações**, selecione a pasta chamada `Mods` do ZZMI (não a pasta do jogo).
4. **Biblioteca** mostra seus mods. Use **Importar arquivo** para ZIP, RAR ou 7z, ou **Instalar pasta** para escolher um diretório com arquivos de mod. Também é possível arrastar um arquivo ou pasta sobre o aplicativo. Para atribuir um mod ao agente certo, use o seletor de agente no card do mod.
5. **Agentes** lista 64 personagens / perfis, permite marcar favoritos, filtrar por facção e salvar ou aplicar perfis **individuais**.

O executável já inclui a interface, o código do catálogo e a lógica de gerenciamento: **não é necessário instalar Python, Node, Go nem abrir terminal**.

## Novidades da v3.3 — Seletor de agentes e teclas dentro do jogo

- A escolha de agente nos cards da Biblioteca usa agora uma janela **escura**, com campo de busca, rolagem interna e favoritos primeiro. O nome selecionado fica visível no próprio card.
- **Comandos no jogo** são extraídos automaticamente dos `.ini` presentes dentro da pasta de cada mod, procurando seções `[Key...]` (incluindo arquivos UTF-8 e UTF-16 com BOM).
- Cada atalho mostra suas teclas (`key =`), tecla de retorno (`back =`, quando existir), nome da seção, tipo (`cycle`, `toggle`, `hold`, `activate`) e condições quando fornecidas.
- Os dois primeiros atalhos ficam visíveis no card. Para mods com mais atalhos, use **Mostrar todos**.
- Quando nenhum `[Key...]` válido é encontrado, o app informa isso em vez de inventar controles. Alguns mods usam substituição automática sem teclas.
- **Somente leitura**: detectar e mostrar os atalhos não altera nenhum `.ini`. Os atalhos só funcionam dentro do jogo quando as condições do mod são atendidas.

## Instalação de mods (v3.2)

- **ZIP, RAR e 7z:** clique em **Importar arquivo** e selecione o arquivo, ou arraste-o para a janela. O gerenciador tem leitores integrados para os três formatos; não precisa instalar WinRAR ou 7-Zip. Pacotes criptografados ou divididos em volumes podem não ser compatíveis.
- **Pasta extraída:** clique em **Instalar pasta** e selecione o diretório do mod pelo seletor de pastas do navegador (Edge/Chrome). Também pode arrastar a pasta inteira para dentro da janela. A estrutura interna dos arquivos é mantida.
- O programa adiciona uma pasta nova em `Mods`, usando sufixos `(2)`, `(3)` etc. caso o nome já exista. **Não sobrescreve nem apaga mods existentes.**
- O pacote precisa conter ao menos um `.ini`. Arquivos executáveis/atalhos, links simbólicos e caminhos que escapam do diretório são bloqueados. O app limita pacotes compactados a 1 GB e o conteúdo extraído a 2 GB ou 8000 entradas.
- A importação por pasta depende do suporte do navegador ao seletor de diretórios / `webkitGetAsEntry` para arrastar. A versão Windows abre no Edge em modo aplicativo quando disponível.

## Atualizações pelo próprio aplicativo (v3.1)

Em **Configurações → Atualizações do aplicativo** (ou pelo botão **Verificar versão** no topo):

1. Clique em **Verificar atualizações**. O aplicativo consulta a versão estável mais recente em https://github.com/Gabriel-Liz2003/ZZZ-MANAGER/releases/latest.
2. Se uma versão superior estiver disponível, clique em **Baixar e instalar**. A atualização é descarregada para uma pasta temporária e validada com o SHA-256 publicado na mesma release.
3. O gerenciador fecha, um script separado aguarda o executável terminar e substitui o arquivo original, criando uma cópia `.backup`. Depois, o aplicativo reinicia automaticamente.

Os mods na pasta do ZZMI e os dados em `%APPDATA%\ZZZ Mod Desk` não são apagados durante a atualização. O executável deve estar num diretório onde o Windows permite escrita. O atualizador funciona apenas no Windows e exige internet. Se a release ainda não foi publicada, o app informa que não há versão disponível.

## Publicar novas versões automaticamente

O arquivo `.github/workflows/release.yml` usa GitHub Actions para compilar o app para Windows, executar testes e publicar um **GitHub Release** com:

- `ZZZ-Mod-Desk.exe`: binário Windows 64 bits;
- `SHA256SUMS.txt`: hash do binário para conferência.

O workflow é executado a cada push para `main` e também pode ser disparado manualmente. Ele só cria uma release se o número em `appVersion` de `updater.go` ainda não foi publicado. Para publicar a próxima atualização, incremente `appVersion` (ex.: 3.1.1) e faça um push no `main`; GitHub Actions cria a nova release. É necessário que o repositório permita GitHub Actions com permissão de escrita em Contents. A compilação na nuvem não apaga dados locais de usuários.

## Funcionalidades da versão 3.1

- Corrigido o erro `Cannot read properties of null (reading 'length')`, provocado por conflitos vazios vindos do servidor como JSON `null`. Agora o servidor envia `[]` e a interface valida os tipos.
- Novo visual, navegação com painel inicial e cards, busca, seletor de agente e contagem de mods/conflitos.
- Catálogo visual de **64 personagens** e cores por facção. Retratos da HoYoverse são carregados pela Internet quando disponíveis. Para agentes sem URL confirmada, ou se a rede estiver offline, o app mostra um avatar visual genérico até você fornecer uma imagem.
- Foto personalizada: no card, clique em **Foto** e escolha PNG/JPG/WEBP de até 8 MB; fica guardada localmente.
- Favoritos persistentes: cards de agentes favoritos aparecem antes dos demais.
- Perfis por agente: salvam só os mods ativos daquele agente e, quando aplicados, alteram somente seus mods. Os mods dos outros agentes continuam intactos.
- Perfis personalizados continuam globais: quando aplicados, alteram a seleção de toda a biblioteca.
- Importação ZIP, RAR, 7z e pastas com verificações contra caminhos inseguros / arquivos executáveis; detecção de hashes INI repetidos; ativação/desativação sem deletar mods.

## Retratos e Internet

O aplicativo referencia imagens de personagens hospedadas no CDN da HoYoverse e faz uso das URLs documentadas por:
- https://github.com/Syqhu/zzz-api/blob/main/src/officialAssets.ts
- https://zenless.hoyoverse.com/en-us/character

As imagens oficiais **não são incluídas no executável**. O app depende de uma conexão para carregar as URLs externas e algumas imagens de agentes recentes ainda não têm URL mapeada. Você pode colocar uma imagem local em qualquer perfil para funcionar totalmente offline.

Imagens e nomes dos agentes pertencem à HoYoverse / miHoYo; o projeto é independente e não tem associação com a desenvolvedora.

## Onde ficam os dados

- Windows: `%APPDATA%\ZZZ Mod Desk\config.json` (alguns sistemas usam o diretório retornado pela API de configuração do Windows)
- Imagens personalizadas: ao lado do arquivo de configuração, na pasta `photos`.

A v3 reaproveita as configurações/perfis da v2. Para segurança, faça backup da sua pasta `Mods` antes de usar um novo gerenciador de mods.

**Segurança e limitações:** a detecção de conflito por hash é apenas uma indicação; nem todo conflito real é reconhecido. Mods não são aprovados pela HoYoverse; usar mods pode violar termos do jogo. O app não remove o risco associado.

## Compilar a partir do código-fonte

Requer Go 1.23+ e conexão para instalar as dependências Go na primeira compilação:

```sh
go mod tidy
```

```sh
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o ZZZ-Mod-Desk.exe .
```

## Testes

```sh
go test -count=1 ./...
```
