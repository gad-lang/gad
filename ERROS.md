# Gadx

# Gad
- [x] a documentação do arquivo hello.gad nao esta renderizando o doc comment a nivel de arquivo.
      <!-- FEITO: o caminho raw (gad doc - / -html -, usado pelo IDE/playground) só
           reconhecia `/*** ***/` como prose de módulo; os samples usam `/** **/`
           detached. rootBlocks agora aceita um `/** **/` líder detached (espelha
           gadbridge.ExtractDoc) e suprime o `# name` sintético quando a prose já
           tem `# Title`. -->
