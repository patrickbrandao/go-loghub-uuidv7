// Exemplo 05: gerar chaves para registros antigos com o instante
// original, preservando a ordem cronológica do índice.
//
//	go run ./skill/examples/05-importar-historico
package main

import (
	"fmt"
	"slices"
	"time"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

const nivel = uuidv7.Level2

type registroAntigo struct {
	CriadoEm time.Time
	Corpo    string
}

func main() {
	// Um histórico fora de ordem, como sai de um arquivo de exportação.
	historico := []registroAntigo{
		{time.Date(2019, 3, 14, 10, 0, 0, 250_000_000, time.UTC), "terceiro"},
		{time.Date(2019, 3, 14, 9, 59, 59, 999_999_000, time.UTC), "primeiro"},
		{time.Date(2019, 3, 14, 10, 0, 0, 0, time.UTC), "segundo"},
	}

	gen := uuidv7.NewGenerator()

	// GenerateAt grava o instante do registro, não o da importação.
	// Com Generate, todos os registros receberiam o carimbo de agora e a
	// chave passaria a refletir a ordem de importação.
	type linha struct {
		ID    uuidv7.UUID
		Corpo string
	}
	var linhas []linha
	for _, r := range historico {
		linhas = append(linhas, linha{ID: gen.GenerateAt(nivel, r.CriadoEm), Corpo: r.Corpo})
	}

	// Ordenar pela chave devolve a ordem cronológica. UUID.Compare serve
	// direto a slices.SortFunc.
	slices.SortFunc(linhas, func(a, b linha) int { return a.ID.Compare(b.ID) })
	for _, l := range linhas {
		quando, _ := l.ID.TimestampWithLevel(nivel)
		fmt.Printf("%s  %s  %s\n", l.ID, quando.Format(time.RFC3339Nano), l.Corpo)
	}

	// GenerateAt é gerador, não construtor determinístico: duas chamadas
	// com o mesmo instante dão UUIDs diferentes, com os campos de tempo
	// iguais. Para o valor determinístico de um instante, use MinAt/MaxAt.
	quando := historico[0].CriadoEm
	a := gen.GenerateAt(nivel, quando)
	b := gen.GenerateAt(nivel, quando)
	ta, _ := a.TimestampWithLevel(nivel)
	tb, _ := b.TimestampWithLevel(nivel)
	fmt.Println("mesmo instante, UUIDs iguais?", a == b, "| instante lido igual?", ta.Equal(tb))

	// Uma lista de UUIDs também se ordena em uma linha.
	ids := []uuidv7.UUID{b, a, linhas[2].ID, linhas[0].ID}
	slices.SortFunc(ids, uuidv7.UUID.Compare)
	fmt.Println("ordenado:", ids[0].Compare(ids[1]) <= 0 && ids[1].Compare(ids[2]) <= 0 && ids[2].Compare(ids[3]) <= 0)

	// Bordas: antes de 1970 vira a própria época; além de
	// 10889-08-02T05:31:50.655999999Z satura no último instante
	// representável. Nenhum dos casos devolve erro.
	antes := gen.GenerateAt(nivel, time.Date(1969, 12, 31, 23, 59, 59, 0, time.UTC))
	depois := gen.GenerateAt(nivel, time.Date(12000, 1, 1, 0, 0, 0, 0, time.UTC))
	tAntes, _ := antes.TimestampWithLevel(nivel)
	tDepois, _ := depois.TimestampWithLevel(nivel)
	fmt.Println("pré-1970 ->", tAntes.Format(time.RFC3339Nano))
	fmt.Println("ano 12000 ->", tDepois.Format(time.RFC3339Nano))
}
