# NEON/MODS — ZZZ Mod Desk v3.6.2

Gerenciador de mods do Zenless Zone Zero para Windows, com suporte à pasta `Mods` do ZZMI / XXMI.

## v3.6.2 — Pacotes com componentes body / face / weapon

O scanner agora reconhece um único mod dividido em pastas de partes do personagem, por exemplo:

```text
Mods/
  Soldier 0-Anby-yyzj-/
    body/
    face/
    weapon/
    说明readme.txt
```

- Se a pasta principal contiver **pelo menos dois componentes reconhecidos** (`body`, `face`, `weapon`, `head`, `hair`, `accessories`) e não contiver subpastas de coleção alheias, o app lê seus `.ini` juntos como **um único mod**, com nome e botão **Abrir pasta** apontando à pasta principal.
- Ligar/desligar renomeia **somente a pasta do pacote inteiro**, nunca as peças internas separadamente. Atalhos, variantes e hashes continuam sendo lidos dos componentes; arquivos em pastas `DISABLED_` continuam inativos.
- Coleções com subpastas de roupas independentes continuam independentes; não agrupamos tudo simplesmente porque está dentro da pasta de um agente.
- Classificações antigas e perfis que faziam referência à subpasta `body`, `face` ou `weapon` continuam recuperáveis quando os metadados são inequívocos. Se houver classificações antigas conflitantes, não é escolhido um traje arbitrário.
- Não move nem apaga nenhuma pasta. A biblioteca passa a mostrar o pacote completo, mesmo que a versão anterior tenha exibido apenas `body`.

## v3.6.1 — Coleções grandes sem cards gigantes

- A aba **Agentes → Trajes oficiais** mostra inicialmente **somente 3 mods por traje**, mantendo os mods ativos no topo e ordenando os demais por nome.
- Quando o traje possui mais de 3 mods, o botão **Ver todos os X mods** expande uma lista com altura máxima de **300 px** (260 px em telas estreitas), rolável independentemente da página. **Recolher lista** devolve o layout compacto.
- O modo expandido inclui **busca por nome** (sem distinguir letras maiúsculas/minúsculas ou acentos) e contagem instantânea de resultados.
- A expansão e a busca são independentes para cada combinação **agente + traje**. A pesquisa não altera arquivos, ativações, presets ou configurações de mods.
- O mesmo comportamento vale para trajes padrão e oficiais alternativos. Trajes com 0 a 3 mods continuam simples, sem botão de expansão.

## v3.6.0 — Identificação automática por hashes de modelos

- Os arquivos `.ini` de cada mod são analisados **somente para leitura**, inclusive quando a pasta principal do mod estiver desativada. Subpastas internas `DISABLED_` são ignoradas na identificação.
- O programa consulta a base pública comunitária de hashes do [ZZZ Mod Hub](https://github.com/the-sea-man/ZZZ-Mod-Hub/blob/main/src/characters.json), que reúne `draw_vb`, `position_vb`, `ib` e hashes dos demais buffers de cada personagem e traje. Uma cópia é salva localmente em `model_hash_database.json`; o aplicativo continua funcionando offline com o último cache.
- A base é sincronizada ao abrir o app e pode ser atualizada em **Configurações → Atualizar hashes e reclassificar**. Roupas oficiais novas conhecidas pela base também entram automaticamente no catálogo, sem esperar uma release do gerenciador.
- Um hash que só aparece em um traje de um agente fornece evidência para classificá-lo; duas ou mais correspondências exclusivas aumentam a confiança. **Hashes compartilhados** não diferenciam as roupas. Se o mod possuir evidências exclusivas de trajes diferentes, aparecerá como **Ambíguo**; se não houver evidência, aparecerá como **Não identificado**. O gerenciador não assume a roupa padrão por causa do nome de arquivo.
- A biblioteca identifica a fonte da classificação (**Hash / Manual / Não identificado**) e seu grau de confiança. **A classificação manual tem prioridade sobre a automática** e pode ser alterada na Biblioteca. Mods desconhecidos não são incluídos silenciosamente em um traje ou no sorteio.
- **Limitações:** uma base comunitária pode ainda não conter modelos recém-lançados e os hashes podem mudar entre patches. Mods de interface, texturas isoladas ou pacotes multiagente podem permanecer sem identificação. Isso não significa que o mod esteja incorreto.

## v3.5.0 — Trajes oficiais separados dos mods

A aba **Agentes** não confunde mais o visual original com um mod. Cada agente aparece com seu **traje padrão oficial** e suas **roupas alternativas oficiais conhecidas**, quando disponíveis. Exemplo: Astra tem **Scarlet Rock** (padrão) e **Chandelier** (oficial alternativo). A miniatura da Chandelier fornecida para o projeto está embutida no programa; para outras roupas, usa-se imagem pública quando disponível, com fallback do retrato do agente se necessário.

- Em **Biblioteca → Traje**, classifique o mod como compatível com o traje padrão ou com uma roupa oficial alternativa. O programa tenta reconhecer nomes explícitos, mas não deduz com segurança quais hashes de modelo o mod substitui; confirme a classificação quando necessário.
- Cada roupa oficial na galeria mostra apenas os mods ligados a ela. Escolher um mod desativa os demais mods do mesmo agente, mantendo **no máximo um mod ativo por personagem**. A opção **Sem mod** deixa o traje selecionado no catálogo, sem substituir os arquivos do jogo.
- Marque **Possuo no jogo** nas roupas alternativas que realmente estão desbloqueadas na sua conta. O gerenciador não acessa a conta da HoYoverse para descobrir isso.
- O botão **Sortear trajes** sorteia entre as roupas possuídas (e o padrão) que tenham mods instalados e sorteia **exatamente um mod** compatível por agente com mod ativo. Agentes sem mod ativo não são alterados.
- **Limitação técnica importante:** selecionar uma roupa oficial no ZZZ Manager não equipa nem compra essa roupa no ZZZ. O usuário deve equipá-la no menu Trajes do jogo. O gerenciador troca apenas os mods da pasta ZZMI; depois pressione **F10** no jogo para recarregá-los.
- O catálogo já inclui os trajes das atualizações **3.1** (Remielle, Sigrid e Lucy), **3.2** (Nangong, Aria e Sunna) e **3.3** (Phoenix e Severian), além de roupas oficiais anteriores verificadas.
- Para um traje cujo retrato de catálogo não carregue, clique em **🖼 Arte** e selecione uma imagem local PNG, JPG ou WEBP. Essa imagem será guardada no diretório de configuração do app e terá prioridade sobre qualquer CDN; **Restaurar** devolve a arte anterior. A miniatura da Chandelier enviada neste projeto já vem incluída.
- Os mods e presets atuais são preservados. Os vínculos de roupas oficiais, marcações de propriedade e a roupa escolhida ficam em `config.json`.

## v3.4.1 — Galeria de skins, sorteio e variações persistentes

- **Galeria por agente**: mostra a aparência original e os mods instalados, com as prévias encontradas nas pastas. Mods sem prévia própria são identificados como tal e usam o retrato do agente como substituto.
- **Uma skin por agente**: na Biblioteca e na galeria, escolher uma opção ativa essa skin e desativa as demais do mesmo personagem. Original desativa todos os mods do personagem. Nenhum outro agente é modificado.
- **🎲 Sortear aparências**: apenas agentes que tinham uma skin ativa participam. Cada um recebe outra skin instalada ou a aparência original; personagens sem skin ativa não são alterados. Operações são revertidas se uma troca falhar; perfis antigos que tentem ativar duas skins para o mesmo agente são bloqueados.
- **F10 para aplicar em jogo**: o app renomeia as pastas, mas o 3DMigoto requer recarregamento para refletir as escolhas.
- **Salvamento opcional de variações**: quando um mod declara `global persist`, o app exibe **Salvar variação atual (após F10)**. Depois de alterar óculos, acessórios, roupas etc. no jogo, pressione F10, retorne ao app e clique no botão. Ele lê `d3dx_user.ini` ao lado da pasta `Mods` e sincroniza APENAS variáveis numéricas de `global persist` com correspondência exata de caminho e nome de variável. O `d3dx_user.ini` é somente leitura e uma cópia `.zzzmanager.bak` é criada antes de modificar cada `.ini`.
- **Limitação importante**: mods que não declaram `global persist`, utilizam namespaces não padronizados, formatos diferentes, ou variáveis que o 3DMigoto não gravou não permitem garantia automática de preservação. O aplicativo informa quando nenhuma variável compatível é encontrada, sem modificar o arquivo.

## Importação RAR reforçada (v3.3.4)

O erro `rardecode: decoded file too short` pode indicar um arquivo RAR truncado, incompatibilidade de descompactação ou corrupção. O gerenciador agora:

- Usa uma versão mais recente de `rardecode` para abrir arquivos RAR.
- Se o leitor interno falhar por erro de decodificação, tenta usar o **tar.exe nativo do Windows** (libarchive), caso o Windows ofereça suporte ao método RAR usado. Não é necessário instalar WinRAR ou 7-Zip para essa segunda tentativa.
- Faz validação dos nomes, impede links/atalhos perigosos e extrai cada arquivo do fallback por streaming com limite total. Não permite que o tar escreva arquivos diretamente em `Mods` ou fora dela.
- Remove a importação parcial se os dois leitores falharem. Exibe uma mensagem com a alternativa **extrair no 7-Zip ou WinRAR e usar Instalar pasta**.
- RAR corrompido, criptografado ou dividido em vários volumes continua podendo exigir o arquivo original, uma senha ou todos os volumes. Não é possível consertar um arquivo incompleto apenas mudando o descompactador.

## Atalhos em subpastas desativadas (v3.3.3)

Pacotes com várias camadas, como `Mods/[HC] Roxy.../[HC] Roxy.../DISABLED_[HC] Roxy.../*.ini`, podem incluir atalhos mesmo nas variantes desativadas. Agora o leitor varre os `.ini` de **todo o pacote**:

- Mostra os atalhos das variantes ativas e também os encontrados em subpastas `DISABLED_`.
- Cada atalho de uma pasta desativada aparece com aviso **Pasta desativada — atalho apenas para referência**. Isso **não ativa** a variante desativada.
- As teclas são ordenadas, priorizando as fontes ativas; a detecção continua em modo somente leitura.
- Pastas `DISABLED_` continuam excluídas da detecção de conflitos de hashes para não gerar falsos avisos.

## Retratos corrigidos (v3.3.1)

Os retratos da **Belle, Wise, Remielle, Sigrid, Claret, Roxy, Phoenix e Severian** agora são incluídos diretamente no executável Windows. Eles aparecem sem conexão com a internet; os demais retratos continuam vindo da CDN oficial, quando disponível. As fotos personalizadas que você já definiu continuam tendo prioridade.

Os arquivos de arte são baixados e normalizados na etapa de compilação `scripts/download_portraits.py`; o GitHub Actions usa Pillow para converter formatos retornados pelas fontes. Os seis agentes mais recentes usam imagens da [HoYoverse](https://zenless.hoyoverse.com/en-us/main?catchSpider=1); Belle e Wise usam os retratos do jogo catalogados pela [Zenless Zone Zero Wiki](https://zenless-zone-zero.fandom.com/wiki/Category:Agent_Portraits). As artes pertencem a HoYoverse / miHoYo.

Para compilar localmente, execute `python -m pip install pillow` e `python scripts/download_portraits.py` antes de `go test ./...` e `go build .`.

## Correção da leitura de pastas (v3.3.2)

O gerenciador reconhece pacotes de mods com pastas internas como `RoxyOveralls/RoxyOveralls/toggles` e `DISABLED_resources`. Agora eles aparecem como **um único mod** chamado `RoxyOveralls`, em vez de entradas separadas para `resources` e `toggles`.

- **Ativar/desativar** atua na pasta principal, sem renomear ou alterar as subpastas de recursos.
- Os atalhos dos `.ini` ativos continuam visíveis no card do mod; os `.ini` de `DISABLED_*` não geram teclas ou conflitos falsos.
- Pastas de coleções com mods independentes (como `Rina/Outfit A` e `Rina/Outfit B`) permanecem separadas.
- Perfis antigos salvos com caminhos de subpastas são interpretados como referência ao novo mod agrupado quando possível.

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
