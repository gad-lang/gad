# TASK: Gadx — três defeitos em atributos de tag
> Created: 2026-09-05 | Updated: 2026-09-05

## Goal
Três formas de atributo que falham em silêncio ou derrubam o processo, achadas
escrevendo um formulário Vue num template. Nenhuma dá erro de sintaxe onde o
erro está.

## Plan
- [x] 1. Nome de atributo entre aspas é descartado sem aviso
- [x] 2. `tag[…]="x"` — grupo seguido de `=` — derruba com nil pointer
- [x] 3. Atributo sem valor desce como `nome=` e o Gad gerado não reparseia

## Os casos

**1. Nome entre aspas some.** Um nome que precisa de aspas (uma diretiva Vue,
por exemplo) é aceito e depois ignorado; os outros atributos do grupo saem
normalmente, então nada indica a perda.

    @main
        div["v-model"="form.name", title="t"] a

    →  <div title="t">a</div>          (o v-model sumiu)

O nome cru funciona: `div[v-model="form.name"]` sai correto. Ou o nome entre
aspas passa a valer, ou tem de ser erro de sintaxe — o que não serve é sumir.

**2. Grupo seguido de `=` derruba o processo.** Escrevendo o valor fora do
grupo, por engano, o writer recebe um nó sem expressão:

    @main
        span["v-text"]="x"

    →  %!v(PANIC=Format method: runtime error: invalid memory address or nil
       pointer dereference)
       em node.(*CodeWriteContext).WriteExprs → (*AssignStmt).WriteCode
       (parser/node/coder.go:974, stmt.go:59)

**3. Atributo sem valor não sobrevive ao transpile.** Renderizar funciona; o
`gad transpile` gera Gad que ele próprio não consegue ler de volta:

    @main
        div[novalidate]
            span x

    gad v.gadx        →  <div novalidate><span>x</span></div>     (ok)
    gad transpile     →  Parse Error: expected operand, found ','

O lowered mostra a causa — `GadxLowered` devolve um argumento nomeado sem
valor:

              8: "\t\t$el := gadx.Tag("
             11: "\t\t\t; novalidate="        ← sem valor

Deveria descer como `novalidate=yes`, que é o que o próprio render faz. Vale
para qualquer atributo-bandeira, e **atinge o sample do repositório**:
`gad transpile samples/gadx/boolean_attribute.gadx` falha em `ba.gad:18:3`. O
`make verify` não pega porque nada transpila os samples.

## Log
### 2026-09-05
- **(1)** `parseAttributeEntry` passou a ler um nome entre aspas
  (`readQuotedAttrName`), e o writer devolve as aspas só quando o nome as
  precisa (`attrName`). `div["x/y"="1", title="t"]` → `<div x/y="1" title="t">`;
  `div["v-model"=…]` volta do fmt como `v-model=…`, sem aspas, porque o nome cru
  já lê inteiro.
- **(2)** A regex do scanner de atribuição tinha o alvo opcional, então `="x"`
  virava atribuição sem alvo e o nó vazio derrubava o writer. Alvo agora é
  obrigatório: `span["v-text"]="x"` lê o resto como texto, sem panic.
- **(3)** `FlagLit.WriteCode` escrevia `e.Literal`, vazio num flag sintetizado,
  gerando `; novalidate=`. Cai para `String()` (`yes`/`no`) quando não há texto
  de origem. E `x=yes` é escrito de volta na forma nua `x` — `x=no` não, que
  omite o atributo e diz o contrário.
- `go test ./...` → `exit=0`.
- Comportamento fixado em teste: `gadx/attr_name_test.go` (nome entre aspas,
  atribuição sem alvo) e `web/gadbridge/flag_attr_test.go` (o flag desce como
  `=yes` e volta nu; `=no` preservado).
- `TestSamplesLowerAndReparse` baixa e reparseia **todos** os samples de
  `samples/gadx` — é o teste que teria pego o (3), que quebrava o
  `boolean_attribute.gadx` do próprio repositório sem nada acusar.

## Current State
Os três corrigidos, com teste. O `make verify` agora cobre o transpile dos
samples, que era o buraco por onde o (3) passou.

# TASK: TextMate — script/style e `#{ … }#` no gadx
> Created: 2026-09-04 | Updated: 2026-09-04

## Goal
No realce do gadx, o corpo de um `script` é JavaScript e o de um `style` é CSS,
com `#{ CODE }#` lido como interpolação e o CODE colorido como gad, igual ao
`{ CODE }` das demais situações. O `#{ … }#` vale também dentro de `@raw_text`.

## Plan
- [x] `script` / `style` em sintaxe de tag: corpo indentado vai para `source.js` / `source.css`
- [x] `<script>` / `<style>` como região HTML inline, atravessando linhas
- [x] `#{= … }#` (saída) e `#{ … }#` (controle) como ilhas de gad no texto cru
- [x] `@raw_text` com o corpo verbatim e as mesmas ilhas
- [x] Testes no tmtest (bun + vscode-textmate)
- [x] Aparar o espaço lateral de dentro de uma tag de bloco (`<p>  x  </p>` → `p x`)
- [x] `<pre>` / `<textarea>` mantêm o whitespace: na leitura do HTML, no lift e no formatter

## Log
### 2026-09-04
- `make grammar-test` → `31 pass, 0 fail` (8 casos novos em `gadx.test.ts`).
- Tokenização do `site_template/index.gadx` real (628 linhas) com a gramática
  nova: 33 tokens de JS, 92 de CSS, 445 nomes de tag, nenhum vazamento (nenhuma
  linha de tag comum dentro de um bloco embutido).
- Dump da tokenização conferido à mão: a linha da própria tag continua tag
  (`script[src="a.js"]` com `entity.name.tag` + `meta.attributes`), o corpo sai
  todo na outra linguagem, uma linha em branco não fecha o bloco e uma linha na
  indentação da tag fecha.

- Espaço lateral de bloco: `go test ./cmd/gad -run TestHTMLToGadx` ok. No
  `index.gadx` real, 627 → 540 linhas e os literais `{= " …"` caíram de 33 para
  9 (os 9 são bordas de elemento inline, que são conteúdo). Comparação de
  layout à moda de navegador — whitespace colapsado, quebra só em borda de
  bloco — entre o render e o `index.html`: 121 linhas idênticas nos dois.

- `<pre>`: `go test ./gadx/ -run TestPreKeepsWhitespace` e
  `go test ./cmd/gad -run TestHTMLToGadx` ok. Transpile de um documento com
  `<pre>`, `<pre><code>` e `<textarea>`: sai em sintaxe de tag e as quatro
  regiões voltam byte a byte iguais à fonte. `index.gadx` segue em 540 linhas,
  com o layout ainda idêntico ao `index.html` (121 linhas).

## Unverified / Pending
- O `source.js` / `source.css` de verdade vem do editor; o teste usa uma
  gramática-sentinela, então o que está provado é que o embed *resolve*, não
  como o JS ou o CSS é colorido.
- Nada commitado foi empurrado: gad-textmate `619199f`, vscode-gad `950217b`,
  intellij-gad `cc8432f` e gad `2f019ff` estão só locais, todos em `main`. O
  `main` do gad-textmate estava atrasado (`8668b60`) em relação ao HEAD
  destacado (`1a1599a`, = `origin/main`); foi fast-forward, sem divergência.

## Errors & Fixes
| Error | Cause | Fix | Evidence |
|-------|-------|-----|----------|
| a linha da tag ganhava o escopo do bloco | o bloco usava `name`, que cobre o `begin` | trocar por `contentName`, que só cobre o corpo | dump mostra `script[…]` sem `meta.embedded.block` |

## Current State
Feito, testado e commitado nos quatro repositórios, nenhum empurrado.

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
