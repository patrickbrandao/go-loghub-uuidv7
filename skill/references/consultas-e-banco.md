# Consulta por intervalo e integração com banco de dados

Leia este arquivo ao usar o UUIDv7 como chave primária: escolher o tipo
da coluna, gravar, ler e responder a uma janela de tempo pelo índice da
própria chave.

## O nível é contrato da coluna

Escolha o nível ao criar a tabela e use **o mesmo** em três lugares:

1. na geração (`Generate(nivel)`, `GenerateAt(nivel, t)`);
2. nas fronteiras de consulta (`MinAt`, `MaxAt`, `RangeAt`);
3. na leitura do instante (`TimestampWithLevel(nivel)`).

Deixe-o em uma constante do projeto, por tabela:

```go
const nivelEventos = uuidv7.Level2
```

**Nunca misture níveis na mesma coluna.** Os bits abaixo do milissegundo
significam coisas diferentes em cada nível, e uma fronteira calculada
para um nível só delimita identificadores gravados naquele nível. O
sintoma é uma consulta que devolve linhas a menos, sem erro nenhum. Se
uma coluna já tiver dados misturados, use as fronteiras de Nível 1 nas
duas pontas (o nível mais permissivo), que devolvem linhas a mais, e
filtre o excesso pelo instante lido.

## Janela de tempo

`RangeAt` devolve o intervalo semiaberto `[inicio, fim)`, a forma do SQL:

```go
lo, hi := uuidv7.RangeAt(nivelEventos, inicio, fim)

rows, err := db.Query(
	`SELECT id, corpo FROM eventos
	  WHERE id >= $1 AND id < $2
	  ORDER BY id`,
	lo.String(), hi.String(),
)
```

O plano é uma varredura de faixa no índice primário: não há
`WHERE criado_em BETWEEN ...`, não há índice secundário de carimbo, e o
`ORDER BY id` já sai cronológico.

Para um intervalo fechado nas duas pontas, `MinAt` e `MaxAt`:

```go
lo := uuidv7.MinAt(nivelEventos, inicio)
hi := uuidv7.MaxAt(nivelEventos, fim)
// WHERE id BETWEEN $1 AND $2
```

A precisão da fronteira é a do nível: no Nível 1 a faixa delimita o
milissegundo inteiro (então `RangeAt` exclui o milissegundo inteiro de
`fim`), no Nível 2 o microssegundo, no Nível 3 o nanossegundo.

Instantes anteriores a 1970 devolvem a fronteira da época; posteriores a
`10889-08-02T05:31:50.655999999Z` devolvem a do último instante
representável. Um intervalo inteiramente fora da faixa é vazio. `RangeAt`
não reordena os argumentos: `fim` anterior a `inicio` dá intervalo vazio.

Chaves "de hoje", "da última hora", "do mês": calcule os dois instantes e
chame `RangeAt`. Não há função de atalho para isso; a única aritmética é
a de `time.Time`.

## Colunas de texto

`Value` grava a string canônica de 36 caracteres, e esse formato não
muda. Use o tipo nativo onde existir:

```sql
-- PostgreSQL: o tipo uuid é nativo e o driver converte o texto
CREATE TABLE eventos (
  id    uuid NOT NULL PRIMARY KEY,
  pai   uuid NULL,
  corpo text
);
```

```go
type Evento struct {
	ID   uuidv7.UUID
	Pai  uuidv7.NullUUID // coluna que aceita NULL
	Corpo string
}

_, err := db.Exec("INSERT INTO eventos (id, pai, corpo) VALUES ($1, $2, $3)", e.ID, e.Pai, e.Corpo)

err = db.QueryRow("SELECT id, pai, corpo FROM eventos WHERE id = $1", chave).
	Scan(&e.ID, &e.Pai, &e.Corpo)
```

`Scan` aceita `NULL`, texto em qualquer formato de `Parse` e 16 bytes
crus. Texto vazio equivale a `NULL` (`DEFAULT ''` não falha).

## Colunas binárias de 16 bytes

Onde não há tipo nativo de UUID (MySQL, MariaDB, SQLite), a coluna
binária economiza mais da metade do espaço por linha, replicado em todo
índice secundário que referencie a chave:

```sql
-- MySQL / MariaDB
CREATE TABLE eventos (
  id    BINARY(16) NOT NULL PRIMARY KEY,
  pai   BINARY(16) NULL,
  corpo TEXT
);

-- SQLite
CREATE TABLE eventos (
  id    BLOB NOT NULL PRIMARY KEY,
  pai   BLOB NULL,
  corpo TEXT
);
```

A escrita binária é uma conversão **no ponto da consulta**; `Value` do
tipo `UUID` continua gravando texto:

```go
_, err := db.Exec(
	"INSERT INTO eventos (id, pai, corpo) VALUES (?, ?, ?)",
	uuidv7.BinaryUUID(e.ID),
	uuidv7.NullBinaryUUID{UUID: e.Pai.UUID, Valid: e.Pai.Valid},
	e.Corpo,
)

// A leitura não muda: Scan de UUID e de NullUUID já aceitam 16 bytes.
err = db.QueryRow("SELECT id, pai FROM eventos WHERE id = ?", uuidv7.BinaryUUID(chave)).
	Scan(&e.ID, &e.Pai)

// A consulta por intervalo também:
lo, hi := uuidv7.RangeAt(nivelEventos, inicio, fim)
rows, err := db.Query(
	"SELECT id, corpo FROM eventos WHERE id >= ? AND id < ? ORDER BY id",
	uuidv7.BinaryUUID(lo), uuidv7.BinaryUUID(hi),
)
```

Não rotacione os bytes. Receitas antigas de MySQL sugerem reordenar os
campos do UUID para melhorar a localidade do índice; o UUIDv7 já nasce
ordenado, e um valor rotacionado não seria lido por nenhuma outra
ferramenta. A ordem é a de rede, a mesma de `MarshalBinary`.

## `NULL` contra UUID nulo

Ausência de valor e UUID nulo são coisas diferentes:

| Valor em Go | Grava |
|:---|:---|
| `NullUUID{Valid: false}` | `NULL` |
| `NullUUID{UUID: uuidv7.Nil, Valid: true}` | `00000000-0000-0000-0000-000000000000` |
| `NullBinaryUUID{Valid: false}` | `NULL` |
| `NullBinaryUUID{UUID: uuidv7.Nil, Valid: true}` | dezesseis bytes zerados |

Uma coluna que misture os dois casos não consegue mais distinguir "não
havia valor" de "o valor era o UUID nulo". Para uma chave estrangeira
opcional, use o tipo anulável e nunca grave `Nil` como "sem pai".

## Importar registros antigos

Ao migrar um histórico, gere a chave com o instante original, e não com o
relógio: a coluna fica em ordem cronológica como se os registros tivessem
sido gravados na época, e a janela de tempo acima passa a funcionar sobre
eles.

```go
for _, r := range historico {
	id := uuidv7.GenerateAtString(nivelEventos, r.CriadoEm)
	if _, err := db.Exec("INSERT INTO eventos (id, corpo) VALUES ($1, $2)", id, r.Corpo); err != nil {
		return err
	}
}
```

`GenerateAt` sorteia os bits livres: dois registros com o mesmo instante
recebem chaves diferentes. Gerando em volume para um **único** instante,
a margem contra colisão é a dos bits livres (74, 62 ou 52 bits conforme
o nível), folgada na prática.

## Ler o instante de uma chave

```go
quando, ok := e.ID.TimestampWithLevel(nivelEventos)
if !ok {
	// a chave não é UUIDv7 (chegou de fora, ou a coluna tem outra coisa)
}
```

Não crie uma coluna `criado_em` para o que a chave já carrega, a menos
que precise de fuso, de edição do valor ou de precisão diferente da do
nível.
