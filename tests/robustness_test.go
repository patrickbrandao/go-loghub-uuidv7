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
