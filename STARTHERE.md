# START HERE — Mapa da biblioteca go-loghub-uuidv7

Ponto de partida para entender o projeto inteiro: estrutura, API, layout
de bits, níveis e onde encontrar cada coisa.

---

## 1. O que é

Biblioteca Go dedicada ao **UUIDv7** (RFC 9562) com **três níveis** de
precisão temporal, nos dois sentidos: **gerar** o identificador com o
instante embutido em milissegundos, microssegundos ou nanossegundos, e
**converter** o UUIDv7 de volta em instante, na mesma precisão. Traz
também análise permissiva de texto, serialização em JSON, integração com
`database/sql` e as fronteiras de tempo para consulta por intervalo.

Só a versão 7 é tratada. As leituras de tempo recusam, com `ErrNotV7`,
qualquer UUID de outra versão. As versões 1, 2, 3, 4, 5, 6 e 8 estão na
biblioteca de origem, `go-loghub-uuid`.

Sem dependências externas; segura para concorrência; otimizada para alto
throughput. A geração não tem trava nem alocação.

---

## 2. Estrutura de arquivos

```
go-loghub-uuidv7/
│
├── go.mod                  # módulo: github.com/patrickbrandao/go-loghub-uuidv7
├── .github/workflows/ci.yml # integração contínua (lint, vet, build, testes, cobertura, Windows/macOS, fuzzing semanal)
├── .golangci.yml           # configuração do linter (golangci-lint v2), a mesma do CI
├── .gitignore
│
│   # PRODUÇÃO — núcleo (caminho quente, sem trava, sem alocação)
├── uuid.go                 # tipos, Generator, geração por nível
├── version7.go             # GenerateV7 e GenerateV7Level1/2/3: a geração pelo nome, por versão e por nível
├── conversion.go           # String()/FromString + apelidos de conversão
├── import.go               # Time, ErrNotV7, Import, ImportBinary
│
│   # PRODUÇÃO — API de apoio
├── inspect.go              # Timestamp e TimestampWithLevel: o instante como time.Time
├── construct.go            # GenerateAt: UUIDv7 de um instante informado, e o empacotamento compartilhado
├── bounds.go               # MinAt, MaxAt e RangeAt: fronteiras para consulta por intervalo
├── parse.go                # Parse permissivo, Validate, FromBytes, MustParse, Must
├── values.go               # Nil, IsZero, IsValid, Compare, URN, Bytes
├── encoding.go             # AppendTo/AppendBinary, MarshalText/Binary e as leituras
├── sql.go                  # Scan, Value, NullUUID, BinaryUUID e NullBinaryUUID
├── entropy.go              # NewGeneratorWithReader e NewCryptoGenerator
├── instant_internal_test.go # teste interno da decomposição do instante
├── example_test.go         # funções Example para o pkg.go.dev (pacote externo uuidv7_test)
│
├── README.md               # descrição rápida + uso rápido
├── STARTHERE.md            # este mapa
├── CHANGELOG.md            # histórico de mudanças e decisões por versão
├── LICENSE                 # MIT
├── SECURITY.md             # como relatar vulnerabilidade e o modelo de ameaça documentado
├── CONTRIBUTING.md         # convenções e verificação local para quem contribui
├── CLAUDE.md               # instruções de manutenção (ferramental)
│
├── docs/                   # especificação, um arquivo por assunto; comece por INDEX.md
│   ├── INDEX.md            # índice, caminhos de leitura, mapa da API
│   ├── 01-escopo-e-layout.md
│   ├── 02-instante-e-entropia.md
│   ├── 03-leitura-do-instante.md
│   ├── 04-construcao-por-instante.md
│   ├── 05-conversao-e-analise.md
│   ├── 06-serializacao-e-banco.md
│   ├── 07-armadilhas.md
│   ├── 08-casos-de-teste.md
│   ├── 09-testes-e-benchmark.md
│   ├── 10-decisoes.md      # decisões firmadas: leia antes de propor mudança
│   └── 11-release.md       # procedimento de publicação de versão
│
├── skill/                  # como usar a biblioteca, para agentes de IA e pessoas
│   ├── SKILL.md            # instruções, regras e receitas (formato Agent Skills)
│   ├── references/         # api.md, consultas-e-banco.md, erros.md
│   └── examples/           # nove programas executáveis: go run ./skill/examples/01-gerar
│
└── tests/                  # tudo que NÃO vai para produção
    ├── doc.go
    ├── generation_test.go      # testes funcionais
    ├── timestamp_test.go       # leituras de tempo, vetor da RFC 9562 e recusa de outras versões
    ├── api_test.go             # análise, serialização, SQL, erros e entropia
    ├── parsing_test.go         # FromString/String e o oráculo exato das mutações de um byte
    ├── layout_test.go          # layout de bits com entropia determinística
    ├── entropy_test.go         # fiação da entropia: palavras, leitor, crypto/rand e independência
    ├── import_test.go          # extração das propriedades de tempo
    ├── ordering_test.go        # ordenação, unicidade e concorrência
    ├── robustness_test.go      # bordas do Generator, consumo de entropia e nível de cada forma de gerar
    ├── alloc_test.go           # travas de alocação
    ├── bounds_test.go          # fronteiras de tempo: MinAt, MaxAt e RangeAt
    ├── construct_test.go       # geração por instante explícito: GenerateAt
    ├── binary_sql_test.go      # BinaryUUID e NullBinaryUUID
    ├── binary_serialization_test.go # serialização JSON, texto e binário dos tipos binários
    ├── golden_test.go          # vetores dourados da extensão multinível
    ├── fuzz_test.go            # FuzzFromString, FuzzParse, FuzzNullUUIDJSON,
    │                            # FuzzInstantArithmetic e FuzzTimeReading
    ├── benchmark_test.go       # benchmarks + massa de 1.000.000
    └── benchmark-bulk/
        └── main.go             # executável: go run ./tests/benchmark-bulk
```

A **raiz** contém apenas o necessário para usar a biblioteca em produção
(os arquivos `.go`, o `go.mod`, README/STARTHERE/CHANGELOG/LICENSE) mais
os arquivos que o GitHub ou o ferramental exigem nesse local: `SECURITY.md`
e `CONTRIBUTING.md` (abas de segurança e de contribuição), `CLAUDE.md`
(instruções de manutenção), `.golangci.yml` (configuração do linter),
`.gitignore` e a integração contínua em `.github/`. Dois arquivos de
teste também ficam na raiz por necessidade: `instant_internal_test.go`,
que exercita a decomposição interna do instante, e `example_test.go`,
cujas funções `Example` só aparecem na documentação do pacote se
estiverem no mesmo diretório. Documentação, especificação, skill e testes ficam
em pastas próprias.

---

## 3. API pública (pacote `uuidv7`)

### 3.1 Geração

**Tipos**
- `Level` — `Level1`, `Level2`, `Level3`.
- `UUID` — `[16]byte`, binário de 128 bits.
- `Generator` — objeto criado no boot, seguro para concorrência.

**Construtores**
- `NewGenerator() *Generator` — padrão rápido (ChaCha8 do runtime, sem lock).
- `NewGeneratorWith(source func() uint64) *Generator` — entropia
  personalizada; `source` precisa ser segura para concorrência e entra em
  pânico se for `nil`.
- `NewGeneratorWithReader(r io.Reader) *Generator` — entropia a partir de
  um `io.Reader` seguro para concorrência.
- `NewCryptoGenerator() *Generator` — entropia de `crypto/rand`.

> O gerador padrão usa o ChaCha8 do runtime do Go, que resiste a
> predição, mas a recomendação para segredos continua sendo
> `crypto/rand`, e o UUIDv7 expõe o instante de criação de qualquer
> forma. Ver a seção "Aviso de segurança" do [README.md](README.md).

**Pelo relógio**
- `(*Generator) Generate(Level) UUID`
- `(*Generator) GenerateString(Level) string`
- `Generate(Level) UUID` — atalho de pacote (gerador padrão interno)
- `GenerateString(Level) string` — atalho de pacote
- `(*Generator) GenerateV7() UUID` e `GenerateV7() UUID` — o UUIDv7
  padrão da RFC 9562 pelo nome da versão; exatamente `Generate(Level1)`,
  com precisão de milissegundo. Não recebe nível nem tem forma em texto.
- `(*Generator) GenerateV7Level1() UUID`, `GenerateV7Level2() UUID`,
  `GenerateV7Level3() UUID` e as funções de pacote de mesmo nome — os
  três níveis pelo nome, sem o argumento: exatamente `Generate` com o
  nível correspondente. `GenerateV7Level1` é o mesmo que `GenerateV7`.
  Sem forma em texto: use `GenerateString(nível)` ou `String()`.

**A partir de um instante**
- `GenerateAt(Level, time.Time) UUID` — um UUIDv7 daquele instante, com
  os bits livres **sorteados**. Também como
  `(*Generator) GenerateAt`, para valer a entropia configurada.
- `GenerateAtString(Level, time.Time) string` — o mesmo, em texto.
- `MinAt(Level, time.Time) UUID` — o menor UUIDv7 gerável naquele
  instante e naquele nível; bits livres de entropia em zero.
- `MaxAt(Level, time.Time) UUID` — o maior; bits livres em um.
- `RangeAt(Level, from, to time.Time) (lo, hi UUID)` — o par de um
  intervalo **semiaberto** `[from, to)`, pronto para
  `WHERE id >= lo AND id < hi`.

> `GenerateAt` é gerador: duas chamadas com o mesmo instante devolvem
> valores diferentes. `MinAt` e `MaxAt` são determinísticas e servem de
> fronteira, não de identificador.
>
> Todas preservam versão 7 e variante RFC. A fronteira só vale para
> UUIDs do **mesmo nível**: os bits abaixo do milissegundo significam
> coisas diferentes em cada um. Ver
> [docs/04-construcao-por-instante.md](docs/04-construcao-por-instante.md) §3.

### 3.2 Leitura do instante

Todas recusam o UUID que não é de versão 7 com a variante da RFC
([docs/03-leitura-do-instante.md](docs/03-leitura-do-instante.md) §2).

- `(UUID) TimestampWithLevel(Level) (time.Time, bool)` — o instante em
  UTC com a precisão do nível; descarta os campos abaixo do milissegundo
  fora de 0 a 999. Falso para UUID que não é v7.
- `(UUID) Timestamp() (time.Time, bool)` — o instante com precisão de
  milissegundo; o mesmo que `TimestampWithLevel(Level1)`.
- `ImportBinary(UUID) (Time, error)` — os quatro campos crus, sem julgar
  o nível; `ErrNotV7` para UUID que não é v7.
- `Import(string) (Time, error)` — o mesmo a partir da string canônica;
  `ErrInvalidFormat` para texto recusado, `ErrNotV7` para outra versão.
- `Time` — `Seconds`, `Milliseconds`, `Microseconds`, `Nanoseconds`.
- `(UUID) IsValid() bool` — versão 7 com a variante da RFC: a condição
  que as leituras exigem.

### 3.3 Conversão e análise de texto

- `(UUID) String() string`
- `(UUID) AppendTo(dst []byte) []byte` — escreve os 36 bytes no buffer do
  chamador; sem alocação quando há capacidade.
- `FromString(string) (UUID, error)` — só a forma canônica.
- `BinaryToString(UUID) string` — apelido de `String()`
- `StringToBinary(string) (UUID, error)` — apelido de `FromString`
- `Parse(string) (UUID, error)` — aceita a forma canônica, entre chaves,
  com prefixo `urn:uuid:` e hexadecimal cru de 32 dígitos.
- `ParseBytes([]byte) (UUID, error)` — o mesmo, sem alocar.
- `MustParse(string) UUID`, `Must(UUID, error) UUID`.
- `Validate(string) error`.
- `FromBytes([]byte) (UUID, error)` — 16 bytes crus.

A análise confere **só a forma**: um UUID bem formado de outra versão é
aceito, e a recusa fica para a leitura de tempo.

### 3.4 Serialização e banco de dados

- `(UUID) MarshalText`, `(*UUID) UnmarshalText`
- `(UUID) AppendText(dst []byte) ([]byte, error)` — `encoding.TextAppender`
  do Go 1.24; o erro é sempre nulo.
- `(UUID) MarshalBinary`, `(*UUID) UnmarshalBinary`
- `(UUID) AppendBinary(dst []byte) ([]byte, error)` —
  `encoding.BinaryAppender` do Go 1.24, o par binário de `AppendText`;
  mesmo conteúdo de `MarshalBinary`, sem alocar quando `dst` tem
  capacidade.
- `(*UUID) Scan(any) error`, `(UUID) Value() (driver.Value, error)`
- `NullUUID` — coluna que aceita `NULL`, com `Scan`, `Value` e as
  serializações em JSON, texto e binário.
- `BinaryUUID` — mesmo UUID, gravado como 16 bytes crus em vez da string
  canônica. Para `BINARY(16)` e `BLOB`, onde não há tipo nativo de UUID.
  A conversão é no ponto da consulta: `uuidv7.BinaryUUID(u)`. Tem
  `MarshalText`/`UnmarshalText` e `MarshalBinary`/`UnmarshalBinary`,
  delegando para `UUID`, para que `encoding/json` grave a string
  canônica e não um vetor de 16 números.
- `NullBinaryUUID` — o mesmo, para coluna que também aceita `NULL`, com
  `Scan`, `Value` e as serializações em JSON, texto e binário,
  espelhando `NullUUID`.

> `Value` de `UUID` grava texto e não vai mudar: misturar os dois
> formatos na mesma coluna faria as linhas antigas sumirem das consultas.
> Cuidado ainda com a distinção entre coluna nula e UUID nulo, que viram
> a mesma linha se forem confundidos.

### 3.5 Inspeção e valores

- `(UUID) Version() byte` — o nibble de versão, lido como está.
- `(UUID) Variant() byte` — os 2 bits de variante; 2 (binário `10`) na RFC.
- `(UUID) IsZero() bool`
- `(UUID) Bytes() []byte` — cópia dos 16 bytes.
- `(UUID) Compare(UUID) int` — serve direto a `slices.SortFunc`.
- `(UUID) URN() string`
- `Nil` — o UUID nulo; não é UUIDv7.

### 3.6 Erros

- `ErrInvalidFormat` — string de UUID em formato inválido.
- `ErrInvalidLength` — comprimento incompatível; embrulha o anterior.
- `ErrInvalidBrackets` — forma entre chaves malformada; idem.
- `ErrNotV7` — UUID bem formado que não é de versão 7 com a variante da
  RFC; família própria, **não** embrulha `ErrInvalidFormat`.
- `ErrInvalidScanType` — tipo não suportado em `Scan`.
- `ErrEntropySource` — falha ao ler da fonte de entropia do chamador;
  vem como valor de pânico.

Os erros de formato são reconhecíveis por
`errors.Is(err, ErrInvalidFormat)`; `FromString` devolve exatamente o
sentinela, também para comparação com `==`.

## 4. Layout de bits (128 bits, big-endian)

```
byte:  0    1    2    3    4    5    6    7    8    9   10   11   12   13   14   15
      [-------- unix_ts_ms (48) --------][V|aa][ aa ][v|bb][----------- rand_b -----------]
                                          7  ^             10 ^
                                          versão            variante
```

- bytes 0..5 → `unix_ts_ms` (milissegundos desde epoch).
- byte 6 → nibble alto `0x7` (versão); nibble baixo = topo de `rand_a`.
- byte 7 → resto de `rand_a` (12 bits no total).
- byte 8 → 2 bits altos `10` (variante); 6 bits baixos = topo de `rand_b`.
- bytes 9..15 → resto de `rand_b` (62 bits no total).

**Onde cada nível grava o tempo sub-ms:**

| Campo            | Nível 1   | Nível 2        | Nível 3                          |
|------------------|-----------|----------------|----------------------------------|
| `rand_a` (12 b)  | aleatório | microssegundos | microssegundos                   |
| `rand_b` topo 10 | aleatório | aleatório      | nanossegundos                    |
| `rand_b` resto   | aleatório | aleatório      | aleatório (52 bits)              |

Microssegundos e nanossegundos são sempre 0..999 (fração do nível
acima). Como ocupam as posições logo após os milissegundos, a ordenação
por string continua cronológica.

---

## 5. Caminhos de leitura recomendados

- **Só quero gerar e ler o instante**: [skill/SKILL.md](skill/SKILL.md)
- **Quero usar tudo, com exemplos executáveis**:
  [skill/SKILL.md](skill/SKILL.md) e `skill/examples/`
- **Quero medir desempenho**:
  [docs/09-testes-e-benchmark.md](docs/09-testes-e-benchmark.md)
- **Quero reimplementar em outra linguagem**:
  [docs/INDEX.md](docs/INDEX.md), arquivos 01 a 08 na ordem
- **Quero propor uma mudança de projeto, ou vou auditar a biblioteca**:
  [docs/10-decisoes.md](docs/10-decisoes.md), o registro de decisões
  firmadas — o que já foi decidido, por quê, e o que justificaria rever.
- **Quero ler o código**: comece por `uuid.go` (geração), depois
  `import.go` e `inspect.go` (leitura) e `conversion.go`.
- **Vou publicar uma versão**: [docs/11-release.md](docs/11-release.md)
- **Vou contribuir ou relatar um problema de segurança**:
  [CONTRIBUTING.md](CONTRIBUTING.md) e [SECURITY.md](SECURITY.md)

---

## 6. Comandos úteis

```bash
go get github.com/patrickbrandao/go-loghub-uuidv7   # instalar
go build ./...                                       # compilar
go vet ./...                                          # análise estática
go test ./tests/ -v                                   # testes
go test ./tests/ -run '^$' -bench Benchmark -benchmem   # benchmarks
go run ./tests/benchmark-bulk                         # 1.000.000 por nível
golangci-lint run ./...                               # linter (configuração em .golangci.yml)
go test ./ -run Example -v                            # exemplos executáveis
```

---

## 7. Desempenho de referência

Em VM modesta (Xeon 2.80 GHz): geração binária ~85–98 ns/UUID com **zero
alocações** e mais de **11 mil UUIDs/ms** por núcleo; string ~165 ns com
1 alocação de 48 bytes. Num Apple M2, a geração fica em ~42 ns e a
leitura dos campos de tempo em ~2,3 ns, também sem alocação. Detalhes e
tabelas em [docs/09-testes-e-benchmark.md](docs/09-testes-e-benchmark.md) §10.
