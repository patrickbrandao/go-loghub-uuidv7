# Uso Completo

Referência de todas as funções da biblioteca `go-loghub-uuidv7`, com
exemplos. A biblioteca trata somente UUIDv7: gera nos três níveis e lê o
instante de volta nas mesmas precisões.

## Instalação

```bash
go get github.com/patrickbrandao/go-loghub-uuidv7
```

> É necessário **Go 1.22 ou superior** (confira com `go version`).

```go
import "github.com/patrickbrandao/go-loghub-uuidv7"
```

> O nome do pacote declarado no código é `uuidv7`, e é por ele que as
> funções são chamadas: `uuidv7.Generate(uuidv7.Level3)`.

---

## Tipos

| Tipo        | Descrição                                                     |
|-------------|---------------------------------------------------------------|
| `Level`     | Nível de precisão: `Level1`, `Level2`, `Level3`.              |
| `UUID`      | `[16]byte` — o valor binário de 128 bits.                     |
| `Generator` | Objeto gerador, criado no boot, seguro para concorrência.     |
| `Time`      | Componentes de tempo importados (segundos/ms/us/ns).          |
| `NullUUID`  | UUID que pode ser `NULL` no banco de dados.                   |
| `BinaryUUID`| UUID gravado no banco como 16 bytes crus.                     |
| `NullBinaryUUID` | O mesmo, em coluna que aceita `NULL`.                    |

```go
type Time struct {
	Seconds      int64 // timestamp Unix (segundos)
	Milliseconds int   // 0..999
	Microseconds int   // 0..999 em Level2/Level3; 0..4095 em Level1 (12 bits aleatorios)
	Nanoseconds  int   // 0..999 em Level3; 0..1023 em Level1/Level2 (10 bits aleatorios)
}
```

> A importação é cega quanto ao nível (ver abaixo): em UUIDs de Nível 1 os
> campos `Microseconds` e `Nanoseconds` são bits aleatórios lidos como se
> fossem tempo, e por isso podem ultrapassar 999. Não valide esses campos
> contra 0..999 sem saber o nível de origem.

---

## Criar geradores

### Gerador padrão (rápido)

```go
g := uuidv7.NewGenerator()
```

A entropia vem do gerador do runtime do Go (`math/rand/v2`): uma
instância de ChaCha8 por thread, semeada pelo sistema operacional na
carga do programa. Sem contenção de lock; ideal para alto volume.

### Gerador com entropia personalizada

Para forçar, por exemplo, entropia **criptográfica** em toda geração:

```go
import (
	crand "crypto/rand"
	"encoding/binary"
)

func cryptoBits() uint64 {
	var b [8]byte
	if _, err := crand.Read(b[:]); err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint64(b[:])
}

g := uuidv7.NewGeneratorWith(cryptoBits)
```

A função fornecida deve devolver 64 bits aleatórios e **ser segura para
uso concorrente** (será chamada por várias goroutines). Ela é chamada uma
vez por UUID nos níveis 2 e 3 e duas vezes no nível 1. Passar `nil` faz
`NewGeneratorWith` entrar em pânico imediatamente, para que o erro de
configuração apareça no boot.

> **Quando isto deixa de ser opcional.** O gerador padrão usa o ChaCha8
> do runtime, que resiste a predição — bem mais forte que o PCG usado
> até a `v0.3.0` —, mas a própria documentação do Go recomenda
> `crypto/rand` para uso sensível a segurança, e a biblioteca não promete
> força criptográfica nessa fonte. Somado a isso, todo UUIDv7 revela o
> instante de criação por construção, qualquer que seja a entropia. Se o
> identificador precisar ser inadivinhável — token de sessão, link
> privado, chave de recuperação — o gerador com entropia criptográfica
> acima é **requisito**, não conveniência. Para chave primária,
> identificador de registro e correlação de log, o gerador padrão é
> adequado.

---

### Gerador com entropia criptográfica

Atalho para o caso mais comum de entropia forte:

```go
var Gen = uuidv7.NewCryptoGenerator()
```

### Gerador a partir de um `io.Reader`

Aceita qualquer fonte no formato da biblioteca padrão. O leitor **precisa
ser seguro para uso concorrente**, porque será chamado por várias
goroutines ao mesmo tempo:

```go
var Gen = uuidv7.NewGeneratorWithReader(crand.Reader)
```

Se uma leitura falhar durante a geração, a chamada entra em pânico: uma
fonte de entropia quebrada não pode degradar em silêncio para um gerador
previsível.

## Gerar

### Binário (128 bits)

```go
u := g.Generate(uuidv7.Level3) // u é um uuidv7.UUID ([16]byte)
```

### String canônica

```go
s := g.GenerateString(uuidv7.Level3) // ex.: 019e99e3-7471-71c2-8e43-f955d7ea2ec6
```

### Atalhos de pacote (gerador padrão interno)

```go
u := uuidv7.Generate(uuidv7.Level2)
s := uuidv7.GenerateString(uuidv7.Level1)
```

### Pelo nome da versão e pelo nome do nível

O UUIDv7 padrão da RFC 9562, o Nível 1, também é pedido pelo nome da
versão, para quem não precisa da extensão de precisão:

```go
u := uuidv7.GenerateV7()          // gerador padrão do pacote
u = g.GenerateV7()              // ou a partir do seu Generator
```

É exatamente `Generate(uuidv7.Level1)`: precisão de milissegundo, 74 bits
aleatórios, zero alocações. Não recebe nível, porque o nome designa o
UUIDv7 da RFC.

Os três níveis também têm nome próprio, sem o argumento de nível, para
que o nível escolhido fique legível no ponto da chamada:

```go
u := uuidv7.GenerateV7Level1()    // o mesmo que GenerateV7 e Generate(uuidv7.Level1)
u = uuidv7.GenerateV7Level2()     // o mesmo que Generate(uuidv7.Level2)
u = uuidv7.GenerateV7Level3()     // o mesmo que Generate(uuidv7.Level3)
u = g.GenerateV7Level3()        // ou a partir do seu Generator
```

Cada um é exatamente `Generate` com o nível correspondente: mesmos
bytes, mesmo consumo de entropia, zero alocações. O nível continua sendo
contrato de toda a coluna: as fronteiras de `MinAt`/`MaxAt` e a leitura
de `TimestampWithLevel` precisam do mesmo nível da geração, e é por isso
que vale tê-lo escrito no nome. Nenhum dos nomes tem forma em texto: use
`GenerateString(uuidv7.Level3)` ou `u.String()`.

---

## Converter

### Binário → string

```go
s := u.String()              // método
s := uuidv7.BinaryToString(u)  // função equivalente
```

### Binário → texto sem alocar

`String` aloca a string devolvida a cada chamada. Quando são milhões de
identificadores por segundo — uma linha de log, um corpo JSON montado à
mão —, `AppendTo` escreve os mesmos 36 bytes no buffer do chamador e não
aloca nada enquanto houver capacidade:

```go
buf := make([]byte, 0, 64)
for _, u := range lista {
	buf = u.AppendTo(buf[:0])
	escreva(buf)
}
```

Medido neste repositório (Apple M2, Go 1.27): `AppendTo` custa ~18,8 ns
e zero alocações, contra ~26 ns e uma alocação de 48 bytes de `String`.
Passar `nil` como `dst` é válido e aloca os 36 bytes.

`AppendText` é o mesmo método com a assinatura de
`encoding.TextAppender` (Go 1.24), devolvendo um erro sempre nulo.

### String → binário

```go
u, err := uuidv7.FromString(s) // método de fábrica
// equivalente: u, err := uuidv7.StringToBinary(s)
if err != nil {
	// uuidv7.ErrInvalidFormat se a string não for canônica
}
```

`FromString` aceita maiúsculas ou minúsculas e exige o formato canônico
`8-4-4-4-12`.

---

## Ler o instante de um UUIDv7

São quatro leituras, todas com o mesmo ponto de partida: **um UUID que
não é de versão 7 com a variante da RFC é recusado**, em vez de ter bits
aleatórios interpretados como data.

| Leitura | Devolve | Recusa |
|---------|---------|--------|
| `u.TimestampWithLevel(nível)` | `time.Time` com a precisão do nível | `false` |
| `u.Timestamp()` | `time.Time` com precisão de milissegundo | `false` |
| `uuidv7.ImportBinary(u)` | os quatro campos crus em `Time` | `ErrNotV7` |
| `uuidv7.Import(s)` | o mesmo, a partir da string canônica | `ErrInvalidFormat` ou `ErrNotV7` |

### Como `time.Time`, na precisão do nível

É a leitura que quase todo código quer:

```go
instante, ok := u.TimestampWithLevel(uuidv7.Level3)
if !ok {
	// u não é UUIDv7
}
```

O nível precisa ser informado, porque ele não pode ser deduzido do UUID.
Informe o nível com que a coluna foi gravada. Se um campo abaixo do
milissegundo estiver fora de 0 a 999, ele denuncia bits aleatórios, e a
leitura devolve só o milissegundo; no Nível 3, os dois campos caem
juntos. Informar o nível errado degrada para o milissegundo, nunca para
uma data errada.

`u.Timestamp()` é o mesmo que `u.TimestampWithLevel(uuidv7.Level1)`.

### Os campos crus

A importação é **cega quanto ao nível**: lê sempre os mesmos campos e
trata os bits como tempo preciso. Para UUIDs de Nível 1, micro/nano
serão aleatórios (esperado).

```go
t, err := uuidv7.Import(s)
switch {
case errors.Is(err, uuidv7.ErrInvalidFormat):
	// o texto não é um UUID canônico
case errors.Is(err, uuidv7.ErrNotV7):
	// é um UUID, mas de outra versão
}
fmt.Println(t.Seconds, t.Milliseconds, t.Microseconds, t.Nanoseconds)
```

A partir de binário:

```go
t, err := uuidv7.ImportBinary(u) // err é nil ou ErrNotV7
```

`ErrNotV7` não é erro de formato: `errors.Is(err, uuidv7.ErrInvalidFormat)`
é falso para ele.

### A análise não confere a versão

`FromString`, `Parse`, `Scan` e os desserializadores aceitam um UUID bem
formado de qualquer versão: guardar, transportar e registrar em log não
dependem dela. Para exigir UUIDv7 logo na entrada, confira depois da
análise:

```go
u, err := uuidv7.Parse(entrada)
if err != nil {
	return err
}
if !u.IsValid() {
	return uuidv7.ErrNotV7
}
```

---

## Gerar a partir de um instante conhecido

`Generate` lê o relógio. `GenerateAt` recebe o instante, e é o sentido
inverso de `Import`: ali se lê o tempo de um identificador, aqui se
constrói um identificador para um tempo.

```go
import "time"

quando := time.Date(2019, 3, 14, 10, 0, 0, 0, time.UTC)

u := uuidv7.GenerateAt(uuidv7.Level3, quando)       // binário
s := uuidv7.GenerateAtString(uuidv7.Level3, quando) // string canônica
```

Também existe como método, para a entropia configurada continuar
valendo:

```go
gen := uuidv7.NewCryptoGenerator()
u := gen.GenerateAt(uuidv7.Level3, quando)
```

### É gerador, não construtor determinístico

**Duas chamadas com o mesmo instante devolvem UUIDs diferentes.** Os
campos de tempo são iguais, os bits livres são sorteados:

```go
a := uuidv7.GenerateAt(uuidv7.Level3, quando)
b := uuidv7.GenerateAt(uuidv7.Level3, quando)
// a != b, mas os campos de tempo lidos por ImportBinary são iguais
```

Isso é proposital. Um gerador que devolvesse sempre o mesmo valor para o
mesmo instante colidiria na primeira repetição. Se o que você quer é o
valor determinístico de um instante, use `MinAt` ou `MaxAt`, da seção
seguinte.

A unicidade vem inteiramente dos bits livres: 74 no Nível 1, 62 no Nível
2 e 52 no Nível 3. Gerando pelo relógio isso nunca é uma escolha sua,
porque o instante avança. Aqui o instante é seu, então vale saber que
gerar em volume para um **único** instante é o caso em que essa margem
importa.

### Reprocessar um histórico preservando a ordem

É o caso de uso principal. Ao importar registros antigos, gerar a chave
com o instante original mantém o índice primário em ordem cronológica,
como se os registros tivessem sido gravados na época:

```go
type Antigo struct {
	CriadoEm time.Time
	Corpo    string
}

gen := uuidv7.NewGenerator()

for _, registro := range historico {
	id := gen.GenerateAtString(uuidv7.Level2, registro.CriadoEm)
	_, err := db.Exec(
		"INSERT INTO eventos (id, corpo) VALUES ($1, $2)",
		id, registro.Corpo,
	)
	if err != nil {
		return err
	}
}
```

Depois disso a consulta por intervalo da seção seguinte funciona sobre
os registros importados, porque a chave carrega o instante de origem.
Se em vez disso você gerasse com `Generate`, todos os registros antigos
receberiam o carimbo do momento da importação e a ordenação da chave
passaria a refletir a ordem de importação, não a do histórico.

Use o **mesmo nível** do resto da tabela. Níveis misturados quebram a
consulta por intervalo, pelo motivo detalhado na seção seguinte.

### Bordas

- Instante anterior a `1970-01-01T00:00:00Z`: degrada para a própria
  época, exatamente como `Generate` faz com um relógio atrasado.
- Instante posterior a `10889-08-02T05:31:50.655999999Z`: satura no
  último instante representável, porque o campo de milissegundos tem 48
  bits.
- Nível desconhecido: tratado como Nível 1, como em `Generate`.

Nenhuma dessas situações devolve erro. Nenhum gerador desta biblioteca
devolve erro, e `GenerateAt` não abre exceção.

---

## Consultar por intervalo de tempo

Este é o motivo prático de adotar UUIDv7 como chave primária: o próprio
índice da chave já está em ordem cronológica, então uma janela de tempo
vira uma varredura de faixa, **sem coluna nem índice de carimbo
temporal**.

O que falta para montar a consulta são os dois identificadores que
delimitam a janela. `MinAt` e `MaxAt` devolvem, respectivamente, o menor
e o maior UUIDv7 que a biblioteca poderia gerar em um instante, e
`RangeAt` devolve o par de um intervalo semiaberto `[from, to)`.

```go
import "time"

fim := time.Now()
inicio := fim.Add(-24 * time.Hour)

lo, hi := uuidv7.RangeAt(uuidv7.Level2, inicio, fim)

rows, err := db.Query(
	`SELECT id, mensagem FROM eventos
	  WHERE id >= $1 AND id < $2
	  ORDER BY id`,
	lo.String(), hi.String(),
)
```

O plano dessa consulta é uma varredura de faixa no índice primário. Não
há `WHERE criado_em BETWEEN ...`, não há índice secundário para manter e
a ordenação por `id` já sai cronológica.

Para um intervalo **fechado** nas duas pontas, use as fronteiras
diretamente:

```go
lo := uuidv7.MinAt(uuidv7.Level2, inicio)
hi := uuidv7.MaxAt(uuidv7.Level2, fim)

rows, err := db.Query(
	"SELECT id, mensagem FROM eventos WHERE id BETWEEN $1 AND $2 ORDER BY id",
	lo.String(), hi.String(),
)
```

### Nunca misture níveis na mesma coluna

**Este é o erro mais provável, e ele não dá mensagem nenhuma: devolve
linhas a menos.**

Os bits abaixo do milissegundo significam coisas diferentes em cada
nível. No Nível 1 os 74 bits abaixo do carimbo são livres; no Nível 2 os
12 bits de `rand_a` carregam os microssegundos exatos; no Nível 3
`rand_a` carrega os microssegundos e os 10 bits altos de `rand_b`
carregam os nanossegundos.

Uma fronteira calculada para um nível só delimita identificadores
gravados **naquele mesmo nível**. Uma fronteira superior de Nível 3, por
exemplo, fica abaixo de boa parte dos identificadores de Nível 1 do
mesmo instante, porque nela `rand_a` vale os microssegundos reais
(0 a 999) enquanto no Nível 1 ele é aleatório (0 a 4095).

Escolha o nível quando criar a tabela e não o mude. Se já houver dados
misturados, a consulta por faixa precisa usar a fronteira do nível mais
permissivo — Nível 1 — nas duas pontas, o que devolve linhas a mais e
exige filtro adicional.

### Precisão da fronteira

A fronteira é tão precisa quanto o nível:

| Nível    | A faixa delimita |
|----------|------------------|
| `Level1` | o milissegundo inteiro |
| `Level2` | o microssegundo |
| `Level3` | o nanossegundo |

No Nível 1, `RangeAt(Level1, inicio, fim)` exclui o milissegundo inteiro
de `fim`. Se a janela precisar terminar dentro daquele milissegundo, o
nível não tem resolução para isso.

### Bordas da faixa representável

O campo de milissegundos tem 48 bits, e as fronteiras saturam nas duas
pontas em vez de dar a volta:

- Instante anterior a `1970-01-01T00:00:00Z`: devolve a fronteira da
  própria época, como faz `Generate`.
- Instante posterior a `10889-08-02T05:31:50.655999999Z`: devolve a
  fronteira do último instante representável.

Saturar mantém as fronteiras monotônicas para qualquer entrada. Em
compensação, dois instantes distintos fora da faixa devolvem o mesmo
valor, então um intervalo inteiramente fora dela é vazio.

`RangeAt` não reordena os argumentos: passar `to` anterior a `from`
devolve um intervalo vazio, e é isso que a comparação vai refletir.

---

## Inspecionar

```go
u := uuidv7.Generate(uuidv7.Level3)

u.Version()  // 7
u.Variant()  // 2 (binário 10, variante RFC)
u.IsValid()  // true: versão 7 com a variante da RFC
```

`Version` e `Variant` leem os bits como estão; num UUID vindo de fora
podem trazer qualquer valor. `IsValid` é a condição que as leituras de
tempo exigem.

Valor especial e comparação:

```go
uuidv7.Nil          // 00000000-0000-0000-0000-000000000000
u.IsZero()          // é o valor nulo?
u.Bytes()           // cópia dos 16 bytes
a.Compare(b)        // -1, 0 ou 1; para igualdade basta a == b
u.URN()             // urn:uuid:0192f7c5-...
```

`Nil` não é UUIDv7: `IsValid` devolve falso para ele. É o valor que
acompanha toda recusa de análise e o que a leitura de banco grava quando
a coluna é `NULL`.

### Ordenar uma lista

A biblioteca não tem tipo de lista. A biblioteca padrão já resolve a
ordenação, com o `Compare` usado direto como função de comparação:

```go
import "slices"

lista := []uuidv7.UUID{c, a, b}
slices.SortFunc(lista, uuidv7.UUID.Compare)
```

`uuidv7.UUID.Compare` aqui é uma expressão de método: vale
`func(uuidv7.UUID, uuidv7.UUID) int`, que é exatamente a assinatura que
`slices.SortFunc` espera. Uma linha, sem alocação, sem comparador
escrito à mão.

Para UUIDv7 essa ordem é cronológica na resolução do nível, porque o
instante ocupa os bits mais altos; dentro do mesmo instante embutido, a
ordem é decidida pelos bits aleatórios. Para ordenar pelo texto o
resultado é o mesmo: a ordem lexicográfica das strings canônicas
acompanha a dos bytes.

`Bytes` devolve uma **cópia**; `u[:]` é mais barato e não copia, mas
aponta para o próprio valor. Use `Bytes` quando o destino guardar a
referência. Serve também para contornar uma armadilha de formatação:
como `UUID` satisfaz `fmt.Stringer`, `%x` sobre um `UUID` formata a
string canônica, não os bytes — `fmt.Printf("%x", u.Bytes())` imprime os
32 dígitos esperados.

## Ler UUIDs escritos em outros formatos

`FromString` aceita apenas a forma canônica. `Parse` aceita quatro
formas, todas com maiúsculas ou minúsculas:

```go
u, err := uuidv7.Parse("0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff")
u, err = uuidv7.Parse("{0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff}")
u, err = uuidv7.Parse("urn:uuid:0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff")
u, err = uuidv7.Parse("0192f7c51a2b7c3d8e4faabbccddeeff")
```

Complementos:

```go
u, err := uuidv7.ParseBytes(linha)   // o mesmo, a partir de []byte, sem alocar
u, err = uuidv7.FromBytes(brutos)    // 16 bytes crus vindos de coluna binária
u = uuidv7.MustParse(constante)      // entra em pânico; só para valores do código
err := uuidv7.Validate(entrada)      // apenas valida
```

Todos os erros de `Parse` são reconhecíveis pelo sentinela de formato, e
os casos específicos pelos próprios erros:

```go
if errors.Is(err, uuidv7.ErrInvalidFormat) { ... }  // qualquer recusa de texto
if errors.Is(err, uuidv7.ErrInvalidLength) { ... }  // comprimento errado
if errors.Is(err, uuidv7.ErrInvalidBrackets) { ... } // chaves malformadas
```

`FromString` devolve sempre exatamente `ErrInvalidFormat`, também para
comparação direta com `==`.

## JSON, texto e binário

O tipo implementa as quatro interfaces de serialização da biblioteca
padrão, então um UUID viaja como string em JSON sem nenhum código extra:

```go
type Evento struct {
	ID uuidv7.UUID `json:"id"`
}

dados, _ := json.Marshal(Evento{ID: uuidv7.Generate(uuidv7.Level3)})
// {"id":"0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff"}
```

`encoding/gob` usa `MarshalBinary` e grava os 16 bytes. Nos dois casos a
leitura confere só a forma, como `Parse`.

## Banco de dados

```go
type Registro struct {
	ID   uuidv7.UUID
	Pai  uuidv7.NullUUID  // coluna que aceita NULL
	Nome string
}

err := db.QueryRow("SELECT id, pai, nome FROM registros WHERE id = $1", chave).
	Scan(&r.ID, &r.Pai, &r.Nome)

_, err = db.Exec("INSERT INTO registros (id, nome) VALUES ($1, $2)", r.ID, r.Nome)
```

`Scan` aceita `NULL`, texto em qualquer formato reconhecido por `Parse` e
16 bytes crus. Texto vazio (string ou bytes) equivale a `NULL`: grava o
UUID nulo sem erro e, em `NullUUID`, deixa `Valid` falso. `Value` grava a
string canônica.

### Coluna binária de 16 bytes

A integração padrão é assimétrica de propósito: a leitura já aceita as
duas formas, mas a escrita é sempre texto. `Value` não vai mudar, porque
uma coluna que já recebeu texto e passasse a receber binário ficaria com
dois formatos misturados e nenhuma consulta acharia as linhas antigas.

Para gravar binário, converta para `BinaryUUID` no ponto da consulta:

```go
_, err := db.Exec(
	"INSERT INTO eventos (id, corpo) VALUES (?, ?)",
	uuidv7.BinaryUUID(r.ID), corpo,
)
```

E `NullBinaryUUID` para a coluna que também aceita `NULL`:

```go
pai := uuidv7.NullBinaryUUID{UUID: chaveDoPai, Valid: temPai}
_, err := db.Exec("INSERT INTO eventos (id, pai) VALUES (?, ?)", uuidv7.BinaryUUID(r.ID), pai)
```

A leitura não muda: os dois tipos delegam ao `Scan` de `UUID` e
continuam aceitando texto e binário. O tipo existe para a escrita.

Vale a pena onde não há tipo nativo de UUID. Trinta e seis bytes de
texto contra dezesseis de binário é mais que o dobro por linha,
replicado em todo índice secundário que referencie a chave:

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

-- PostgreSQL: o tipo e nativo, o driver converte o texto e nao ha ganho
CREATE TABLE eventos (
  id    uuid NOT NULL PRIMARY KEY,
  pai   uuid NULL,
  corpo text
);
```

> **A armadilha.** Ausência de valor e UUID nulo são coisas diferentes e
> viram a mesma linha se você confundir. `NullBinaryUUID` com `Valid`
> falso grava `NULL`; para gravar dezesseis bytes zerados é preciso
> `Valid` verdadeiro com o UUID igual a `Nil`. Uma coluna que misture os
> dois casos não consegue mais distinguir "não havia valor" de "o valor
> era o UUID nulo".

> **Não rotacione os bytes.** A ordem é a de rede, a mesma de
> `MarshalBinary`. Algumas receitas de MySQL sugerem rotacionar os campos
> do identificador para melhorar a localidade do índice. É desnecessário
> no UUIDv7, que já nasce ordenado, e produziria um valor que nenhuma
> outra ferramenta lê.

A consulta por intervalo funciona igual na coluna binária, com as
fronteiras convertidas do mesmo jeito:

```go
lo, hi := uuidv7.RangeAt(uuidv7.Level2, inicio, fim)
rows, err := db.Query(
	"SELECT id, corpo FROM eventos WHERE id >= ? AND id < ? ORDER BY id",
	uuidv7.BinaryUUID(lo), uuidv7.BinaryUUID(hi),
)
```

## Quando usar cada nível

| Nível    | Precisão embutida        | Aleatoriedade restante | Uso típico                                   |
|----------|--------------------------|------------------------|----------------------------------------------|
| `Level1` | milissegundos            | 74 bits                | UUIDv7 padrão; máxima compatibilidade        |
| `Level2` | + microssegundos         | 62 bits                | ordenação mais fina dentro do mesmo ms       |
| `Level3` | + micro e nanossegundos  | 52 bits                | logs/eventos de altíssima frequência         |

Quanto maior o nível, mais bits de tempo e menos bits aleatórios. Em
todos, a colisão é praticamente desprezível para volumes normais, mas se
sua aplicação depende criticamente de unicidade entre máquinas, combine
com um identificador de origem fora do UUID.

> **Resolução do relógio do host.** O Nível 3 só grava nanossegundos
> reais se `time.Now()` os fornecer. Em hosts cujo relógio tem resolução
> de microssegundo, como o macOS, o campo de nanossegundos sai sempre
> zero: são dez bits de aleatoriedade trocados por nada, sem ganho de
> ordenação em relação ao Nível 2. `TestTieRateReport` informa quantos
> instantes distintos o relógio do host oferece; use-o para escolher o
> nível.

> **Relógio do sistema atrasado.** O UUIDv7 lê o relógio de parede a cada
> geração. Se ele for atrasado (ajuste manual ou salto de NTP), os UUIDs
> seguintes ficam lexicograficamente antes dos anteriores até o relógio
> alcançar o instante antigo; a RFC 9562 permite esse comportamento e a
> biblioteca não tenta compensá-lo, porque isso exigiria estado
> compartilhado entre goroutines no caminho quente.

---

## Exemplo completo

```go
package main

import (
	"fmt"
	"time"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

var Gen = uuidv7.NewGenerator()

func main() {
	// gerar
	u := Gen.Generate(uuidv7.Level3)
	s := u.String()
	fmt.Println("uuid:", s)

	// converter ida e volta
	v, _ := uuidv7.FromString(s)
	fmt.Println("igual:", v == u)

	// ler o instante na precisão do nível
	instante, _ := v.TimestampWithLevel(uuidv7.Level3)
	fmt.Println("instante:", instante.Format(time.RFC3339Nano))

	// importar os campos crus
	t, err := uuidv7.ImportBinary(v)
	if err != nil {
		panic(err)
	}
	fmt.Printf("seg=%d ms=%03d us=%03d ns=%03d\n",
		t.Seconds, t.Milliseconds, t.Microseconds, t.Nanoseconds)
}
```
