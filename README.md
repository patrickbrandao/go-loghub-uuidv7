# go-loghub-uuidv7

[![ci](https://github.com/patrickbrandao/go-loghub-uuidv7/actions/workflows/ci.yml/badge.svg)](https://github.com/patrickbrandao/go-loghub-uuidv7/actions/workflows/ci.yml)

Biblioteca Go leve e rápida dedicada ao **UUIDv7** (RFC 9562) com **três
níveis de precisão temporal**, nos dois sentidos: **gerar** o
identificador com milissegundos, microssegundos ou nanossegundos
embutidos, e **converter** o UUIDv7 de volta em instante, na mesma
precisão.

- **Rápida**: geração binária em dezenas de nanossegundos, **zero
  alocações**, mais de **11 mil UUIDs/ms** por núcleo (em VM modesta).
- **Concorrente**: um único `Generator` criado no boot atende centenas
  de goroutines sem trava global.
- **Sem dependências**: apenas a biblioteca padrão do Go.
- **Multinível**: do milissegundo padrão até nanossegundos embutidos.
- **Leitura de tempo segura**: um UUID de outra versão é recusado com
  `ErrNotV7`, em vez de virar uma data sem sentido.
- **Consultável por intervalo**: `MinAt`, `MaxAt` e `RangeAt` dão os
  UUIDs que delimitam uma janela de tempo, para responder a ela com o
  índice da própria chave primária, sem coluna nem índice de carimbo.
- **Integrável**: análise permissiva de texto, serialização em JSON e
  integração com `database/sql`, inclusive em coluna binária de 16 bytes.

Só a versão 7 é tratada. As demais versões da RFC 9562 (1, 2, 3, 4, 5, 6
e 8) estão na biblioteca de origem deste projeto,
[go-loghub-uuid](https://github.com/patrickbrandao/go-loghub-uuid).

## Níveis

| Nível    | Precisão embutida           | Compatível UUIDv7 | Bits aleatórios |
|----------|-----------------------------|:-----------------:|:---------------:|
| `Level1` | milissegundos (padrão)      | Sim               | 74              |
| `Level2` | + microssegundos em rand_a  | Sim               | 62              |
| `Level3` | + nanossegundos em rand_b   | Sim               | 52              |

Todos preservam versão 7 e variante RFC.

Os microssegundos e os nanossegundos são gravados como contagens decimais
de 0 a 999, e não como a fração binária do Método 3 da RFC 9562. Qualquer
biblioteca lê o milissegundo destes identificadores; o sub-milissegundo só
esta biblioteca, ou uma implementação de [docs/SPEC.md](docs/SPEC.md), lê
corretamente. O motivo está na seção 11.3 da especificação.

## Gerar

| Função                       | O que devolve                                   |
|------------------------------|-------------------------------------------------|
| `Generate(nível)`            | o UUIDv7 binário (`[16]byte`) do instante atual |
| `GenerateString(nível)`      | o mesmo, já na string canônica                  |
| `GenerateV7()`               | o UUIDv7 padrão da RFC: `Generate(Level1)`      |
| `GenerateV7Level1..3()`      | os três níveis pelo nome, sem argumento         |
| `GenerateAt(nível, instante)`| um UUIDv7 para um instante que você informa     |

Todas existem também como métodos do `Generator`.

## Converter em instante

```go
u, _ := uuidv7.FromString("019b76da-a87b-71c8-b150-000000000000") // gravado em Nível 3

t, ok := u.TimestampWithLevel(uuidv7.Level3)
// t = 2026-01-01T00:00:00.123456789Z, ok = true

campos, err := uuidv7.ImportBinary(u)
// campos = {Seconds:1767225600 Milliseconds:123 Microseconds:456 Nanoseconds:789}, err = nil
```

O nível não pode ser deduzido do UUID: quem lê informa o nível com que a
coluna foi gravada. `TimestampWithLevel` descarta os campos abaixo do
milissegundo que não podem ser tempo (fora de 0 a 999), e `Import` e
`ImportBinary` devolvem os campos crus.

As quatro leituras de tempo (`Import`, `ImportBinary`, `Timestamp` e
`TimestampWithLevel`) recusam qualquer UUID que não seja de versão 7 com
a variante da RFC: as duas primeiras com `ErrNotV7`, as duas últimas com
`false`. A análise de texto (`FromString`, `Parse`, `Scan`, JSON) confere
só a forma; para exigir a versão logo na entrada, use `u.IsValid()`.

## Consulta por intervalo de tempo

É o motivo prático de usar UUIDv7 como chave primária. O índice da chave
já está em ordem cronológica, então uma janela de tempo vira varredura de
faixa:

```go
lo, hi := uuidv7.RangeAt(uuidv7.Level2, inicio, fim)

rows, err := db.Query(
	"SELECT id, corpo FROM eventos WHERE id >= $1 AND id < $2 ORDER BY id",
	lo.String(), hi.String(),
)
```

`RangeAt` devolve o intervalo semiaberto `[inicio, fim)`. Para as
fronteiras separadas, `MinAt` e `MaxAt`. As três respeitam o nível: no
Nível 2 o campo `rand_a` carrega os microssegundos reais do instante, e
zerá-lo daria uma fronteira errada.

Para gravar a chave de um instante conhecido, ao reprocessar um
histórico, `GenerateAt(nível, instante)` gera com o tempo que você
informa, preservando a ordenação da chave.

> A fronteira só vale para UUIDs gravados no **mesmo nível**. Misturar
> níveis na mesma coluna faz a consulta devolver linhas a menos, sem erro
> nenhum. Detalhes em [docs/DEPLOY-FULL.md](docs/DEPLOY-FULL.md).

## Instalação

```bash
go get github.com/patrickbrandao/go-loghub-uuidv7
```

## Uso rápido no Linux

Instalar Go (é necessário **Go 1.22 ou superior**; confira com
`go version` — se a distribuição empacotar versão inferior, use o
instalador oficial):
```bash
apt-get update;
apt-get install -y golang-go;
```

Arquivo go.mod:
```go
module uuid-test

go 1.22

require github.com/patrickbrandao/go-loghub-uuidv7 v0.0.1
```

> O `require` acima é preenchido automaticamente pelo `go get` mostrado
> na seção de compilação; declará-lo à mão é opcional.

Arquivo test-uuid.go:
```go
package main

import (
	"fmt"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

func main() {
	// Gera um UUIDv7 em string (gerador padrão interno, pronto para uso).
	fmt.Println(uuidv7.GenerateString(uuidv7.Level1))
	// ex.: 019e99e3-42f0-7882-9719-2305ff84949c
}
```

Compilar:
```bash
go mod download;
go build -o test-uuid test-uuid.go;
```

Compilar (alternativa):
```bash
go get github.com/patrickbrandao/go-loghub-uuidv7@latest;
go mod tidy;
go build -o test-uuid test-uuid.go;
```

Compilar (multi plataforma):
```
# Windows
GOOS=windows GOARCH=amd64 go build -o test-uuid.exe test-uuid.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o test-uuid test-uuid.go

# Linux (outros processadores)
GOOS=linux GOARCH=arm64 go build -o test-uuid test-uuid.go
```

Rodar:
```bash
./test-uuid;
    # 019e9ace-a992-79d2-9460-e33944a68428
```

Em serviços de alto volume, crie **um** gerador no boot e reutilize:

```go
var Gen = uuidv7.NewGenerator()

func newID() string { return Gen.GenerateString(uuidv7.Level3) }
```

## Aviso de segurança

O gerador padrão (`NewGenerator`, e as funções de pacote `Generate`,
`GenerateString`, `GenerateAt`, `GenerateV7` e `GenerateV7Level1` a
`GenerateV7Level3`) tira entropia do gerador do runtime do Go, uma
instância de **ChaCha8** por thread semeada pelo sistema operacional. É
uma cifra de fluxo, resistente a predição, mas a própria documentação do
Go recomenda `crypto/rand` para uso sensível a segurança. E,
independentemente da fonte, todo UUIDv7 expõe o instante de criação por
construção.

**Não use estes UUIDs como segredo** — token de sessão, link privado,
chave de recuperação ou senha de uso único. Para identificadores que
precisem ser inadivinháveis, monte o gerador com entropia criptográfica:

```go
var Gen = uuidv7.NewCryptoGenerator()
```

`NewGeneratorWithReader` aceita qualquer `io.Reader` seguro para uso
concorrente, e `NewGeneratorWith` aceita uma função que devolve 64 bits.

Como identificador de registro, chave primária ou correlação de log — o
uso a que a biblioteca se destina — o gerador padrão é adequado.

## Mais

- **Mapa completo do projeto**: [STARTHERE.md](STARTHERE.md)
- **Histórico de mudanças**: [CHANGELOG.md](CHANGELOG.md)
- Uso rápido: [docs/DEPLOY-FAST.md](docs/DEPLOY-FAST.md)
- Uso completo (todas as funções): [docs/DEPLOY-FULL.md](docs/DEPLOY-FULL.md)
- Testes e benchmark: [docs/TEST-AND-BENCHMARK.md](docs/TEST-AND-BENCHMARK.md)
- Especificação de desenvolvimento (agnóstica de linguagem):
  [docs/SPEC.md](docs/SPEC.md)
- Como contribuir: [CONTRIBUTING.md](CONTRIBUTING.md)
- Como relatar uma vulnerabilidade: [SECURITY.md](SECURITY.md)

## Licença

[MIT](LICENSE).
