# TASK: Gadx — `gad transpile page.html` em sintaxe de tags
> Created: 2026-09-04 | Updated: 2026-09-04

## Goal
`gad transpile arquivo.html` gera `arquivo.gadx` na sintaxe de tags do gadx
(`div.card`, corpo indentado) em vez de HTML inline, com o espaço entre
elementos inline legível e sem perder nada do que a página renderiza.

## Plan
- [x] Levantar o HTML e formatar pelo mesmo passo do `gad fmt`
- [x] Descartar o espaço de indentação entre tags não-inline
- [x] Linha de interpolação única sem o prefixo `| `; corpo de uma linha sobe para a tag
- [x] `*` como marcador de um único espaço
- [x] Orçamento de 200 colunas no transpile de HTML (o `gad fmt` mantém o dele)
- [x] `#id` / `.class` na forma curta, nomes especiais entre aspas
- [x] `script` / `style` em sintaxe de tag, corpo indentado lido verbatim
- [x] Samples (`text_space`, `shorthand`) + doc + `make verify`
- [x] `<!-- … -->` renderiza como comentário HTML e permanece no fmt e no transpile
- [x] `tag(3)` indica repetição `3x`: o formatter escreve `a link\na link` como
      `a(2) link`, e `(EXPR)` — sempre no fim da tag, antes do texto — replica a
      tag com toda a sua sub-árvore. Contagem literal sai como cópias (até 32),
      o resto vira laço contado; escrever as cópias é o que deixa `a(2) x` e
      dois `a x` serem o mesmo template, sem o qual o formatter não poderia
      fundir um no outro.
- [ ] faça com que o textmate format o gadx `script` como javascript e `style` como css, reconhecendo o `#{ CODE }#` 
  como interpolação, colorindo CODE como codigo gad corretamente como em `{ CODE }` nas nas demais situações.
  o `#{ CODE }#` tabem deve ser reconhecido em `@raw_text`. faça os testes no textmate.

## Log
### 2026-09-04
- `go test ./...` → `exit=0`, nenhum FAIL.
- `make verify` → `==> verify OK`, `rc=0` (regen, docs, staticcheck, vet, testes, CLI e WASM).
- `gad transpile site_template/index.html` da planaheadconstructions: 757 linhas
  de HTML → 598 de gadx, `grep -c '<[a-z]'` → 0 (nenhum HTML inline restante).
- Render do `.gadx` comparado com o `.html` de origem via HTMLParser:
  744 tags/atributos idênticos, 140 nós de texto idênticos, script/style
  idênticos ignorando a indentação externa.
- Comentários HTML: `go test ./gadx/ -run TestHTMLComment` ok; a validação do
  `index.gadx` passou de 744 para 785 nós casando, os 41 comentários incluídos.
- Repetição: `go test ./gadx/ -run TestTagRepeat` e
  `go test ./web/gadbridge -run TestFormatFolds` ok. No `index.gadx` o fmt
  fundiu três grupos de 5 ícones em `i.fa-solid.fa-star(5)`, 639 → 627 linhas,
  com a validação contra o `index.html` ainda em 785/140 nós idênticos.
- `*` round-trip: `gad samples/gadx/text_space.gadx` →
  `<p><a href="#one">one</a> <a href="#two">two</a></p>…`; `gad fmt` no arquivo
  não muda nada. Idem `samples/gadx/shorthand.gadx`.

## Unverified / Pending
- `.class ? cond` na forma curta não condiciona a classe, e `?` dentro de um
  grupo `[…]` não é lido como condição. Verificado igual em HEAD antes destas
  mudanças — é anterior, não regressão.

## Errors & Fixes
| Error | Cause | Fix | Evidence |
|-------|-------|-----|----------|
| `*` não renderizava espaço | run de texto só-espaço é podado ao baixar | o scanner entrega `{= " " }` | render mostra `one</a> <a` |
| corpo raw hoisted para fora da tag | `rawTextStmts` devolvia os stmts crus | envolver num `TextStmt`, como a região HTML faz | `<div><script>a = {…}</script></div>` |
| formatter recusava o arquivo | corpo raw abrindo com linha em branco não tem indentação para ler | `rawTextBlockSafe`, senão mantém a região HTML | `TestFormatKeepsRawTextElements` ok |
| forma curta quebrava o guard | `hoistID` só corria no `parseTag`, não no caminho da região HTML | aplicar também em `html_build.go` | `lowered equal=true`, `idempotent=true` |
| `TestPorted_Id` invertido | `hoistID` movia o segundo id à frente do primeiro | não hoistar quando já há um id antes | `go test ./gadx/...` ok |
| `(N)` saía como `()` | `gnode.Int` não preenche `IntLit.Literal`, que é o que o writer imprime | montar o `IntLit` com o literal | fmt escreve `a(3) link` |
| fmt recusava o repeat dinâmico | contador `$rep<pos>` mudava de nome a cada reformatação | nome fixo `$rep` | `samples/gadx/repeat.gadx` estável no fmt |

## Current State
Tudo do Plan está feito e verificado. O transpile de HTML sai em sintaxe de
tags, com `#id`/`.class` na forma curta, `script`/`style` com corpo indentado,
`*` para o espaço único entre elementos inline, `{= "…" }` para os demais, os
comentários HTML preservados e as tags repetidas fundidas em `(N)`.
