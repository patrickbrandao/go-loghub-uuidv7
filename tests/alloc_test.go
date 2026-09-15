package tests

import (
	"testing"
	"time"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

// Este arquivo trava as alocações exigidas por docs/08-casos-de-teste.md,
// caso 16. As medições de referência são feitas sem o detector de
// corrida; ver docs/09-testes-e-benchmark.md seção 2.

// TestGenerateZeroAllocations trava a propriedade de "zero alocações" da
// geração binária. Uma regressão aqui indica que algum caminho quente
// passou a escapar para o heap.
func TestGenerateZeroAllocations(t *testing.T) {
	g := uuidv7.NewGenerator()
	for _, level := range []uuidv7.Level{uuidv7.Level1, uuidv7.Level2, uuidv7.Level3} {
		allocs := testing.AllocsPerRun(1_000, func() {
			sinkU = g.Generate(level)
		})
		if allocs != 0 {
			t.Errorf("Generate(nível %d): %.0f alocações por chamada, esperado 0", level, allocs)
		}
	}
}

// TestGenerateStringSingleAllocation trava o limite de uma única
// alocação (a string final) na geração em texto.
func TestGenerateStringSingleAllocation(t *testing.T) {
	g := uuidv7.NewGenerator()
	for _, level := range []uuidv7.Level{uuidv7.Level1, uuidv7.Level2, uuidv7.Level3} {
		allocs := testing.AllocsPerRun(1_000, func() {
			sinkS = g.GenerateString(level)
		})
		if allocs > 1 {
			t.Errorf("GenerateString(nível %d): %.0f alocações por chamada, esperado no máximo 1", level, allocs)
		}
	}
}

// TestFromStringZeroAllocations confere que a análise da string não
// aloca: ela escreve em um UUID por valor, devolvido na pilha.
func TestFromStringZeroAllocations(t *testing.T) {
	allocs := testing.AllocsPerRun(1_000, func() {
		sinkU, _ = uuidv7.FromString(canonical)
	})
	if allocs != 0 {
		t.Errorf("FromString: %.0f alocações por chamada, esperado 0", allocs)
	}
}

// TestTimeReadingZeroAllocations confere que as leituras de tempo não
// alocam, nem quando aceitam nem quando recusam: ErrNotV7 é um valor
// pronto, e a recusa não pode passar a montar um erro por chamada.
func TestTimeReadingZeroAllocations(t *testing.T) {
	u := uuidv7.Generate(uuidv7.Level3)
	v4 := uuidv7.MustParse(rfc9562Vectors["A.3 versão 4"])
	cases := map[string]func(){
		"ImportBinary":                 func() { sinkT, sinkErr = uuidv7.ImportBinary(u) },
		"ImportBinary recusando":       func() { sinkT, sinkErr = uuidv7.ImportBinary(v4) },
		"Import":                       func() { sinkT, sinkErr = uuidv7.Import(canonical) },
		"Import recusando":             func() { sinkT, sinkErr = uuidv7.Import(rfc9562Vectors["A.3 versão 4"]) },
		"Timestamp":                    func() { sinkTime, sinkOK = u.Timestamp() },
		"TimestampWithLevel":           func() { sinkTime, sinkOK = u.TimestampWithLevel(uuidv7.Level3) },
		"TimestampWithLevel recusando": func() { sinkTime, sinkOK = v4.TimestampWithLevel(uuidv7.Level3) },
		"IsValid":                      func() { sinkOK = u.IsValid() },
	}
	for name, call := range cases {
		if allocs := testing.AllocsPerRun(1_000, call); allocs != 0 {
			t.Errorf("%s: %.0f alocações por chamada, esperado 0", name, allocs)
		}
	}
}

// Sumidouros que evitam que o compilador elimine as leituras de tempo.
var (
	sinkT    uuidv7.Time
	sinkErr  error
	sinkTime time.Time
	sinkOK   bool
)

// TestGenerateV7ZeroAllocations estende a trava aos nomes do UUIDv7, por
// versão e por nível, que precisam custar o mesmo que Generate no nível
// correspondente: nenhuma alocação, porque são apelidos que o compilador
// embute, e não caminhos próprios.
func TestGenerateV7ZeroAllocations(t *testing.T) {
	g := uuidv7.NewGenerator()
	names := []struct {
		name string
		gen  func(*uuidv7.Generator) uuidv7.UUID
	}{
		{"GenerateV7", (*uuidv7.Generator).GenerateV7},
		{"GenerateV7Level1", (*uuidv7.Generator).GenerateV7Level1},
		{"GenerateV7Level2", (*uuidv7.Generator).GenerateV7Level2},
		{"GenerateV7Level3", (*uuidv7.Generator).GenerateV7Level3},
	}
	for _, n := range names {
		allocs := testing.AllocsPerRun(1_000, func() {
			sinkU = n.gen(g)
		})
		if allocs != 0 {
			t.Errorf("%s: %.0f alocações por chamada, esperado 0", n.name, allocs)
		}
	}
}

// TestAppendToZeroAllocations trava a razão de existir de AppendTo: com
// capacidade sobrando no buffer do chamador, a escrita não aloca nada,
// enquanto String aloca a string devolvida em toda chamada.
func TestAppendToZeroAllocations(t *testing.T) {
	u := uuidv7.Generate(uuidv7.Level3)
	buf := make([]byte, 0, 64)
	allocs := testing.AllocsPerRun(1_000, func() {
		sinkB = u.AppendTo(buf[:0])
	})
	if allocs != 0 {
		t.Errorf("AppendTo: %.0f alocações por chamada, esperado 0", allocs)
	}
}

// sinkB evita que o compilador elimine as chamadas de AppendTo.
var sinkB []byte

// TestAppendBinaryZeroAllocations confere a mesma propriedade de
// AppendTo no lado binário: com capacidade sobrando no buffer do
// chamador, a escrita dos 16 bytes não aloca nada, enquanto
// MarshalBinary devolve uma fatia apoiada no valor recebido.
func TestAppendBinaryZeroAllocations(t *testing.T) {
	u := uuidv7.Generate(uuidv7.Level3)
	buf := make([]byte, 0, 32)
	allocs := testing.AllocsPerRun(1_000, func() {
		sinkB, _ = u.AppendBinary(buf[:0])
	})
	if allocs != 0 {
		t.Errorf("AppendBinary: %.0f alocações por chamada, esperado 0", allocs)
	}
}

// TestParseZeroAllocations confere que o analisador permissivo não paga
// alocação em nenhum dos quatro formatos, nem a partir de bytes.
func TestParseZeroAllocations(t *testing.T) {
	raw := []byte(canonical)
	cases := map[string]func(){
		"Parse canônico":        func() { sinkU, _ = uuidv7.Parse(canonical) },
		"Parse entre chaves":    func() { sinkU, _ = uuidv7.Parse("{" + canonical + "}") },
		"Parse URN":             func() { sinkU, _ = uuidv7.Parse("urn:uuid:" + canonical) },
		"Parse hexadecimal cru": func() { sinkU, _ = uuidv7.Parse("0192f7c51a2b7c3d8e4faabbccddeeff") },
		"ParseBytes":            func() { sinkU, _ = uuidv7.ParseBytes(raw) },
	}
	for name, call := range cases {
		if allocs := testing.AllocsPerRun(1_000, call); allocs != 0 {
			t.Errorf("%s: %.0f alocações por chamada, esperado 0", name, allocs)
		}
	}
}

// TestBoundsZeroAllocations trava a ausência de alocações nas fronteiras
// de tempo. Elas montam um UUID por valor, devolvido na pilha, e são
// chamadas uma vez por consulta — mas uma alocação aqui denunciaria que
// o instante ou o nível passaram a escapar para o heap.
func TestBoundsZeroAllocations(t *testing.T) {
	instante := time.Now()
	fim := instante.Add(time.Second)
	for _, level := range []uuidv7.Level{uuidv7.Level1, uuidv7.Level2, uuidv7.Level3} {
		cases := map[string]func(){
			"MinAt":   func() { sinkU = uuidv7.MinAt(level, instante) },
			"MaxAt":   func() { sinkU = uuidv7.MaxAt(level, instante) },
			"RangeAt": func() { sinkU, sinkU = uuidv7.RangeAt(level, instante, fim) },
		}
		for name, call := range cases {
			if allocs := testing.AllocsPerRun(1_000, call); allocs != 0 {
				t.Errorf("%s(nível %d): %.0f alocações por chamada, esperado 0", name, level, allocs)
			}
		}
	}
}

// TestGenerateAtZeroAllocations trava a mesma propriedade de Generate na
// geração por instante explícito: zero alocações na forma binária e no
// máximo uma, a string final, na forma em texto.
func TestGenerateAtZeroAllocations(t *testing.T) {
	g := uuidv7.NewGenerator()
	instante := time.Now()
	for _, level := range []uuidv7.Level{uuidv7.Level1, uuidv7.Level2, uuidv7.Level3} {
		allocs := testing.AllocsPerRun(1_000, func() {
			sinkU = g.GenerateAt(level, instante)
		})
		if allocs != 0 {
			t.Errorf("GenerateAt(nível %d): %.0f alocações por chamada, esperado 0", level, allocs)
		}

		allocs = testing.AllocsPerRun(1_000, func() {
			sinkS = g.GenerateAtString(level, instante)
		})
		if allocs > 1 {
			t.Errorf("GenerateAtString(nível %d): %.0f alocações por chamada, esperado no máximo 1", level, allocs)
		}
	}
}
