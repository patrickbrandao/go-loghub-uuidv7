# Uso Rápido

Como incluir a biblioteca em um projeto Go e gerar um UUIDv7 em string
em poucos segundos.

## 1. Instalar

No diretório do seu projeto (que já tem um `go.mod`):

```bash
go get github.com/patrickbrandao/go-loghub-uuidv7
```

## 2. Gerar um UUIDv7 em string

```go
package main

import (
	"fmt"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

func main() {
	// Forma mais curta: funções de pacote usam um gerador padrão interno,
	// já pronto e seguro para concorrência.
	s1 := uuidv7.GenerateString(uuidv7.Level1) // só milissegundos (UUIDv7 padrão)
	s2 := uuidv7.GenerateString(uuidv7.Level2) // até microssegundos (UUIDv7 + rand_a)
	s3 := uuidv7.GenerateString(uuidv7.Level3) // até nanossegundos (UUIDv7 + rand_a + rand_b)
	fmt.Println(s1)                        // ex.: 019e99e3-42f0-7882-9719-2305ff84949c
	fmt.Println(s2)                        // ex.: 019e99e3-42f0-71a2-9719-2305ff84949c
	fmt.Println(s3)                        // ex.: 019e99e3-42f0-71a2-9719-81a2ff84949c
}
```

Pronto. É isso para o caso mais comum.

## 3. Escolher o nível de precisão

```go
uuidv7.GenerateString(uuidv7.Level1) // milissegundos
uuidv7.GenerateString(uuidv7.Level2) // + microssegundos embutidos
uuidv7.GenerateString(uuidv7.Level3) // + microssegundos e nanossegundos embutidos
```

Todos os níveis produzem UUIDv7 válidos (versão 7, variante RFC). Os
três também existem pelo nome, sem o argumento de nível, devolvendo o
binário:

```go
uuidv7.GenerateV7()       // o UUIDv7 padrão da RFC: o mesmo que Generate(uuidv7.Level1)
uuidv7.GenerateV7Level1() // o mesmo que GenerateV7
uuidv7.GenerateV7Level2() // o mesmo que Generate(uuidv7.Level2)
uuidv7.GenerateV7Level3() // o mesmo que Generate(uuidv7.Level3)
```

## 4. Se você gera muito (recomendado em serviços)

Crie **um** gerador no boot e reutilize-o em todas as goroutines:

```go
var Gen = uuidv7.NewGenerator() // crie uma vez, no início do programa

func handler() string {
	return Gen.GenerateString(uuidv7.Level3)
}
```

O mesmo `*Generator` pode ser chamado por centenas de goroutines ao mesmo
tempo, sem trava global.

## 5. Consultar por intervalo de tempo

A chave já está em ordem cronológica, então a janela de tempo vira
varredura de faixa no índice primário, sem coluna de carimbo:

```go
lo, hi := uuidv7.RangeAt(uuidv7.Level3, inicio, fim) // intervalo [inicio, fim)

rows, err := db.Query(
	"SELECT id, corpo FROM eventos WHERE id >= $1 AND id < $2 ORDER BY id",
	lo.String(), hi.String(),
)
```

Use **o mesmo nível** com que os identificadores foram gravados. Níveis
misturados na mesma coluna fazem a consulta devolver linhas a menos, sem
erro nenhum.

## 6. Gerar para um instante que você já tem

Ao importar registros antigos, gerar a chave com o instante original
mantém o índice em ordem cronológica, como se tivessem sido gravados na
época:

```go
id := uuidv7.GenerateAtString(uuidv7.Level2, registro.CriadoEm)
```

Com `Generate`, todos os registros importados receberiam o carimbo do
momento da importação.

## 7. Ler o instante de volta

```go
u, err := uuidv7.Parse(texto)
if err != nil {
	return err // texto que não é UUID
}
quando, ok := u.TimestampWithLevel(uuidv7.Level3) // o nível com que a coluna foi gravada
if !ok {
	return uuidv7.ErrNotV7 // UUID bem formado, mas de outra versão
}
fmt.Println(quando.Format(time.RFC3339Nano))
```

A análise aceita qualquer versão; é a leitura de tempo que recusa o que
não é UUIDv7. Informe sempre o nível com que os identificadores foram
gravados: ele não pode ser deduzido do UUID.

---

Para todas as funções (binário, conversões, importação de tempo, coluna
binária de 16 bytes), veja [DEPLOY-FULL.md](DEPLOY-FULL.md).
