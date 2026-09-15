// Exemplo 04: consultar por intervalo de tempo usando o índice da
// própria chave primária, com as fronteiras MinAt, MaxAt e RangeAt.
//
//	go run ./skill/examples/04-consulta-por-intervalo
//
// Não há banco aqui: o exemplo monta a consulta e os parâmetros, e prova
// com Compare que os identificadores gerados dentro da janela caem entre
// as fronteiras e os de fora não.
package main

import (
	"fmt"
	"time"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

// O nível é contrato de toda a coluna: o mesmo na geração, na fronteira
// e na leitura. Deixe-o em uma constante do projeto.
const nivel = uuidv7.Level2

func main() {
	inicio := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	fim := inicio.Add(time.Hour)

	// RangeAt devolve o intervalo semiaberto [inicio, fim): lo = MinAt(inicio)
	// e hi = MinAt(fim). Corresponde a id >= lo AND id < hi.
	lo, hi := uuidv7.RangeAt(nivel, inicio, fim)
	fmt.Println("SELECT id, corpo FROM eventos WHERE id >= $1 AND id < $2 ORDER BY id")
	fmt.Println("  $1 =", lo)
	fmt.Println("  $2 =", hi)

	// Para um intervalo fechado nas duas pontas, use MinAt e MaxAt:
	// id BETWEEN MinAt(inicio) AND MaxAt(fim).
	fmt.Println("MaxAt(fim) =", uuidv7.MaxAt(nivel, fim), "(limite superior fechado do instante fim)")

	// Prova: um identificador gerado dentro da janela fica entre as
	// fronteiras; um gerado no instante fim fica fora (semiaberto).
	dentro := uuidv7.GenerateAt(nivel, inicio.Add(30*time.Minute))
	noFim := uuidv7.GenerateAt(nivel, fim)
	antes := uuidv7.GenerateAt(nivel, inicio.Add(-time.Nanosecond))
	fmt.Println("dentro da janela:", dentroDe(dentro, lo, hi))
	fmt.Println("no instante fim: ", dentroDe(noFim, lo, hi))
	fmt.Println("antes do início: ", dentroDe(antes, lo, hi))

	// A precisão da fronteira é a do nível. No Nível 2, dois instantes
	// separados por um microssegundo têm faixas disjuntas; no Nível 1 a
	// fronteira delimita o milissegundo inteiro.
	t := time.Date(2026, 3, 10, 12, 0, 0, 123_456_000, time.UTC)
	fmt.Println("Nível 1:", uuidv7.MinAt(uuidv7.Level1, t), "..", uuidv7.MaxAt(uuidv7.Level1, t))
	fmt.Println("Nível 2:", uuidv7.MinAt(uuidv7.Level2, t), "..", uuidv7.MaxAt(uuidv7.Level2, t))
	fmt.Println("Nível 3:", uuidv7.MinAt(uuidv7.Level3, t), "..", uuidv7.MaxAt(uuidv7.Level3, t))

	// A ARMADILHA: fronteiras de um nível não delimitam identificadores
	// gravados em outro. Um UUID de Nível 1 do mesmo milissegundo pode
	// ficar acima da fronteira superior de Nível 3, porque em Nível 1
	// rand_a é aleatório (0..4095) e em Nível 3 vale os microssegundos
	// reais (0..999). A consulta devolve linhas a menos, sem erro nenhum.
	gen := uuidv7.NewGeneratorWith(func() uint64 { return ^uint64(0) }) // bits livres em um, para o pior caso
	l1 := gen.GenerateAt(uuidv7.Level1, t)
	fmt.Println("UUID de Nível 1 dentro da faixa de Nível 3 do mesmo instante?",
		dentroDe(l1, uuidv7.MinAt(uuidv7.Level3, t), uuidv7.MaxAt(uuidv7.Level3, t)))
	fmt.Println("... e dentro da faixa de Nível 1?",
		dentroDe(l1, uuidv7.MinAt(uuidv7.Level1, t), uuidv7.MaxAt(uuidv7.Level1, t)))
}

// dentroDe informa se u está no intervalo [lo, hi], pela comparação de
// bytes que o índice do banco também faz.
func dentroDe(u, lo, hi uuidv7.UUID) bool {
	return u.Compare(lo) >= 0 && u.Compare(hi) <= 0
}
