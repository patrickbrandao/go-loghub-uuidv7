// Exemplo 02: ler o instante gravado em um UUIDv7, nas duas políticas
// de leitura, e recusar o que não é UUIDv7.
//
//	go run ./skill/examples/02-ler-instante
//
// Usa vetores fixos, então a saída é sempre a mesma.
package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

func main() {
	// Gerado em Nível 3 para 2026-01-01T00:00:00.123456789Z com os bits
	// livres em zero (é a fronteira inferior desse instante). Os campos:
	// ms=123, us=456 (0x1c8 em rand_a), ns=789 (0x315 no topo de rand_b).
	u := uuidv7.MustParse("019b76da-a87b-71c8-b150-000000000000")

	// TimestampWithLevel: um time.Time em UTC com a precisão do nível.
	// O nível não é dedutível do UUID; informe o nível com que a coluna
	// foi gravada.
	t3, ok := u.TimestampWithLevel(uuidv7.Level3)
	fmt.Println("Nível 3:", t3.Format(time.RFC3339Nano), ok)

	// Informar um nível menor descarta a precisão que ele não grava.
	t2, _ := u.TimestampWithLevel(uuidv7.Level2)
	t1, _ := u.Timestamp() // o mesmo que TimestampWithLevel(Level1)
	fmt.Println("Nível 2:", t2.Format(time.RFC3339Nano))
	fmt.Println("Nível 1:", t1.Format(time.RFC3339Nano))

	// ImportBinary: os quatro campos crus, sem julgar o nível. Em um
	// UUID de Nível 1, Microseconds e Nanoseconds seriam bits aleatórios
	// e poderiam passar de 999.
	campos, err := uuidv7.ImportBinary(u)
	if err != nil {
		panic(err)
	}
	fmt.Printf("campos crus: seg=%d ms=%03d us=%03d ns=%03d\n",
		campos.Seconds, campos.Milliseconds, campos.Microseconds, campos.Nanoseconds)

	// Um UUIDv7 de outro gerador (o exemplo A.6 da RFC 9562) é aceito:
	// rand_a vale 3267, fora de 0..999, então a leitura por nível
	// descarta o sub-milissegundo e devolve só o milissegundo.
	rfc := uuidv7.MustParse("017f22e2-79b0-7cc3-98c4-dc0c0c07398f")
	tr, _ := rfc.TimestampWithLevel(uuidv7.Level3)
	cr, _ := uuidv7.ImportBinary(rfc)
	fmt.Println("RFC A.6 por nível:", tr.Format(time.RFC3339Nano), "| crus: us =", cr.Microseconds, "ns =", cr.Nanoseconds)

	// Um UUIDv4 é bem formado, então Parse o aceita; as leituras de tempo
	// o recusam, porque os seus bits altos não são um instante.
	v4, err := uuidv7.Parse("919108f7-52d1-4320-9bac-f847db4148a8")
	fmt.Println("Parse aceita o UUIDv4:", err == nil, "| IsValid:", v4.IsValid())

	_, ok = v4.Timestamp()
	_, err = uuidv7.ImportBinary(v4)
	fmt.Println("Timestamp ok:", ok, "| ImportBinary:", err)
	fmt.Println("ErrNotV7 é erro de formato?", errors.Is(err, uuidv7.ErrInvalidFormat))

	// Import lê a partir do texto canônico. Texto malformado devolve
	// exatamente ErrInvalidFormat; texto bem formado de outra versão
	// devolve ErrNotV7.
	_, err = uuidv7.Import("isto não é um uuid")
	fmt.Println("Import de texto inválido:", err)
}
