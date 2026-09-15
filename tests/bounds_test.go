package tests

import (
	"testing"
	"time"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

// levelResolution é o menor passo de tempo que cada nível distingue.
// Duas fronteiras separadas por pelo menos esse passo têm de ficar em
// ordem estrita.
var levelResolution = map[uuidv7.Level]time.Duration{
	uuidv7.Level1: time.Millisecond,
	uuidv7.Level2: time.Microsecond,
	uuidv7.Level3: time.Nanosecond,
}

// allLevels é a lista dos três níveis, na ordem natural.
var allLevels = []uuidv7.Level{uuidv7.Level1, uuidv7.Level2, uuidv7.Level3}

// baseInstant é um instante fixo com milissegundo, microssegundo e
// nanossegundo distintos e diferentes de zero (123, 456 e 789), para que
// uma troca de campos apareça no teste em vez de passar despercebida.
var baseInstant = time.Date(2026, 9, 11, 12, 34, 56, 123_456_789, time.UTC)

// TestBoundsContainRealGeneration confere a propriedade central: os
// UUIDs realmente gerados entre dois instantes caem todos dentro das
// fronteiras desses instantes, nos três níveis.
func TestBoundsContainRealGeneration(t *testing.T) {
	g := uuidv7.NewGenerator()
	for _, level := range allLevels {
		inicio := time.Now()
		amostras := make([]uuidv7.UUID, 20_000)
		for i := range amostras {
			amostras[i] = g.Generate(level)
		}
		fim := time.Now()

		lo := uuidv7.MinAt(level, inicio)
		hi := uuidv7.MaxAt(level, fim)

		for i, u := range amostras {
			if u.Compare(lo) < 0 {
				t.Fatalf("nível %d, amostra %d: %s ficou abaixo da fronteira inferior %s",
					level, i, u, lo)
			}
			if u.Compare(hi) > 0 {
				t.Fatalf("nível %d, amostra %d: %s ficou acima da fronteira superior %s",
					level, i, u, hi)
			}
		}
	}
}

// TestBoundsPreserveVersionAndVariant confere que as duas fronteiras
// continuam sendo UUIDv7 de variante RFC, que é o que as torna
// comparáveis com os identificadores reais gravados na tabela.
func TestBoundsPreserveVersionAndVariant(t *testing.T) {
	instantes := []time.Time{
		time.Unix(0, 0).UTC(),                           // época
		time.Date(1969, 7, 20, 20, 17, 40, 0, time.UTC), // antes da época
		baseInstant,
		time.Now(),
		time.Date(20_000, 1, 1, 0, 0, 0, 0, time.UTC), // acima da faixa
	}
	for _, level := range allLevels {
		for _, instante := range instantes {
			for nome, u := range map[string]uuidv7.UUID{
				"MinAt": uuidv7.MinAt(level, instante),
				"MaxAt": uuidv7.MaxAt(level, instante),
			} {
				if u.Version() != 7 {
					t.Errorf("%s(nível %d, %s): versão %d, esperado 7",
						nome, level, instante, u.Version())
				}
				if u.Variant() != 0b10 {
					t.Errorf("%s(nível %d, %s): variante %b, esperado 10",
						nome, level, instante, u.Variant())
				}
				if !u.IsValid() {
					t.Errorf("%s(nível %d, %s): %s reprovado por IsValid",
						nome, level, instante, u)
				}
			}
		}
	}
}

// TestBoundsOrdering confere as duas relações de ordem que fazem a
// consulta por intervalo funcionar: a fronteira inferior nunca passa da
// superior no mesmo instante, e instantes separados pela resolução do
// nível produzem faixas disjuntas e em ordem.
func TestBoundsOrdering(t *testing.T) {
	for _, level := range allLevels {
		t0 := baseInstant
		lo, hi := uuidv7.MinAt(level, t0), uuidv7.MaxAt(level, t0)
		if lo.Compare(hi) > 0 {
			t.Errorf("nível %d: MinAt %s ficou acima de MaxAt %s", level, lo, hi)
		}

		t1 := t0.Add(levelResolution[level])
		if depois := uuidv7.MinAt(level, t1); hi.Compare(depois) >= 0 {
			t.Errorf("nível %d: MaxAt(t0)=%s não ficou abaixo de MinAt(t0+%s)=%s",
				level, hi, levelResolution[level], depois)
		}
	}
}

// TestBoundsAreMonotonic confere que avançar o instante nunca faz a
// fronteira regredir, inclusive fora da faixa representável, onde a
// saturação é o que garante a propriedade.
func TestBoundsAreMonotonic(t *testing.T) {
	instantes := []time.Time{
		time.Date(-300_000_000, 1, 1, 0, 0, 0, 0, time.UTC), // estoura a multiplicação
		time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Unix(0, 0).UTC(),
		baseInstant,
		baseInstant.Add(time.Nanosecond),
		baseInstant.Add(time.Microsecond),
		baseInstant.Add(time.Millisecond),
		time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC),

		// A travessia da borda superior, onde a saturação precisa levar
		// junto os campos abaixo do milissegundo: se ela os zerasse, o
		// instante seguinte regrediria em relação ao último milissegundo
		// cheio e a monotonicidade cairia aqui.
		time.UnixMilli(int64(1)<<48 - 1).UTC(),
		time.UnixMilli(int64(1)<<48 - 1).UTC().Add(999*time.Microsecond + 999*time.Nanosecond),
		time.Date(20_000, 1, 1, 0, 0, 0, 0, time.UTC),      // acima da faixa
		time.Date(300_000_000, 1, 1, 0, 0, 0, 0, time.UTC), // estoura a multiplicação
	}
	for _, level := range allLevels {
		for i := 1; i < len(instantes); i++ {
			anterior, atual := instantes[i-1], instantes[i]
			if a, b := uuidv7.MinAt(level, anterior), uuidv7.MinAt(level, atual); a.Compare(b) > 0 {
				t.Errorf("nível %d: MinAt regrediu de %s para %s (%s -> %s)",
					level, anterior, atual, a, b)
			}
			if a, b := uuidv7.MaxAt(level, anterior), uuidv7.MaxAt(level, atual); a.Compare(b) > 0 {
				t.Errorf("nível %d: MaxAt regrediu de %s para %s (%s -> %s)",
					level, anterior, atual, a, b)
			}
		}
	}
}

// TestBoundsRoundTripTimestamp confere a ida e volta: ler a fronteira
// com TimestampWithLevel devolve o instante de origem, truncado à
// resolução do nível.
func TestBoundsRoundTripTimestamp(t *testing.T) {
	truncagem := map[uuidv7.Level]time.Duration{
		uuidv7.Level1: time.Millisecond,
		uuidv7.Level2: time.Microsecond,
		uuidv7.Level3: time.Nanosecond,
	}
	for _, level := range allLevels {
		esperado := baseInstant.Truncate(truncagem[level]).UTC()
		for nome, u := range map[string]uuidv7.UUID{
			"MinAt": uuidv7.MinAt(level, baseInstant),
			"MaxAt": uuidv7.MaxAt(level, baseInstant),
		} {
			lido, ok := u.TimestampWithLevel(level)
			if !ok {
				t.Fatalf("%s(nível %d): TimestampWithLevel recusou a fronteira", nome, level)
			}
			if !lido.Equal(esperado) {
				t.Errorf("%s(nível %d): leu %s, esperado %s", nome, level, lido, esperado)
			}
		}
	}
}

// TestBoundsBeforeEpoch confere o piso na época Unix: qualquer instante
// anterior a 1970 devolve a mesma fronteira da própria época, com os 48
// bits de milissegundos zerados.
func TestBoundsBeforeEpoch(t *testing.T) {
	epoca := time.Unix(0, 0).UTC()

	// O valor zero de time.Time é o ano 1, bem antes da época; nomeá-lo
	// deixa isso explícito na lista.
	var valorZero time.Time

	anteriores := []time.Time{
		time.Date(1969, 12, 31, 23, 59, 59, 999_999_999, time.UTC),
		time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
		valorZero,
		time.Date(-300_000_000, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	for _, level := range allLevels {
		for _, instante := range anteriores {
			if got, want := uuidv7.MinAt(level, instante), uuidv7.MinAt(level, epoca); got != want {
				t.Errorf("MinAt(nível %d, %s): %s, esperado a fronteira da época %s",
					level, instante, got, want)
			}
			if got, want := uuidv7.MaxAt(level, instante), uuidv7.MaxAt(level, epoca); got != want {
				t.Errorf("MaxAt(nível %d, %s): %s, esperado a fronteira da época %s",
					level, instante, got, want)
			}
			for i := 0; i < 6; i++ {
				if b := uuidv7.MinAt(level, instante)[i]; b != 0 {
					t.Errorf("MinAt(nível %d, %s): byte %d = %#x, esperado 0",
						level, instante, i, b)
				}
			}
		}
	}
}

// TestBoundsSaturateAboveRange confere o teto: instantes além de
// 10889-08-02 saturam no último milissegundo representável em vez de dar
// a volta, inclusive quando o instante é grande o bastante para estourar
// a multiplicação por mil dentro da decomposição.
func TestBoundsSaturateAboveRange(t *testing.T) {
	// O último instante representável é o fim do último milissegundo,
	// não o começo dele: 10889-08-02T05:31:50.655999999Z. É nele que a
	// saturação tem de travar, senão os campos abaixo do milissegundo
	// regrediriam ao cruzar a borda e a monotonicidade cairia.
	ultimoMilissegundo := time.UnixMilli(int64(1)<<48 - 1).UTC()
	limite := ultimoMilissegundo.Add(999*time.Microsecond + 999*time.Nanosecond)
	acima := []time.Time{
		ultimoMilissegundo.Add(time.Millisecond),
		time.Date(20_000, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(300_000_000, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	for _, level := range allLevels {
		teto := uuidv7.MaxAt(level, limite)
		for _, instante := range acima {
			if got := uuidv7.MaxAt(level, instante); got != teto {
				t.Errorf("MaxAt(nível %d, %s): %s, esperado a saturação %s",
					level, instante, got, teto)
			}
			for i := 0; i < 6; i++ {
				if b := uuidv7.MinAt(level, instante)[i]; b != 0xFF {
					t.Errorf("MinAt(nível %d, %s): byte %d = %#x, esperado 0xff",
						level, instante, i, b)
				}
			}
		}
	}
}

// TestBoundsExactValuesAtTheTopOfTheRange trava os valores exatos em volta
// do teto de 48 bits, onde a saturação começa. Os testes de monotonicidade
// e de saturação conferem a ordem e o valor saturado, mas não o ponto em
// que a saturação entra: uma campanha de mutação mostrou que saturar desde
// o início do último segundo representável, ou já no último milissegundo,
// preserva a ordem e passava pela suíte. Cada linha confere as duas
// fronteiras nos três níveis, a geração por instante com entropia nula, que
// usa a mesma decomposição, e a leitura por nível do valor produzido.
//
// Os vetores foram calculados por uma implementação independente, a partir de
// docs/02-instante-e-entropia.md seção 1 e de
// docs/04-construcao-por-instante.md seção 2.
func TestBoundsExactValuesAtTheTopOfTheRange(t *testing.T) {
	// Segundos Unix de 10889-08-02T05:31:50Z, o último segundo cujo
	// milissegundo inicial cabe em 48 bits.
	const lastSecond = int64(281_474_976_710)
	cases := []struct {
		name     string
		instant  time.Time
		min, max [3]string        // níveis 1, 2 e 3
		readBack [3]time.Duration // o que a leitura por nível soma a lastSecond
	}{
		{
			"início do último segundo, 10889-08-02T05:31:50Z",
			time.Unix(lastSecond, 0),
			[3]string{"ffffffff-fd70-7000-8000-000000000000", "ffffffff-fd70-7000-8000-000000000000", "ffffffff-fd70-7000-8000-000000000000"},
			[3]string{"ffffffff-fd70-7fff-bfff-ffffffffffff", "ffffffff-fd70-7000-bfff-ffffffffffff", "ffffffff-fd70-7000-800f-ffffffffffff"},
			[3]time.Duration{0, 0, 0},
		},
		{
			"último milissegundo, 10889-08-02T05:31:50.655Z",
			time.Unix(lastSecond, 655_000_000),
			[3]string{"ffffffff-ffff-7000-8000-000000000000", "ffffffff-ffff-7000-8000-000000000000", "ffffffff-ffff-7000-8000-000000000000"},
			[3]string{"ffffffff-ffff-7fff-bfff-ffffffffffff", "ffffffff-ffff-7000-bfff-ffffffffffff", "ffffffff-ffff-7000-800f-ffffffffffff"},
			[3]time.Duration{655 * time.Millisecond, 655 * time.Millisecond, 655 * time.Millisecond},
		},
		{
			"dentro do último milissegundo, 10889-08-02T05:31:50.655456789Z",
			time.Unix(lastSecond, 655_456_789),
			[3]string{"ffffffff-ffff-7000-8000-000000000000", "ffffffff-ffff-71c8-8000-000000000000", "ffffffff-ffff-71c8-b150-000000000000"},
			[3]string{"ffffffff-ffff-7fff-bfff-ffffffffffff", "ffffffff-ffff-71c8-bfff-ffffffffffff", "ffffffff-ffff-71c8-b15f-ffffffffffff"},
			[3]time.Duration{655 * time.Millisecond, 655_456 * time.Microsecond, 655_456_789 * time.Nanosecond},
		},
		{
			"primeiro milissegundo acima da faixa, saturado",
			time.Unix(lastSecond, 656_000_000),
			[3]string{"ffffffff-ffff-7000-8000-000000000000", "ffffffff-ffff-73e7-8000-000000000000", "ffffffff-ffff-73e7-be70-000000000000"},
			[3]string{"ffffffff-ffff-7fff-bfff-ffffffffffff", "ffffffff-ffff-73e7-bfff-ffffffffffff", "ffffffff-ffff-73e7-be7f-ffffffffffff"},
			[3]time.Duration{655 * time.Millisecond, 655_999 * time.Microsecond, 655_999_999 * time.Nanosecond},
		},
		{
			"primeiro segundo acima da faixa, saturado pela guarda dos segundos",
			time.Unix(lastSecond+1, 0),
			[3]string{"ffffffff-ffff-7000-8000-000000000000", "ffffffff-ffff-73e7-8000-000000000000", "ffffffff-ffff-73e7-be70-000000000000"},
			[3]string{"ffffffff-ffff-7fff-bfff-ffffffffffff", "ffffffff-ffff-73e7-bfff-ffffffffffff", "ffffffff-ffff-73e7-be7f-ffffffffffff"},
			[3]time.Duration{655 * time.Millisecond, 655_999 * time.Microsecond, 655_999_999 * time.Nanosecond},
		},
	}
	zeroEntropy := uuidv7.NewGeneratorWith(constantSource(0))
	for _, c := range cases {
		for i, level := range allLevels {
			lo, hi := uuidv7.MinAt(level, c.instant), uuidv7.MaxAt(level, c.instant)
			if lo.String() != c.min[i] {
				t.Errorf("%s, nível %d: MinAt = %s, esperado %s", c.name, level, lo, c.min[i])
			}
			if hi.String() != c.max[i] {
				t.Errorf("%s, nível %d: MaxAt = %s, esperado %s", c.name, level, hi, c.max[i])
			}
			if got := zeroEntropy.GenerateAt(level, c.instant); got.String() != c.min[i] {
				t.Errorf("%s, nível %d: GenerateAt com entropia nula = %s, esperado %s", c.name, level, got, c.min[i])
			}
			want := time.Unix(lastSecond, 0).UTC().Add(c.readBack[i])
			if got, ok := lo.TimestampWithLevel(level); !ok || !got.Equal(want) {
				t.Errorf("%s, nível %d: leitura da fronteira = %s, %v; esperado %s",
					c.name, level, got.Format(time.RFC3339Nano), ok, want.Format(time.RFC3339Nano))
			}
		}
	}
}

// TestBoundsMillisecondTurn confere a virada de milissegundo: o último
// nanossegundo de um milissegundo e o primeiro do seguinte produzem
// fronteiras em ordem estrita, e no Nível 1 a fronteira inferior do
// milissegundo seguinte fica acima da superior do anterior.
func TestBoundsMillisecondTurn(t *testing.T) {
	fim := time.Date(2026, 9, 11, 12, 34, 56, 123_999_999, time.UTC)
	inicio := fim.Add(time.Nanosecond) // primeiro nanossegundo do ms 124

	for _, level := range allLevels {
		if a, b := uuidv7.MaxAt(level, fim), uuidv7.MinAt(level, inicio); a.Compare(b) >= 0 {
			t.Errorf("nível %d: a virada de milissegundo não separou as faixas (%s >= %s)",
				level, a, b)
		}
	}

	// A virada tem de aparecer nos 48 bits de milissegundos.
	antes := uuidv7.MinAt(uuidv7.Level1, fim)
	depois := uuidv7.MinAt(uuidv7.Level1, inicio)
	if antes[5]+1 != depois[5] {
		t.Errorf("virada de milissegundo: byte 5 foi de %#x para %#x, esperado incremento de 1",
			antes[5], depois[5])
	}
}

// TestRangeAtIsHalfOpen confere o contrato semiaberto de RangeAt: o
// limite inferior é a fronteira mínima de "from", o superior é a
// fronteira mínima de "to", e um identificador gerado exatamente em "to"
// fica de fora.
func TestRangeAtIsHalfOpen(t *testing.T) {
	from := baseInstant
	to := baseInstant.Add(5 * time.Millisecond)

	for _, level := range allLevels {
		lo, hi := uuidv7.RangeAt(level, from, to)
		if want := uuidv7.MinAt(level, from); lo != want {
			t.Errorf("nível %d: limite inferior %s, esperado MinAt %s", level, lo, want)
		}
		if want := uuidv7.MinAt(level, to); hi != want {
			t.Errorf("nível %d: limite superior %s, esperado MinAt %s", level, hi, want)
		}
		if lo.Compare(hi) >= 0 {
			t.Errorf("nível %d: limite inferior %s não ficou abaixo do superior %s", level, lo, hi)
		}

		// O instante final está excluído: a menor fronteira dele já é o
		// próprio limite superior, portanto não é menor que ele.
		if uuidv7.MinAt(level, to).Compare(hi) < 0 {
			t.Errorf("nível %d: o instante final deveria estar fora do intervalo", level)
		}

		// Um intervalo invertido é vazio, não é reordenado.
		vazioLo, vazioHi := uuidv7.RangeAt(level, to, from)
		if vazioLo.Compare(vazioHi) <= 0 {
			t.Errorf("nível %d: RangeAt invertido devolveu %s..%s, esperado intervalo vazio",
				level, vazioLo, vazioHi)
		}
	}
}

// TestRangeAtContainsGenerationInsideWindow confere que RangeAt delimita
// a geração real: tudo que foi gerado antes de "to" cai dentro de
// [lo, hi), nos três níveis.
func TestRangeAtContainsGenerationInsideWindow(t *testing.T) {
	g := uuidv7.NewGenerator()
	for _, level := range allLevels {
		from := time.Now()
		amostras := make([]uuidv7.UUID, 5_000)
		for i := range amostras {
			amostras[i] = g.Generate(level)
		}
		// "to" precisa ser estritamente posterior à última geração, e no
		// Nível 1 a resolução é o milissegundo: um milissegundo cheio de
		// folga cobre os três níveis.
		to := time.Now().Add(time.Millisecond)

		lo, hi := uuidv7.RangeAt(level, from, to)
		for i, u := range amostras {
			if u.Compare(lo) < 0 || u.Compare(hi) >= 0 {
				t.Fatalf("nível %d, amostra %d: %s ficou fora de [%s, %s)",
					level, i, u, lo, hi)
			}
		}
	}
}

// TestBoundsOfDifferentLevelsDoNotCompose torna concreto o aviso mais
// importante da API: uma fronteira de Nível 3 não delimita
// identificadores gravados em Nível 1, porque os bits abaixo do
// milissegundo significam coisas diferentes em cada nível.
//
// No instante de referência os microssegundos valem 456, bem abaixo dos
// 4095 que rand_a comporta quando é aleatório, então o teto de Nível 3
// fica estritamente abaixo do teto de Nível 1 do mesmo instante — e
// portanto abaixo de parte dos identificadores de Nível 1 geráveis ali.
func TestBoundsOfDifferentLevelsDoNotCompose(t *testing.T) {
	tetoNivel1 := uuidv7.MaxAt(uuidv7.Level1, baseInstant)
	tetoNivel3 := uuidv7.MaxAt(uuidv7.Level3, baseInstant)
	if tetoNivel3.Compare(tetoNivel1) >= 0 {
		t.Fatalf("o teto de Nível 3 (%s) não ficou abaixo do de Nível 1 (%s): o aviso da documentação deixou de valer",
			tetoNivel3, tetoNivel1)
	}

	pisoNivel1 := uuidv7.MinAt(uuidv7.Level1, baseInstant)
	pisoNivel3 := uuidv7.MinAt(uuidv7.Level3, baseInstant)
	if pisoNivel1.Compare(pisoNivel3) >= 0 {
		t.Fatalf("o piso de Nível 1 (%s) não ficou abaixo do de Nível 3 (%s): o aviso da documentação deixou de valer",
			pisoNivel1, pisoNivel3)
	}
}

// TestBoundsUnknownLevelBehavesAsLevel1 confere que níveis fora da faixa
// conhecida caem no Nível 1, como faz Generate.
func TestBoundsUnknownLevelBehavesAsLevel1(t *testing.T) {
	for _, level := range []uuidv7.Level{0, 4, 99, 255} {
		if got, want := uuidv7.MinAt(level, baseInstant), uuidv7.MinAt(uuidv7.Level1, baseInstant); got != want {
			t.Errorf("MinAt(nível %d): %s, esperado o de Nível 1 %s", level, got, want)
		}
		if got, want := uuidv7.MaxAt(level, baseInstant), uuidv7.MaxAt(uuidv7.Level1, baseInstant); got != want {
			t.Errorf("MaxAt(nível %d): %s, esperado o de Nível 1 %s", level, got, want)
		}
	}
}

// TestBoundsFixedVectors trava o layout de bits das fronteiras contra um
// vetor calculado à mão, para uma troca de campos ser detectada sem
// depender do relógio.
//
// O instante de referência está a 1_789_130_096_123 ms da época
// (0x01A0_9076_BDFB nos 48 bits), com microssegundos 456 (0x1C8) e
// nanossegundos 789 (0x315).
func TestBoundsFixedVectors(t *testing.T) {
	casos := []struct {
		nome  string
		got   uuidv7.UUID
		quero string
	}{
		// Nível 1: rand_a e rand_b inteiramente livres.
		{"MinAt Nível 1", uuidv7.MinAt(uuidv7.Level1, baseInstant), "01a09076-bdfb-7000-8000-000000000000"},
		{"MaxAt Nível 1", uuidv7.MaxAt(uuidv7.Level1, baseInstant), "01a09076-bdfb-7fff-bfff-ffffffffffff"},

		// Nível 2: rand_a = 456 = 0x1c8 nos dois casos; rand_b livre.
		{"MinAt Nível 2", uuidv7.MinAt(uuidv7.Level2, baseInstant), "01a09076-bdfb-71c8-8000-000000000000"},
		{"MaxAt Nível 2", uuidv7.MaxAt(uuidv7.Level2, baseInstant), "01a09076-bdfb-71c8-bfff-ffffffffffff"},

		// Nível 3: rand_a = 0x1c8 e os 10 bits altos de rand_b = 789 =
		// 0x315, que ocupam os 6 bits baixos do byte 8 (0b110001 = 0x31,
		// somado à variante dá 0xb1) e os 4 bits altos do byte 9 (0x5).
		{"MinAt Nível 3", uuidv7.MinAt(uuidv7.Level3, baseInstant), "01a09076-bdfb-71c8-b150-000000000000"},
		{"MaxAt Nível 3", uuidv7.MaxAt(uuidv7.Level3, baseInstant), "01a09076-bdfb-71c8-b15f-ffffffffffff"},
	}
	for _, caso := range casos {
		if got := caso.got.String(); got != caso.quero {
			t.Errorf("%s: %s, esperado %s", caso.nome, got, caso.quero)
		}
	}
}
