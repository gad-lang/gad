# Gadx

- [x] quando clica para ir para a definicao do IDENT nao faz nada
      <!-- FEITO: `gad def`/`gad complete` agora tratam `.gadx` (langsymParse parseia
           pelo front-end gadx e lowre para gad com posições preservadas). O plugin
           passa --stdin-name para detectar o dialeto. -->
- [x] (gad) go-to-def levava o cursor ao início do arquivo
      <!-- FEITO: o handler navega para o offset exato da declaração via
           FakePsiElement + OpenFileDescriptor (antes findElementAt caía no offset 0). -->
- [x] em goland, nao está colorindo doc comments como markdown
      <!-- FEITO: a grammar gadx agora marca o corpo de `/** … **/` como
           meta.embedded.block.markdown (text.html.markdown). Reinstale/reinicie o GoLand. -->
- [x] formatar `<span>one</span>` deveria gerar `span one` e não `span\n| one`
      <!-- FEITO: uma tag cujo corpo é um único texto curto de uma linha é inlinada
           como `tag texto` (idempotente; edge-whitespace tratado). -->
- [x] formatar `@TOKEN` deve ser precedido por uma linha em braco se nao houver doc commente, caso contrario a linha vazia
      fica antes do doc comment `@param ...\n@comp ...` deve ficar `@param ...\n\n@comp ...`
      <!-- FEITO: diretivas de declaração top-level (@comp/@func/@param/@test/…) são
           separadas por uma linha em branco, inserida antes do doc comment. -->
- [x] formatar `div[a="v"][b="x"]` para `div[a="v", b="x"]` e quebrar linha quando estourar MAX_COLUMN
      <!-- FEITO: grupos de atributos mescláveis (sem condição/spread) fundem em
           `[a, b]`; quando a linha estoura MAX_COLUMN o grupo quebra 1 item/linha. -->
- [x] `~ EXPR` e `+ EXPR` (para componentes), com ou sem espaço depois de `~` e `+`, não está sendo colorido no goland como código gad nativo, como em `~~ ... ~~`
      <!-- FEITO: a grammar gadx colore `~ EXPR` (codeline) e `+ EXPR` (component)
           embutindo source.gad, como `~~ … ~~`. -->
- [x] doc comments de bloco `/** … **/` removiam a quebra de linha final `…\n**/`
      <!-- FEITO: um doc multi-linha mantém `/**` e `**/` em linhas próprias. -->

# Gad
- [x] quando clica para ir para a definicao do IDENT o cursor vai para o inicio do arquivo
      <!-- FEITO: o handler agora navega para o offset exato via FakePsiElement +
           OpenFileDescriptor (o motor `gad def` já retornava o offset correto). -->
