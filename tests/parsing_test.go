package tests

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

// canonical é uma string de UUIDv7 válida usada como base para mutações.
// Os campos abaixo do milissegundo estão fora da faixa de 0 a 999, o que a
// torna também o vetor da extração cega (docs/SPEC.md seção 10).
const canonical = "0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff"

// allOnes é o UUID com os 128 bits em um. Não é UUIDv7 (versão 15,
// variante 3): serve de extremo superior da comparação de bytes e de
// valor que a serialização aceita e as leituras de tempo recusam.
var allOnes = uuidv7.UUID{
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
}

// mustParse falha o teste se a string não for aceita.
func mustParse(t *testing.T, s string) uuidv7.UUID {
	t.Helper()
	u, err := uuidv7.FromString(s)
	if err != nil {
		t.Fatalf("FromString(%q) devolveu erro inesperado: %v", s, err)
	}
	return u
}

// safeFromString executa FromString capturando pânico, para que um
// defeito de limite de índice apareça como falha de teste e não derrube
// toda a suíte.
func safeFromString(s string) (u uuidv7.UUID, panicked any, err error) {
	defer func() { panicked = recover() }()
	u, err = uuidv7.FromString(s)
	return
}

// textForm é uma das quatro formas textuais aceitas por Parse, escrita
// sobre a string canônica de referência.
type textForm struct {
	name      string
	text      string
	bodyStart int  // deslocamento do primeiro caractere do corpo
	hyphens   bool // corpo 8-4-4-4-12, com 36 caracteres, ou 32 dígitos crus
}

// canonicalForm é a única forma que FromString aceita.
var canonicalForm = textForm{"canônica", canonical, 0, true}

// textForms são as quatro formas de canonical que o analisador permissivo
// aceita.
var textForms = []textForm{
	canonicalForm,
	{"entre chaves", "{" + canonical + "}", 1, true},
	{"URN", "urn:uuid:" + canonical, 9, true},
	{"hexadecimal cru", strings.ReplaceAll(canonical, "-", ""), 0, false},
}

// expectMutation é o oráculo das mutações de um byte, independente da
// biblioteca: diz se a forma continua válida com o byte c na posição p e,
// se continua, que valor ela representa, decodificado por encoding/hex.
// Posição de dígito aceita qualquer dígito hexadecimal, em qualquer caixa;
// posição de hífen, só o hífen; as chaves, só elas mesmas; e o prefixo URN,
// as mesmas letras em qualquer caixa e os mesmos dois-pontos.
func expectMutation(f textForm, p int, c byte) (uuidv7.UUID, bool) {
	mutated := []byte(f.text)
	mutated[p] = c
	bodyLen := 32
	if f.hyphens {
		bodyLen = 36
	}
	digits := make([]byte, 0, 32)
	for i, got := range mutated {
		offset := i - f.bodyStart
		switch {
		case offset < 0 || offset >= bodyLen:
			if 'A' <= got && got <= 'Z' {
				got += 'a' - 'A'
			}
			if got != f.text[i] {
				return uuidv7.Nil, false
			}
		case f.hyphens && (offset == 8 || offset == 13 || offset == 18 || offset == 23):
			if got != '-' {
				return uuidv7.Nil, false
			}
		default:
			digits = append(digits, got)
		}
	}
	raw, err := hex.DecodeString(string(digits))
	if err != nil {
		return uuidv7.Nil, false
	}
	return uuidv7.UUID(raw), true
}

// TestFromStringNeverPanics verifica que nenhuma entrada malformada de 36
// caracteres provoca pânico e, sobre as mesmas 36 x 256 mutações de um
// único byte, que FromString aceita exatamente as que mantêm a forma
// canônica, com o valor que o oráculo expectMutation decodifica por conta
// própria. Conferir só a ausência de pânico deixava passar um analisador
// que aceitasse 'G' ou ':' como dígito: uma campanha de mutação mostrou os
// dois defeitos sem nenhuma falha.
//
// REGRESSÃO: antes da correção, um hífen colocado nos deslocamentos
// pares do último grupo (24, 26, 28, 30, 32 e 34) fazia o laço de
// FromString ler s[36] e estourar o índice.
func TestFromStringNeverPanics(t *testing.T) {
	base := []byte(canonical)
	for i := 0; i < len(base); i++ {
		for c := 0; c < 256; c++ {
			mutated := make([]byte, len(base))
			copy(mutated, base)
			mutated[i] = byte(c)
			s := string(mutated)

			u, panicked, err := safeFromString(s)
			if panicked != nil {
				t.Fatalf("FromString entrou em pânico na posição %d com o byte %#x (%q): %v",
					i, c, s, panicked)
			}
			want, accept := expectMutation(canonicalForm, i, byte(c))
			if accept != (err == nil) || u != want {
				t.Fatalf("FromString(%q), posição %d, byte %#x: devolveu %s com erro %v; esperado aceitar = %v, com %s",
					s, i, c, u, err, accept, want)
			}
		}
	}
}

// TestParseSingleByteMutations estende a varredura de mutações de um byte
// às quatro formas do analisador permissivo, com o oráculo exato de
// expectMutation: Parse, ParseBytes e Validate aceitam exatamente as
// mutações que mantêm a forma válida, Parse e ParseBytes devolvem o valor
// decodificado pelo oráculo, e toda recusa é erro de formato com o UUID
// nulo.
//
// Uma campanha de mutação mostrou que Parse podia deixar de conferir o
// hífen das posições 8 ou 23, ou os dois-pontos do prefixo URN, sem
// nenhuma falha. Nem FuzzParse percebia: a entrada aceita volta pela forma
// canônica ao mesmo valor, e a ida e volta é tudo o que ele confere.
func TestParseSingleByteMutations(t *testing.T) {
	for _, f := range textForms {
		for p := 0; p < len(f.text); p++ {
			for c := 0; c < 256; c++ {
				mutated := []byte(f.text)
				mutated[p] = byte(c)
				want, accept := expectMutation(f, p, byte(c))

				got, err := uuidv7.Parse(string(mutated))
				switch {
				case accept && (err != nil || got != want):
					t.Fatalf("forma %s, posição %d, byte %#x (%q): Parse devolveu %s com erro %v, esperado %s",
						f.name, p, c, mutated, got, err, want)
				case !accept && (err == nil || !errors.Is(err, uuidv7.ErrInvalidFormat) || got != uuidv7.Nil):
					t.Fatalf("forma %s, posição %d, byte %#x (%q): Parse devolveu %s com erro %v, esperado a recusa com o UUID nulo",
						f.name, p, c, mutated, got, err)
				}

				if gotBytes, errBytes := uuidv7.ParseBytes(mutated); gotBytes != got || (errBytes == nil) != (err == nil) {
					t.Fatalf("forma %s, posição %d, byte %#x: ParseBytes devolveu %s com erro %v, e Parse %s com erro %v",
						f.name, p, c, gotBytes, errBytes, got, err)
				}
				if errValidate := uuidv7.Validate(string(mutated)); (errValidate == nil) != (err == nil) {
					t.Fatalf("forma %s, posição %d, byte %#x: Validate devolveu %v, e Parse %v",
						f.name, p, c, errValidate, err)
				}
			}
		}
	}
}

// TestFromStringExtraHyphen isola o caso mínimo do defeito acima: uma
// string de 36 caracteres com os quatro hífens canônicos nas posições
// corretas mais um hífen extra dentro do último grupo.
//
// REGRESSÃO: antes da correção, cada um desses casos causava pânico.
func TestFromStringExtraHyphen(t *testing.T) {
	for _, pos := range []int{24, 26, 28, 30, 32, 34} {
		mutated := []byte(canonical)
		mutated[pos] = '-'
		s := string(mutated)

		_, panicked, err := safeFromString(s)
		if panicked != nil {
			t.Errorf("hífen extra na posição %d causou pânico: %v", pos, panicked)
			continue
		}
		if err == nil {
			t.Errorf("hífen extra na posição %d deveria produzir ErrInvalidFormat", pos)
		}
	}
}

// TestFromStringLengthBoundaries confere que tamanhos diferentes de 36
// são rejeitados, inclusive a string vazia e entradas muito longas.
func TestFromStringLengthBoundaries(t *testing.T) {
	for n := 0; n <= 72; n++ {
		if n == 36 {
			continue
		}
		s := strings.Repeat("a", n)
		if _, err := uuidv7.FromString(s); !errors.Is(err, uuidv7.ErrInvalidFormat) {
			t.Fatalf("tamanho %d: esperava ErrInvalidFormat, obteve %v", n, err)
		}
	}
}

// TestFromStringCaseInsensitive garante que maiúsculas e minúsculas
// produzem exatamente os mesmos 16 bytes.
func TestFromStringCaseInsensitive(t *testing.T) {
	lower := mustParse(t, canonical)
	upper := mustParse(t, strings.ToUpper(canonical))
	if lower != upper {
		t.Fatalf("maiúsculas e minúsculas divergiram: %v vs %v", lower, upper)
	}
	// String() sempre reemite em minúsculas.
	if got := upper.String(); got != canonical {
		t.Fatalf("String() = %q, esperado %q", got, canonical)
	}
}

// TestFromStringHyphenPositions confere que os quatro hífens canônicos
// são obrigatórios exatamente nas posições 8, 13, 18 e 23.
func TestFromStringHyphenPositions(t *testing.T) {
	for _, pos := range []int{8, 13, 18, 23} {
		mutated := []byte(canonical)
		mutated[pos] = 'a'
		if _, err := uuidv7.FromString(string(mutated)); !errors.Is(err, uuidv7.ErrInvalidFormat) {
			t.Fatalf("hífen removido da posição %d deveria ser rejeitado", pos)
		}
	}
}

// TestFromStringErrorReturnsZero garante que o valor devolvido junto com
// um erro é sempre o UUID zerado, sem bytes parcialmente preenchidos.
func TestFromStringErrorReturnsZero(t *testing.T) {
	invalid := []string{
		"",
		"abc",
		"0192f7c5-1a2b-7c3d-8e4f-aabbccddeegg",
		"0192f7c5+1a2b-7c3d-8e4f-aabbccddeeff",
		"zzzzzzzz-1a2b-7c3d-8e4f-aabbccddeeff",
	}
	for _, s := range invalid {
		u, err := uuidv7.FromString(s)
		if err == nil {
			t.Fatalf("esperava erro para %q", s)
		}
		if u != (uuidv7.UUID{}) {
			t.Fatalf("FromString(%q) devolveu %v junto com o erro; esperado UUID zerado", s, u)
		}
	}
}

// TestStringRoundTripExtremes exercita os valores de borda dos 16 bytes.
func TestStringRoundTripExtremes(t *testing.T) {
	cases := []uuidv7.UUID{
		{},
		{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x76, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
	}
	for _, want := range cases {
		s := want.String()
		if len(s) != 36 {
			t.Fatalf("String() devolveu %d caracteres: %q", len(s), s)
		}
		got, err := uuidv7.FromString(s)
		if err != nil {
			t.Fatalf("FromString(%q) falhou: %v", s, err)
		}
		if got != want {
			t.Fatalf("round-trip divergiu: %v -> %q -> %v", want, s, got)
		}
	}
}

// TestStringKnownVector fixa a formatação canônica para um valor
// conhecido, travando a posição dos hífens e a ordem dos bytes.
func TestStringKnownVector(t *testing.T) {
	u := uuidv7.UUID{
		0x01, 0x92, 0xf7, 0xc5, 0x1a, 0x2b, 0x7c, 0x3d,
		0x8e, 0x4f, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff,
	}
	if got := u.String(); got != canonical {
		t.Fatalf("String() = %q, esperado %q", got, canonical)
	}
	if got := uuidv7.BinaryToString(u); got != canonical {
		t.Fatalf("BinaryToString() = %q, esperado %q", got, canonical)
	}
}
