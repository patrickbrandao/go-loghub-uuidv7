# Testes e Benchmark

Como rodar a suíte de testes, os benchmarks e a geração em massa de até
1.000.000 de UUIDv7 por nível, medindo o tempo consumido.

Todos os arquivos de teste ficam na pasta `tests/` (fora da raiz de
produção) e importam a biblioteca pelo seu caminho público, como um
consumidor externo faria.

---

## 1. Testes funcionais

```bash
go test ./tests/ -v
```

Cobrem versão/variante, round-trip de conversão, faixas de micro/nano,
coerência da importação, rejeição de strings inválidas, ordenação e
concorrência.

Cobrem também a leitura de tempo nos dois sentidos: o vetor de UUIDv7 da
RFC 9562 (apêndice A.6), o descarte por faixa da leitura por nível, a
recusa das outras versões nas 64 combinações de versão e variante e nos
exemplos da RFC das versões 1, 3, 4, 5, 6 e 8, as fronteiras e a geração
por instante, a análise permissiva de texto nos quatro formatos, a
serialização em JSON e binário e a integração com `database/sql`.

Para rodar apenas um desses grupos:

```bash
go test ./tests/ -run 'TestTimestamp|TestTimeReading|TestRFC9562|TestImport' -v  # leitura de tempo
go test ./tests/ -run 'TestParse|TestJSON|TestSQL|TestNull|TestErrorTaxonomy' -v  # API de apoio
```

Modo rápido (pula os testes de massa de 1 milhão):

```bash
go test ./tests/ -short -v
```

### Detector de corrida (race detector)

A biblioteca é concorrente por projeto. A suíte inteira deve passar sob
o detector de corrida sem registrar alertas:

```bash
go test ./... -race
```

As travas de alocação passam com e sem o detector. O gerador padrão lê do
gerador do runtime e não tem estado a recriar, então o detector não muda
a contagem; ainda assim o passo dedicado, sem detector, é a medição de
referência:

```bash
go test ./tests/ -short -run 'Allocations|SingleAllocation' -v
```

### Fuzzing

São cinco alvos, em dois grupos. Os três primeiros varrem **texto**,
onde o risco é leitura fora dos limites; os dois últimos varrem **tempo**
nos dois sentidos, onde o risco é saturação, estouro de sinal e bits
aleatórios lidos como instante.

```bash
# Fuzz do analisador estrito
go test ./tests/ -run '^$' -fuzz FuzzFromString -fuzztime 60s

# Fuzz do analisador permissivo (quatro formatos)
go test ./tests/ -run '^$' -fuzz FuzzParse -fuzztime 60s

# Fuzz do leitor de JSON de NullUUID (concordância com o tipo UUID)
go test ./tests/ -run '^$' -fuzz FuzzNullUUIDJSON -fuzztime 60s

# Fuzz das fronteiras e da geração por instante (seções 3.5 e 3.6)
go test ./tests/ -run '^$' -fuzz FuzzInstantArithmetic -fuzztime 60s

# Fuzz da leitura de tempo a partir de 16 bytes crus (seções 4 e 7)
go test ./tests/ -run '^$' -fuzz FuzzTimeReading -fuzztime 60s
```

Os dois alvos de tempo existem porque a suíte cobre essas bordas por
tabela, com valores escolhidos à mão, e tabela não varre faixa.
`FuzzInstantArithmetic` recebe dois instantes e exige versão 7, variante
RFC, fronteira inferior nunca acima da superior, o valor gerado dentro
das fronteiras e monotonicidade quando o instante não regride, nos três
níveis mais um nível desconhecido. `FuzzTimeReading` recebe 16 bytes
arbitrários e exige que as quatro leituras concordem com `IsValid` ao
aceitar ou recusar, que a extração cega devolva o carimbo exato e os
campos crus na faixa dos bits, que a leitura por nível some os campos
abaixo do milissegundo exatamente quando cabem em 0 a 999, e que o
instante lido no Nível 3 regere por `MinAt` os mesmos campos de tempo.

Quando uma campanha encontra uma entrada que quebra o alvo, o Go a grava
em `tests/testdata/fuzz/<Alvo>/<hash>` (diretório ignorado pelo Git) e a
partir daí ela passa a rodar como caso de teste comum, sem `-fuzz`:

```bash
go test ./tests/ -run 'FuzzParse/<hash>' -v
```

### Exemplos executáveis

As funções `Example` de `example_test.go`, na raiz, reproduzem trechos
da documentação de uso e são compiladas e executadas pela suíte; as que
declaram `// Output:` usam só vetores fixos. Elas aparecem na página do
pacote em pkg.go.dev.

```bash
go test ./ -run Example -v     # executa
go test ./ -list Example       # lista
```

### Linter

Além do `go vet`, o projeto roda o `golangci-lint` (versão 2) com a
configuração de `.golangci.yml`, a mesma usada pelo CI. Instalação e
uso:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
golangci-lint run ./...
```

Toda marcação `//nolint` precisa nomear o linter e trazer o motivo; o
próprio linter (`nolintlint`) recusa marcações sem motivo ou que não
silenciam nada. A regra `G115` do `gosec` (truncamento em conversão de
inteiro) está desligada de propósito: empacotar campos em bytes por
deslocamento e truncamento é o que a biblioteca faz, e o caminho quente
não ganha máscaras para calar um aviso.

### Cobertura

A suíte vive em `./tests/`, outro pacote, então o `-cover` padrão não
conta o pacote da raiz: é preciso `-coverpkg`.

```bash
go test ./tests/ ./ -short -coverpkg=github.com/patrickbrandao/go-loghub-uuidv7 -coverprofile=cover.out
go tool cover -func=cover.out            # por função, com o total na última linha
go tool cover -html=cover.out            # abre o relatório no navegador
```

O CI publica o relatório (`cover.out` e `cover.html`) como artefato
`cobertura` do job `test` e falha se o total ficar abaixo de 100%. A
cobertura de instruções é **total**, sem exceção documentada: todo ramo
do pacote, inclusive a falha de leitura da fonte de entropia (provocada
por um leitor esgotado) e as recusas das leituras de tempo, é exercitado
pela suíte curta. Se `go tool cover` apontar um bloco descoberto, é
lacuna de teste.

### Integração contínua

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
go test ./tests/ -run '^$' -bench 'BenchmarkGenerateLevel|BenchmarkFromString' -benchmem -benchtime 200000x
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

Uma tag só deve ser publicada com o fluxo `test` verde no commit
correspondente; o procedimento está em [RELEASE.md](RELEASE.md).

#### Reproduzir uma falha de fuzzing do CI

Quando uma campanha do job `deep` falha, a entrada que quebrou o alvo é
publicada como artefato `fuzz-corpus` do job (retenção de 30 dias), com a
mesma árvore que o Go usa localmente: `<Alvo>/<hash>`. Para reproduzir:

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
   permanente com `f.Add(...)` na função de fuzzing correspondente em
   `tests/fuzz_test.go`; o diretório `tests/testdata/fuzz/` continua fora
   do Git.

---

## 2. Benchmarks (estilo `go test -bench`)

```bash
go test ./tests/ -run '^$' -bench Benchmark -benchmem
```

Mede nanossegundos por operação e alocações:

- `BenchmarkGenerateLevel1/2/3` — geração binária por nível.
- `BenchmarkGenerateStringLevel1/3` — geração com serialização em string.
- `BenchmarkGenerateLevel3Parallel` — throughput com várias goroutines.
- `BenchmarkFromString` — análise estrita no formato canônico.
- `BenchmarkImportBinary` — leitura dos campos de tempo, com a
  verificação de versão.
- `BenchmarkImport` — o mesmo a partir da string canônica.
- `BenchmarkTimestampWithLevel3` — leitura do instante como `time.Time`.
- `BenchmarkGenerateV7` — o nome por versão do UUIDv7; deve medir o mesmo
  que `BenchmarkGenerateLevel1`, porque é um apelido de `Generate(Level1)`
  embutido pelo compilador.
- `BenchmarkGenerateV7Level1/2/3` — os nomes por nível; cada um deve medir
  o mesmo que `BenchmarkGenerateLevel1/2/3`, pelo mesmo motivo.
- `BenchmarkParse` — análise permissiva no formato canônico.
- `BenchmarkMinAtLevel1/3`, `BenchmarkMaxAtLevel3` e
  `BenchmarkRangeAtLevel3` — fronteiras de tempo para consulta por
  intervalo. Medem sobre um instante fixo, para não somar a leitura do
  relógio ao resultado.
- `BenchmarkGenerateAtLevel1/3` e `BenchmarkGenerateAtStringLevel3` —
  geração a partir de instante explícito. Saem mais baratas que
  `BenchmarkGenerateLevel1/3` porque não pagam a leitura do relógio.

Os números de referência de todos eles estão na seção 4.

Para conferir que uma alteração não tocou o caminho quente, compare os
benchmarks de geração antes e depois dela, com o coletor de lixo
desligado para reduzir o ruído:

```bash
GOGC=off GOMAXPROCS=4 go test ./tests/ -run '^$' \
  -bench 'BenchmarkGenerateLevel3' -benchtime 5000000x -count 6
```

---

## 3. Geração em massa de 1.000.000 por nível

### Via teste com relatório

```bash
go test ./tests/ -run 'TestMassOneMillion|TestMassConcurrent' -v
```

Gera 1.000.000 de UUIDs em cada cenário (binário e string, por nível) e
registra tempo total, ns por UUID e UUIDs/ms; além de uma variante
concorrente com 256 goroutines.

### Via executável autônomo

```bash
go run ./tests/benchmark-bulk            # 1.000.000 por cenário
go run ./tests/benchmark-bulk -n 5000000 # quantidade personalizada
```

Imprime uma tabela com tempo total, ns por UUID e UUIDs/ms para cada
nível, em binário e string.

---

## 4. Resultados de referência

> **Sobre as tabelas anteriores.** Até a revisão de 2026-08-27, tanto
> `TestMassOneMillion` quanto `benchmark-bulk` mediam os cenários em
> sequência **sem passagem de aquecimento**. O primeiro cenário medido
> pagava sozinho o custo de aquecer cache de instruções e o escalonamento
> de frequência da CPU — e como o Nível 1 é sempre o primeiro, aparecia
> mais lento do que realmente era. Ambos os
> medidores passaram a descartar uma passagem de aquecimento, e o cálculo
> da taxa deixou de usar `dur.Milliseconds()+1` (que arredondava para
> baixo e ainda somava 1 ms) em favor de nanossegundos. Os números abaixo
> foram remedidos com o código corrigido.

### Host A — Apple M2, macOS, Go 1.27, GOMAXPROCS=8

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

**Por que o Nível 1 agora é o mais lento.** Não é mais viés de
aquecimento: é real e esperado. Nos níveis 2 e 3 o campo `rand_a` carrega
os microssegundos, então basta **uma** palavra de 64 bits do gerador
pseudoaleatório; só o Nível 1 (e os níveis desconhecidos, que se
comportam como ele) precisa de **duas**. A diferença é exatamente o custo
de um sorteio extra.

#### Leitura de tempo e análise (Host A, 2026-09-13)

Medido no mesmo host, com Go 1.27.0 e `GOMAXPROCS=8`, por
`go test ./tests/ -run '^$' -bench 'BenchmarkGenerateLevel[13]$|BenchmarkGenerateLevel3Parallel$|BenchmarkFromString$|BenchmarkParse$|BenchmarkImportBinary$|BenchmarkImport$|BenchmarkTimestampWithLevel3$' -benchmem -benchtime 2s -count 3`;
cada linha é a mediana das três execuções.

| Benchmark                  |  ns/op | alloc/op | bytes/op |
| -------------------------- | -----: | -------: | -------: |
| `ImportBinary`             |  ~2,21 |        0 |        0 |
| `TimestampWithLevel3`      |  ~6,29 |        0 |        0 |
| `Import` (texto)           |  ~33,7 |        0 |        0 |
| `FromString`               |  ~32,2 |        0 |        0 |
| `Parse` (canônico)         |  ~33,3 |        0 |        0 |

Para comparar com a geração na mesma sessão: `GenerateLevel1` ~45,1 ns,
`GenerateLevel3` ~40,4 ns e `GenerateLevel3Parallel` ~7,9 ns, todos sem
alocação.

Leitura dos números:

- **A verificação de versão custa ~0,6 ns.** `ImportBinary` passou de
  ~1,7 ns, na biblioteca de origem, onde lia qualquer UUID como v7, para
  ~2,2 ns com a recusa de `docs/SPEC.md` seção 4. A recusa devolve um
  erro pré-alocado e continua sem alocação. Reduzir a verificação a uma
  comparação só foi medido e saiu mais lento.
- **`TimestampWithLevel` custa ~4 ns a mais que `ImportBinary`** pela
  montagem do `time.Time` e pela soma das durações; é a leitura
  recomendada quando o destino é um instante.
- **`Import` é praticamente `FromString`**: a análise do texto domina, e
  a extração acrescenta os mesmos ~2 ns da forma binária.
- `Parse` custa ~1 ns a mais que `FromString` pelo despacho entre os
  quatro formatos; nenhum dos dois aloca.

### Host B — VM modesta, Intel Xeon @ 2.80 GHz, Go 1.22, núcleo único

Piso de referência para máquinas lentas (números da revisão anterior,
medidos sem aquecimento — leia o Nível 1 com essa ressalva):

| Cenário           | Tempo total | ns/UUID | UUIDs/ms |
| ----------------- | ----------: | ------: | -------: |
| Nível 1 (binário) |      ~90 ms |     ~90 |  ~11.100 |
| Nível 2 (binário) |      ~88 ms |     ~88 |  ~11.330 |
| Nível 3 (binário) |      ~88 ms |     ~88 |  ~11.370 |
| Nível 1 (string)  |     ~165 ms |    ~165 |   ~6.060 |
| Nível 2 (string)  |     ~163 ms |    ~163 |   ~6.120 |
| Nível 3 (string)  |     ~165 ms |    ~165 |   ~6.045 |

Leitura dos números: a geração binária custa **dezenas de
nanossegundos** e **zero alocações**; passa de **11 mil UUIDs por
milissegundo** por núcleo até na VM lenta, e de **24 mil** em CPU
moderna. A serialização em string adiciona uma alocação de 48 bytes (a
própria string). A meta de "milhares de UUIDs por milissegundo" é
atingida com folga.

**Onde está o teto.** Cerca de dois terços do custo de gerar um UUID é a
leitura do relógio (`time.Now()`), e esse custo é irredutível — a
precisão sub-milissegundo é a razão de ser da biblioteca. Micro-otimizar
a montagem dos 16 bytes não move o número.

### Troca da fonte de entropia do gerador padrão

Medição da substituição do `sync.Pool` de PRNGs PCG pelo gerador do
runtime (`math/rand/v2`, ChaCha8 por thread), feita na biblioteca de
origem em 2026-09-11, com o mesmo caminho de geração. Apple M2, macOS, Go 1.27,
`GOGC=off GOMAXPROCS=4`, média de seis execuções de 5.000.000 iterações
cada:

```bash
GOGC=off GOMAXPROCS=4 go test ./tests/ -run '^$' -bench 'BenchmarkGenerate' \
  -benchmem -benchtime 5000000x -count 6
```

| Benchmark                  | Pool + PCG | ChaCha8 do runtime |      Δ |
| -------------------------- | ---------: | -----------------: | -----: |
| `GenerateLevel1`           |   45,55 ns |           44,85 ns |  -1,5% |
| `GenerateLevel2`           |   41,46 ns |           39,31 ns |  -5,2% |
| `GenerateLevel3`           |   42,05 ns |           39,80 ns |  -5,4% |
| `GenerateLevel3Parallel`   |   16,93 ns |           11,06 ns | -34,7% |
| `GenerateStringLevel1`     |   69,92 ns |           66,71 ns |  -4,6% |
| `GenerateStringLevel3`     |   65,73 ns |           67,06 ns |  +2,0% |

Leitura dos números:

- **O ganho real está no paralelo**: -34,7% no `Level3Parallel`. O par
  `Get`/`Put` do pool era o gargalo sob concorrência; o gerador do
  runtime não tem nenhum.
- **Em série o ganho é modesto**, 2% a 6%, porque `time.Now()` domina o
  custo. O +2,0% de `GenerateStringLevel3` é ruído: destoa do
  `GenerateLevel3` (-5,4%) logo acima.

---

## 5. Reproduzir

```bash
git clone https://github.com/patrickbrandao/go-loghub-uuidv7
cd go-loghub-uuidv7
go test ./tests/ -v
go run ./tests/benchmark-bulk
```
