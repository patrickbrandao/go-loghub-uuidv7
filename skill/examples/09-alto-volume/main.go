// Exemplo 09: gerar em alto volume com um único Generator compartilhado
// por muitas goroutines, provar a unicidade, e serializar sem alocar com
// AppendTo em um buffer reaproveitado.
//
//	go run ./skill/examples/09-alto-volume
//	go run ./skill/examples/09-alto-volume -n 1000000 -g 256
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

// Um único gerador para o processo inteiro. Não há trava global: cada
// thread lê do seu próprio ChaCha8 do runtime.
var gen = uuidv7.NewGenerator()

func main() {
	n := flag.Int("n", 200_000, "quantidade total de UUIDs")
	g := flag.Int("g", 64, "quantidade de goroutines")
	flag.Parse()

	porGoroutine := *n / *g
	resultados := make([][]uuidv7.UUID, *g)

	inicio := time.Now()
	var wg sync.WaitGroup
	for i := range resultados {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			lote := make([]uuidv7.UUID, porGoroutine)
			for j := range lote {
				lote[j] = gen.Generate(uuidv7.Level3) // binário: zero alocações
			}
			resultados[i] = lote
		}(i)
	}
	wg.Wait()
	duracao := time.Since(inicio)

	// Unicidade: nenhum UUID se repete, mesmo com empates de instante,
	// porque os bits livres desempatam.
	vistos := make(map[uuidv7.UUID]struct{}, *n)
	for _, lote := range resultados {
		for _, u := range lote {
			if _, repetido := vistos[u]; repetido {
				panic("UUID repetido: " + u.String())
			}
			vistos[u] = struct{}{}
		}
	}
	total := len(vistos)
	fmt.Printf("%d UUIDs de Nível 3 em %d goroutines: %v (%.0f ns por UUID no total, todos únicos)\n",
		total, *g, duracao.Round(time.Microsecond), float64(duracao.Nanoseconds())/float64(total))

	// Serializar em volume: String() aloca a cada chamada; AppendTo
	// escreve os 36 bytes no buffer do chamador e não aloca enquanto
	// houver capacidade. O mesmo buffer serve para todas as linhas.
	w := bufio.NewWriter(io.Discard) // troque por os.Stdout ou um arquivo
	buf := make([]byte, 0, 64)
	inicio = time.Now()
	for _, lote := range resultados {
		for _, u := range lote {
			buf = u.AppendTo(buf[:0])
			buf = append(buf, '\n')
			if _, err := w.Write(buf); err != nil {
				panic(err)
			}
		}
	}
	if err := w.Flush(); err != nil {
		panic(err)
	}
	fmt.Printf("%d linhas escritas com AppendTo em %v\n", total, time.Since(inicio).Round(time.Microsecond))

	// A ordem: os primeiros e os últimos UUIDs do primeiro lote, lidos de
	// volta no Nível 3. Dentro do mesmo instante embutido a ordem é
	// aleatória; entre instantes distintos é cronológica.
	lote := resultados[0]
	primeiro, _ := lote[0].TimestampWithLevel(uuidv7.Level3)
	ultimo, _ := lote[len(lote)-1].TimestampWithLevel(uuidv7.Level3)
	fmt.Println("primeiro:", lote[0], primeiro.Format(time.RFC3339Nano))
	fmt.Println("último:  ", lote[len(lote)-1], ultimo.Format(time.RFC3339Nano))
}
