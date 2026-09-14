package tests

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

// TestNewGeneratorWithNilSourcePanics confere que uma fonte de entropia
// nula é rejeitada na configuração, e não na primeira geração.
//
// REGRESSÃO: antes da correção, NewGeneratorWith(nil) devolvia um
// Generator aparentemente válido que só estourava com "nil pointer
// dereference" dentro de Generate — ou seja, o erro de configuração
// aparecia em produção, e não no boot.
func TestNewGeneratorWithNilSourcePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewGeneratorWith(nil) deveria entrar em pânico")
		}
	}()
	uuidv7.NewGeneratorWith(nil)
}

// TestZeroGeneratorUsesDefaultEntropy confere que um Generator obtido
// fora dos construtores — valor zero embutido em outra struct, ou
// ponteiro nulo — ainda produz UUIDs válidos em vez de derrubar o
// processo.
//
// REGRESSÃO: antes da correção, ambos os casos causavam "nil pointer
// dereference". O primeiro é comum: como Generator é exportado, embuti-lo
// por valor em outra struct compila e parece correto.
func TestZeroGeneratorUsesDefaultEntropy(t *testing.T) {
	var byValue uuidv7.Generator
	var byPointer *uuidv7.Generator

	check := func(name string, u uuidv7.UUID) {
		t.Helper()
		if u.Version() != 7 || u.Variant() != 0b10 {
			t.Fatalf("%s: UUID inválido %s (versão %d, variante %b)",
				name, u, u.Version(), u.Variant())
		}
		if u == (uuidv7.UUID{}) {
			t.Fatalf("%s: devolveu o UUID zerado", name)
		}
	}

	for _, level := range []uuidv7.Level{uuidv7.Level1, uuidv7.Level2, uuidv7.Level3} {
		check("Generator zerado", byValue.Generate(level))
		check("ponteiro nulo", byPointer.Generate(level))

		s := byPointer.GenerateString(level)
		if len(s) != 36 {
			t.Fatalf("ponteiro nulo: GenerateString devolveu %d caracteres: %q", len(s), s)
		}
	}

	// A tolerância é regra do tipo inteiro (docs/SPEC.md seção 5.2, caso
	// 3): vale também para os nomes do UUIDv7, por versão e por nível, e
	// para a geração por instante, que não passa por Generate.
	for name, g := range map[string]*uuidv7.Generator{"Generator zerado": &byValue, "ponteiro nulo": byPointer} {
		if u := g.GenerateV7(); u.Version() != 7 || u.Variant() != 0b10 || u.IsZero() {
			t.Fatalf("%s: GenerateV7 devolveu %s", name, u)
		}
		if u := g.GenerateV7Level1(); u.Version() != 7 || u.Variant() != 0b10 || u.IsZero() {
			t.Fatalf("%s: GenerateV7Level1 devolveu %s", name, u)
		}
		if u := g.GenerateV7Level2(); u.Version() != 7 || u.Variant() != 0b10 || u.IsZero() {
			t.Fatalf("%s: GenerateV7Level2 devolveu %s", name, u)
		}
		if u := g.GenerateV7Level3(); u.Version() != 7 || u.Variant() != 0b10 || u.IsZero() {
			t.Fatalf("%s: GenerateV7Level3 devolveu %s", name, u)
		}
		for _, level := range allLevels {
			if u := g.GenerateAt(level, baseInstant); !u.IsValid() || u.IsZero() {
				t.Fatalf("%s: GenerateAt(nível %d) devolveu %s", name, level, u)
			}
			if s := g.GenerateAtString(level, baseInstant); len(s) != 36 {
				t.Fatalf("%s: GenerateAtString(nível %d) devolveu %q", name, level, s)
			}
		}
	}
}

// countingSource devolve uma fonte de entropia determinística que conta
// quantas vezes foi chamada.
func countingSource(calls *atomic.Int64, v uint64) func() uint64 {
	return func() uint64 {
		calls.Add(1)
		return v
	}
}

// TestEntropyDrawsPerLevel trava quantas palavras de 64 bits cada nível
// consome. Nos níveis 2 e 3 rand_a carrega os microssegundos, portanto
// uma única palavra basta; só o nível 1 (e os níveis desconhecidos, que
// se comportam como ele) precisa de duas.
//
// Importa para quem usa NewGeneratorWith com crypto/rand: cada chamada
// extra é uma leitura de entropia criptográfica desperdiçada.
func TestEntropyDrawsPerLevel(t *testing.T) {
	cases := []struct {
		level uuidv7.Level
		draws int64
	}{
		{uuidv7.Level1, 2},
		{uuidv7.Level2, 1},
		{uuidv7.Level3, 1},
		{uuidv7.Level(0), 2},
		{uuidv7.Level(99), 2},
	}
	for _, c := range cases {
		var calls atomic.Int64
		g := uuidv7.NewGeneratorWith(countingSource(&calls, 0))
		g.Generate(c.level)
		if got := calls.Load(); got != c.draws {
			t.Errorf("nível %d: fonte chamada %d vezes, esperado %d", c.level, got, c.draws)
		}
	}

	// Cada nome consome o mesmo que o nível que ele apelida: duas palavras
	// no nome por versão e no Nível 1, uma nos níveis 2 e 3.
	names := []struct {
		name  string
		gen   func(*uuidv7.Generator) uuidv7.UUID
		draws int64
	}{
		{"GenerateV7", (*uuidv7.Generator).GenerateV7, 2},
		{"GenerateV7Level1", (*uuidv7.Generator).GenerateV7Level1, 2},
		{"GenerateV7Level2", (*uuidv7.Generator).GenerateV7Level2, 1},
		{"GenerateV7Level3", (*uuidv7.Generator).GenerateV7Level3, 1},
	}
	for _, n := range names {
		var calls atomic.Int64
		g := uuidv7.NewGeneratorWith(countingSource(&calls, 0))
		n.gen(g)
		if got := calls.Load(); got != n.draws {
			t.Errorf("%s: fonte chamada %d vezes, esperado %d", n.name, got, n.draws)
		}
	}
}

// TestUnknownLevelBehavesAsLevel1 confere que um nível desconhecido não
// grava tempo em rand_a nem no topo de rand_b: os bits têm de vir da
// entropia, exatamente como no nível 1.
func TestUnknownLevelBehavesAsLevel1(t *testing.T) {
	g := uuidv7.NewGeneratorWith(constantSource(^uint64(0)))
	for _, level := range []uuidv7.Level{uuidv7.Level(0), uuidv7.Level(4), uuidv7.Level(99), uuidv7.Level(255)} {
		u := g.Generate(level)
		if u[6] != 0x7F || u[7] != 0xFF {
			t.Errorf("nível %d: rand_a = %#x%02x, esperado 0xfff (aleatório, como no nível 1)",
				level, u[6]&0x0F, u[7])
		}
		if u[8] != 0xBF {
			t.Errorf("nível %d: byte 8 = %#x, esperado 0xbf (variante 10 + rand_b aleatório)", level, u[8])
		}
	}
}

// TestGenerateV7IsLevel1 confere que o nome por versão do UUIDv7 é
// exatamente Generate(Level1): com entropia constante em um, rand_a e o
// topo de rand_b saem inteiros da fonte, sem microssegundos nem
// nanossegundos gravados, e o carimbo de milissegundos é o do relógio.
func TestGenerateV7IsLevel1(t *testing.T) {
	g := uuidv7.NewGeneratorWith(constantSource(^uint64(0)))
	before := time.Now().Truncate(time.Millisecond)
	u := g.GenerateV7()
	after := time.Now()

	checkV7(t, "GenerateV7", u)
	if u[6] != 0x7F || u[7] != 0xFF {
		t.Errorf("rand_a = %#x%02x, esperado 0xfff (aleatório, como no Nível 1)", u[6]&0x0F, u[7])
	}
	if u[8] != 0xBF {
		t.Errorf("byte 8 = %#x, esperado 0xbf (variante 10 + rand_b aleatório)", u[8])
	}

	ts, ok := u.Timestamp()
	if !ok {
		t.Fatal("Timestamp deveria devolver verdadeiro para a versão 7")
	}
	if ts.Before(before) || ts.After(after) {
		t.Errorf("carimbo %s fora da janela [%s, %s]",
			ts.Format(time.RFC3339Nano), before.Format(time.RFC3339Nano), after.Format(time.RFC3339Nano))
	}
}

// TestGenerateV7LevelNamesMatchLevels prova que cada nome gera no nível
// que o nome diz, e não em outro. Com entropia constante em um, o UUID
// gerado pelo nome é lido de volta com o nível esperado e regerado por
// instante nesse mesmo nível: os 16 bytes têm de coincidir. Um nome que
// chamasse outro nível seria pego, porque os campos de tempo
// sub-milissegundo ficam em 0..999 e a entropia constante os deixa em
// 0xfff e 0x3ff, valores que nenhum campo de tempo assume. Os campos
// também são conferidos um a um, para a falha dizer qual bit divergiu.
func TestGenerateV7LevelNamesMatchLevels(t *testing.T) {
	names := []struct {
		name  string
		level uuidv7.Level
		gen   func(*uuidv7.Generator) uuidv7.UUID
	}{
		{"GenerateV7", uuidv7.Level1, (*uuidv7.Generator).GenerateV7},
		{"GenerateV7Level1", uuidv7.Level1, (*uuidv7.Generator).GenerateV7Level1},
		{"GenerateV7Level2", uuidv7.Level2, (*uuidv7.Generator).GenerateV7Level2},
		{"GenerateV7Level3", uuidv7.Level3, (*uuidv7.Generator).GenerateV7Level3},
	}
	for _, n := range names {
		g := uuidv7.NewGeneratorWith(constantSource(^uint64(0)))
		u := n.gen(g)
		checkV7(t, n.name, u)

		// rand_a: entropia (0xfff) no Nível 1, microssegundos (0..999) nos
		// níveis 2 e 3.
		randA := uint16(u[6]&0x0F)<<8 | uint16(u[7])
		if n.level == uuidv7.Level1 && randA != 0x0FFF {
			t.Errorf("%s: rand_a = %#x, esperado 0xfff (aleatório, como no Nível 1)", n.name, randA)
		}
		if n.level != uuidv7.Level1 && randA > 999 {
			t.Errorf("%s: rand_a = %#x, esperado microssegundos em 0..999", n.name, randA)
		}

		// Topo de rand_b: entropia (0x3ff) nos níveis 1 e 2, nanossegundos
		// (0..999) no Nível 3.
		topB := uint16(u[8]&0x3F)<<4 | uint16(u[9]>>4)
		if n.level != uuidv7.Level3 && topB != 0x03FF {
			t.Errorf("%s: topo de rand_b = %#x, esperado 0x3ff (aleatório)", n.name, topB)
		}
		if n.level == uuidv7.Level3 && topB > 999 {
			t.Errorf("%s: topo de rand_b = %#x, esperado nanossegundos em 0..999", n.name, topB)
		}

		// Ida e volta pelo instante embutido, no nível que o nome declara.
		instante, ok := u.TimestampWithLevel(n.level)
		if !ok {
			t.Fatalf("%s: TimestampWithLevel recusou um UUID recém-gerado", n.name)
		}
		if porInstante := g.GenerateAt(n.level, instante); porInstante != u {
			t.Errorf("%s: gerou em outro nível que não o do nome\n  pelo nome:    %s\n  por instante: %s",
				n.name, u, porInstante)
		}
	}
}

// TestPackageLevelShortcuts cobre diretamente os atalhos de pacote, que
// usam o gerador padrão interno e até aqui só eram exercitados de forma
// indireta.
func TestPackageLevelShortcuts(t *testing.T) {
	for _, level := range []uuidv7.Level{uuidv7.Level1, uuidv7.Level2, uuidv7.Level3} {
		u := uuidv7.Generate(level)
		if u.Version() != 7 || u.Variant() != 0b10 {
			t.Fatalf("Generate(nível %d): UUID inválido %s", level, u)
		}

		s := uuidv7.GenerateString(level)
		back, err := uuidv7.FromString(s)
		if err != nil {
			t.Fatalf("GenerateString(nível %d) devolveu %q, que FromString rejeitou: %v", level, s, err)
		}
		if back.String() != s {
			t.Fatalf("GenerateString(nível %d): round-trip divergiu (%q -> %q)", level, s, back.String())
		}
		if back.Version() != 7 || back.Variant() != 0b10 {
			t.Fatalf("GenerateString(nível %d): UUID inválido %s", level, s)
		}

		// Os atalhos precisam produzir valores distintos a cada chamada.
		if uuidv7.Generate(level) == u {
			t.Fatalf("Generate(nível %d) repetiu o mesmo valor em chamadas consecutivas", level)
		}
	}

	// Os nomes do UUIDv7, por versão e por nível, também têm atalho de
	// pacote sobre o gerador padrão.
	names := []struct {
		name string
		gen  func() uuidv7.UUID
	}{
		{"GenerateV7", uuidv7.GenerateV7},
		{"GenerateV7Level1", uuidv7.GenerateV7Level1},
		{"GenerateV7Level2", uuidv7.GenerateV7Level2},
		{"GenerateV7Level3", uuidv7.GenerateV7Level3},
	}
	for _, n := range names {
		u := n.gen()
		if u.Version() != 7 || u.Variant() != 0b10 {
			t.Fatalf("%s: UUID inválido %s", n.name, u)
		}
		if n.gen() == u {
			t.Fatalf("%s repetiu o mesmo valor em chamadas consecutivas", n.name)
		}
	}
}

// observedLevel infere em que nível uma forma de gerar grava, pela
// assinatura estatística dos bits livres, sem precisar injetar entropia.
// É a única prova possível para as funções de pacote, que usam o gerador
// padrão e não aceitam fonte (docs/SPEC.md seção 11.2).
//
// A assinatura decorre da seção 3.1: no Nível 1, rand_a é aleatório e
// passa de 999 em 3096 de cada 4096 amostras; nos níveis 2 e 3 carrega os
// microssegundos e nunca passa. O topo de rand_b é aleatório no Nível 2 e
// passa de 999 em 24 de cada 1024 amostras; no Nível 3 carrega os
// nanossegundos e nunca passa. Com 4.000 amostras, confundir o Nível 2 com
// o 3 tem probabilidade abaixo de 10^-41, e o Nível 1 com qualquer outro,
// muito menor; os níveis 2 e 3 nunca são confundidos com o 1.
func observedLevel(gen func() uuidv7.UUID) uuidv7.Level {
	const samples = 4_000
	randAAbove999, topBAbove999 := false, false
	for i := 0; i < samples; i++ {
		u := gen()
		if uint16(u[6]&0x0F)<<8|uint16(u[7]) > 999 {
			randAAbove999 = true
		}
		if uint16(u[8]&0x3F)<<4|uint16(u[9]>>4) > 999 {
			topBAbove999 = true
		}
	}
	switch {
	case randAAbove999:
		return uuidv7.Level1
	case topBAbove999:
		return uuidv7.Level2
	}
	return uuidv7.Level3
}

// TestEveryGenerationFormWritesItsLevel estende o caso 1 da seção 10 a
// todas as formas de gerar. TestGenerateV7LevelNamesMatchLevels prova o
// nível dos nomes pelo método, com entropia injetada; as funções de pacote
// não recebem fonte, e uma campanha de mutação mostrou que Generate,
// GenerateString, GenerateAt e GenerateAtString podiam ignorar o nível
// pedido, e os nomes GenerateV7 e GenerateV7Level1 a GenerateV7Level3
// podiam chamar o nível errado, sem nenhuma falha: todos continuam
// produzindo UUIDv7 válidos e distintos. A assinatura de observedLevel
// pega os dois defeitos, nas funções de pacote e nos métodos.
func TestEveryGenerationFormWritesItsLevel(t *testing.T) {
	g := uuidv7.NewGenerator()
	forms := []struct {
		name string
		gen  func(uuidv7.Level) uuidv7.UUID
	}{
		{"Generate (pacote)", uuidv7.Generate},
		{"Generate (método)", g.Generate},
		{"GenerateString (pacote)", func(l uuidv7.Level) uuidv7.UUID { return mustParse(t, uuidv7.GenerateString(l)) }},
		{"GenerateString (método)", func(l uuidv7.Level) uuidv7.UUID { return mustParse(t, g.GenerateString(l)) }},
		{"GenerateAt (pacote)", func(l uuidv7.Level) uuidv7.UUID { return uuidv7.GenerateAt(l, instanteConhecido) }},
		{"GenerateAt (método)", func(l uuidv7.Level) uuidv7.UUID { return g.GenerateAt(l, instanteConhecido) }},
		{"GenerateAtString (pacote)", func(l uuidv7.Level) uuidv7.UUID {
			return mustParse(t, uuidv7.GenerateAtString(l, instanteConhecido))
		}},
		{"GenerateAtString (método)", func(l uuidv7.Level) uuidv7.UUID {
			return mustParse(t, g.GenerateAtString(l, instanteConhecido))
		}},
	}
	levels := []struct {
		asked, want uuidv7.Level
	}{
		{uuidv7.Level1, uuidv7.Level1},
		{uuidv7.Level2, uuidv7.Level2},
		{uuidv7.Level3, uuidv7.Level3},
		{uuidv7.Level(0), uuidv7.Level1}, // desconhecidos se comportam como o Nível 1
		{uuidv7.Level(9), uuidv7.Level1},
	}
	for _, f := range forms {
		for _, l := range levels {
			if got := observedLevel(func() uuidv7.UUID { return f.gen(l.asked) }); got != l.want {
				t.Errorf("%s(nível %d): os bits livres são de nível %d, esperado %d", f.name, l.asked, got, l.want)
			}
		}
	}

	names := []struct {
		name string
		want uuidv7.Level
		gen  func() uuidv7.UUID
	}{
		{"GenerateV7 (pacote)", uuidv7.Level1, uuidv7.GenerateV7},
		{"GenerateV7Level1 (pacote)", uuidv7.Level1, uuidv7.GenerateV7Level1},
		{"GenerateV7Level2 (pacote)", uuidv7.Level2, uuidv7.GenerateV7Level2},
		{"GenerateV7Level3 (pacote)", uuidv7.Level3, uuidv7.GenerateV7Level3},
		{"GenerateV7 (método)", uuidv7.Level1, g.GenerateV7},
		{"GenerateV7Level1 (método)", uuidv7.Level1, g.GenerateV7Level1},
		{"GenerateV7Level2 (método)", uuidv7.Level2, g.GenerateV7Level2},
		{"GenerateV7Level3 (método)", uuidv7.Level3, g.GenerateV7Level3},
	}
	for _, n := range names {
		if got := observedLevel(n.gen); got != n.want {
			t.Errorf("%s: os bits livres são de nível %d, esperado %d", n.name, got, n.want)
		}
	}
}
