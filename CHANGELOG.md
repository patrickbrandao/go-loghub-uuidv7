# Histórico de mudanças — go-loghub-uuidv7

Guia do histórico do projeto: o que mudou em cada versão, por quê, e o
que isso significa para quem consome a biblioteca. As versões seguem o
versionamento semântico e as tags publicadas são imutáveis (nunca são
movidas com `-f`, para não quebrar o `sum.golang.org` dos usuários).

Convenções de cada seção:

- **Adicionado**: API ou capacidade nova.
- **Alterado**: comportamento existente que mudou; leia com atenção ao
  atualizar.
- **Removido**: API ou capacidade que deixou de existir.
- **Corrigido**: defeito removido.
- **Documentação**: mudanças só em texto.
- **Decisões**: propostas avaliadas, aceitas ou rejeitadas, com o motivo,
  para que não voltem a ser sugeridas sem argumento novo.

**O registro canônico de decisões é a seção 11 do
[docs/SPEC.md](docs/SPEC.md)**, que lista cada uma com o motivo e o que
justificaria revê-la. Este arquivo guarda o histórico — quando cada
decisão foi tomada e o que mudou junto —, mas quem for propor ou auditar
deve ler a especificação primeiro. Decisão nova entra nos dois lugares.

---

## [Não publicado]

## [v0.0.2] — 2026-09-14

Nenhuma mudança no código de produção. Uma revisão em 2026-09-13 aplicou
à suíte, que já tinha 100% de cobertura de instruções, uma campanha de
mutação dirigida: de 45 defeitos injetados um por vez, em pontos
suspeitos, 34 passavam sem nenhuma falha. O código estava correto em
todos esses pontos, e cada um foi conferido à parte; faltavam os testes
que impedissem a regressão. Com os testes abaixo, os 45 são detectados.

### Adicionado

- **Fiação da entropia** (`tests/entropy_test.go`, arquivo novo).
  `TestEntropyWordsFeedTheirFields` trava, com palavras distintas, que no
  Nível 1 `rand_a` vem de `r1` e `rand_b` de `r2`, e que nos níveis 2 e 3
  `rand_b` vem da única palavra sorteada, pelo relógio e por instante.
  `TestReaderGeneratorByteOrderAndShortReads` trava a ordem de rede, a
  ordem `r1`, `r2`, o consumo de 8 bytes por palavra e a leitura curta
  completada, com um leitor que entrega um byte por chamada.
  `TestCryptoGeneratorReadsCryptoRandReader` prova que
  `NewCryptoGenerator` lê `crypto/rand.Reader`, trocado só durante a
  construção, e `TestSuiteHasNoParallelTests` falha se algum arquivo de
  teste do pacote chamar `Parallel`, o que tornaria a troca uma corrida de
  dados. `TestTextFormsUseTheGeneratorEntropy` prova que
  `GenerateString` e `GenerateAtString` usam a fonte do próprio gerador.
  `TestDefaultGeneratorDrawsIndependentWords` confere em volume que o
  gerador padrão sorteia duas palavras independentes no Nível 1.
- **Nível de todas as formas de gerar** (`tests/robustness_test.go`).
  `TestEveryGenerationFormWritesItsLevel` confere, pela assinatura
  estatística dos bits livres, o nível de `Generate`, `GenerateString`,
  `GenerateAt` e `GenerateAtString`, como função de pacote e como método,
  nos três níveis e em níveis desconhecidos, e dos quatro nomes. Antes, só
  os nomes chamados como método tinham o nível provado.
- **Oráculo exato das mutações de um byte** (`tests/parsing_test.go`).
  `TestFromStringNeverPanics` passa a exigir, nas 36 x 256 mutações, que
  `FromString` aceite exatamente as que mantêm a forma canônica, com o
  valor decodificado por `encoding/hex`, e não só que não entre em pânico.
  `TestParseSingleByteMutations` estende o critério a `Parse`,
  `ParseBytes` e `Validate` nas quatro formas, inclusive chaves e prefixo
  URN. `FuzzParse` (`tests/fuzz_test.go`) ganha o mesmo tipo de oráculo
  para entradas arbitrárias: toda entrada aceita tem de ser, sem distinção
  de caixa, uma das quatro formas do valor lido, porque a ida e volta
  sozinha não percebe um hífen ou um dois-pontos que deixaram de ser
  conferidos.
- **Valores exatos no teto de 48 bits** (`tests/bounds_test.go`).
  `TestBoundsExactValuesAtTheTopOfTheRange` confere as fronteiras dos três
  níveis no início do último segundo representável, no último
  milissegundo, dentro dele e nos dois primeiros instantes saturados, mais
  a geração por instante com entropia nula e a leitura de volta.
- `TestSplitUnixInstantFloorsPreEpoch` (`instant_internal_test.go`) passa
  a ser tabela, com o último nanossegundo antes da época, em que o
  carimbo vale -1, e o valor zero de `time.Time`.
- `TestTimeReadingsReturnUTC` (`tests/timestamp_test.go`): as duas
  leituras de instante devolvem `time.UTC` em todos os níveis, conferido
  por identidade, o que também vale numa máquina com `TZ=UTC`.
- Em `tests/api_test.go`: `TestVersionAndVariantReadTheirBits`, sobre
  todas as versões e variantes; `TestCompareAndIsZeroSeeEveryByte`, sobre
  cada uma das 16 posições; `TestScanAndMustParseAcceptEveryParseFormat`,
  nos quatro tipos de banco e em `MustParse`; e, em `TestErrorTaxonomy`,
  três casos de JSON com sequência de escape, que passam pelo
  `encoding/json` e precisam manter o erro específico de `Parse`. O caso
  que se chamava "string com escape e dígito inválido" não tinha escape e
  passou a se chamar "string com dígito inválido".
- Em `tests/binary_serialization_test.go`:
  `TestBinaryDecodersRejectEveryOtherLength`, de 0 a 64 bytes, e
  `TestGobRoundTrip`, com os quatro tipos numa estrutura.

### Documentação

- `docs/SPEC.md`: a seção 3.1 registra que a codificação sub-milissegundo
  é decimal, e não a do Método 3 da RFC; a 3.3 torna normativa a
  correspondência entre palavras e campos; a 3.5 registra a divergência do
  truncamento da RFC e o ponto exato em que a saturação entra; a 5.3 torna
  normativa a formação das palavras a partir do leitor e a prova de origem
  do gerador criptográfico. Na seção 10, os casos 1, 2, 3, 4, 10 e 17
  ganharam os requisitos acima, e o caso 19, novo, reúne a coerência da
  API de apoio.
- `README.md`: nota sobre a codificação decimal e a leitura do
  sub-milissegundo por outras bibliotecas.
- `CLAUDE.md`, `CONTRIBUTING.md`, `STARTHERE.md` e
  `docs/TEST-AND-BENCHMARK.md`: o arquivo novo de testes, a proibição de
  `t.Parallel` na suíte e as decisões novas.

### Decisões

Registradas em `docs/SPEC.md` seção 11, com o que justificaria revê-las.

- **Os campos sub-milissegundo são decimais, e não a fração binária do
  Método 3 da RFC 9562 §6.2** (11.3). É o que permite à leitura por nível
  reconhecer ruído pela faixa; o custo é a leitura errada do
  sub-milissegundo por bibliotecas que sigam o Método 3.
- **A saturação acima de 48 bits diverge do truncamento da RFC 9562
  §6.1** (11.3), só na construção a partir de um instante.
- **O gerador sobre leitor usa ordem de rede, completa leituras curtas e
  lê `r1` antes de `r2`**, agora como contrato (11.1).
- **Recusado um ponto de injeção no gerador padrão para testar as funções
  de pacote** (11.2): a prova é a assinatura estatística, sem estado
  global mutável.
- **A prova de origem do gerador criptográfico troca
  `crypto/rand.Reader` durante a construção**, e por isso a suíte não usa
  `t.Parallel` (11.2).

---

## [v0.0.1] — 2026-09-13

Primeira versão publicada. Criação do projeto, em 2026-09-13, a partir de `go-loghub-uuid` na
`v0.6.0` mais os três commits não publicados que a seguiam (até
`bd7d920`). O objetivo é uma biblioteca com um único propósito: gerar
UUIDv7 nos níveis 1 a 3 e converter UUIDv7 em instante nas mesmas
precisões. Tudo que não está diretamente ligado a gerar ou a processar
UUIDv7 foi descartado.

Para quem vem de `go-loghub-uuid` usando só UUIDv7, a migração é a troca
do caminho de importação e do nome do pacote, mais os três pontos da
seção "Alterado": `ImportBinary` passa a devolver erro, `IsValid` passa a
reconhecer só UUIDv7, e as leituras de tempo recusam outras versões.

### Removido

- **As versões 1, 2, 3, 4, 5, 6 e 8** e tudo que só existia por elas:
  `clock.go` (`GregorianTime`, `GetTime`, `ClockSequence`,
  `SetClockSequence`, `NodeID`, `SetNodeID` e o piso de relógio por
  sequência), `version1.go` (`GenerateV1`, `GenerateV6`), `version2.go`
  (`Domain`, `GenerateV2`, `GenerateV2Person`, `GenerateV2Group`),
  `version4.go` (`GenerateV4`), `version8.go` (`GenerateV8`,
  `GenerateV8Random`), `namebased.go` (`NameSpaceDNS`, `NameSpaceURL`,
  `NameSpaceOID`, `NameSpaceX500`, `GenerateV3`, `GenerateV5`,
  `GenerateHash`) e, em `inspect.go`, os métodos `GregorianTime`,
  `ClockSequence`, `NodeID`, `Domain` e `ID` de `UUID`.
- **A camada de compatibilidade com `github.com/google/uuid`**
  (`compat.go`: `New`, `NewString`, `NewRandom`, `NewRandomFromReader`,
  `NewUUID`, `NewV6`, `NewV7`, `NewV7FromReader`, `NewMD5`, `NewSHA1`,
  `NewHash`, `NewDCESecurity`, `NewDCEPerson`, `NewDCEGroup`), o guia
  `docs/MIGRATION.md` e o módulo aninhado `tests/compare`, que comparava
  o comportamento de v1 com aquele pacote.
- **`Max`, `IsMax`, o tipo `UUIDs`, `VersionString`, `VariantString`**
  (`values.go`) e **`IsInvalidLengthError`** (`parse.go`). A faixa de
  UUIDv7 é dada por `MaxAt`; `errors.Is(err, ErrInvalidLength)` e
  `slices.SortFunc(lista, UUID.Compare)` substituem os outros dois em uma
  linha.
- Os testes, benchmarks, exemplos, alvos de fuzzing, vetores dourados e
  casos da especificação das versões removidas: `tests/versions_test.go`,
  `tests/clockstate_test.go`, a tabela gregoriana de
  `tests/golden_test.go`, `FuzzGregorianUnixTime`, os benchmarks
  `GenerateV1`, `GenerateV4`, `GenerateV5`, `GenerateV6` e
  `GenerateV1Parallel`, `ExampleGenerateV5`, e a seção 4 antiga e os
  casos 18 a 21 antigos de `docs/SPEC.md`.

### Alterado

- **Caminho de módulo e nome do pacote.** O módulo passa a ser
  `github.com/patrickbrandao/go-loghub-uuidv7` e o pacote, `uuidv7`
  (antes `github.com/patrickbrandao/go-loghub-uuid` e `loghubuuid`). As
  mensagens de erro e de pânico passam a começar por `uuidv7:`, e quatro
  delas ganharam a acentuação que faltava. `go.mod`, todos os arquivos.
- **`ImportBinary(UUID) (Time, error)`** (antes `ImportBinary(UUID)
  Time`), e as quatro leituras de tempo recusam o UUID que não é de
  versão 7 com a variante da RFC. `Import` e `ImportBinary` devolvem o
  novo `ErrNotV7` com a estrutura zerada; `Timestamp` e
  `TimestampWithLevel` devolvem `false` com o instante zero. Antes, a
  extração lia qualquer UUID como UUIDv7, devolvendo uma data sem sentido
  para um UUIDv4, e `TimestampWithLevel` conferia a versão mas não a
  variante. `Timestamp` deixa de ler as versões 1 e 6. `import.go`,
  `inspect.go`; normativo em `docs/SPEC.md` seção 4.

  Os analisadores, a leitura de banco e os desserializadores **não**
  mudaram: continuam conferindo só a forma, e aceitam um UUID bem formado
  de qualquer versão.

  Custo medido num Apple M2: `ImportBinary` de ~1,7 ns para ~2,2 ns, sem
  alocação, também na recusa. A geração não foi tocada.
- **`IsValid` reconhece UUIDv7**: versão 7 e variante `0b10`. Antes
  aceitava as versões 1 a 8 e os valores especiais nulo e máximo. `Nil`
  passa a ser recusado. `values.go`.

### Adicionado

- **`ErrNotV7`**, o erro das leituras de tempo para UUID de outra versão
  ou variante. Família própria: não embrulha `ErrInvalidFormat`.
  `import.go`.
- **Testes da leitura de tempo** (`tests/timestamp_test.go`):
  `TestTimeReadingsRejectNonV7` varre as 64 combinações de versão e
  variante e exige as quatro leituras de acordo com `IsValid`;
  `TestTimeReadingsRejectRealUUIDsOfOtherVersions` repete a recusa sobre
  os exemplos da RFC 9562 das versões 1, 3, 4, 5, 6 e 8, o nulo e o UUID
  com todos os bits em um, e exige que os analisadores os aceitem;
  `TestRFC9562Version7Vector` lê o exemplo de UUIDv7 do apêndice A.6 da
  RFC, o único vetor de leitura de tempo calculado fora do projeto; e os
  testes de `Timestamp` e `TimestampWithLevel` que viviam no arquivo das
  versões, agora com os limites 999 e 1000 da faixa e os níveis
  desconhecidos.
- **`FuzzTimeReading`**, quinto alvo de fuzzing, no lugar do gregoriano:
  recebe 16 bytes crus e exige que as leituras concordem com `IsValid`,
  que a extração devolva o carimbo exato e os campos crus na faixa dos
  bits, que a leitura por nível some os campos exatamente quando cabem em
  0 a 999, e que o instante lido regere por `MinAt` os mesmos campos.
  `tests/fuzz_test.go`, `.github/workflows/ci.yml`.
- **`TestTimeReadingZeroAllocations`** trava as quatro leituras e
  `IsValid` sem alocação, aceitando e recusando. `tests/alloc_test.go`.
- **`BenchmarkImport`** e **`BenchmarkTimestampWithLevel3`**
  (`tests/benchmark_test.go`), e os dois no benchmark curto do CI.
- **`ExampleImport`**, com o exemplo de UUIDv7 e o de UUIDv4 da RFC 9562.
  `example_test.go`.
- Na taxonomia de erros (`tests/api_test.go`), o UUIDv4 recusado por
  `Import` e `ImportBinary` com `ErrNotV7` e aceito por `Parse`, e o
  valor do pânico de `ErrEntropySource` com leitor esgotado, que antes só
  era exercitado pelos apelidos de compatibilidade.
- `TestCompareOrdersLikeStrings` passa a conferir pares aleatórios nos
  dois sentidos, a antissimetria e `slices.SortFunc` com `UUID.Compare`.

### Documentação

- **`docs/SPEC.md` passa a especificar só UUIDv7.** A seção 4 foi
  reescrita como a regra normativa de recusa das demais versões na
  leitura de tempo; as seções 1, 2, 5, 6.4, 7 e 8 perderam o que era das
  outras versões e da compatibilidade; os itens 4 e 7 da seção 9 passaram
  a registrar os dois defeitos de leitura de tempo corrigidos aqui; os
  casos 5, 7 e 9 da seção 10 passaram a ser o vetor da RFC, a recusa das
  outras versões e a independência de ordem, os casos gregorianos saíram
  e o antigo caso 20 virou o 18. A numeração das seções e dos casos 1 a
  17 foi preservada, porque código e testes a citam.
- `README.md`, `STARTHERE.md`, `CLAUDE.md`, `CONTRIBUTING.md`,
  `SECURITY.md`, `docs/DEPLOY-FAST.md`, `docs/DEPLOY-FULL.md`,
  `docs/TEST-AND-BENCHMARK.md` e `docs/RELEASE.md` reescritos para o
  escopo, o pacote e o módulo novos. Os guias de uso ganharam a leitura
  do instante como seção própria. A tabela de referência de desempenho
  das demais versões foi substituída pela da leitura de tempo, medida
  em 2026-09-13.
- `clock_internal_test.go` virou `instant_internal_test.go`, só com os
  testes de `splitUnixInstant`.
- `.gitignore`, `.golangci.yml` e `.github/workflows/ci.yml` vieram do
  projeto de origem, sem o passo da suíte comparativa, com
  `FuzzTimeReading` no lugar de `FuzzGregorianUnixTime` e com a cobertura
  mínima elevada de 95% para 100%: sem as versões removidas, não sobra
  ramo inalcançável, e a suíte curta cobre todas as instruções.

### Decisões

Todas registradas em `docs/SPEC.md` seção 11.3, com o que justificaria
revê-las.

- **A biblioteca trata só UUIDv7.** O que ficou de apoio — análise
  permissiva, serialização, integração com banco, fronteiras e geração
  por instante — ficou porque serve para receber, guardar e consultar
  UUIDv7.
- **As leituras de tempo recusam outras versões; os analisadores
  continuam só de forma.** Recusar na análise impediria até de registrar
  em log o valor que chegou errado, e guardar ou transportar um
  identificador não depende da versão.
- **`ErrNotV7` não embrulha `ErrInvalidFormat`**, porque o texto foi
  aceito.
- **O pacote chama-se `uuidv7`.**
- **Os apelidos `GenerateV7` e `GenerateV7Level1` a `GenerateV7Level3`
  ficam.** O motivo original, acompanhar os nomes das demais versões,
  saiu com elas; a permanência foi decidida de novo, pelo nome por versão
  designar o UUIDv7 padrão sem extensão e pelos nomes por nível deixarem
  o nível legível na chamada.
- **Recusada a verificação de versão em uma comparação só**
  (`(uint16(u[6])<<8|uint16(u[8]))&0xF0C0 == 0x7080`). Medida contra as
  duas comparações legíveis, saiu ~0,1 ns mais lenta em `ImportBinary` e
  ~0,3 ns em `TimestampWithLevel`.
- **Mantidos por herança, com o motivo reescrito**: o sentinela puro do
  analisador estrito, a assimetria do prefixo URN e a permanência em
  `v0.x`.

---

## Origem

O histórico anterior à criação deste projeto — da `v0.1.0` à `v0.6.0` de
`go-loghub-uuid`, com as decisões que deram forma aos níveis, às
fronteiras, à geração por instante, à política de entropia e às travas de
alocação — está no `CHANGELOG.md` daquele repositório.

[Não publicado]: https://github.com/patrickbrandao/go-loghub-uuidv7/compare/v0.0.2...HEAD
[v0.0.2]: https://github.com/patrickbrandao/go-loghub-uuidv7/compare/v0.0.1...v0.0.2
[v0.0.1]: https://github.com/patrickbrandao/go-loghub-uuidv7/releases/tag/v0.0.1
