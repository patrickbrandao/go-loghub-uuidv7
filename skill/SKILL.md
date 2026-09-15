---
name: go-loghub-uuidv7
description: >
  Use a biblioteca Go go-loghub-uuidv7 (pacote uuidv7, sem dependências,
  Go 1.22+) para gerar UUIDv7 (RFC 9562) com precisão de milissegundo,
  microssegundo ou nanossegundo, ler o instante de volta de um UUIDv7,
  validar e analisar UUIDs em texto, consultar registros por intervalo de
  tempo usando a própria chave primária (MinAt, MaxAt, RangeAt), gerar
  chaves para instantes passados (GenerateAt) e integrar com JSON e
  database/sql, inclusive colunas BINARY(16). Use sempre que o usuário
  pedir UUIDv7, "uuid v7", "uuid ordenado por tempo", identificador
  ordenável, chave primária cronológica, "extrair a data de um UUID" ou
  "buscar por período pelo id" em Go, mesmo que não cite o nome da
  biblioteca.
license: MIT
metadata:
  author: patrickbrandao
  library: github.com/patrickbrandao/go-loghub-uuidv7
  version: "0.0.2"
---

# go-loghub-uuidv7

Biblioteca Go dedicada ao UUIDv7 com três níveis de precisão temporal:
gera o identificador com o instante embutido (ms, us ou ns) e lê o
instante de volta na mesma precisão. Zero alocações na geração binária,
um `Generator` compartilhado por todas as goroutines, só biblioteca
padrão.

```bash
go get github.com/patrickbrandao/go-loghub-uuidv7
```

```go
import "github.com/patrickbrandao/go-loghub-uuidv7" // o pacote chama-se uuidv7
```

Os programas em `examples/` compilam e rodam; copie deles. As assinaturas
completas estão em `references/api.md`.

## Regras que não podem ser violadas

1. **O nível é contrato da coluna.** Use o **mesmo** `Level` ao gerar,
   ao calcular fronteiras (`MinAt`/`MaxAt`/`RangeAt`) e ao ler
   (`TimestampWithLevel`). Guarde-o em uma constante por tabela. Níveis
   misturados na mesma coluna fazem a consulta por intervalo devolver
   linhas a menos, **sem erro nenhum**.
2. **O nível não é dedutível do UUID.** `TimestampWithLevel(nivel)` precisa
   dele; `Timestamp()` lê só o milissegundo. Informar um nível maior que o
   gravado degrada para o milissegundo, nunca para uma data errada.
3. **A análise aceita qualquer versão; a leitura de tempo recusa.**
   `Parse`, `FromString`, `Scan` e JSON aceitam um UUIDv4 bem formado.
   `Timestamp`/`TimestampWithLevel` devolvem `false` e `Import`/`ImportBinary`
   devolvem `ErrNotV7` para tudo que não é versão 7 com variante RFC. Para
   exigir UUIDv7 na entrada, chame `u.IsValid()` depois de `Parse`.
4. **`ImportBinary` é cega quanto ao nível**: devolve `rand_a` como
   microssegundos e o topo de `rand_b` como nanossegundos mesmo quando são
   bits aleatórios (Nível 1: `Microseconds` até 4095, `Nanoseconds` até
   1023). Para um `time.Time` que descarta o que não é tempo, use
   `TimestampWithLevel`.
5. **`GenerateAt` sorteia os bits livres**: duas chamadas com o mesmo
   instante dão UUIDs diferentes. O valor determinístico de um instante é
   `MinAt` (bits em zero) ou `MaxAt` (bits em um), que são fronteiras, não
   identificadores.
6. **`Value` grava a string canônica e isso não muda.** Para coluna
   `BINARY(16)`/`BLOB`, converta no ponto da consulta: `uuidv7.BinaryUUID(u)`
   e `uuidv7.NullBinaryUUID{...}`. `Scan` já lê as duas formas. Nunca
   rotacione os bytes.
7. **`NULL` e UUID nulo são valores distintos.** `NullUUID{Valid: false}`
   grava `NULL`; `Nil` com `Valid: true` grava zeros. Não use `Nil` como
   "sem valor" em coluna anulável.
8. **O gerador padrão não é para segredos.** Token de sessão, link
   privado, chave de recuperação: `uuidv7.NewCryptoGenerator()`. Todo
   UUIDv7 expõe o instante de criação, com qualquer entropia.
9. **Nenhuma geração devolve erro.** Instantes antes de 1970 viram a época;
   depois de `10889-08-02T05:31:50.655999999Z` saturam. Não escreva
   `if err != nil` onde não há erro.
10. **Dentro do mesmo instante a ordem é aleatória, por projeto.** Não há
    contador monotônico e não se deve acrescentar um por fora; empates de
    instante são comuns (sempre no Nível 1). A ordem cronológica vale entre
    instantes distintos, na resolução do nível.
11. **`FromString` devolve exatamente `ErrInvalidFormat`** (compare com
    `==` ou `errors.Is`); `Parse` devolve `ErrInvalidLength` e
    `ErrInvalidBrackets`, que embrulham o sentinela. `ErrNotV7` **não**
    embrulha `ErrInvalidFormat`.
12. **`%x` sobre um `UUID` formata o texto** (72 caracteres), porque o tipo
    é `fmt.Stringer`. Para os bytes: `fmt.Printf("%x", u.Bytes())`.
13. **Um `Generator` por processo**, criado no boot (`var gen =
    uuidv7.NewGenerator()`), ou as funções de pacote, que usam um gerador
    interno. Não crie um gerador por requisição.
14. **No macOS o relógio tem resolução de microssegundo**: o Nível 3 grava
    nanossegundos zerados ali. Não escreva teste que espere nanossegundos
    reais no host.

## Escolher o nível

| Nível | Precisão embutida | Bits aleatórios | Quando |
|:---|:---|:---|:---|
| `Level1` | milissegundo | 74 | UUIDv7 padrão da RFC; compatibilidade máxima; qualquer biblioteca lê o instante |
| `Level2` | + microssegundo | 62 | ordenação mais fina dentro do mesmo ms; padrão sensato para chave primária |
| `Level3` | + nanossegundo | 52 | logs e eventos de altíssima frequência em hosts com relógio de nanossegundo |

O sub-milissegundo é gravado como contagem decimal (0..999), não como a
fração binária do Método 3 da RFC: outras bibliotecas leem o milissegundo
corretamente e o sub-milissegundo errado. Só esta biblioteca lê os três
níveis.

## Tarefas

### Gerar

```go
var gen = uuidv7.NewGenerator() // uma vez, no boot; seguro para concorrência

u := gen.Generate(uuidv7.Level2)       // uuidv7.UUID ([16]byte), zero alocações
s := gen.GenerateString(uuidv7.Level2) // string canônica, 36 caracteres
s = uuidv7.GenerateString(uuidv7.Level2) // o mesmo, pelo gerador padrão do pacote
u = uuidv7.GenerateV7()               // o UUIDv7 padrão da RFC: Generate(Level1)
u = uuidv7.GenerateV7Level3()         // os níveis também existem pelo nome
```

Ver `examples/01-gerar`.

### Ler o instante de volta

```go
quando, ok := u.TimestampWithLevel(uuidv7.Level2) // o nível com que a coluna foi gravada
if !ok {
	return uuidv7.ErrNotV7 // bem formado, mas não é UUIDv7
}
fmt.Println(quando.UTC().Format(time.RFC3339Nano)) // já vem em UTC

campos, err := uuidv7.ImportBinary(u) // Seconds, Milliseconds, Microseconds, Nanoseconds crus
```

Ver `examples/02-ler-instante`.

### Receber um UUID de fora (texto)

```go
u, err := uuidv7.Parse(entrada) // canônico, {chaves}, urn:uuid:, 32 hex; qualquer caixa
if err != nil {
	return err // errors.Is(err, uuidv7.ErrInvalidFormat) é sempre verdadeiro aqui
}
if !u.IsValid() {
	return uuidv7.ErrNotV7 // se a API exige UUIDv7
}
```

`FromString` aceita só a forma canônica. `MustParse` é para constantes do
código. Ver `examples/03-analisar-texto` e `references/erros.md`.

### Consultar por intervalo de tempo

```go
const nivelEventos = uuidv7.Level2

lo, hi := uuidv7.RangeAt(nivelEventos, inicio, fim) // intervalo semiaberto [inicio, fim)
rows, err := db.Query(
	"SELECT id, corpo FROM eventos WHERE id >= $1 AND id < $2 ORDER BY id",
	lo.String(), hi.String(),
)
```

Sem coluna nem índice de carimbo: a faixa é varrida no índice primário.
Intervalo fechado: `MinAt(nivel, inicio)` e `MaxAt(nivel, fim)` com
`BETWEEN`. Ver `examples/04-consulta-por-intervalo` e
`references/consultas-e-banco.md`.

### Importar registros antigos preservando a ordem

```go
for _, r := range historico {
	id := gen.GenerateAtString(nivelEventos, r.CriadoEm) // o instante original, não o de agora
	// INSERT ... (id, corpo)
}
```

Com `Generate` todos receberiam o carimbo da importação. Ver
`examples/05-importar-historico`.

### JSON, texto, binário e gob

Sem código extra: `UUID` viaja como string canônica em JSON, `NullUUID`
como string ou `null`, `gob` grava os 16 bytes.

```go
type Evento struct {
	ID  uuidv7.UUID     `json:"id"`
	Pai uuidv7.NullUUID `json:"pai"` // null quando Valid=false
}
```

Ver `examples/06-json-e-gob`.

### Banco de dados

```go
type Registro struct {
	ID  uuidv7.UUID
	Pai uuidv7.NullUUID // coluna que aceita NULL
}
_, err := db.Exec("INSERT INTO registros (id, pai) VALUES ($1, $2)", r.ID, r.Pai)             // texto
_, err = db.Exec("INSERT INTO eventos (id) VALUES (?)", uuidv7.BinaryUUID(r.ID))              // BINARY(16)
err = db.QueryRow("SELECT id, pai FROM registros WHERE id = $1", chave).Scan(&r.ID, &r.Pai) // lê texto ou 16 bytes
```

Ver `examples/07-banco-de-dados` e `references/consultas-e-banco.md`,
que tem o DDL para PostgreSQL, MySQL e SQLite.

### Entropia criptográfica e testes reprodutíveis

```go
var gen = uuidv7.NewCryptoGenerator()                 // crypto/rand em toda geração
gen = uuidv7.NewGeneratorWithReader(crand.Reader)     // qualquer io.Reader seguro para concorrência
gen = uuidv7.NewGeneratorWith(func() uint64 { ... })  // função própria, segura para concorrência

// Em testes: leitor determinístico + GenerateAt = sempre o mesmo UUID.
fixo := uuidv7.NewGeneratorWithReader(bytes.NewReader(bytesConhecidos))
u := fixo.GenerateAt(uuidv7.Level3, instanteFixo)
```

Uma fonte que falha durante a geração provoca pânico com
`ErrEntropySource`; fonte `nil` provoca pânico na construção. Ver
`examples/08-entropia`.

### Alto volume sem alocar

```go
buf := make([]byte, 0, 64)
for _, u := range lote {
	buf = u.AppendTo(buf[:0]) // 36 bytes, sem alocar; String() alocaria a cada chamada
	buf = append(buf, '\n')
	w.Write(buf)
}
```

Ordenar uma lista: `slices.SortFunc(lista, uuidv7.UUID.Compare)`. Ver
`examples/09-alto-volume`.

## Exemplos executáveis

Rodam de dentro do repositório da biblioteca, sem dependências. Os que
usam vetores fixos têm saída estável; os demais mudam a cada execução.

| Comando | Mostra |
|:---|:---|
| `go run ./skill/examples/01-gerar` | os três níveis, funções de pacote e `Generator`, os nomes `GenerateV7*`, `Compare` |
| `go run ./skill/examples/02-ler-instante` | `TimestampWithLevel`, `Timestamp`, `ImportBinary`, o vetor da RFC, a recusa de um UUIDv4 |
| `go run ./skill/examples/03-analisar-texto` | `FromString`, as quatro formas de `Parse`, a taxonomia de erros, `IsValid` na entrada |
| `go run ./skill/examples/04-consulta-por-intervalo` | `RangeAt`, `MinAt`, `MaxAt`, a precisão por nível, a armadilha de misturar níveis |
| `go run ./skill/examples/05-importar-historico` | `GenerateAt` preservando a ordem, não determinismo, ordenação com `slices.SortFunc`, bordas |
| `go run ./skill/examples/06-json-e-gob` | JSON, `MarshalText`/`MarshalBinary`, `AppendTo`/`AppendBinary`, `gob`, receptor intacto |
| `go run ./skill/examples/07-banco-de-dados` | `Scan` e `Value` de `UUID`, `BinaryUUID`, `NullUUID`, `NullBinaryUUID`; `NULL` contra `Nil` |
| `go run ./skill/examples/08-entropia` | os quatro construtores, leitor determinístico, entropia zero igual a `MinAt`, pânicos |
| `go run ./skill/examples/09-alto-volume` | um gerador em 64 goroutines, unicidade, `AppendTo` em buffer reaproveitado |

## Verificar o que você escreveu

```bash
go build ./... && go vet ./...   # no projeto do usuário
```

Para conferir um valor sem relógio, gere com entropia fixa e compare com
`MinAt`: `uuidv7.NewGeneratorWith(func() uint64 { return 0 }).GenerateAt(n, t) == uuidv7.MinAt(n, t)`.
Vetores de referência: `019b76da-a87b-71c8-b150-000000000000` é
`MinAt(Level3, 2026-01-01T00:00:00.123456789Z)`, e o exemplo A.6 da RFC
`017f22e2-79b0-7cc3-98c4-dc0c0c07398f` lê `2022-02-22T19:22:22Z`.

## Referências

- Leia `references/api.md` **quando** precisar da assinatura exata de um
  símbolo ou da lista completa de métodos de um tipo.
- Leia `references/consultas-e-banco.md` **quando** for criar a tabela,
  escolher entre coluna de texto e binária, montar a consulta por período
  ou migrar dados antigos.
- Leia `references/erros.md` **quando** for escrever o tratamento de erro
  de análise, de leitura de tempo, de `Scan` ou de fonte de entropia.
- A especificação completa está no repositório da biblioteca, em
  `docs/INDEX.md`; as decisões de projeto que não devem ser reabertas
  estão em `docs/10-decisoes.md`.
