# Gadx

- [ ] quando clica para ir para a definicao do IDENT nao faz nada
      <!-- VERIFICADO: `gad def` não parseia `.gadx` (usa o front-end gad e falha:
           "Parse Error"). O langsym precisa compilar `.gadx` pelo front-end gadx
           (ModuleFile=.gadx) para o go-to-def funcionar. PENDENTE. -->
- [x] em goland, nao está colorindo doc comments como markdown
      <!-- FEITO: a grammar gadx agora marca o corpo de `/** … **/` como
           meta.embedded.block.markdown (text.html.markdown). Reinstale/reinicie o GoLand. -->
- [ ] formatar `<span>one</span>` deveria gerar `span one` e não `span\n| one`
      <!-- VERIFICADO: ainda gera `span` + `\t| one`. PENDENTE (formatter:
           texto único e curto de uma tag deveria ficar inline `tag texto`). -->
- [ ] formatar `@TOKEN` deve ser precedido por uma linha em braco se nao houver doc commente, caso contrario a linha vazia
      fica antes do doc comment `@param ...\n@comp ...` deve ficar `@param ...\n\n@comp ...`
      <!-- VERIFICADO: nenhuma linha em branco é inserida entre diretivas. PENDENTE
           (formatter: inserir blank line antes de cada `@TOKEN`/doc de topo). -->
- [ ] formatar `div[a="v"][b="x"]` para `div[a="v", b="x"]` e quebrar linha quando estourar MAX_COLUMN
      <!-- VERIFICADO: os grupos permanecem `[a="v"][b="x"]` (não mesclados). PENDENTE
           (formatter: fundir grupos de atributos e quebrar por MAX_COLUMN). -->
- [x] `~ EXPR` e `+ EXPR` (para componentes), com ou sem espaço depois de `~` e `+`, não está sendo colorido no goland como código gad nativo, como em `~~ ... ~~`
      <!-- FEITO: a grammar gadx colore `~ EXPR` (codeline) e `+ EXPR` (component)
           embutindo source.gad, como `~~ … ~~`. -->

# Gad
- [ ] quando clica para ir para a definicao do IDENT o cursor vai para o inicio do arquivo
      <!-- VERIFICADO: o motor `gad def` retorna o offset CORRETO (ex.: uso de `y`
           -> decl na linha 2, offset 7). O bug está no PLUGIN IntelliJ (conversão
           byte→char / navegação PSI cai no offset 0). PENDENTE (plugin). -->
