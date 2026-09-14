package tests

import (
	"errors"
	"testing"
	"time"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

// checkV7 confere versão, variante e IsValid de um UUID que o teste sabe
// ter sido produzido por esta biblioteca.
func checkV7(t *testing.T, name string, u uuidv7.UUID) {
	t.Helper()
	if got := u.Version(); got != 7 {
		t.Errorf("%s: versão %d, esperado 7", name, got)
	}
	if got := u.Variant(); got != 0b10 {
		t.Errorf("%s: variante %d, esperado 2", name, got)
	}
	if !u.IsValid() {
		t.Errorf("%s: IsValid devolveu falso para %s", name, u)
	}
}

// mustImportBinary extrai o tempo de um UUID que o teste sabe ser UUIDv7,
// falhando o teste se ImportBinary o recusar. Não pode ser chamada de
// outra goroutine que não a do teste.
func mustImportBinary(tb testing.TB, u uuidv7.UUID) uuidv7.Time {
	tb.Helper()
	tm, err := uuidv7.ImportBinary(u)
	if err != nil {
		tb.Fatalf("ImportBinary(%s): erro inesperado %v", u, err)
	}
	return tm
}

// rfc9562Vectors são os exemplos publicados nos apêndices A e B da RFC
// 9562, um por versão, transcritos como estão no texto da RFC. O de
// versão 7 serve de vetor externo para a leitura de tempo; os demais são
// UUIDs reais de outras versões, que as leituras de tempo têm de recusar.
var rfc9562Vectors = map[string]string{
	"A.1 versão 1": "c232ab00-9414-11ec-b3c8-9f6bdeced846",
	"A.2 versão 3": "5df41881-3aed-3515-88a7-2f4a814cf09e",
	"A.3 versão 4": "919108f7-52d1-4320-9bac-f847db4148a8",
	"A.4 versão 5": "2ed6657d-e927-568b-95e1-2665a8aea6a2",
	"A.5 versão 6": "1ec9414c-232a-6b00-b3c8-9f6bdeced846",
	"A.6 versão 7": "017f22e2-79b0-7cc3-98c4-dc0c0c07398f",
	"B.2 versão 8": "5c146b14-3c52-8afd-938a-375d0df1fbf6",
}

// TestTimestampRoundTrip confere que o instante extraído fica próximo do
// relógio do sistema no momento da geração, nos três níveis.
func TestTimestampRoundTrip(t *testing.T) {
	for _, level := range allLevels {
		before := time.Now().Add(-2 * time.Millisecond)
		u := uuidv7.Generate(level)
		after := time.Now().Add(2 * time.Millisecond)

		got, ok := u.Timestamp()
		if !ok {
			t.Errorf("nível %d: Timestamp devolveu falso", level)
			continue
		}
		if got.Before(before) || got.After(after) {
			t.Errorf("nível %d: instante %v fora da janela [%v, %v]", level, got, before, after)
		}
		if got.Location() != time.UTC {
			t.Errorf("nível %d: instante em %v, esperado UTC", level, got.Location())
		}
	}
}

// TestTimeReadingsReturnUTC confere que as duas leituras de instante
// devolvem UTC em todos os níveis, inclusive nos desconhecidos e no caminho
// do descarte por faixa (docs/SPEC.md seção 7). A comparação é pela
// identidade de time.UTC, e não pelo nome do fuso: numa máquina com TZ=UTC,
// como os runners da integração contínua, o fuso local também se chama
// "UTC". Antes deste teste só TestTimestampRoundTrip conferia o fuso, e só
// de Timestamp; uma campanha de mutação mostrou que TimestampWithLevel podia
// devolver o horário local e só um exemplo falhava, e só fora de UTC.
func TestTimeReadingsReturnUTC(t *testing.T) {
	cases := map[string]uuidv7.UUID{
		"campos na faixa":      uuidv7.MinAt(uuidv7.Level3, baseInstant),
		"campos fora da faixa": uuidv7.MustParse(canonical),
		"gerado pelo relógio":  uuidv7.Generate(uuidv7.Level3),
	}
	levels := []uuidv7.Level{uuidv7.Level1, uuidv7.Level2, uuidv7.Level3, uuidv7.Level(0), uuidv7.Level(9)}
	for name, u := range cases {
		if got, ok := u.Timestamp(); !ok || got.Location() != time.UTC {
			t.Errorf("%s: Timestamp = %v em %v, %v; esperado UTC", name, got, got.Location(), ok)
		}
		for _, level := range levels {
			if got, ok := u.TimestampWithLevel(level); !ok || got.Location() != time.UTC {
				t.Errorf("%s: TimestampWithLevel(%d) = %v em %v, %v; esperado UTC", name, level, got, got.Location(), ok)
			}
		}
	}
}

// TestTimestampWithLevelRecoversSubMillisecond confere que a precisão
// gravada pelos níveis 2 e 3 é recuperada como time.Time.
func TestTimestampWithLevelRecoversSubMillisecond(t *testing.T) {
	for _, level := range []uuidv7.Level{uuidv7.Level2, uuidv7.Level3} {
		u := uuidv7.Generate(level)

		coarse, ok := u.Timestamp()
		if !ok {
			t.Fatalf("nível %d: Timestamp devolveu falso", level)
		}
		fine, ok := u.TimestampWithLevel(level)
		if !ok {
			t.Fatalf("nível %d: TimestampWithLevel devolveu falso", level)
		}

		delta := fine.Sub(coarse)
		if delta < 0 || delta >= time.Millisecond {
			t.Errorf("nível %d: refinamento de %v fora da faixa de um milissegundo", level, delta)
		}

		imported := mustImportBinary(t, u)
		if got := fine.Nanosecond() / 1_000 % 1_000; got != imported.Microseconds {
			t.Errorf("nível %d: microssegundos %d, esperado %d", level, got, imported.Microseconds)
		}
		if level == uuidv7.Level3 {
			if got := fine.Nanosecond() % 1_000; got != imported.Nanoseconds {
				t.Errorf("nível 3: nanossegundos %d, esperado %d", got, imported.Nanoseconds)
			}
		}
	}
}

// TestTimestampWithLevelDiscardsOutOfRangeFields confere que, quando os
// campos sub-milissegundo denunciam bits aleatórios (fora de 0..999),
// TimestampWithLevel devolve apenas o milissegundo, sem somar o campo que
// por acaso ainda caiba na faixa (docs/SPEC.md seção 7).
func TestTimestampWithLevelDiscardsOutOfRangeFields(t *testing.T) {
	// UUIDv7 montado à mão: rand_a = 0xFFF (4095, fora da faixa) e
	// topo de rand_b = 5 nanossegundos (dentro da faixa).
	base := uuidv7.UUID{
		0x01, 0x92, 0xf7, 0xc5, 0x1a, 0x2b, // unix_ts_ms
		0x7F, 0xFF, // versão 7 + rand_a = 0xFFF
		0x80, 0x50, // variante 10 + nano = (0x00 << 4) | (0x50 >> 4) = 5
		0, 0, 0, 0, 0, 0,
	}
	coarse, ok := base.Timestamp()
	if !ok {
		t.Fatal("Timestamp devolveu falso para um UUIDv7")
	}

	for _, level := range []uuidv7.Level{uuidv7.Level2, uuidv7.Level3} {
		fine, ok := base.TimestampWithLevel(level)
		if !ok {
			t.Fatalf("nível %d: TimestampWithLevel devolveu falso", level)
		}
		if !fine.Equal(coarse) {
			t.Errorf("nível %d: rand_a fora da faixa deveria devolver só o milissegundo; obtido %v, esperado %v",
				level, fine, coarse)
		}
	}

	// A extração cega, sobre os mesmos bytes, devolve os campos crus: as
	// duas leituras divergindo é o comportamento correto.
	if raw := mustImportBinary(t, base); raw.Microseconds != 0xFFF || raw.Nanoseconds != 5 {
		t.Errorf("ImportBinary deveria devolver os campos crus (4095, 5); obtido (%d, %d)",
			raw.Microseconds, raw.Nanoseconds)
	}

	// Caso simétrico: micro válido, nano fora da faixa (1023) no nível 3.
	other := base
	other[6], other[7] = 0x70, 0x07 // rand_a = 7 microssegundos
	other[8], other[9] = 0xBF, 0xF0 // nano = (0x3F << 4) | 0xF = 1023
	fine, _ := other.TimestampWithLevel(uuidv7.Level3)
	if !fine.Equal(coarse) {
		t.Errorf("nível 3 com nano fora da faixa deveria devolver só o milissegundo; obtido %v", fine)
	}
	// No nível 2 o nano é ignorado, então os 7 microssegundos valem.
	fine, _ = other.TimestampWithLevel(uuidv7.Level2)
	if want := coarse.Add(7 * time.Microsecond); !fine.Equal(want) {
		t.Errorf("nível 2 deveria somar 7 microssegundos; obtido %v, esperado %v", fine, want)
	}

	// Caso válido nos dois campos: 7 microssegundos e 5 nanossegundos.
	valid := base
	valid[6], valid[7] = 0x70, 0x07
	fine, _ = valid.TimestampWithLevel(uuidv7.Level3)
	if want := coarse.Add(7*time.Microsecond + 5*time.Nanosecond); !fine.Equal(want) {
		t.Errorf("nível 3 válido deveria somar 7 us e 5 ns; obtido %v, esperado %v", fine, want)
	}

	// Os limites da faixa: 999 é tempo, 1000 já não é.
	edge := base
	edge[6], edge[7] = 0x73, 0xE7 // rand_a = 999
	edge[8], edge[9] = 0xBE, 0x70 // nano = (0x3E << 4) | 0x7 = 999
	fine, _ = edge.TimestampWithLevel(uuidv7.Level3)
	if want := coarse.Add(999*time.Microsecond + 999*time.Nanosecond); !fine.Equal(want) {
		t.Errorf("nível 3 com 999 us e 999 ns deveria somar os dois; obtido %v, esperado %v", fine, want)
	}
	edge[6], edge[7] = 0x73, 0xE8 // rand_a = 1000
	fine, _ = edge.TimestampWithLevel(uuidv7.Level2)
	if !fine.Equal(coarse) {
		t.Errorf("nível 2 com rand_a = 1000 deveria devolver só o milissegundo; obtido %v", fine)
	}

	// Nível 1 e níveis desconhecidos devolvem só o milissegundo, mesmo com
	// os dois campos válidos.
	for _, level := range []uuidv7.Level{uuidv7.Level1, uuidv7.Level(0), uuidv7.Level(9)} {
		if fine, _ := valid.TimestampWithLevel(level); !fine.Equal(coarse) {
			t.Errorf("nível %d deveria devolver só o milissegundo; obtido %v", level, fine)
		}
	}
}

// TestRFC9562Version7Vector lê o exemplo de UUIDv7 publicado no apêndice
// A.6 da RFC 9562. É o único vetor de tempo desta suíte calculado fora do
// projeto: a RFC declara unix_ts_ms = 0x017F22E279B0, o instante
// 2022-02-22T14:22:22-05:00, rand_a = 0xCC3 e rand_b = 0x18C4DC0C0C07398F.
//
// rand_a (3267) e o topo de rand_b (396) caem fora e dentro da faixa de
// 0 a 999, respectivamente, então o vetor também exercita o descarte da
// leitura por nível num UUID que esta biblioteca não produziu.
func TestRFC9562Version7Vector(t *testing.T) {
	u := uuidv7.MustParse(rfc9562Vectors["A.6 versão 7"])
	want := time.Date(2022, 2, 22, 19, 22, 22, 0, time.UTC)

	checkV7(t, "RFC 9562 A.6", u)

	tm, err := uuidv7.ImportBinary(u)
	if err != nil {
		t.Fatalf("ImportBinary recusou o vetor da RFC: %v", err)
	}
	if tm != (uuidv7.Time{Seconds: 1645557742, Milliseconds: 0, Microseconds: 0xCC3, Nanoseconds: 0x18C}) {
		t.Errorf("ImportBinary = %+v, esperado {1645557742 0 3267 396}", tm)
	}
	if fromText, err := uuidv7.Import("017F22E2-79B0-7CC3-98C4-DC0C0C07398F"); err != nil || fromText != tm {
		t.Errorf("Import em maiúsculas, como na RFC = %+v, erro %v; esperado %+v", fromText, err, tm)
	}

	if got, ok := u.Timestamp(); !ok || !got.Equal(want) {
		t.Errorf("Timestamp = %v, %v; esperado %v", got, ok, want)
	}
	for _, level := range []uuidv7.Level{uuidv7.Level1, uuidv7.Level2, uuidv7.Level3} {
		if got, ok := u.TimestampWithLevel(level); !ok || !got.Equal(want) {
			t.Errorf("TimestampWithLevel(%d) = %v, %v; esperado só o milissegundo %v", level, got, ok, want)
		}
	}
}

// TestTimeReadingsRejectNonV7 varre as 64 combinações de nibble de versão
// e código de variante sobre os mesmos 16 bytes e exige que as cinco
// leituras de tempo concordem com IsValid: aceitam só a versão 7 com a
// variante 0b10, e recusam todo o resto — Import e ImportBinary com
// exatamente ErrNotV7 e a estrutura zerada, Timestamp e TimestampWithLevel
// com falso e o instante zero.
//
// A varredura é exaustiva de propósito: uma implementação que confira só
// a versão, ou só a variante, passaria num teste com um único UUIDv4 e
// falharia aqui.
func TestTimeReadingsRejectNonV7(t *testing.T) {
	base := uuidv7.MustParse(canonical)
	levels := []uuidv7.Level{uuidv7.Level1, uuidv7.Level2, uuidv7.Level3, uuidv7.Level(0), uuidv7.Level(9)}

	for version := 0; version < 16; version++ {
		for variant := 0; variant < 4; variant++ {
			u := base
			u[6] = u[6]&0x0F | byte(version)<<4
			u[8] = u[8]&0x3F | byte(variant)<<6
			accept := version == 7 && variant == 0b10

			if u.IsValid() != accept {
				t.Errorf("versão %d, variante %d: IsValid = %v, esperado %v", version, variant, u.IsValid(), accept)
			}

			tm, err := uuidv7.ImportBinary(u)
			text, errText := uuidv7.Import(u.String())
			ts, okTS := u.Timestamp()
			if accept {
				if err != nil || errText != nil || !okTS {
					t.Errorf("versão 7, variante 2: recusado (%v, %v, %v)", err, errText, okTS)
				}
				if text != tm {
					t.Errorf("versão 7, variante 2: Import %+v diverge de ImportBinary %+v", text, tm)
				}
			} else {
				if err != uuidv7.ErrNotV7 || errText != uuidv7.ErrNotV7 { //nolint:errorlint // o sentinela puro é o contrato testado
					t.Errorf("versão %d, variante %d: erros (%v, %v), esperado exatamente ErrNotV7",
						version, variant, err, errText)
				}
				if tm != (uuidv7.Time{}) || text != (uuidv7.Time{}) {
					t.Errorf("versão %d, variante %d: estrutura não zerada (%+v, %+v)", version, variant, tm, text)
				}
				if okTS || !ts.IsZero() {
					t.Errorf("versão %d, variante %d: Timestamp = %v, %v; esperado zero e falso",
						version, variant, ts, okTS)
				}
			}

			for _, level := range levels {
				got, ok := u.TimestampWithLevel(level)
				if ok != accept {
					t.Errorf("versão %d, variante %d, nível %d: TimestampWithLevel ok = %v, esperado %v",
						version, variant, level, ok, accept)
				}
				if !accept && !got.IsZero() {
					t.Errorf("versão %d, variante %d, nível %d: instante %v junto da recusa, esperado zero",
						version, variant, level, got)
				}
			}
		}
	}
}

// TestTimeReadingsRejectRealUUIDsOfOtherVersions repete a recusa sobre
// UUIDs reais, e não montados à mão: os exemplos da RFC 9562 das outras
// versões, o UUID nulo e o UUID com todos os bits em um. É o caso de uso
// que motivou a recusa: um UUIDv4 lido como UUIDv7 devolveria um instante
// sem sentido, e sem erro.
func TestTimeReadingsRejectRealUUIDsOfOtherVersions(t *testing.T) {
	cases := map[string]uuidv7.UUID{
		"nulo":       uuidv7.Nil,
		"bits em um": allOnes,
	}
	for name, s := range rfc9562Vectors {
		if name != "A.6 versão 7" {
			cases["RFC 9562 "+name] = uuidv7.MustParse(s)
		}
	}

	for name, u := range cases {
		if u.IsValid() {
			t.Errorf("%s: IsValid devolveu verdadeiro", name)
		}
		if _, err := uuidv7.ImportBinary(u); !errors.Is(err, uuidv7.ErrNotV7) {
			t.Errorf("%s: ImportBinary devolveu %v, esperado ErrNotV7", name, err)
		}
		if _, err := uuidv7.Import(u.String()); !errors.Is(err, uuidv7.ErrNotV7) {
			t.Errorf("%s: Import devolveu %v, esperado ErrNotV7", name, err)
		}
		if _, ok := u.Timestamp(); ok {
			t.Errorf("%s: Timestamp devolveu verdadeiro", name)
		}
		if _, ok := u.TimestampWithLevel(uuidv7.Level3); ok {
			t.Errorf("%s: TimestampWithLevel devolveu verdadeiro", name)
		}

		// A análise continua sendo só de forma: o mesmo texto é aceito por
		// todos os analisadores, e a recusa fica para a leitura de tempo.
		if back, err := uuidv7.Parse(u.String()); err != nil || back != u {
			t.Errorf("%s: Parse deveria aceitar um UUID de outra versão; obtido %s, erro %v", name, back, err)
		}
		if back, err := uuidv7.FromString(u.String()); err != nil || back != u {
			t.Errorf("%s: FromString deveria aceitar um UUID de outra versão; obtido %s, erro %v", name, back, err)
		}
	}
}
