# TASK: gaddoc `*_api.gad` + tipos (interfaces/unions) + fixes de VM

> Created: 2026-08-10 | Updated: 2026-08-11

## Goal
Fazer `cmd/gaddoc` gerar arquivos `.gad` que documentam a API pública de cada
módulo com `export` de funções tipadas (params + retorno) e doc comments. Rodar
`gad doc <api>.gad` produz a documentação. Saída: `samples/stdlib/<mod>_api.gad`
(fmt, strings, time, json) e `samples/api.gad` (builtins root). Ligar ao
`go generate`.

## Decisões do usuário
- Builtins root: **adicionar gad:doc tipado no source Go** e depois gerar.
- ~11 sigs com notação `[opcional]`: **corrigir no source gad:doc** (Gad válido).
- Assinatura renderizada como bloco ```` ```gad ```` abaixo do header (não inline).
- Ligar geração ao `go generate` (//go:generate).

## Plan
- [x] Validar fluxo `export nome(params) <ret> => nil` → `gad doc`.
- [x] Assinatura tipada abaixo do header (gadbridge + md.gadx/html.gadx x2).
- [x] Gerador `gaddoc api <src> <out.gad> <module>` (extração + emissão tipada).
- [x] Corrigir 12 sigs `[opcional]` no source (module_fmt/strings/time*.go).
- [x] Gerar samples/stdlib/{fmt,strings,time,json}_api.gad + `gad doc` verde.
- [x] Root builtins (44 documentados) → samples/builtins_api.gad.
- [x] Overloads no gerador (`export func NAME { … }`).
- [x] export const + runtime const (StdModuleData).
- [x] Interfaces builtin + type union + sintaxe `type <…>`.
- [x] Testes + regenerar doc/samples/stdlib + go test ./... verde.
- [x] //go:generate + Makefile wiring (gaddoc_generate.go + `go generate -run gaddoc`).
- [x] Auditar sigs dos builtins root restantes (~31) — classificados; 4 construtores
      documentados; interfaces/erros/internos categorizados no header.

## Log
### 2026-08-10
- gadbridge/doc.go: export func expõe `Params.String()+FormatFuncReturn` (sem o
  nome) — validado (`add(a int, b int) <int>` abaixo do header).
- md.gadx + html.gadx (repo + embutidas): assinatura em bloco ```gad abaixo do
  h3/###. Validado via probe.
- cmd/gaddoc/api.go: modo `gaddoc api <srcdir> <out.gad> <module>`. Captura
  estruturada (func sig+desc, const name+value) no docgroup (main.go). normalizeSig
  converte o dialeto de doc p/ Gad válido: `->`→`<>`, `<nil>` dropado, `<[x]>`→
  `<array>`, união `<a|b>`→`<_ a|b>`, param tipo-função sem tipo, `on/off`→bool.
  Fallback `(*args)` + warning p/ sig não-parseável; skip p/ nome reservado (`in`).
  Doc de cada export em bloco `/** **/`.
- Corrigidas 12 sigs `[opcional]` no source (module_fmt/strings/time*.go).
- GERADOS e validados por `gad doc` (EXIT 0): samples/stdlib/{fmt,strings,time,
  json}_api.gad. 1 warning só: time `in` (keyword) pulado.

## Unverified / Pending
- **Consts NÃO capturados**: causa = source gad:doc quebrado. Em module_time.go
  o bloco `## Constants`/`### Months` (linha ~47) está SEM `// gad:doc` (nunca
  extraído); Weekdays/Layouts caem no bucket Types (falta header `## Constants` no
  grupo). Precisa reestruturar o gad:doc no source.
- **Assinaturas corretas por tipo real** (pedido do usuário): auditar cada função
  rastreando a impl `func(c Call){…}` + testes VM (*_test.go). NÃO feito.
- **Múltiplas assinaturas (overloads)**: usuário quer `met`/multi-sig. Precisa (a)
  sintaxe de export overload que gadbridge renderize, (b) suporte no gadbridge
  (hoje só ExportStmt→FuncExpr/escalar), (c) captura no gerador. NÃO feito.
- **Builtins root → samples/api.gad**: 354 builtins sem gad:doc/Header. Usuário
  quer adicionar gad:doc tipado no source (rastreando impl+testes). NÃO feito.
- go:generate + Makefile wiring. NÃO feito.
- Sync test TestDocTemplatesInSyncWithEmbedded (cópias iguais) — NÃO rodado.
- Regenerar doc/samples + doc/stdlib com novo layout (assinatura em bloco) —
  pendente; muda muitos .md. `go test ./...` — NÃO rodado após mudanças.

### 2026-08-10 (cont.)
- Overloads: DocSymbol.Overloads + DocOverload em gadbridge/doc.go; gadDocData trata
  `export func NAME { (sig)=>… }` (FuncWithMethodsExpr) e `export const` (nil val);
  gadExportName trata FuncWithMethodsExpr. RenderMarkdown + md.gadx/html.gadx (4
  cópias) renderizam 1 bloco ```gad + doc por overload. Validado.
- `export const NAME [= value]`: NOVO no parser (ParseExportStmt case token.Const) +
  ExportStmt.Const + String/WriteCode. Parseia e documenta (Pi=3.14, Answer=42).
  Compilador ignora o flag Const por ora (runtime const-enforcement NÃO feito).
- Gerador emite `export const NAME = value`. 4 arquivos stdlib regerados, `gad doc`
  EXIT 0.
- Testes golden atualizados (doc_template_test.go: assinatura agora em bloco abaixo
  do header). `go test ./...` TODO VERDE. `go build ./...` OK.

## Requisitos novos do usuário (pendentes)
- **Estrutura API**: overloads via `export func NAME { (sig)=>nil … }`; consts via
  `export const NAME`. [parse+doc PRONTO; gerador ainda emite single-sig p/ funcs]
- **Runtime const**: alterar ModuleData p/ suportar constantes e REJEITAR IndexSet
  em const. (ModuleData é interface; data padrão Dict{}.) NÃO feito.
- **Exemplos de uso dentro do gad:doc**: mover os samples/stdlib/use_*.gad p/ dentro
  dos comentários gad:doc, com validação de resultado/output (doctests `>>>`), de
  modo que use_*.gad fiquem obsoletos e a doc seja automatizada. NÃO feito.
- **Assinaturas corretas por tipo real**: auditar cada função (impl `func(c Call)` +
  testes VM) e escrever sigs corretas + overloads no gad:doc. NÃO feito.
- **354 builtins root → samples/api.gad**: gad:doc tipado no source. NÃO feito.
- **Fix consts capture**: gad:doc do source quebrado (## Constants sem gad:doc etc).
- **go:generate + Makefile**; regenerar doc/ com novo layout.

### 2026-08-10 (cont. 2) — COMMITADO
- fc51f89 feat(export): `export const`. 32804a9 feat(doc): sigs+overloads abaixo do
  header. 98858d2 feat(gaddoc): gera *_api.gad + fixes de sig. 12c0191 feat(module):
  StdModuleData (Vars/Consts/Funcs; IndexSet só Vars; @vars/@consts/@funcs vivos;
  Set roteia por tipo; SetConst). Testes verdes, go test ./... verde.

## Próximos (pós-StdModuleData)
- Compilador: `export const` → SetConst; módulos compilados usarem StdModuleData e
  rotear const/func/var. (Hoje StdModuleData é opt-in; default ainda Dict{}.)
- Migrar módulos Go stdlib p/ popular Vars/Consts/Funcs.
- (demais itens grandes já listados: use-examples no gad:doc, builtins root, sigs
  corretas por tipo, fix consts capture, go:generate, gerador emitir overloads.)

### 2026-08-10 (cont. 3) — módulos stdlib + time enums/durations
- 8c646b0 stdlib modules build StdModuleData (const buckets); registerBuiltinModule
  recebe StdModuleData; Base64Module→StdModuleData; AddressOf robusto (valor).
- ae5b83c compilador: export const → SetConst (OpExtendModuleConst); NewModule
  default = StdModuleData.
- 61c2e05 time: Months/Weekdays viram enum; durations viram tipo Duration;
  StdModuleData{Consts,Funcs} explícito; Int*Duration comutativo; ToGoInt64 aceita
  Duration.
- fd383a0 fmt/strings/reflect/json/base64: buckets explícitos StdModuleData{Funcs|Consts}.
- 64b877e time: assinaturas gad:doc com duration/Months/Weekdays; parseDuration/
  round/truncate/since/until/sub/`time-time` retornam Duration; ToGoInt64 aceita
  EnumValue. 81d0b33 regenera doc/stdlib-time.md.
- go test ./... VERDE após cada passo.

### 2026-08-10 (cont. 4) — gaddoc overloads
- 4a13c6d gaddoc emite `export func NAME { (sig)=>nil … }` p/ assinaturas múltiplas
  (agrupa linhas consecutivas de mesmo nome); single-sig mantém forma flat;
  fallback stub se não parseia. apiSym.overloads + captureFuncSig/appendFuncDesc.
  Testes: TestEmitAPIOverloads, TestGroupOverloadsCapture. gad doc renderiza 1
  bloco+doc por overload. 3d274e8 regenera time_api.gad.
- Nenhum gad:doc de módulo tem overloads hoje (feature p/ builtins root/futuro).

### 2026-08-10 (cont. 5) — builtins root (lote 1)
- e179777 pipeline: gaddoc moduleData caso "builtins" (enumera NewBuiltins);
  builtins_doc.go com `# builtins module` gad:doc. `gaddoc api . samples/
  builtins_api.gad builtins` gera 27 exports (len/cap/typeName/typeof/chars/copy/
  dcopy/repeat/contains/repr + 17 is*). gad doc valida.
- Nota: `len` tem Header runtime que sobrepõe o gad:doc (perde `<int>`); menor.
- 161a2ac lote 2: iteração (filter/map/each/reduce/keys/values/items/iterate/
  enumerate/collect/toArray) + sort/sortReverse + IO (print/printf/println/sprintf).
  Total 44 exports. gad doc valida.
- FALTAM: conversões (str/int/uint/float/bool/char/bytes/decimal/array/dict/…),
  meta (cast/wrap/is/implements/Class/addMethod/rawCaller/typeof/enter/exit/close/
  read/write/flush/stdio/obstart/obend/userData), operadores (binOp/unOp/selfAssignOp).

### 2026-08-10 (cont. 6) — interfaces builtin + type union
- 5709ed4 `iterable` interface (Interface.Native predicate). 7272afa callable/
  lengther/indexable/indexAssignable/indexDeletable (+ fix lastBuiltinType).
  67d473c callback callable em filter/map/each/reduce.
- 76a7d69 TypeUnion + builtin `number` (int|uint|float|decimal). Funciona como
  param/return/`::`; nested `str|number`. members preenchidos em init (ordem de
  var-init). Testes: TestNumberTypeUnion.

## Pedidos PENDENTES do usuário (type union feature)
- **Sintaxe `type <int|uint>`** — FEITO (4ff424a). Expr `type <int|uint>` +
  stmt `type NAME <int|uint>` (sugar p/ const). Keyword contextual (`type`+`<`);
  nó TypeUnionExpr; OpMakeTypeUnion. `type` segue ident normal (.type/type=/:=).
  Round-trip fmt corrige `< >`. Testes parser+VM. (Sintaxe final: `< >`, não bare.)
- **Documentar definição de interfaces/unions por Go** — FEITO (b211e48).
  doc/embedding.md seção "Type unions and interfaces from Go": NewTypeUnion +
  &Interface{Native:...} via globals, param/return/`::`. Snippets verificados.
- **Exemplos bem documentados** dessas features (sample .gad + gad:doc). NÃO feito.
- **Usar interfaces nas assinaturas** dos builtins — FEITO (adc9d09).
  contains(o iterable), chars(s str|bytes), repeat(o str|bytes|array). len/cap/
  sort ficam `any` (toleram tudo / Sorter custom). Callbacks já eram callable,
  iteração já era iterable. Nota: builtins root não têm Header → gad:doc é
  documentação (não valida runtime); interfaces tornam a doc precisa.

### 2026-08-10 (cont. 7) — protocolo iterator + bug de dispatch
- Confirmado: interface `iterable` JÁ reconhece o protocolo de overload por classe
  (met iterator(T) / iterator(T,state)) — isIterable(Range())=true, for-in ok.
- f0920fe: ToIterator usa vm.Call (não NewInvoker); ParamType.AcceptResolve
  resolve símbolos de tipo free contra o closure da função (não vm.curFrame).
  Suíte verde.
- **BUG CORRIGIDO** (74667f6) "expected Range, found Range": iterar instância de
  classe (values/collect/for-in) após passar por PARÂMETRO de função. Causa raiz:
  o símbolo de tipo `Range` do `met iterator` é compilado como ScopeLocal (não
  promovido a free var — resolução de tipo-param roda no escopo EXTERNO, não no da
  função), então re-valida contra o frame do caller (errado). Fix: ToIterator
  chama start/next/len com SafeArgs (dispatch já casou por tipo → re-validação é
  redundante). Fix de compilador (resolver tipos no escopo da função) foi tentado
  mas quebra registro de método de classe (`this cls` vira closure) — revertido.
  Regressões em TestVMIterator.

### 2026-08-11 — doc do protocolo iterator
- 163e57b: seção "Custom iterables (the iterator protocol)" em
  samples/06_control_flow.gad com exemplo runnable (class Range com met
  iterator start/next; forma key/value `[(k)=v]`; for-in, collect(values),
  func(x iterable)). gad doc valida, outputs conferidos. 7703cd4 regenera índice.

## Current State (o Log abaixo detalha cada passo; este bloco é o resumo do AGORA)
`go test ./...` VERDE. Muitos commits locais (NADA PUSHED — usuário revisa/pushe).
Estado consolidado (ver Log cont.1..5 para detalhes/hashes):

- **gaddoc**: `gaddoc api <src> <out.gad> <mod>` gera exports tipados + doc
  `/** **/`; overloads `export func NAME { … }`; consts `export const`. gad doc
  renderiza assinatura/overloads em blocos ```gad abaixo do header. Arquivos:
  samples/stdlib/{fmt,strings,time,json}.gad + samples/builtins.gad (**66 builtins
  root documentados**, incl. construtores str/int/…). Target Makefile `generate-api`
  (samples-doc depende).
- **`export const`** → runtime const-enforcement via **StdModuleData** (Vars/Consts/
  Funcs; OpExtendModuleConst; default do NewModule). Módulos stdlib em buckets
  explícitos. time: Months/Weekdays enums, durations tipo Duration.
- **10 interfaces builtin** (Interface.Native): iterable/callable/lengther/indexable/
  indexAssignable/indexDeletable/classInstance/classType/readable/writable. Usadas
  nas assinaturas (read/write/flush → readable/writable). `readable`/`writable` (não
  reader/writer, que são tipos builtin estreitos). Commit 81bd499.
- **TypeUnion + `number`** + sintaxe **`type <int|uint>`** (expr) / `type NAME <…>`
  (stmt). OpMakeTypeUnion. classType→classInstance documentado.
- **fmt scan (sscan/sscanf/sscanln) PROPAGA erro (throw)**, não retorna &Error{};
  idem chars/repeat/contains/sort → sigs sem `| error`.
- **BUG corrigido** (74667f6): iterar instância de classe passada por parâmetro.
- **Docs/samples**: doc/embedding.md (módulos/interfaces/unions por Go); protocolo
  iterator em 06_control_flow.gad; **samples/35_type_unions.gad** (unions+interfaces,
  runnable/doctested, no langOrder).

### 2026-08-11 (cont.) — MODULE.gad rename + fmt scan throw
- a53fbfc/bfeee4c: renomeou *_api.gad → MODULE.gad (samples/stdlib/{fmt,strings,
  time,json}.gad, samples/builtins.gad); target `generate-api` no Makefile
  (samples-doc depende dele). 63590ae: fmt Scan Examples ```go→```gad ignore.
- d7d25a9: **fmt scan agora PROPAGA erro (throw)** em vez de retornar &Error{}
  como valor. sscan/sscanf/sscanln sig `<int>` (não `<int|error>`); exemplos
  try/catch; fmtPostScan retorna o erro; partial scan preservado; testes do fmt
  reescritos (try/catch + str(err.cause)).
- 6a68aad: mesmo bug em chars/repeat/contains/sort/sortReverse — throw, não
  error-value → sigs sem `| error`, descrições "throws". + is/implements/wrap
  documentados.

### 2026-08-11 (cont. 2) — builtins root lote 3
- 6a68aad: is/implements/wrap. 7291838: Class/addMethod/obstart/obend/read/write/
  close/flush. 54 builtins root documentados. enter/exit ficam de fora (namespace
  gad, não top-level; usados pelo `with`). `Class` sem return (`<class>` não
  parseia — keyword). stdio pulado (identifiers incertos).
- FALTAM builtins root: conversões (str/int/uint/float/bool/char/bytes/decimal/
  array/dict — são TIPOS, não funcs), operadores (binOp/unOp/selfAssignOp — doc no
  sample 14), cast/stdio/userData + internos (namedParamTypeCheck/methodFromArgs/
  rawCaller/vmPushWriter/vmPopWriter/iteratorInput).

### 2026-08-11 (cont. 3) — classInstance/classType interfaces
- 66afc59: interface native `classInstance` (casa *ClassInstance — instância de
  classe). ae23b65: interface `classType` (casa *Class — a classe); Class typed
  `<classType>`; doc `classType(…) <classInstance>` (chamar classe→instância).
  Confirmado: Class(...) retorna *Class (a classe), não instância.

### 2026-08-11 (cont. 4) — sample de type unions
- 9760399: samples/35_type_unions.gad (runnable, doctested): type unions (`type
  <…>` expr/stmt, builtin number, nesting, type-como-ident), 8 interfaces
  behavioural com tabela, classType→classInstance. No langOrder após 24_interfaces.
  Snippets validados por gad doc.

### 2026-08-11 (cont. 5) — construtores de tipo
- 5b1b4fc: documentados os 11 construtores/conversões (str/rawstr/int/uint/float/
  decimal/char/bool/bytes/array/dict) sob ## Functions (header próprio não é
  reconhecido pelo docgroup). Throw em conversão inválida. 66 builtins root
  documentados. gad doc valida.

### 2026-08-11 (cont. 6) — readable/writable + cast/userData
- 81bd499: interfaces `readable`/`writable` (native, ReaderFrom/WriterFrom;
  read/write/flush tipados). Nomes adjetivos pois reader/writer são tipos builtin
  estreitos. 7fead71: cast/userData documentados. **68 builtins root documentados.**
  Header de builtins_doc.go atualizado.

### 2026-08-11 (cont. 7) — stdio + tabela sample 35
- 8e728b5: stdio documentado (IN/OUT/ERR → readable/writable). **69 builtins root
  documentados.** Tabela de interfaces do sample 35 inclui readable/writable (10).

### 2026-08-11 (cont. 8) — go:generate wiring + audit dos builtins restantes
- **go:generate wiring**: gaddoc_generate.go (root, package gad) com 5 //go:generate
  `go run ./cmd/gaddoc api …` (fonte única). Makefile `generate-api` agora roda
  `go generate -run gaddoc ./...` (só as directives de gaddoc, pula mkcallable/
  update-delve). Prova: `go generate -n -run gaddoc ./...` lista as 5; `go generate
  -run gaddoc ./...` EXIT 0, regenera só os stubs.
- **Audit dos ~31 builtins não-documentados** (rastreado por probe na VM + fonte):
  - 4 construtores CHAMÁVEIS documentados em ## Functions: `iterator(it iterable)
    <iterator>`, `zip(*iterables) <iterator>` (ENCADEIA iteráveis — confirmado por
    probe: itera o 1º todo, depois o 2º), `keyValue(key,value) <keyValue>`,
    `keyValueArray(*pairs) <keyValueArray>`. **73 builtins documentados.**
  - 10 interfaces + `number`: value-types (não funcs) → sample 35. 10 erros
    predefinidos + DISCARD_WRITER: consts. binOp/unOp/selfAssignOp: sample 14.
    Internos (namedParamTypeCheck/methodFromArgs/rawCaller/vmPushWriter/
    vmPopWriter/iteratorInput). Tudo categorizado no header de builtins_doc.go.
  - PROVAS: `go run ./cmd/gaddoc api . samples/builtins.gad builtins` EXIT 0 (73
    exports); `gad doc samples/builtins.gad` EXIT 0; `make samples-doc` EXIT 0
    (builtins.md renderiza os 4); `go test ./...` VERDE; check-delve up to date;
    go build/vet EXIT 0.

## Nova feature: tipos paramétricos (generics com constraints) — 2026-08-11 cont. 11
Sintaxe: `func mySet[T indexable, K int|uint, V number](target T, k K, v V) <T> { }`.
Lista `[IDENT TYPE, ...]` após `func` (e após nome se houver), antes de `(`. TYPE é
"como param type" (pode ser união/interface). Deve valer em: func nomeada/anônima,
`const f = func [T ...](...)`, header, meti, e method shorthand em dict/named params.
**Design**: type param = ALIAS de constraint resolvido por SUBSTITUIÇÃO em
compile-time (`T`→exprs da constraint). Sem opcodes/runtime novos. Reusa
nameSymbolsOfTypedIdent/returnTypesOf (expandem type-param → símbolos da constraint).
- [x] AST: FuncHeader.TypeParams + ClosureExpr.TypeParams + String/WriteCode
      (FormatTypeParams). Round-trip por gad fmt confirmado.
- [x] Parser: parseTypeParams (ParseTypedIdent) + peekTypeParamsAfterLBrack
      (desambigua `[` index × type params por `[IDENT IDENT`).
- [x] Parser: wired em func nomeada/anon (ParseFuncExprT), meti+header
      (parseInterfaceHeader/ParseFuncHeaderExpr), dict method shorthand
      (ParseFuncDefLit/ParseDictElementLit).
- [x] Compiler: c.typeParams map + withTypeParams helper; typeExprSymbols expande
      type-param→símbolos da constraint (com guard de recursão); aplicado em
      compileFunc/buildFuncHeaderObject/buildCtxFuncHeaderObject; refatorado
      nameSymbolsOfTypedIdent + returnTypesOf.
- [x] Testes: type_params_test.go (VM, 5 funcs) + parser/type_params_test.go
      (round-trip 6 contextos). docs: samples/36_type_parameters.gad (doctested,
      no langOrder após 35). PROVAS: go test ./... VERDE; make samples-doc EXIT 0
      (5 doctests); delve up to date; vet limpo.
  NOTA semântica (pré-existente, não do feature): named-func dispatch é permissivo
  p/ tipos interface/união (só concretos como int|uint são checados no dispatch);
  anon/closure validam tudo. return types <T> não são runtime-enforced (nem
  diretos). Type params se comportam idêntico a escrever a constraint direto.

## 2026-08-11 cont. 14 — gad:samples (exemplos de uso no MODULE.gad) + doctest inline
- **Diretiva** `// gad:samples [module,auto] <path>` (usuário adicionou em
  module_time.go) → gaddoc captura path+flags (reSamplesDir em main.go).
- **Gerador** (api.go): parseSampleFile lê regiões `//snippet NAME … //endsnippet`
  (formato doctest EXISTENTE, não inventa novo) → snippet por membro. `auto`:
  scaffoldMissingSamples cria a samples file com um `//snippet NAME`/`//endsnippet`
  por membro exportado sem exemplo. emitAPIGad/emitSingleFunc/emitOverloadedFunc:
  para cada item exportado, mescla o exemplo como subseção `## Example` (bloco
  ```gad ignore) JUNTO da assinatura `export … => nil`.
- **Doctest inline** (novo, cmd/gad/doc_snippet.go): `//= EXPR` (valor) e `//< TEXT`
  (saída) — forma terse de `/**= … **/` / `/**< … **/`. Teste
  TestExtractSnippetsInlineMarkers. `//` é só comentário em Gad (sem conflito).
- **stdlib/time/samples.gad**: criado (auto-scaffold, 41 membros); semeados 6
  exemplos determinísticos com `//= VALOR`; **validado por `gad doc` (doctest EXIT
  0)**. time.gad regenerado com `## Example` por membro.
- PROVAS: `go build/test ./...` VERDE; vet limpo; `gad doc stdlib/time/samples.gad`
  EXIT 0; merge visível (monthString → `## Example` + `time.monthString(1)` +
  `//= "January"`). make samples-doc EXIT 0.
- **Múltiplos doctests por snippet** (cont. 15): um snippet pode ter vários
  `//= VALOR`/`//< OUT`, cada um verifica o statement ANTERIOR (o valor/saída do
  código cumulativo até a linha). snippetCheck{lineEnd,kind,expected} +
  parseSnippetChecks + runSnippetChecks; renderSnippet insere `// => valor` inline
  após cada statement verificado. Testes TestExtractSnippetsMultipleChecks +
  TestExtractSnippetsInlineMarkers. Ex.: `1\n//= 1\n\n"x"\n//= "x"` → 2 checks.
- **Rollout (cont. 16)**: diretiva `gad:samples [module,auto]` em module_fmt.go/
  module_strings.go/stdlib/json/module.go (paths repo-relative stdlib/<m>/samples.gad).
  Fix bug: header do scaffold continha literal `//snippet NAME` → parser capturava
  "NAME" como membro; header reescrito. samples files gerados: time(41)/fmt(9)/
  strings(45)/json(10). **Exemplos autorados+validados**: fmt (todos 9: sprint/f/ln
  //=, print/f/ln retornam byte-count //=, scan família), json (Marshal/MarshalIndent/
  Unmarshal/Valid, cada com `json := import("json")`). MODULE.gad regenerado com
  `## Example`. Todos os 4 samples files: `gad doc` doctest OK.
- RESTA: autorar exemplos p/ strings(45) + time(35 restantes); dobrar use_time/
  use_strings; REMOVER use_*.gad; minerar testes.

## 2026-08-11 cont. 13 — stdlib refs: sig em ```gad, doc/stdlib/, submenu Stdlib
- **Assinaturas de função** em stdlib-*.md saíam inline (`` `local() <Location>` ``);
  agora em bloco ```gad (cmd/gaddoc/main.go processFuncBlock).
- **Path**: geram em `doc/stdlib/MODULE.md` (antes `doc/stdlib-MODULE.md`). Makefile
  atualizado; writeToFile faz MkdirAll do dir pai. Movido `doc/stdlib-test.md`→
  `doc/stdlib/test.md`; removidos os flat antigos.
- **Menu do site**: novo submenu (navGroup) **"Stdlib"** em site.go com os itens
  corretos: strings, fmt, json, time, test (via collectStdlibPages, lê doc/stdlib/).
  Rotas `ref-stdlib-<mod>` PRESERVADAS (slug estável). refOrder perde os stdlib-*.
- PROVAS: build-website `build --no-wasm` EXIT 0; content.json tem grupo "Stdlib"
  com os 5 (ref-stdlib-{strings,fmt,json,time,test}); time.md Functions em ```gad;
  `go build/test/vet ./...` VERDE.

## 2026-08-11 cont. 12 — tipos globais fora de time.gad → Built-in Types
- **Bug**: `samples/stdlib/time.gad` continha os tipos GLOBAIS FuncHeaderObject,
  Interface, MethodInterface, Prop (não são do time). Causa: gaddoc associa blocos
  gad:doc órfãos (sem `# X module`) ao header de módulo que os precede; `module_time.go`
  ('t') é o último `module_*.go` antes dos `objects_*.go` (alfabético) → vazam p/ time.
- **Fix**: consolidei os 4 blocos gad:doc num novo `builtin_types_doc.go` com header
  `# types module` (ordena "b", antes de `module_*`, então nada vaza p/ time). Removi
  os blocos free-floating dos 4 `objects_*.go` (cada tipo já tinha Go doc próprio na
  decl). gaddoc: `moduleData("types")` = dict vazio (só-doc); //go:generate p/ types.
  Título emitido: `moduleTitle()` rende **"# Built-in Types"** p/ types (não
  "`types` module" — são globais, não módulo; pedido do usuário). `# types module` no
  source é só marcador de fronteira.
- **Varredura de outros casos**: só os 4. `module_time_location.go`/`module_time_time.go`
  têm `## Types` mas documentam Location/Time (tipos DO time, sob `# time module`
  corretamente) — não são vazamentos.
- PROVAS: `go build/test ./...` VERDE; delve up to date; vet limpo. time.gad/
  stdlib-time.md SEM os tipos globais (grep FuncHeaderObject só em types.gad/types.md);
  samples/types.gad + doc/samples/types.md geram "# Built-in Types" (make samples-doc
  EXIT 0). builtins.gad inalterado.

## Restam — checklist acionável (2026-08-11 cont. 9)
- [x] **Bug de fundo do compilador** — CORRIGIDO (fix particionado, ver cont.10).
- [ ] **Revisar/pushar** os commits locais desta sessão. Usuário pediu NÃO pushar
      agora (revisar antes). Ação EXTERNA → aguarda ok explícito.

### 2026-08-11 (cont. 10) — FIX do bug de fundo (partição free-de-tipo × free-de-corpo)
- **compiler_nodes.go compileFunc**: após `fork.Compile(body)`, capturo símbolos de
  tipo de param/named/return com Scope Local/Free (referenciam escopo externo) para
  os free vars DESTA função via `st.Resolve(sym.Name)` — assim
  `paramTypeSymbolValue` os lê de `o.Free` (frame certo), não de `vm.curFrame`.
  `AllowMethods` passa a ser gated por `bodyFreeCount` (frees ANTES da captura de
  tipos), não por `len(freeSymbols)` — capturas só-de-tipo não desabilitam registro
  de método. O receptor `this` injetado (`compiler_class.go thisParam`, tipado com
  o `cls` externo) é PULADO na captura: métodos de classe despacham com SafeArgs →
  o tipo nunca é lido em runtime; capturá-lo tornava o método um closure anônimo.
- PROVA repro (`.__tmp/bug1.gad`): antes `TypeError: expected MyInt, found int`;
  agora `outer()(41)`→42 e `apply(outer(),41)`→42. Sample 06 (protocolo iterator
  real com `met`) segue EXIT 0.
- **Teste de regressão**: `TestParamTypeFromEnclosingLocal` (builtin_interfaces_test.go)
  — union em local capturado usado como param-type de closure retornado (direto e
  via apply), rejeição de valor inválido do frame errado, e método de classe cujo
  `this cls` referencia o local do define permanece registrável. PASS.
- PROVAS: `go test ./...` VERDE; `go run ./cmd/update-delve check` = up to date;
  `go build ./...`/`go vet` EXIT 0; `gofmt -l` limpo. SafeArgs (74667f6) mantido —
  otimização ortogonal (pula re-validação redundante pós-dispatch), ainda válida.

### Assessment do bug de fundo (2026-08-11 cont. 9)
- **Repro mínimo** (`.__tmp/bug1.gad`):
  ```gad
  outer := func() { MyInt := type <int|uint>; return func(v MyInt) => v + 1 }
  outer()(41)   // TypeError: expected MyInt, found int  — falha até na chamada DIRETA
  ```
  Isolado: falha SÓ quando o tipo é um **local capturado (free var)** usado como
  param-type E a função roda noutro frame. Casos que FUNCIONAM: tipo global (A),
  tipo local mesmo-escopo (B), tipo builtin em closure retornado (C).
- **Raiz**: `compiler_nodes.go:1954` `compileFunc` resolve os símbolos de tipo via
  `nameSymbolsOfTypedIdent`→`requireSymbol`→`c.symbolTable.Resolve` no **compilador
  EXTERNO `c`** (antes do fork `st`). Logo `MyInt` vira ScopeLocal do escopo externo.
  Em runtime `CompiledFunction.paramTypeSymbolValue` (bytecode.go:529) só trata
  ScopeFree lendo `o.Free`; ScopeLocal cai em `vm.GetSymbolValue`→`vm.curFrame`
  (frame ERRADO). O tipo nunca é promovido a free var da função.
- **Por que o fix ingênuo regride** (resolver tipos em `st`): promove `MyInt` a free
  → `freeSymbols>0` (compiler_nodes.go:2050) → função vira **closure** (OpClosure) e
  `AllowMethods=false`. Métodos de classe cujo param-type referencia a própria classe
  (`this cls`, ou o nome da classe) viram closures anônimos → `objects_class.go:1052`
  `ErrClassMethodRegister: method N: is anonymous`. Foi tentado e revertido antes.
- **Fix real (não-cirúrgico)**: PARTICIONAR capturas de free "de-tipo" das de "corpo"
  — capturar o valor do tipo em `o.Free` sem flipar a função p/ closure-mode que
  quebra registro de método. Toca: compileFunc (coleta separada), emissão OpClosure/
  layout de `o.Free`, paramTypeSymbolValue/ParamTypes, serialização de bytecode
  (Fprint/Equal) e **re-sync do delve** (vm_loop_debug.go). Blast radius alto no path
  mais quente do compilador/VM, p/ um edge case JÁ contornado (SafeArgs, 74667f6,
  cobre o caminho user-facing de iterators). **Não landeado** — aguarda decisão.

### Excluídos por decisão (não são pendências)
- **Builtins internos NÃO-user-facing** — permanecem sem gad:doc de propósito
  (documentar como API pública seria incorreto): `binOp`/`unOp`/`selfAssignOp`
  (sample 14_user_operators), `namedParamTypeCheck`/`methodFromArgs`/`rawCaller`/
  `vmPushWriter`/`vmPopWriter`/`iteratorInput`. `enter`/`exit` → namespace `gad`.
  Categorizados no header de builtins_doc.go (commit 8924557). DECISÃO FINAL.
- **use_*.gad → gad:doc c/ doctests** — usuário mandou IGNORAR/pular.
