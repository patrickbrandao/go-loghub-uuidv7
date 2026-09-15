# Referência da API (pacote `uuidv7`)

Leia este arquivo quando precisar da assinatura exata de um símbolo. O
contrato completo de cada operação está na especificação do repositório
(`docs/INDEX.md`); esta cópia existe para a skill funcionar fora dele.

```go
import "github.com/patrickbrandao/go-loghub-uuidv7" // pacote: uuidv7
```

## Tipos

| Tipo | Definição | Uso |
|:---|:---|:---|
| `Level` | `uint8`; constantes `Level1`, `Level2`, `Level3` | precisão embutida: ms; ms+us; ms+us+ns. Valores desconhecidos agem como `Level1` |
| `UUID` | `[16]byte`, ordem de rede | o identificador; `==` compara, `u[:]` acessa os bytes |
| `Generator` | struct; criar com um construtor | seguro para concorrência; um por processo. O valor zero e o ponteiro nulo funcionam (usam a fonte padrão), mas não são a forma prevista |
| `Time` | `{Seconds int64; Milliseconds, Microseconds, Nanoseconds int}` | campos crus devolvidos por `Import`/`ImportBinary` |
| `NullUUID` | `{UUID UUID; Valid bool}` | coluna que aceita `NULL`; `Scan`, `Value`, JSON, texto, binário |
| `BinaryUUID` | `UUID` | grava 16 bytes crus no banco; `Scan`, `Value`, `String`, texto, binário |
| `NullBinaryUUID` | `{UUID UUID; Valid bool}` | o anterior, para coluna que aceita `NULL` |

## Construtores de gerador

| Assinatura | Entropia |
|:---|:---|
| `NewGenerator() *Generator` | ChaCha8 do runtime (`math/rand/v2`), por thread, sem trava. O padrão |
| `NewCryptoGenerator() *Generator` | `crypto/rand`; para identificadores que precisam ser inadivinháveis |
| `NewGeneratorWithReader(r io.Reader) *Generator` | 8 bytes big-endian por palavra; `r` precisa ser seguro para concorrência; pânico se `nil`, e pânico com `ErrEntropySource` se uma leitura falhar |
| `NewGeneratorWith(source func() uint64) *Generator` | função própria, segura para concorrência; pânico se `nil` |

Consumo: uma palavra de 64 bits por UUID nos níveis 2 e 3, duas no
Nível 1 (`r1` alimenta `rand_a`, `r2` alimenta `rand_b`).

## Gerar pelo relógio

Cada função existe como método de `*Generator` e como função de pacote
(que usa um gerador padrão interno).

| Assinatura | Devolve |
|:---|:---|
| `Generate(level Level) UUID` | UUIDv7 binário do instante atual |
| `GenerateString(level Level) string` | o mesmo, na forma canônica (1 alocação) |
| `GenerateV7() UUID` | o UUIDv7 padrão da RFC: `Generate(Level1)` |
| `GenerateV7Level1() UUID`, `GenerateV7Level2() UUID`, `GenerateV7Level3() UUID` | `Generate` com o nível do nome |

Nenhuma devolve erro. Binário: zero alocações.

## Construir a partir de um instante

| Assinatura | Devolve |
|:---|:---|
| `GenerateAt(level Level, t time.Time) UUID` (também método) | UUIDv7 do instante `t`, bits livres sorteados; duas chamadas diferem |
| `GenerateAtString(level Level, t time.Time) string` (também método) | o mesmo, em texto |
| `MinAt(level Level, t time.Time) UUID` | o menor UUIDv7 possível em `t`; bits livres em zero |
| `MaxAt(level Level, t time.Time) UUID` | o maior; bits livres em um; limite superior fechado |
| `RangeAt(level Level, from, to time.Time) (lo, hi UUID)` | `[from, to)`: `lo = MinAt(from)`, `hi = MinAt(to)`; para `id >= lo AND id < hi` |

Antes de 1970: a época. Depois de `10889-08-02T05:31:50.655999999Z`: o
último instante representável. Nunca erro.

## Ler o instante

| Assinatura | Devolve | Recusa não-v7 com |
|:---|:---|:---|
| `(UUID) Timestamp() (time.Time, bool)` | UTC, precisão de milissegundo | `false`, instante zero |
| `(UUID) TimestampWithLevel(level Level) (time.Time, bool)` | UTC com a precisão do nível; descarta campos sub-ms fora de 0..999 (no Nível 3, os dois juntos) | `false`, instante zero |
| `ImportBinary(u UUID) (Time, error)` | campos crus, sem julgar o nível | `ErrNotV7`, `Time{}` |
| `Import(s string) (Time, error)` | o mesmo a partir da forma canônica (analisador estrito) | `ErrInvalidFormat` (texto) ou `ErrNotV7` |
| `(UUID) IsValid() bool` | versão 7 e variante `0b10`; a condição que as quatro leituras exigem | |

## Texto

| Assinatura | Comportamento |
|:---|:---|
| `(UUID) String() string` | forma canônica, minúsculas, 36 caracteres; 1 alocação |
| `BinaryToString(u UUID) string` | apelido de `String` |
| `(UUID) AppendTo(dst []byte) []byte` | os 36 bytes no fim de `dst`; sem alocar se houver capacidade; `nil` é válido |
| `(UUID) AppendText(dst []byte) ([]byte, error)` | `AppendTo` com a assinatura de `encoding.TextAppender`; erro sempre `nil` |
| `(UUID) URN() string` | `urn:uuid:` + forma canônica |
| `FromString(s string) (UUID, error)` | só a forma canônica, qualquer caixa; erro é exatamente `ErrInvalidFormat` (`==` vale) |
| `StringToBinary(s string) (UUID, error)` | apelido de `FromString` |
| `Parse(s string) (UUID, error)` | canônica, `{...}`, `urn:uuid:...`, 32 hex; qualquer caixa; erros específicos que embrulham `ErrInvalidFormat` |
| `ParseBytes(b []byte) (UUID, error)` | `Parse` sobre bytes, sem alocar |
| `Validate(s string) error` | só o erro de `Parse` |
| `MustParse(s string) UUID` | `Parse` com pânico; só para constantes |
| `Must(u UUID, err error) UUID` | pânico com `err` se não for `nil` |

## Bytes e comparação

| Assinatura | Comportamento |
|:---|:---|
| `FromBytes(b []byte) (UUID, error)` | exatamente 16 bytes; senão `ErrInvalidLength` |
| `(UUID) Bytes() []byte` | cópia dos 16 bytes (`%x` sobre um `UUID` formata o texto, não os bytes) |
| `(UUID) Compare(other UUID) int` | -1, 0, 1 byte a byte; serve a `slices.SortFunc(lista, uuidv7.UUID.Compare)` |
| `(UUID) IsZero() bool` | é `Nil`? |
| `(UUID) Version() byte`, `(UUID) Variant() byte` | os bits como estão (7 e 2 nos gerados) |
| `Nil` | `var Nil UUID`, todos os bytes em zero; não é UUIDv7; não altere |

## Serialização

| Assinatura | Formato |
|:---|:---|
| `(UUID) MarshalText() ([]byte, error)` / `(*UUID) UnmarshalText([]byte) error` | forma canônica; é o que faz `encoding/json` gravar string. Leitura aceita as formas de `Parse` |
| `(UUID) MarshalBinary() ([]byte, error)` / `(*UUID) UnmarshalBinary([]byte) error` | 16 bytes; leitura exige exatamente 16 (`ErrInvalidLength`). Usado por `encoding/gob` |
| `(UUID) AppendBinary(dst []byte) ([]byte, error)` | os 16 bytes no fim de `dst`; `encoding.BinaryAppender` |
| `NullUUID`, `NullBinaryUUID`: `MarshalJSON`, `UnmarshalJSON`, `MarshalText`, `UnmarshalText`, `MarshalBinary`, `UnmarshalBinary` | ausência: `null` em JSON, sequência vazia em texto e binário; entrada vazia vira ausência |
| `BinaryUUID`: `MarshalText`, `UnmarshalText`, `MarshalBinary`, `UnmarshalBinary`, `String` | delegam a `UUID`; JSON grava string, não vetor |

Entrada inválida: erro e receptor intacto (identificador e `Valid`).

## Banco de dados (`database/sql`)

| Assinatura | Comportamento |
|:---|:---|
| `(*UUID) Scan(src any) error` | aceita `nil`, `string` (formas de `Parse`; vazia = `Nil`), `[]byte` (16 = binário; vazio = `Nil`; outro tamanho = texto); senão `ErrInvalidScanType` |
| `(UUID) Value() (driver.Value, error)` | a string canônica; formato estável |
| `(BinaryUUID) Value()` / `(*BinaryUUID) Scan` | grava 16 bytes crus; lê como `UUID` |
| `(*NullUUID) Scan` / `(NullUUID) Value` | `NULL`/vazio = `Valid=false` sem erro; erro derruba `Valid` e mantém o UUID |
| `(*NullBinaryUUID) Scan` / `(NullBinaryUUID) Value` | idem, gravando 16 bytes quando `Valid` |

## Erros

`ErrInvalidFormat`, `ErrInvalidLength`, `ErrInvalidBrackets` (os dois
últimos embrulham o primeiro), `ErrNotV7`, `ErrInvalidScanType`,
`ErrEntropySource` (valor de pânico). Ver `erros.md`.
