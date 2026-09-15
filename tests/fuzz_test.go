package tests

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

// FuzzFromString procura entradas que façam FromString entrar em pânico
// ou aceitar uma string que não sobreviva ao round-trip. Rode com:
//
//	go test ./tests/ -run '^$' -fuzz FuzzFromString -fuzztime 30s
func FuzzFromString(f *testing.F) {
	f.Add(canonical)
	f.Add(strings.ToUpper(canonical))
	f.Add("")
	f.Add("00000000-0000-0000-0000-000000000000")
	f.Add("ffffffff-ffff-ffff-ffff-ffffffffffff")
	f.Add("0192f7c5-1a2b-7c3d-8e4f--abbccddeeff") // hífen extra: armadilha 1 de docs/07-armadilhas.md
	f.Add("0192f7c5-1a2b-7c3d-8e4f-aabbccddee-f")
	f.Add(uuidv7.GenerateString(uuidv7.Level3))

	f.Fuzz(func(t *testing.T, s string) {
		// FromString jamais pode entrar em pânico, qualquer que seja a entrada.
		u, err := uuidv7.FromString(s)
		if err != nil {
			if u != (uuidv7.UUID{}) {
				t.Fatalf("FromString(%q) devolveu erro e um UUID não zerado: %v", s, u)
			}
			return
		}

		// Se aceitou, a string reemitida precisa ser a forma canônica
		// minúscula da entrada e voltar aos mesmos 16 bytes.
		canon := u.String()
		if canon != strings.ToLower(s) {
			t.Fatalf("FromString(%q) aceitou, mas String() devolveu %q", s, canon)
		}
		again, err := uuidv7.FromString(canon)
		if err != nil {
			t.Fatalf("round-trip falhou para %q: %v", canon, err)
		}
		if again != u {
			t.Fatalf("round-trip divergiu para %q: %v vs %v", s, u, again)
		}

		// Para uma string já aceita, Import só pode recusar por versão, e
		// recusa exatamente quando IsValid recusa.
		tm, err := uuidv7.Import(s)
		switch {
		case u.IsValid() && err != nil:
			t.Fatalf("Import(%q) recusou um UUIDv7 que FromString aceitou: %v", s, err)
		case !u.IsValid() && err != uuidv7.ErrNotV7: //nolint:errorlint // o sentinela puro é o contrato testado
			t.Fatalf("Import(%q) de UUID não v7 devolveu %v, esperado exatamente ErrNotV7", s, err)
		case !u.IsValid() && tm != (uuidv7.Time{}):
			t.Fatalf("Import(%q) recusou mas devolveu %+v, esperado estrutura zerada", s, tm)
		}
	})
}

// FuzzParse procura entradas que façam o analisador permissivo entrar em
// pânico, aceitar algo que não seja uma das quatro formas do valor lido, ou
// aceitar algo que não sobreviva ao round-trip.
//
// Vale mais que FuzzFromString porque Parse tem quatro caminhos e usa
// aritmética de índice — recortes como v[9:] e v[1:37] só são seguros
// por causa do teste de comprimento que os precede. Rode com:
//
//	go test ./tests/ -run '^$' -fuzz FuzzParse -fuzztime 30s
func FuzzParse(f *testing.F) {
	f.Add(canonical)
	f.Add(strings.ToUpper(canonical))
	f.Add("{" + canonical + "}")
	f.Add("urn:uuid:" + canonical)
	f.Add("URN:UUID:" + canonical)
	f.Add(strings.ReplaceAll(canonical, "-", ""))
	f.Add("")
	f.Add("{")
	f.Add("}")
	f.Add("urn:uuid:")
	f.Add("{" + canonical)
	f.Add(canonical + "}")

	f.Fuzz(func(t *testing.T, s string) {
		// Parse jamais pode entrar em pânico, qualquer que seja a entrada.
		u, err := uuidv7.Parse(s)

		// ParseBytes precisa concordar com Parse em todos os casos.
		ub, errb := uuidv7.ParseBytes([]byte(s))
		if u != ub || (err == nil) != (errb == nil) {
			t.Fatalf("Parse e ParseBytes divergiram para %q: %v/%v e %v/%v", s, u, err, ub, errb)
		}

		// Validate precisa concordar com Parse quanto a aceitar ou recusar.
		if (uuidv7.Validate(s) == nil) != (err == nil) {
			t.Fatalf("Validate divergiu de Parse para %q", s)
		}

		if err != nil {
			if !u.IsZero() {
				t.Fatalf("Parse(%q) devolveu erro e um UUID não zerado: %v", s, u)
			}
			if !errors.Is(err, uuidv7.ErrInvalidFormat) {
				t.Fatalf("Parse(%q): erro %v não é reconhecível como ErrInvalidFormat", s, err)
			}
			return
		}

		// Se aceitou, a entrada é, sem distinção de caixa, exatamente uma das
		// quatro formas escritas a partir do valor lido. A ida e volta sozinha
		// não percebe um hífen ou um dois-pontos que deixaram de ser
		// conferidos: a entrada aceita indevidamente volta ao mesmo valor.
		canon := u.String()
		matched := false
		for _, form := range []string{canon, "{" + canon + "}", "urn:uuid:" + canon, strings.ReplaceAll(canon, "-", "")} {
			if strings.EqualFold(s, form) {
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("Parse(%q) aceitou uma entrada que não é nenhuma das quatro formas de %s", s, canon)
		}

		// Se aceitou, as três reemissões precisam voltar ao mesmo valor.
		for name, text := range map[string]string{
			"String": u.String(),
			"URN":    u.URN(),
		} {
			again, err := uuidv7.Parse(text)
			if err != nil {
				t.Fatalf("round-trip por %s falhou para %q: %v", name, s, err)
			}
			if again != u {
				t.Fatalf("round-trip por %s divergiu para %q: %v vs %v", name, s, u, again)
			}
		}

		// A serialização em texto precisa ser lida de volta sem perda.
		text, err := u.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText falhou para %q: %v", s, err)
		}
		var back uuidv7.UUID
		if err := back.UnmarshalText(text); err != nil || back != u {
			t.Fatalf("volta pelo texto divergiu para %q: %v vs %v, erro %v", s, u, back, err)
		}
	})
}

// FuzzNullUUIDJSON procura entradas que façam NullUUID.UnmarshalJSON
// entrar em pânico ou divergir do tipo UUID lido pelo encoding/json.
// NullUUID tem um leitor de JSON próprio, com caminho direto sem escapes
// e delegação ao encoding/json quando há barra invertida; os dois
// caminhos precisam concordar com o que o encoding/json faria para um
// UUID comum. Rode com:
//
//	go test ./tests/ -run '^$' -fuzz FuzzNullUUIDJSON -fuzztime 30s
func FuzzNullUUIDJSON(f *testing.F) {
	f.Add([]byte(`"` + canonical + `"`))
	f.Add([]byte(`"{` + canonical + `}"`))
	f.Add([]byte(`"urn:uuid:` + canonical + `"`))
	f.Add([]byte(`"` + strings.ReplaceAll(canonical, "-", "") + `"`))
	f.Add([]byte(`"` + strings.ReplaceAll(canonical, "0", `\u0030`) + `"`))
	f.Add([]byte(`"` + strings.ReplaceAll(canonical, "-", `\u002d`) + `"`))
	f.Add([]byte(`"\x00"`))
	f.Add([]byte(`null`))
	f.Add([]byte(`""`))
	f.Add([]byte(`"`))
	f.Add([]byte(`123`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		// A chamada direta jamais pode entrar em pânico, e todo erro dela
		// precisa ser reconhecível como erro de formato.
		var direct uuidv7.NullUUID
		if err := direct.UnmarshalJSON(data); err != nil && !errors.Is(err, uuidv7.ErrInvalidFormat) {
			t.Fatalf("UnmarshalJSON(%q): erro %v não é reconhecível como ErrInvalidFormat", data, err)
		}

		// Pelo encoding/json, NullUUID e UUID precisam aceitar e recusar as
		// mesmas entradas e produzir o mesmo valor. O literal null é a
		// exceção: vira valor ausente em NullUUID e não toca um UUID.
		var viaNull uuidv7.NullUUID
		var viaUUID uuidv7.UUID
		errNull := json.Unmarshal(data, &viaNull)
		errUUID := json.Unmarshal(data, &viaUUID)
		if (errNull == nil) != (errUUID == nil) {
			t.Fatalf("NullUUID e UUID divergiram ao aceitar %q: %v vs %v", data, errNull, errUUID)
		}
		if errNull != nil {
			return
		}
		if viaNull.Valid {
			if viaNull.UUID != viaUUID {
				t.Fatalf("NullUUID e UUID divergiram no valor de %q: %s vs %s", data, viaNull.UUID, viaUUID)
			}
		} else if !viaUUID.IsZero() {
			t.Fatalf("NullUUID ausente mas UUID leu %s de %q", viaUUID, data)
		}
	})
}

// FuzzInstantArithmetic procura instantes que quebrem as invariantes da
// construção a partir de um instante explícito: as fronteiras e a
// geração de docs/04-construcao-por-instante.md.
//
// É o alvo que falta ao lado dos três de texto, e vale por um motivo
// diferente deles. Ali a entrada é uma string e o risco é leitura fora
// dos limites; aqui a entrada é um instante e o risco é aritmético:
// saturação nas duas pontas, o estouro da multiplicação por mil e a
// divisão que decide o piso na época. A suíte cobre esses pontos por
// tabela, com valores escolhidos à mão; o fuzzing varre a faixa inteira
// de segundos, incluindo as duas metades do inteiro com sinal, que é
// onde o estouro troca o sinal.
//
// Rode com:
//
//	go test ./tests/ -run '^$' -fuzz FuzzInstantArithmetic -fuzztime 30s
func FuzzInstantArithmetic(f *testing.F) {
	f.Add(int64(0), int64(0), int64(0), int64(0))
	f.Add(int64(1767225600), int64(123456789), int64(1767225600), int64(123456790))
	f.Add(int64(-1), int64(999999999), int64(0), int64(0))            // travessia da época
	f.Add(int64(1)<<48/1000, int64(0), int64(1)<<48/1000+1, int64(0)) // borda dos 48 bits
	f.Add(int64(1)<<62, int64(0), int64(1)<<62+1, int64(0))           // estouro da multiplicação
	f.Add(-(int64(1) << 62), int64(0), -(int64(1)<<62)+1, int64(0))   // estouro ao contrário
	f.Add(int64(math.MaxInt64/1000), int64(999999999), int64(math.MaxInt64/1000), int64(0))

	f.Fuzz(func(t *testing.T, sec1, nsec1, sec2, nsec2 int64) {
		// time.Unix normaliza a fração, então qualquer par de entrada
		// produz um instante válido; é justamente o que se quer varrer.
		t1 := time.Unix(sec1, nsec1)
		t2 := time.Unix(sec2, nsec2)

		for _, level := range []uuidv7.Level{uuidv7.Level1, uuidv7.Level2, uuidv7.Level3, uuidv7.Level(9)} {
			lo := uuidv7.MinAt(level, t1)
			hi := uuidv7.MaxAt(level, t1)

			// Versão e variante sobrevivem ao preenchimento, nas duas
			// pontas e em qualquer instante. É o que torna a fronteira um
			// limite correto.
			for nome, u := range map[string]uuidv7.UUID{"MinAt": lo, "MaxAt": hi} {
				if u.Version() != 7 {
					t.Fatalf("nível %d, %s(%v): versão %d, esperado 7", level, nome, t1, u.Version())
				}
				if u.Variant() != 0b10 {
					t.Fatalf("nível %d, %s(%v): variante %#b, esperado 10", level, nome, t1, u.Variant())
				}
			}

			// A fronteira inferior nunca passa da superior do mesmo instante.
			if lo.Compare(hi) > 0 {
				t.Fatalf("nível %d, instante %v: MinAt %v acima de MaxAt %v", level, t1, lo, hi)
			}

			// O valor gerado para o instante cai dentro das fronteiras dele.
			gerado := uuidv7.GenerateAt(level, t1)
			if gerado.Compare(lo) < 0 || gerado.Compare(hi) > 0 {
				t.Fatalf("nível %d, instante %v: GenerateAt %v fora de [%v, %v]", level, t1, gerado, lo, hi)
			}

			// Monotonicidade: instante que não regride não produz fronteira
			// que regride. É a propriedade que a saturação por truncamento
			// quebraria em silêncio, e a razão de a saturação levar micro e
			// nano a 999 no teto.
			if !t2.Before(t1) {
				if uuidv7.MinAt(level, t2).Compare(lo) < 0 {
					t.Fatalf("nível %d: instante %v não é anterior a %v, mas MinAt regrediu", level, t2, t1)
				}
				if uuidv7.MaxAt(level, t2).Compare(hi) < 0 {
					t.Fatalf("nível %d: instante %v não é anterior a %v, mas MaxAt regrediu", level, t2, t1)
				}
			}
		}

		// O intervalo semiaberto nunca tem o limite superior abaixo do
		// inferior quando os argumentos estão em ordem.
		if !t2.Before(t1) {
			lo, hi := uuidv7.RangeAt(uuidv7.Level3, t1, t2)
			if lo.Compare(hi) > 0 {
				t.Fatalf("RangeAt(%v, %v) devolveu intervalo invertido: %v acima de %v", t1, t2, lo, hi)
			}
		}
	})
}

// FuzzTimeReading procura UUIDs cuja leitura de tempo quebre as
// invariantes de docs/03-leitura-do-instante.md: é o alvo do sentido
// inverso da geração, e recebe os 16 bytes crus, sem passar por texto.
//
// As invariantes valem para qualquer entrada: as leituras nunca entram em
// pânico; todas concordam com IsValid ao aceitar ou recusar; a extração
// cega devolve os campos crus nas faixas dos bits; e a leitura por nível
// devolve um instante dentro do milissegundo do carimbo, somando os campos
// sub-milissegundo exatamente quando eles cabem em 0 a 999. Por fim, o
// instante lido no Nível 3 regera, por MinAt, os mesmos campos de tempo:
// é a ida e volta que liga a leitura à construção por instante.
//
// Rode com:
//
//	go test ./tests/ -run '^$' -fuzz FuzzTimeReading -fuzztime 30s
func FuzzTimeReading(f *testing.F) {
	for _, s := range rfc9562Vectors {
		f.Add(uuidv7.MustParse(s).Bytes())
	}
	f.Add(uuidv7.MustParse(canonical).Bytes())
	f.Add(uuidv7.Nil.Bytes())
	f.Add(allOnes.Bytes())
	f.Add(uuidv7.MinAt(uuidv7.Level3, time.Unix(0, 0)).Bytes())
	f.Add(uuidv7.MaxAt(uuidv7.Level3, time.Unix(1<<47, 0)).Bytes())
	f.Add(uuidv7.GenerateAt(uuidv7.Level3, time.Unix(1767225600, 123456789)).Bytes())

	f.Fuzz(func(t *testing.T, data []byte) {
		u, err := uuidv7.FromBytes(data)
		if err != nil {
			return
		}

		tm, errImport := uuidv7.ImportBinary(u)
		ts, okTS := u.Timestamp()
		if !u.IsValid() {
			if errImport != uuidv7.ErrNotV7 || tm != (uuidv7.Time{}) { //nolint:errorlint // o sentinela puro é o contrato testado
				t.Fatalf("%s: ImportBinary devolveu %+v, %v; esperado estrutura zerada e ErrNotV7", u, tm, errImport)
			}
			if okTS {
				t.Fatalf("%s: Timestamp aceitou um UUID que IsValid recusa", u)
			}
			for _, level := range []uuidv7.Level{uuidv7.Level1, uuidv7.Level2, uuidv7.Level3} {
				if _, ok := u.TimestampWithLevel(level); ok {
					t.Fatalf("%s: TimestampWithLevel(%d) aceitou um UUID que IsValid recusa", u, level)
				}
			}
			return
		}
		if errImport != nil || !okTS {
			t.Fatalf("%s: UUIDv7 válido recusado: %v, %v", u, errImport, okTS)
		}

		// Extração cega: os campos crus, nas faixas dos bits.
		ms := int64(u[0])<<40 | int64(u[1])<<32 | int64(u[2])<<24 | int64(u[3])<<16 | int64(u[4])<<8 | int64(u[5])
		if tm.Seconds*1_000+int64(tm.Milliseconds) != ms || tm.Milliseconds < 0 || tm.Milliseconds > 999 {
			t.Fatalf("%s: carimbo lido como %+v, esperado %d ms", u, tm, ms)
		}
		if tm.Microseconds < 0 || tm.Microseconds > 4095 || tm.Nanoseconds < 0 || tm.Nanoseconds > 1023 {
			t.Fatalf("%s: campos crus fora da faixa dos bits: %+v", u, tm)
		}
		if ts.UnixMilli() != ms || ts.Nanosecond()%1_000_000 != 0 {
			t.Fatalf("%s: Timestamp = %v, esperado exatamente %d ms", u, ts, ms)
		}

		// Leitura por nível: soma os campos exatamente quando cabem. No
		// Nível 3 os dois caem juntos se qualquer um não couber.
		microOK := tm.Microseconds <= 999
		nanoOK := tm.Nanoseconds <= 999
		var extra2, extra3 time.Duration
		if microOK {
			extra2 = time.Duration(tm.Microseconds) * time.Microsecond
		}
		if microOK && nanoOK {
			extra3 = extra2 + time.Duration(tm.Nanoseconds)
		}
		for _, c := range []struct {
			level uuidv7.Level
			extra time.Duration
		}{
			{uuidv7.Level1, 0},
			{uuidv7.Level2, extra2},
			{uuidv7.Level3, extra3},
			{uuidv7.Level(9), 0},
		} {
			got, ok := u.TimestampWithLevel(c.level)
			if !ok {
				t.Fatalf("%s: TimestampWithLevel(%d) recusou um UUIDv7 válido", u, c.level)
			}
			if want := ts.Add(c.extra); !got.Equal(want) {
				t.Fatalf("%s: TimestampWithLevel(%d) = %v, esperado %v", u, c.level, got, want)
			}
		}

		// Ida e volta: o instante lido no Nível 3 regera os mesmos campos de
		// tempo. Quando os campos sub-milissegundo não cabem, a leitura
		// devolve só o milissegundo e a regeração sai com eles zerados.
		fine, _ := u.TimestampWithLevel(uuidv7.Level3)
		back := mustImportBinary(t, uuidv7.MinAt(uuidv7.Level3, fine))
		want := uuidv7.Time{Seconds: tm.Seconds, Milliseconds: tm.Milliseconds}
		if microOK && nanoOK {
			want.Microseconds, want.Nanoseconds = tm.Microseconds, tm.Nanoseconds
		}
		if back != want {
			t.Fatalf("%s: regerado por MinAt(Nível 3, %v) leu %+v, esperado %+v", u, fine, back, want)
		}
	})
}
