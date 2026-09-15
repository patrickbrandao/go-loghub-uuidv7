# 09. Testes e benchmark

Como rodar a suíte de testes da implementação de referência, o fuzzing, a
cobertura, o linter e a integração contínua, e como medir o desempenho:
benchmarks, geração em massa de 1.000.000 de UUIDv7 por nível e os
resultados de referência. O que a suíte é obrigada a provar está em
[08-casos-de-teste.md](08-casos-de-teste.md); este arquivo é o guia de
execução.

Todos os arquivos de teste ficam na pasta `tests/`, fora da raiz de
produção, e importam a biblioteca pelo seu caminho de módulo, como um
consumidor externo faria. Por isso `go test` roda contra `./tests/`, e
não contra a raiz, com duas exceções na raiz: `instant_internal_test.go`,
que cobre a decomposição pura do instante (`splitUnixInstant`), que não
pode ser exercitada de fora do pacote porque o relógio não é injetável, e
`example_test.go`, com as funções `Example`, que o godoc só associa ao
pacote quando vivem no diretório dele.

---

## 1. Testes funcionais

```bash
go test ./tests/ -v            # suíte completa, com os testes de massa de 1.000.000
go test ./tests/ -short -v     # modo rápido: pula os testes de massa
go test ./tests/ -run TestRoundTripString   # um teste pelo nome
```

O que cada arquivo cobre:

| Arquivo | Cobre |
|:---|:---|
| `generation_test.go` | versão e variante, ida e volta entre binário e texto, apelidos de conversão, texto inválido, extração em Nível 3 e `TestMonotonicity`, que dorme entre gerações para o instante embutido avançar de verdade |
| `layout_test.go` | layout de bits com entropia determinística (zero e um), versão e variante em todos os níveis, carimbo e sub-milissegundo conferidos contra o relógio |
| `entropy_test.go` | fiação da entropia: palavras, leitor, `crypto/rand` e independência das duas palavras do gerador padrão |
| `robustness_test.go` | bordas do `Generator` (fonte nula, valor zero, referência nula), consumo de entropia por nível, níveis desconhecidos, os nomes `GenerateV7*`, os atalhos de pacote e o nível de cada forma de gerar (`TestEveryGenerationFormWritesItsLevel`) |
| `ordering_test.go` | ordenação, unicidade e concorrência; `TestOrderingFollowsEmbeddedTime` é a invariante independente do relógio e `TestTieRateReport` informa a taxa de empates do host |
| `timestamp_test.go` | as leituras de instante, o vetor A.6 da RFC 9562, o descarte por faixa, a recusa nas 64 combinações de versão e variante e nos exemplos da RFC das versões 1, 3, 4, 5, 6 e 8, e o retorno em UTC |
| `import_test.go` | o vetor da extração cega, texto inválido, as faixas por nível em volume e a ida e volta a partir do relógio |
| `parsing_test.go` | `FromString`/`String` e o oráculo exato das 36 × 256 mutações de um byte nas quatro formas |
| `api_test.go` | análise permissiva e estrita, os valores (`Nil`, `Compare`, `URN`, `Bytes`, `IsValid`, `Version`, `Variant`), os anexadores, serialização em texto, binário e JSON, SQL e `NullUUID`, os geradores sobre leitor e criptográfico, e a taxonomia de erros (`TestErrorTaxonomy`) |
| `bounds_test.go` | `MinAt`, `MaxAt` e `RangeAt`, inclusive os valores exatos no teto de 48 bits |
| `construct_test.go` | `GenerateAt`, inclusive a não divergência com a geração pelo relógio |
| `golden_test.go` | os vetores dourados da extensão multinível |
| `binary_sql_test.go` | `BinaryUUID` e `NullBinaryUUID` no banco |
| `binary_serialization_test.go` | serialização JSON, texto, binário e `gob` dos quatro tipos |
| `alloc_test.go` | travas de alocação |
| `fuzz_test.go` | os cinco alvos de fuzzing (seção 3) |
| `benchmark_test.go` | benchmarks e testes de massa de 1.000.000 |
| `benchmark-bulk/main.go` | o executável de geração em massa (seção 9) |

Para rodar só um grupo:

```bash
go test ./tests/ -run 'TestTimestamp|TestTimeReading|TestRFC9562|TestImport' -v  # leitura de tempo
go test ./tests/ -run 'TestParse|TestJSON|TestSQL|TestNull|TestErrorTaxonomy' -v  # API de apoio
go test ./tests/ -run 'Entropy|Reader|Crypto|TextForms|EveryGenerationForm' -v    # fiação da entropia e níveis
go test ./tests/ -run 'TestBounds|TestRangeAt|TestGenerateAt|TestGolden' -v       # construção por instante
```

**Cobertura total é necessária, não suficiente.** A campanha de mutação
de 2026-09-13 encontrou 34 de 45 defeitos injetados passando por uma
suíte com 100% de cobertura de instruções. Os testes que os detectam
travam valores exatos em vez de validade ou ida e volta: de onde cada
campo livre tira os bits, o nível de cada forma de gerar pela assinatura
estatística dos bits livres, a aceitação exata das mutações de um byte
com um decodificador independente como oráculo, e os valores exatos no
teto de 48 bits. Ao acrescentar um teste, pergunte qual defeito de uma
linha ainda passaria ([08-casos-de-teste.md](08-casos-de-teste.md)).

**Nenhum teste de `./tests/` pode usar `t.Parallel`.**
`TestCryptoGeneratorReadsCryptoRandReader` troca `crypto/rand.Reader`
durante a construção do gerador, e um teste paralelo que lesse a mesma
variável seria corrida de dados. `TestSuiteHasNoParallelTests` confere os
arquivos de teste do pacote e falha se algum chamar `Parallel`
([10-decisoes.md](10-decisoes.md) §2).

**Independência de ordem e de repetição** ([08-casos-de-teste.md](08-casos-de-teste.md),
caso 9):

```bash
go test ./... -short -count 3 -shuffle on
```

## 2. Detector de corrida e travas de alocação

A biblioteca é concorrente por projeto. A suíte inteira deve passar sob o
detector de corrida sem registrar alertas:

```bash
go test ./... -race -short
```

As travas de alocação passam com e sem o detector: o gerador padrão lê do
gerador do runtime e não tem estado a recriar, então o detector não muda
a contagem. Ainda assim o passo dedicado, sem detector, é a medição de
referência, porque o detector altera o código gerado e pode contar
alocações que não existem em produção
([08-casos-de-teste.md](08-casos-de-teste.md), caso 16):

```bash
go test ./tests/ -short -run 'Allocations|SingleAllocation' -v
```

Uma falha aqui nunca se resolve elevando o limite tolerado.

## 3. Fuzzing

São cinco alvos, em dois grupos. Os três primeiros varrem **texto**, onde
o risco é leitura fora dos limites; os dois últimos varrem **tempo** nos
dois sentidos, onde o risco é saturação, estouro de sinal e bits
aleatórios lidos como instante.

```bash
go test ./tests/ -run '^$' -fuzz FuzzFromString -fuzztime 60s        # analisador estrito
go test ./tests/ -run '^$' -fuzz FuzzParse -fuzztime 60s             # analisador permissivo, quatro formatos
go test ./tests/ -run '^$' -fuzz FuzzNullUUIDJSON -fuzztime 60s      # leitor de JSON de NullUUID
go test ./tests/ -run '^$' -fuzz FuzzInstantArithmetic -fuzztime 60s # fronteiras e geração por instante
go test ./tests/ -run '^$' -fuzz FuzzTimeReading -fuzztime 60s       # leitura de tempo a partir de 16 bytes crus
```

Os oráculos de cada alvo estão no caso 2 de
[08-casos-de-teste.md](08-casos-de-teste.md). Os dois alvos de tempo
existem porque a suíte cobre essas bordas por tabela, com valores
escolhidos à mão, e tabela não varre faixa.

Quando uma campanha encontra uma entrada que quebra o alvo, o Go a grava
em `tests/testdata/fuzz/<Alvo>/<hash>` (diretório ignorado pelo Git) e a
partir daí ela passa a rodar como caso de teste comum, sem `-fuzz`:

```bash
go test ./tests/ -run 'FuzzParse/<hash>' -v
```

Corrija o defeito e transforme a entrada em caso de regressão permanente
com `f.Add(...)` na função de fuzzing correspondente em
`tests/fuzz_test.go`.

## 4. Exemplos executáveis

As funções `Example` de `example_test.go`, na raiz, reproduzem trechos da
documentação de uso e são compiladas e executadas pela suíte; as que
declaram `// Output:` usam só vetores fixos, nunca o relógio ou o gerador
pseudoaleatório. Elas aparecem na página do pacote em pkg.go.dev.

```bash
go test ./ -run Example -v     # executa
go test ./ -list Example       # lista
```

Os programas de exemplo da skill, em `skill/examples/`, também compilam
dentro do módulo (`go build ./...`) e são a forma de ver a biblioteca em
uso de ponta a ponta:

```bash
go run ./skill/examples/01-gerar
```

## 5. Linter

Além do `go vet`, o projeto roda o `golangci-lint` (versão 2) com a
configuração de `.golangci.yml`, a mesma usada pelo CI:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
golangci-lint run ./...
```

Linters ativos: `errcheck`, `govet`, `staticcheck`, `unused`,
`ineffassign`, `gosec`, `errorlint`, `revive` (regra de comentário em
identificador exportado) e `nolintlint`; `misspell` fica desligado porque
os comentários são em português. Toda marcação `//nolint` precisa nomear
o linter e trazer o motivo; o próprio `nolintlint` recusa marcações sem
motivo ou que não silenciam nada. A regra `G115` do `gosec` (truncamento
em conversão de inteiro) está desligada de propósito: empacotar campos em
bytes por deslocamento e truncamento é o que a biblioteca faz, e o
caminho quente não ganha máscaras para calar um aviso. Os testes que
comparam um sentinela com `==` de propósito (`ErrInvalidFormat` vindo de
`FromString`, `ErrNotV7`, o valor de pânico `ErrEntropySource`) carregam
`//nolint:errorlint` com esse motivo.

## 6. Cobertura

A suíte vive em `./tests/`, outro pacote, então o `-cover` padrão não
conta o pacote da raiz: é preciso `-coverpkg`.

```bash
go test ./tests/ ./ -short -coverpkg=github.com/patrickbrandao/go-loghub-uuidv7 -coverprofile=cover.out
go tool cover -func=cover.out            # por função, com o total na última linha
go tool cover -html=cover.out            # abre o relatório no navegador
```

A cobertura de instruções é **total**, sem exceção documentada: todo ramo
do pacote, inclusive a falha de leitura da fonte de entropia (provocada
por um leitor esgotado) e as recusas das leituras de tempo, é exercitado
pela suíte curta. O CI publica o relatório como artefato `cobertura` do
job `test` e falha se o total ficar abaixo de 100% (`COVERAGE_MIN` em
`ci.yml`). Se `go tool cover` apontar um bloco descoberto, é lacuna de
teste: acrescente o teste, nunca baixe o limiar.

## 7. Integração contínua

O arquivo `.github/workflows/ci.yml` define três jobs.

**`test`**, em Linux, a cada push, pull request e tag, na versão mínima
declarada em `go.mod` (1.22) e na versão estável mais recente:

```bash
gofmt -l .                      # só na versão estável
go vet ./...
go build ./...
GOOS=windows GOARCH=amd64 go build ./... && go vet ./...   # idem para darwin/arm64 e linux/arm64
golangci-lint run ./...         # só na versão estável
go test ./... -race -short
go test ./tests/ -short -run 'Allocations|SingleAllocation' -v   # travas de alocação, sem -race
go test ./tests/ -run '^$' -bench 'BenchmarkGenerateLevel|BenchmarkFromString|BenchmarkImportBinary|BenchmarkTimestampWithLevel3' -benchmem -benchtime 200000x
go test ./tests/ ./ -short -coverpkg=... -coverprofile=cover.out   # só na versão estável; falha abaixo de 100%
```

**`test-os`**, em `windows-latest` e `macos-latest` com a versão estável:
`go vet`, `go build` e `go test ./... -short`, sem `-race` (no Windows o
detector exige CGO e é bem mais lento; a corrida já é verificada em
Linux). Para poupar os runners mais caros, este job não roda a cada push
em `main`: só em pull request, tag, no agendamento semanal e sob demanda.
É ele que verifica a resolução do relógio e o comportamento específico de
cada sistema.

**`deep`**, toda segunda-feira e sob demanda pela aba Actions: a suíte
completa (com os testes de massa de 1.000.000) sob o detector de corrida
e 60 segundos de fuzzing em cada um dos cinco alvos. Os cinco rodam
sempre, mesmo que um falhe, e o job falha ao final se algum tiver
falhado.

Para disparar `test-os` e `deep` manualmente:

```bash
gh workflow run ci.yml
```

Regras de manutenção do fluxo: cada passo `actions/*` fica preso a uma
versão principal explícita cujo `action.yml` declare `using: node24`
(confira no repositório da ação antes de atualizar); `setup-go` passa
`cache: false` nos três jobs, porque sem dependências não há `go.sum` e
nada a cachear, e sem a opção toda execução imprime um aviso de
"Dependencies file is not found". Não silencie o aviso com um `go.sum`
vazio: seria um arquivo mentiroso, e a raiz fica mínima. `go list -m
all` na raiz deve imprimir só este módulo.

**Go 1.22 é imposto pelo CI, não pela ferramenta local.** Um `go` local
mais novo aceita API da biblioteca padrão posterior à 1.22 mesmo com
`go 1.22` no `go.mod`. Antes de depender de algo recente, teste com uma
ferramenta 1.22 (o CI compila e testa nessa versão).

Uma tag só deve ser publicada com o fluxo `test` verde no commit
correspondente; o procedimento está em [11-release.md](11-release.md).

### Reproduzir uma falha de fuzzing do CI

Quando uma campanha do job `deep` falha, a entrada que quebrou o alvo é
publicada como artefato `fuzz-corpus` do job (retenção de 30 dias), com a
mesma árvore que o Go usa localmente: `<Alvo>/<hash>`.

1. Baixe o artefato pela página da execução na aba Actions, ou pela CLI:

   ```bash
   gh run list --workflow ci.yml --limit 5          # identifique a execução
   gh run download <id> --name fuzz-corpus --dir tests/testdata/fuzz
   ```

2. Confirme que o arquivo ficou em `tests/testdata/fuzz/<Alvo>/<hash>` e
   rode só ele, como caso de teste comum:

   ```bash
   go test ./tests/ -run 'FuzzParse/<hash>' -v
   ```

3. Corrija o defeito e transforme a entrada em caso de regressão
   permanente com `f.Add(...)` em `tests/fuzz_test.go`; o diretório
   `tests/testdata/fuzz/` continua fora do Git.

---

## 8. Benchmarks

```bash
go test ./tests/ -run '^$' -bench Benchmark -benchmem
```

Mede nanossegundos por operação e alocações:

- `BenchmarkGenerateLevel1/2/3`: geração binária por nível.
- `BenchmarkGenerateStringLevel1/3`: geração com serialização em string.
- `BenchmarkGenerateLevel3Parallel`: throughput com várias goroutines.
- `BenchmarkGenerateV7` e `BenchmarkGenerateV7Level1/2/3`: os nomes por
  versão e por nível; cada um deve medir o mesmo que o nível que apelida,
  porque é embutido pelo compilador.
- `BenchmarkFromString`: análise estrita no formato canônico.
- `BenchmarkParse`: análise permissiva no formato canônico.
- `BenchmarkImportBinary`: leitura dos campos de tempo, com a
  verificação de versão.
- `BenchmarkImport`: o mesmo a partir da string canônica.
- `BenchmarkTimestampWithLevel3`: leitura do instante como `time.Time`.
- `BenchmarkMinAtLevel1/3`, `BenchmarkMaxAtLevel3` e
  `BenchmarkRangeAtLevel3`: fronteiras de tempo. Medem sobre um instante
  fixo, para não somar a leitura do relógio ao resultado.
- `BenchmarkGenerateAtLevel1/3` e `BenchmarkGenerateAtStringLevel3`:
  geração a partir de instante explícito. Saem mais baratas que
  `BenchmarkGenerateLevel1/3` porque não pagam a leitura do relógio.

Para conferir que uma alteração não tocou o caminho quente, compare os
benchmarks de geração antes e depois dela, com o coletor de lixo
desligado para reduzir o ruído:

```bash
GOGC=off GOMAXPROCS=4 go test ./tests/ -run '^$' \
  -bench 'BenchmarkGenerateLevel3' -benchtime 5000000x -count 6
```

Toda alteração em `uuid.go`, `conversion.go` ou `import.go` exige essa
medição antes e depois, inclusive uma feita só para satisfazer o linter.

## 9. Geração em massa de 1.000.000 por nível

Via teste com relatório:

```bash
go test ./tests/ -run 'TestMassOneMillion|TestMassConcurrent' -v
```

Gera 1.000.000 de UUIDs em cada cenário (binário e string, por nível) e
registra tempo total, ns por UUID e UUIDs/ms; além de uma variante
concorrente com 256 goroutines.

Via executável autônomo:

```bash
go run ./tests/benchmark-bulk            # 1.000.000 por cenário
go run ./tests/benchmark-bulk -n 5000000 # quantidade personalizada
```

Imprime uma tabela com tempo total, ns por UUID e UUIDs/ms para cada
nível, em binário e string. Os dois medidores descartam uma passagem de
aquecimento: sem ela o primeiro cenário medido paga sozinho o custo de
aquecer o cache de instruções e o escalonamento de frequência da CPU, e
como o Nível 1 é sempre o primeiro, aparecia mais lento do que era. A
taxa é calculada em nanossegundos, e não em `dur.Milliseconds()+1`, que
arredondava para baixo e ainda somava 1 ms.

---

## 10. Resultados de referência

### Host A: Apple M2, macOS, Go 1.27, GOMAXPROCS=8

Benchmark (`go test ./tests/ -run '^$' -bench Benchmark -benchmem -benchtime=2s`):

| Benchmark                  | ns/op | alloc/op | bytes/op |
| -------------------------- | ----: | -------: | -------: |
| `GenerateLevel1` (binário) | ~43,5 |        0 |        0 |
| `GenerateLevel2` (binário) | ~41,1 |        0 |        0 |
| `GenerateLevel3` (binário) | ~42,0 |        0 |        0 |
| `GenerateStringLevel1`     | ~67,9 |        1 |       48 |
| `GenerateStringLevel3`     | ~64,4 |        1 |       48 |
| `GenerateLevel3Parallel`   |  ~9,7 |        0 |        0 |
| `String` (UUID já pronto)  | ~26,3 |        1 |       48 |
| `AppendTo` (buffer reusado)| ~18,8 |        0 |        0 |
| `FromString`               | ~31,6 |        0 |        0 |

Geração em massa de 1.000.000 (`go run ./tests/benchmark-bulk`):

| Cenário           | Tempo total | ns/UUID | UUIDs/ms |
| ----------------- | ----------: | ------: | -------: |
| Nível 1 (binário) |      ~52 ms |     ~52 |  ~19.100 |
| Nível 2 (binário) |      ~41 ms |     ~41 |  ~24.500 |
| Nível 3 (binário) |      ~41 ms |     ~41 |  ~24.300 |
| Nível 1 (string)  |      ~68 ms |     ~68 |  ~14.600 |
| Nível 2 (string)  |      ~65 ms |     ~65 |  ~15.400 |
| Nível 3 (string)  |      ~69 ms |     ~69 |  ~14.400 |

Concorrente (1.000.000 de Nível 3 em 256 goroutines): ~7,8 ms,
~128.000 UUIDs/ms agregados.

**Por que o Nível 1 é o mais lento.** Não é viés de aquecimento: é real e
esperado. Nos níveis 2 e 3 o campo `rand_a` carrega os microssegundos,
então basta **uma** palavra de 64 bits do gerador pseudoaleatório; só o
Nível 1 (e os níveis desconhecidos) precisa de **duas**. A diferença é
exatamente o custo de um sorteio extra
([02-instante-e-entropia.md](02-instante-e-entropia.md) §3).

**Leitura de tempo e análise** (mesmo host, 2026-09-13, mediana de três
execuções de `-benchtime 2s -count 3`):

| Benchmark                  |  ns/op | alloc/op | bytes/op |
| -------------------------- | -----: | -------: | -------: |
| `ImportBinary`             |  ~2,21 |        0 |        0 |
| `TimestampWithLevel3`      |  ~6,29 |        0 |        0 |
| `Import` (texto)           |  ~33,7 |        0 |        0 |
| `FromString`               |  ~32,2 |        0 |        0 |
| `Parse` (canônico)         |  ~33,3 |        0 |        0 |

Na mesma sessão: `GenerateLevel1` ~45,1 ns, `GenerateLevel3` ~40,4 ns e
`GenerateLevel3Parallel` ~7,9 ns, todos sem alocação.

- **A verificação de versão custa ~0,6 ns.** `ImportBinary` passou de
  ~1,7 ns, na biblioteca de origem, onde lia qualquer UUID como v7, para
  ~2,2 ns com a recusa de [03-leitura-do-instante.md](03-leitura-do-instante.md)
  §2. A recusa devolve um erro pré-alocado e continua sem alocação.
  Reduzir a verificação a uma comparação só foi medido e saiu mais lento.
- **`TimestampWithLevel` custa ~4 ns a mais que `ImportBinary`** pela
  montagem do `time.Time` e pela soma das durações; é a leitura
  recomendada quando o destino é um instante.
- **`Import` é praticamente `FromString`**: a análise do texto domina, e
  a extração acrescenta os mesmos ~2 ns da forma binária.
- `Parse` custa ~1 ns a mais que `FromString` pelo despacho entre os
  quatro formatos; nenhum dos dois aloca.

### Host B: VM modesta, Intel Xeon a 2,80 GHz, Go 1.22, núcleo único

Piso de referência para máquinas lentas (números da revisão anterior,
medidos sem aquecimento; leia o Nível 1 com essa ressalva):

| Cenário           | Tempo total | ns/UUID | UUIDs/ms |
| ----------------- | ----------: | ------: | -------: |
| Nível 1 (binário) |      ~90 ms |     ~90 |  ~11.100 |
| Nível 2 (binário) |      ~88 ms |     ~88 |  ~11.330 |
| Nível 3 (binário) |      ~88 ms |     ~88 |  ~11.370 |
| Nível 1 (string)  |     ~165 ms |    ~165 |   ~6.060 |
| Nível 2 (string)  |     ~163 ms |    ~163 |   ~6.120 |
| Nível 3 (string)  |     ~165 ms |    ~165 |   ~6.045 |

A geração binária custa **dezenas de nanossegundos** e **zero
alocações**; passa de **11 mil UUIDs por milissegundo** por núcleo até na
VM lenta, e de **24 mil** em CPU moderna. A serialização em string
adiciona uma alocação de 48 bytes (a própria string).

**Onde está o teto.** Cerca de dois terços do custo de gerar um UUID é a
leitura do relógio (`time.Now()`), e esse custo é irredutível: a
precisão sub-milissegundo é a razão de ser da biblioteca. Micro-otimizar
a montagem dos 16 bytes não move o número.

### Troca da fonte de entropia do gerador padrão

Medição da substituição do `sync.Pool` de PRNGs PCG pelo gerador do
runtime (`math/rand/v2`, ChaCha8 por thread), feita na biblioteca de
origem em 2026-09-11, com o mesmo caminho de geração. Apple M2, macOS, Go
1.27, `GOGC=off GOMAXPROCS=4`, média de seis execuções de 5.000.000
iterações cada:

| Benchmark                  | Pool + PCG | ChaCha8 do runtime |      Δ |
| -------------------------- | ---------: | -----------------: | -----: |
| `GenerateLevel1`           |   45,55 ns |           44,85 ns |  -1,5% |
| `GenerateLevel2`           |   41,46 ns |           39,31 ns |  -5,2% |
| `GenerateLevel3`           |   42,05 ns |           39,80 ns |  -5,4% |
| `GenerateLevel3Parallel`   |   16,93 ns |           11,06 ns | -34,7% |
| `GenerateStringLevel1`     |   69,92 ns |           66,71 ns |  -4,6% |
| `GenerateStringLevel3`     |   65,73 ns |           67,06 ns |  +2,0% |

- **O ganho real está no paralelo**: -34,7% no `Level3Parallel`. O par
  `Get`/`Put` do pool era o gargalo sob concorrência; o gerador do
  runtime não tem nenhum.
- **Em série o ganho é modesto**, 2% a 6%, porque `time.Now()` domina o
  custo. O +2,0% de `GenerateStringLevel3` é ruído: destoa do
  `GenerateLevel3` (-5,4%) logo acima.

A decisão está em [10-decisoes.md](10-decisoes.md) §1.

## 11. Reproduzir

```bash
git clone https://github.com/patrickbrandao/go-loghub-uuidv7
cd go-loghub-uuidv7
go test ./tests/ -v
go run ./tests/benchmark-bulk
```
