package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

// TestParseAcceptsFourFormats confere que Parse aceita as quatro formas
// usuais de escrever um UUID, todas resultando no mesmo valor.
func TestParseAcceptsFourFormats(t *testing.T) {
	reference, err := uuidv7.FromString(canonical)
	if err != nil {
		t.Fatalf("FromString rejeitou a referência: %v", err)
	}

	compact := ""
	for _, c := range canonical {
		if c != '-' {
			compact += string(c)
		}
	}

	cases := map[string]string{
		"canônico":        canonical,
		"maiúsculas":      "0192F7C5-1A2B-7C3D-8E4F-AABBCCDDEEFF",
		"entre chaves":    "{" + canonical + "}",
		"URN":             "urn:uuid:" + canonical,
		"URN maiúsculo":   "URN:UUID:" + canonical,
		"hexadecimal cru": compact,
	}
	for name, input := range cases {
		got, err := uuidv7.Parse(input)
		if err != nil {
			t.Errorf("Parse(%s): erro inesperado %v", name, err)
			continue
		}
		if got != reference {
			t.Errorf("Parse(%s): %s, esperado %s", name, got, reference)
		}
		if got, err := uuidv7.ParseBytes([]byte(input)); err != nil || got != reference {
			t.Errorf("ParseBytes(%s): %s, erro %v", name, got, err)
		}
		if err := uuidv7.Validate(input); err != nil {
			t.Errorf("Validate(%s): erro inesperado %v", name, err)
		}
	}
}

// TestParseRejectsMalformedInput confere as recusas de Parse e garante
// que todo erro continua reconhecível como ErrInvalidFormat.
func TestParseRejectsMalformedInput(t *testing.T) {
	invalid := []string{
		"",
		"nao-e-um-uuid",
		canonical + "0",
		canonical[:35],
		"{" + canonical,
		"urn:uiid:" + canonical,
		"0192f7c5-1a2b-7c3d-8e4f-aabbccddeeg f",
		"0192f7c51a2b7c3d8e4faabbccddeeg0",
	}
	for _, input := range invalid {
		got, err := uuidv7.Parse(input)
		if err == nil {
			t.Errorf("Parse(%q) deveria falhar", input)
			continue
		}
		if !errors.Is(err, uuidv7.ErrInvalidFormat) {
			t.Errorf("Parse(%q): erro %v não é reconhecível como ErrInvalidFormat", input, err)
		}
		if !got.IsZero() {
			t.Errorf("Parse(%q) devolveu %s em vez do UUID nulo", input, got)
		}
	}
}

// TestInvalidLengthIsDistinguishable confere que o erro de comprimento é
// identificável sem deixar de ser um erro de formato.
func TestInvalidLengthIsDistinguishable(t *testing.T) {
	_, err := uuidv7.Parse("abc")
	if !errors.Is(err, uuidv7.ErrInvalidLength) {
		t.Errorf("erro de comprimento não reconhecido: %v", err)
	}
	if !errors.Is(err, uuidv7.ErrInvalidFormat) {
		t.Error("o erro de comprimento deveria continuar sendo um erro de formato")
	}

	_, err = uuidv7.Parse("0192f7c5-1a2b-7c3d-8e4f-aabbccddeezz")
	if errors.Is(err, uuidv7.ErrInvalidLength) {
		t.Error("dígito inválido foi classificado como erro de comprimento")
	}

	// O reconhecimento atravessa camadas que embrulham o erro com %w, e o
	// erro embrulhado continua sendo erro de formato.
	_, err = uuidv7.Parse("abc")
	wrapped := fmt.Errorf("lendo identificador: %w", err)
	if !errors.Is(wrapped, uuidv7.ErrInvalidLength) || !errors.Is(wrapped, uuidv7.ErrInvalidFormat) {
		t.Errorf("erro de comprimento embrulhado não reconhecido: %v", wrapped)
	}
}

// TestFromStringReturnsBareSentinel trava o contrato do analisador
// estrito: FromString devolve exatamente ErrInvalidFormat em toda recusa,
// inclusive para quem compara com o operador de igualdade em vez de
// errors.Is (docs/05-conversao-e-analise.md seção 5).
func TestFromStringReturnsBareSentinel(t *testing.T) {
	for _, input := range []string{"", "curto", canonical + "x", "0192f7c5+1a2b-7c3d-8e4f-aabbccddeeff"} {
		if _, err := uuidv7.FromString(input); err != uuidv7.ErrInvalidFormat { //nolint:errorlint // a comparação direta é o próprio contrato testado
			t.Errorf("FromString(%q): erro %v, esperado exatamente ErrInvalidFormat", input, err)
		}
	}
}

// TestFromStringRejectsExtendedFormats confere que a permissividade fica
// restrita a Parse: FromString aceita só a forma canônica.
func TestFromStringRejectsExtendedFormats(t *testing.T) {
	for _, input := range []string{"{" + canonical + "}", "urn:uuid:" + canonical, "0192f7c51a2b7c3d8e4faabbccddeeff"} {
		if _, err := uuidv7.FromString(input); err == nil {
			t.Errorf("FromString(%q) deveria recusar", input)
		}
	}
}

// TestFromBytes confere a construção a partir de 16 bytes crus.
func TestFromBytes(t *testing.T) {
	reference := uuidv7.Generate(uuidv7.Level3)

	got, err := uuidv7.FromBytes(reference[:])
	if err != nil || got != reference {
		t.Errorf("FromBytes: %s, erro %v", got, err)
	}
	if _, err := uuidv7.FromBytes(reference[:15]); !errors.Is(err, uuidv7.ErrInvalidLength) {
		t.Errorf("FromBytes com 15 bytes: erro %v, esperado erro de comprimento", err)
	}
}

// TestMustParseAndMust confere as variantes que entram em pânico.
func TestMustParseAndMust(t *testing.T) {
	if uuidv7.MustParse(canonical).String() != canonical {
		t.Error("MustParse não devolveu o valor esperado")
	}
	if uuidv7.Must(uuidv7.Parse(canonical)).String() != canonical {
		t.Error("Must não devolveu o valor esperado")
	}

	defer func() {
		if recover() == nil {
			t.Error("MustParse deveria entrar em pânico com entrada inválida")
		}
	}()
	uuidv7.MustParse("nao-e-um-uuid")
}

// TestNil confere o valor nulo: a forma em texto, o predicado e a recusa
// por IsValid, porque o nulo não carrega versão nem variante.
func TestNil(t *testing.T) {
	if uuidv7.Nil.String() != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("Nil: %s", uuidv7.Nil)
	}
	if !uuidv7.Nil.IsZero() {
		t.Error("IsZero devolveu falso para Nil")
	}
	if uuidv7.Nil.IsValid() {
		t.Error("IsValid devolveu verdadeiro para Nil, que não é UUIDv7")
	}
	if allOnes.IsZero() {
		t.Error("IsZero devolveu verdadeiro para o UUID com todos os bits em um")
	}
	if uuidv7.Generate(uuidv7.Level1).IsZero() {
		t.Error("um UUID gerado nunca deveria ser nulo")
	}
}

// TestCompareOrdersLikeStrings confere que Compare concorda com a ordem
// lexicográfica das strings canônicas, sobre pares aleatórios que caem
// dos dois lados, e que serve diretamente a slices.SortFunc.
func TestCompareOrdersLikeStrings(t *testing.T) {
	if uuidv7.Nil.Compare(allOnes) != -1 {
		t.Error("Nil deveria vir antes do UUID com todos os bits em um")
	}
	if allOnes.Compare(uuidv7.Nil) != 1 {
		t.Error("o UUID com todos os bits em um deveria vir depois de Nil")
	}
	if allOnes.Compare(allOnes) != 0 {
		t.Error("Compare de um valor com ele mesmo deveria ser zero")
	}

	// Pares de Nível 1 com o carimbo sobrescrito por entropia: sem isso,
	// quase todos cairiam no mesmo milissegundo e a comparação só olharia
	// para o fim dos bytes.
	previous := uuidv7.Generate(uuidv7.Level1)
	for i := 0; i < 1_000; i++ {
		current := uuidv7.Generate(uuidv7.Level1)
		noise := uuidv7.Generate(uuidv7.Level1)
		copy(current[0:6], noise[10:16])
		byBytes := previous.Compare(current)
		byText := 0
		switch {
		case previous.String() < current.String():
			byText = -1
		case previous.String() > current.String():
			byText = 1
		}
		if byBytes != byText {
			t.Fatalf("divergência na comparação %d: bytes %d, texto %d", i, byBytes, byText)
		}
		if current.Compare(previous) != -byBytes {
			t.Fatalf("comparação %d não é antissimétrica", i)
		}
		previous = current
	}

	// A forma documentada de ordenar uma lista precisa dar a mesma ordem
	// das strings.
	list := make([]uuidv7.UUID, 500)
	texts := make([]string, len(list))
	for i := range list {
		list[i] = uuidv7.Generate(uuidv7.Level3)
		noise := uuidv7.Generate(uuidv7.Level1)
		copy(list[i][0:6], noise[10:16])
		texts[i] = list[i].String()
	}
	slices.SortFunc(list, uuidv7.UUID.Compare)
	sort.Strings(texts)
	for i := range list {
		if list[i].String() != texts[i] {
			t.Fatalf("slices.SortFunc com Compare divergiu da ordem das strings na posição %d", i)
		}
	}
}

// TestURN confere a forma URN e o caminho de volta por Parse.
func TestURN(t *testing.T) {
	u := uuidv7.MustParse(canonical)
	urn := u.URN()
	if urn != "urn:uuid:"+canonical {
		t.Errorf("URN: %s", urn)
	}
	if back, err := uuidv7.Parse(urn); err != nil || back != u {
		t.Errorf("volta pela URN: %s, erro %v", back, err)
	}
}

// TestTextMarshaling confere as interfaces de texto da biblioteca padrão.
func TestTextMarshaling(t *testing.T) {
	u := uuidv7.MustParse(canonical)

	text, err := u.MarshalText()
	if err != nil || string(text) != canonical {
		t.Fatalf("MarshalText: %q, erro %v", text, err)
	}

	var back uuidv7.UUID
	if err := back.UnmarshalText(text); err != nil || back != u {
		t.Errorf("UnmarshalText: %s, erro %v", back, err)
	}
	// A leitura de texto aceita os mesmos formatos de Parse.
	if err := back.UnmarshalText([]byte("{" + canonical + "}")); err != nil || back != u {
		t.Errorf("UnmarshalText entre chaves: %s, erro %v", back, err)
	}

	original := back
	if err := back.UnmarshalText([]byte("invalido")); err == nil {
		t.Error("UnmarshalText deveria recusar entrada inválida")
	} else if back != original {
		t.Error("UnmarshalText alterou o receptor mesmo falhando")
	}
}

// TestAppendTo confere a escrita no buffer do chamador: o conteúdo
// anexado, a preservação do que já estava em dst, o buffer nulo e a
// reutilização com dst[:0].
func TestAppendTo(t *testing.T) {
	u := uuidv7.MustParse(canonical)

	if got := string(u.AppendTo(nil)); got != canonical {
		t.Errorf("AppendTo(nil) = %q, esperado %q", got, canonical)
	}

	prefixo := []byte("id=")
	if got := string(u.AppendTo(prefixo)); got != "id="+canonical {
		t.Errorf("AppendTo sobre prefixo = %q, esperado %q", got, "id="+canonical)
	}
	if string(prefixo) != "id=" {
		t.Errorf("AppendTo alterou o comprimento de dst: %q", prefixo)
	}

	// Reutilizar o mesmo buffer é o uso previsto pela documentação.
	buf := make([]byte, 0, 64)
	for _, esperado := range []uuidv7.UUID{u, uuidv7.Nil, allOnes} {
		buf = esperado.AppendTo(buf[:0])
		if string(buf) != esperado.String() {
			t.Errorf("AppendTo reutilizando o buffer = %q, esperado %q", buf, esperado)
		}
	}

	// AppendText é a mesma escrita com a assinatura de encoding.TextAppender.
	texto, err := u.AppendText([]byte("x"))
	if err != nil || string(texto) != "x"+canonical {
		t.Errorf("AppendText = %q, erro %v", texto, err)
	}
}

// TestAppendBinary confere a escrita dos 16 bytes crus no fim do buffer
// do chamador, que é a assinatura de encoding.BinaryAppender. O conteúdo
// tem de ser idêntico ao de MarshalBinary: se divergirem, um dos dois
// caminhos de serialização binária está errado.
func TestAppendBinary(t *testing.T) {
	u := uuidv7.MustParse(canonical)

	cru, err := u.AppendBinary(nil)
	if err != nil {
		t.Fatalf("AppendBinary(nil) devolveu erro %v, esperado nulo", err)
	}
	if !bytes.Equal(cru, u[:]) {
		t.Errorf("AppendBinary(nil) = %x, esperado %x", cru, u[:])
	}

	marshalado, err := u.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary devolveu erro %v", err)
	}
	if !bytes.Equal(cru, marshalado) {
		t.Errorf("AppendBinary = %x, mas MarshalBinary = %x; os dois têm de coincidir", cru, marshalado)
	}

	prefixo := []byte{0xAA}
	comPrefixo, err := u.AppendBinary(prefixo)
	if err != nil {
		t.Fatalf("AppendBinary sobre prefixo devolveu erro %v", err)
	}
	if !bytes.Equal(comPrefixo, append([]byte{0xAA}, u[:]...)) {
		t.Errorf("AppendBinary sobre prefixo = %x", comPrefixo)
	}
	if len(prefixo) != 1 {
		t.Errorf("AppendBinary alterou o comprimento de dst: %x", prefixo)
	}

	// A ida e volta fecha com UnmarshalBinary, e os dois extremos entram
	// porque não carregam versão nem variante: a serialização não confere
	// versão.
	for _, esperado := range []uuidv7.UUID{u, uuidv7.Nil, allOnes} {
		buf, err := esperado.AppendBinary(make([]byte, 0, 16))
		if err != nil {
			t.Fatalf("%v: AppendBinary devolveu erro %v", esperado, err)
		}
		var volta uuidv7.UUID
		if err := volta.UnmarshalBinary(buf); err != nil {
			t.Fatalf("%v: UnmarshalBinary devolveu erro %v", esperado, err)
		}
		if volta != esperado {
			t.Errorf("ida e volta binária = %v, esperado %v", volta, esperado)
		}
	}
}

// TestBytes confere que Bytes devolve uma cópia independente dos 16
// bytes, ao contrário de u[:].
func TestBytes(t *testing.T) {
	u := uuidv7.MustParse(canonical)

	raw := u.Bytes()
	if len(raw) != 16 {
		t.Fatalf("Bytes devolveu %d bytes, esperado 16", len(raw))
	}
	if uuidv7.UUID(raw[0:16]) != u {
		t.Errorf("Bytes = %x, esperado %x", raw, u[:])
	}

	raw[0] = 0xFF
	if u[0] == 0xFF {
		t.Error("Bytes devolveu uma fatia sobre o próprio valor de origem")
	}
}

// TestIsValid confere a semântica adotada: IsValid reconhece um UUIDv7,
// isto é, versão 7 com a variante da RFC 9562, e nada mais. A varredura
// exaustiva das combinações está em TestTimeReadingsRejectNonV7.
func TestIsValid(t *testing.T) {
	validos := map[string]uuidv7.UUID{
		"nível 1":          uuidv7.Generate(uuidv7.Level1),
		"nível 2":          uuidv7.Generate(uuidv7.Level2),
		"nível 3":          uuidv7.Generate(uuidv7.Level3),
		"fronteira mínima": uuidv7.MinAt(uuidv7.Level1, baseInstant),
		"fronteira máxima": uuidv7.MaxAt(uuidv7.Level1, baseInstant),
		"canônico":         uuidv7.MustParse(canonical),
		"RFC 9562 A.6":     uuidv7.MustParse(rfc9562Vectors["A.6 versão 7"]),
	}
	for nome, u := range validos {
		if !u.IsValid() {
			t.Errorf("%s: IsValid devolveu falso para %s", nome, u)
		}
	}

	base := uuidv7.MustParse(canonical)

	versaoQuatro := base
	versaoQuatro[6] = (versaoQuatro[6] & 0x0F) | 0x40
	if versaoQuatro.IsValid() {
		t.Error("IsValid deveria recusar versão 4")
	}

	for _, variante := range []byte{0x00, 0x40, 0xC0, 0xE0} {
		naoRFC := base
		naoRFC[8] = (naoRFC[8] & 0x3F) | variante
		if naoRFC.IsValid() {
			t.Errorf("IsValid deveria recusar a variante %#02x (codigo %d)", variante, naoRFC.Variant())
		}
	}

	if uuidv7.Nil.IsValid() || allOnes.IsValid() {
		t.Error("IsValid deveria recusar o nulo e o UUID com todos os bits em um")
	}
}

// TestVersionAndVariantReadTheirBits confere que Version e Variant leem os
// campos como estão, para qualquer valor, e não só nos UUIDv7 que a
// biblioteca produz (docs/03-leitura-do-instante.md seção 5). Nenhum
// teste chamava os dois sobre um UUID de outra versão, e uma campanha de
// mutação mostrou que um Version com 7 fixo, ou um Variant com 2 fixo,
// passava pela suíte inteira.
// Os demais bits do byte não podem interferir: cada valor é conferido sobre
// o UUID nulo e sobre o UUID com todos os bits em um.
func TestVersionAndVariantReadTheirBits(t *testing.T) {
	for _, base := range []uuidv7.UUID{uuidv7.Nil, allOnes} {
		for version := 0; version < 16; version++ {
			u := base
			u[6] = u[6]&0x0F | byte(version)<<4
			if got := u.Version(); got != byte(version) {
				t.Errorf("byte 6 = %#02x: Version = %d, esperado %d", u[6], got, version)
			}
		}
		for variant := 0; variant < 4; variant++ {
			u := base
			u[8] = u[8]&0x3F | byte(variant)<<6
			if got := u.Variant(); got != byte(variant) {
				t.Errorf("byte 8 = %#02x: Variant = %d, esperado %d", u[8], got, variant)
			}
		}
	}
}

// TestCompareAndIsZeroSeeEveryByte confere que Compare e IsZero olham as 16
// posições. Os testes de ordenação comparam UUIDs que já diferem nos
// primeiros bytes, e os de IsZero usam valores que diferem do nulo logo no
// primeiro: uma campanha de mutação mostrou que um Compare que ignorasse o
// último byte, ou um IsZero que só olhasse as pontas, passava pela suíte.
// Cada posição é conferida sozinha, com os demais bytes em zero e em um, e
// com 0x7f contra 0x80, que também pega comparação com sinal.
func TestCompareAndIsZeroSeeEveryByte(t *testing.T) {
	for i := 0; i < 16; i++ {
		for _, base := range []uuidv7.UUID{uuidv7.Nil, allOnes} {
			lower, upper := base, base
			lower[i], upper[i] = 0x7F, 0x80
			if got := lower.Compare(upper); got != -1 {
				t.Errorf("byte %d, 0x7f contra 0x80: Compare = %d, esperado -1", i, got)
			}
			if got := upper.Compare(lower); got != 1 {
				t.Errorf("byte %d, 0x80 contra 0x7f: Compare = %d, esperado 1", i, got)
			}
			if got := upper.Compare(upper); got != 0 {
				t.Errorf("byte %d: Compare de um valor com ele mesmo = %d, esperado 0", i, got)
			}
		}

		var single uuidv7.UUID
		single[i] = 0x01
		if single.IsZero() {
			t.Errorf("IsZero devolveu verdadeiro com o byte %d diferente de zero", i)
		}
	}

	// O menor UUIDv7 que existe começa e termina em zero, e não é o nulo:
	// a versão e a variante ficam no meio.
	if uuidv7.MinAt(uuidv7.Level1, time.Unix(0, 0)).IsZero() {
		t.Error("IsZero devolveu verdadeiro para a fronteira mínima da época, que é UUIDv7")
	}
}

// safeMustParse executa MustParse capturando o pânico, para que uma recusa
// indevida apareça como falha do caso e não derrube a suíte.
func safeMustParse(s string) (u uuidv7.UUID, panicked any) {
	defer func() { panicked = recover() }()
	return uuidv7.MustParse(s), nil
}

// TestScanAndMustParseAcceptEveryParseFormat confere o que a documentação
// de Scan e de MustParse promete (docs/05-conversao-e-analise.md seção 4 e
// docs/06-serializacao-e-banco.md seção 3): qualquer
// formato aceito por Parse, e não só o canônico. Os testes de banco e de
// MustParse usavam só a forma canônica, e uma campanha de mutação mostrou
// que restringir os dois a ela passava pela suíte. A leitura de banco é
// conferida nos quatro tipos, com o texto em string e em fatia de bytes.
func TestScanAndMustParseAcceptEveryParseFormat(t *testing.T) {
	reference := uuidv7.MustParse(canonical)
	forms := map[string]string{
		"canônica":        canonical,
		"maiúsculas":      strings.ToUpper(canonical),
		"entre chaves":    "{" + canonical + "}",
		"URN":             "urn:uuid:" + canonical,
		"URN maiúsculo":   "URN:UUID:" + canonical,
		"hexadecimal cru": strings.ReplaceAll(canonical, "-", ""),
	}
	for name, text := range forms {
		if got, panicked := safeMustParse(text); panicked != nil || got != reference {
			t.Errorf("MustParse(%s): %s, pânico %v; esperado %s", name, got, panicked, reference)
		}
		for _, src := range []any{text, []byte(text)} {
			var u uuidv7.UUID
			if err := u.Scan(src); err != nil || u != reference {
				t.Errorf("UUID.Scan(%s, %T): %s, erro %v", name, src, u, err)
			}
			var b uuidv7.BinaryUUID
			if err := b.Scan(src); err != nil || uuidv7.UUID(b) != reference {
				t.Errorf("BinaryUUID.Scan(%s, %T): %s, erro %v", name, src, b, err)
			}
			var n uuidv7.NullUUID
			if err := n.Scan(src); err != nil || !n.Valid || n.UUID != reference {
				t.Errorf("NullUUID.Scan(%s, %T): %+v, erro %v", name, src, n, err)
			}
			var nb uuidv7.NullBinaryUUID
			if err := nb.Scan(src); err != nil || !nb.Valid || nb.UUID != reference {
				t.Errorf("NullBinaryUUID.Scan(%s, %T): %+v, erro %v", name, src, nb, err)
			}
		}
	}
}

// TestBinaryMarshaling confere as interfaces binárias da biblioteca
// padrão, inclusive a garantia de que o resultado não aponta para o valor
// de origem.
func TestBinaryMarshaling(t *testing.T) {
	u := uuidv7.MustParse(canonical)

	raw, err := u.MarshalBinary()
	if err != nil || len(raw) != 16 {
		t.Fatalf("MarshalBinary: %d bytes, erro %v", len(raw), err)
	}
	raw[0] = 0xFF
	if u[0] == 0xFF {
		t.Error("MarshalBinary devolveu uma fatia sobre o próprio valor de origem")
	}

	raw, _ = u.MarshalBinary()
	var back uuidv7.UUID
	if err := back.UnmarshalBinary(raw); err != nil || back != u {
		t.Errorf("UnmarshalBinary: %s, erro %v", back, err)
	}
	if err := back.UnmarshalBinary(raw[:10]); !errors.Is(err, uuidv7.ErrInvalidLength) {
		t.Errorf("UnmarshalBinary com 10 bytes: erro %v, esperado erro de comprimento", err)
	}
}

// TestJSONRoundTrip confere que o UUID viaja como string em JSON, e não
// como lista de 16 números.
func TestJSONRoundTrip(t *testing.T) {
	type record struct {
		ID uuidv7.UUID `json:"id"`
	}
	original := record{ID: uuidv7.MustParse(canonical)}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if string(encoded) != `{"id":"`+canonical+`"}` {
		t.Errorf("JSON gerado: %s", encoded)
	}

	var back record
	if err := json.Unmarshal(encoded, &back); err != nil || back != original {
		t.Errorf("json.Unmarshal: %+v, erro %v", back, err)
	}
}

// TestSQLScanAndValue confere a integração com database/sql.
func TestSQLScanAndValue(t *testing.T) {
	reference := uuidv7.MustParse(canonical)

	value, err := reference.Value()
	if err != nil || value != canonical {
		t.Errorf("Value: %v, erro %v", value, err)
	}

	var u uuidv7.UUID
	if err := u.Scan(canonical); err != nil || u != reference {
		t.Errorf("Scan de string: %s, erro %v", u, err)
	}
	if err := u.Scan([]byte(canonical)); err != nil || u != reference {
		t.Errorf("Scan de texto em bytes: %s, erro %v", u, err)
	}
	if err := u.Scan(reference[:]); err != nil || u != reference {
		t.Errorf("Scan de 16 bytes: %s, erro %v", u, err)
	}
	if err := u.Scan(nil); err != nil || !u.IsZero() {
		t.Errorf("Scan de NULL: %s, erro %v", u, err)
	}
	if err := u.Scan(42); !errors.Is(err, uuidv7.ErrInvalidScanType) {
		t.Errorf("Scan de inteiro: erro %v, esperado ErrInvalidScanType", err)
	}

	// Texto vazio é ausência de valor: grava o UUID nulo e não devolve erro.
	// Colunas de texto com a string vazia como valor padrão dependem disso.
	u = reference
	if err := u.Scan(""); err != nil || !u.IsZero() {
		t.Errorf("Scan de string vazia: %s, erro %v, esperado UUID nulo sem erro", u, err)
	}
	u = reference
	if err := u.Scan([]byte{}); err != nil || !u.IsZero() {
		t.Errorf("Scan de bytes vazios: %s, erro %v, esperado UUID nulo sem erro", u, err)
	}

	// Em caso de erro o receptor não é alterado.
	u = reference
	if err := u.Scan("invalido"); err == nil || u != reference {
		t.Errorf("Scan de texto inválido: %s, erro %v, esperado erro sem alterar o receptor", u, err)
	}

	// Fatia de bytes que não está vazia nem tem 16 bytes cai no caminho de
	// texto, e um texto inválido ali precisa falhar sem alterar o receptor.
	// É o ramo que distingue "bytes crus" de "texto em bytes".
	u = reference
	if err := u.Scan([]byte("nao-e-um-uuid")); !errors.Is(err, uuidv7.ErrInvalidFormat) || u != reference {
		t.Errorf("Scan de bytes de texto inválido: %s, erro %v, esperado erro de formato sem alterar o receptor", u, err)
	}

	// A leitura de banco confere só a forma, como Parse: um UUID de outra
	// versão é lido sem erro, e a recusa fica para as leituras de tempo.
	v4 := rfc9562Vectors["A.3 versão 4"]
	if err := u.Scan(v4); err != nil || u.String() != v4 {
		t.Errorf("Scan de UUIDv4: %s, erro %v, esperado a leitura sem erro", u, err)
	}
}

// TestNullUUID confere o tipo que aceita coluna nula.
func TestNullUUID(t *testing.T) {
	reference := uuidv7.MustParse(canonical)

	var n uuidv7.NullUUID
	if err := n.Scan(nil); err != nil || n.Valid {
		t.Errorf("Scan de NULL: %+v, erro %v", n, err)
	}
	if value, err := n.Value(); err != nil || value != nil {
		t.Errorf("Value sem valor: %v, erro %v", value, err)
	}
	if encoded, err := json.Marshal(n); err != nil || string(encoded) != "null" {
		t.Errorf("JSON sem valor: %s, erro %v", encoded, err)
	}

	if err := n.Scan(canonical); err != nil || !n.Valid || n.UUID != reference {
		t.Errorf("Scan com valor: %+v, erro %v", n, err)
	}

	// Escapes JSON precisam ser interpretados como o encoding/json faria
	// para o tipo UUID: \u0030 vale 0, \u002d vale o hífen. Passar o
	// conteúdo entre aspas cru a ParseBytes recusaria qualquer escape como
	// formato inválido.
	escaped := `"\u0030192f7c5\u002d1a2b-7c3d-8e4f-aabbccddeeff"`
	var fromEscaped uuidv7.NullUUID
	if err := json.Unmarshal([]byte(escaped), &fromEscaped); err != nil || !fromEscaped.Valid || fromEscaped.UUID != reference {
		t.Errorf("JSON com escapes: %+v, erro %v, esperado %s válido", fromEscaped, err, reference)
	}
	// O mesmo JSON precisa produzir o mesmo valor no tipo UUID.
	var plain uuidv7.UUID
	if err := json.Unmarshal([]byte(escaped), &plain); err != nil || plain != fromEscaped.UUID {
		t.Errorf("UUID e NullUUID divergiram ao ler escapes: %s vs %s, erro %v", plain, fromEscaped.UUID, err)
	}
	// Escape inválido e conteúdo inválido após o escape são erros de
	// formato, e o receptor não é alterado. A chamada é direta ao método
	// porque json.Unmarshal rejeita o documento com escape inválido antes
	// de chegar a ele; o que se testa aqui é a conversão feita pelo método.
	kept := fromEscaped
	for _, bad := range []string{`"\x0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff"`, `"\u0030192f7c5-1a2b-7c3d-8e4f-aabbccddeezz"`} {
		err := fromEscaped.UnmarshalJSON([]byte(bad))
		if !errors.Is(err, uuidv7.ErrInvalidFormat) {
			t.Errorf("JSON %s: erro %v, esperado ErrInvalidFormat", bad, err)
		}
		if fromEscaped != kept {
			t.Errorf("JSON %s alterou o receptor mesmo falhando: %+v", bad, fromEscaped)
		}
	}

	// Texto vazio produz valor ausente, sem erro.
	for name, empty := range map[string]any{"string vazia": "", "bytes vazios": []byte{}} {
		n.Scan(canonical) //nolint:errcheck // reposiciona um valor válido antes de cada caso
		if err := n.Scan(empty); err != nil || n.Valid || !n.UUID.IsZero() {
			t.Errorf("Scan de %s: %+v, erro %v, esperado valor ausente sem erro", name, n, err)
		}
	}
	if err := n.Scan("invalido"); err == nil || n.Valid {
		t.Errorf("Scan de texto inválido: %+v, erro %v, esperado erro com Valid falso", n, err)
	}

	if err := n.Scan(canonical); err != nil || !n.Valid || n.UUID != reference {
		t.Errorf("Scan com valor após ausência: %+v, erro %v", n, err)
	}
	encoded, err := json.Marshal(n)
	if err != nil || string(encoded) != `"`+canonical+`"` {
		t.Fatalf("JSON com valor: %s, erro %v", encoded, err)
	}

	var back uuidv7.NullUUID
	if err := json.Unmarshal(encoded, &back); err != nil || back != n {
		t.Errorf("volta pelo JSON: %+v, erro %v", back, err)
	}
	if err := json.Unmarshal([]byte("null"), &back); err != nil || back.Valid {
		t.Errorf("volta de null: %+v, erro %v", back, err)
	}
}

// TestNullUUIDRejectsUnsupportedType confere que um valor de tipo
// inesperado não é confundido com ausência de valor: o auxiliar que
// reconhece NULL, texto vazio e fatia vazia precisa recusar qualquer
// outro tipo, e o erro tem de chegar ao chamador com Valid falso.
func TestNullUUIDRejectsUnsupportedType(t *testing.T) {
	n := uuidv7.NullUUID{UUID: uuidv7.MustParse(canonical), Valid: true}
	if err := n.Scan(42); !errors.Is(err, uuidv7.ErrInvalidScanType) {
		t.Errorf("Scan de inteiro: erro %v, esperado ErrInvalidScanType", err)
	}
	if n.Valid {
		t.Error("Scan com erro deveria deixar Valid falso")
	}

	// Um float também não é ausência de valor, nem um booleano: o auxiliar
	// só trata NULL, string e fatia de bytes.
	for _, src := range []any{3.14, true, struct{}{}} {
		var outro uuidv7.NullUUID
		if err := outro.Scan(src); !errors.Is(err, uuidv7.ErrInvalidScanType) {
			t.Errorf("Scan de %T: erro %v, esperado ErrInvalidScanType", src, err)
		}
		if outro.Valid {
			t.Errorf("Scan de %T deveria deixar Valid falso", src)
		}
	}
}

// TestGeneratorWithReader confere a fonte de entropia em formato io.Reader.
func TestGeneratorWithReader(t *testing.T) {
	// Dezesseis bytes zerados: o nível 1 consome as duas palavras, e todos
	// os bits aleatórios saem em zero. Sobram apenas tempo, versão e
	// variante.
	g := uuidv7.NewGeneratorWithReader(bytes.NewReader(make([]byte, 16)))
	u := g.Generate(uuidv7.Level1)
	if u.Version() != 7 || u.Variant() != 0b10 {
		t.Fatalf("UUID malformado: %s", u)
	}
	if u[6]&0x0F != 0 || u[7] != 0 {
		t.Errorf("rand_a deveria estar zerado: %s", u)
	}

	// Fonte curta: a leitura falha no meio da geração.
	defer func() {
		if recover() == nil {
			t.Error("uma fonte de entropia esgotada deveria entrar em pânico")
		}
	}()
	uuidv7.NewGeneratorWithReader(bytes.NewReader(make([]byte, 4))).Generate(uuidv7.Level1)
}

// TestGeneratorWithNilReaderPanics confere que o leitor nulo é rejeitado
// na configuração, e não na primeira geração.
func TestGeneratorWithNilReaderPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("NewGeneratorWithReader(nil) deveria entrar em pânico")
		}
	}()
	uuidv7.NewGeneratorWithReader(nil)
}

// TestCryptoGenerator confere que o gerador criptográfico produz UUIDs
// válidos em todos os níveis e nos nomes do UUIDv7 por versão e por nível.
func TestCryptoGenerator(t *testing.T) {
	g := uuidv7.NewCryptoGenerator()
	for _, level := range []uuidv7.Level{uuidv7.Level1, uuidv7.Level2, uuidv7.Level3} {
		u := g.Generate(level)
		if u.Version() != 7 || u.Variant() != 0b10 {
			t.Errorf("nível %d: UUID malformado %s", level, u)
		}
	}
	if u := g.GenerateV7(); u.Version() != 7 || u.Variant() != 0b10 {
		t.Errorf("versão 7 pelo nome malformada: %s", u)
	}
	if u := g.GenerateV7Level1(); u.Version() != 7 || u.Variant() != 0b10 {
		t.Errorf("Nível 1 pelo nome malformado: %s", u)
	}
	if u := g.GenerateV7Level2(); u.Version() != 7 || u.Variant() != 0b10 {
		t.Errorf("Nível 2 pelo nome malformado: %s", u)
	}
	if u := g.GenerateV7Level3(); u.Version() != 7 || u.Variant() != 0b10 {
		t.Errorf("Nível 3 pelo nome malformado: %s", u)
	}
	if u := g.GenerateAt(uuidv7.Level3, baseInstant); !u.IsValid() {
		t.Errorf("geração por instante malformada: %s", u)
	}
}

// TestNullUUIDTextAndBinary confere as serializações em texto e binário
// de NullUUID, nos dois estados: com valor e ausente. Entrada vazia
// produz valor ausente sem erro; entrada inválida devolve erro e deixa
// Valid falso.
func TestNullUUIDTextAndBinary(t *testing.T) {
	reference := uuidv7.MustParse(canonical)
	present := uuidv7.NullUUID{UUID: reference, Valid: true}
	var absent uuidv7.NullUUID

	// Texto.
	if text, err := present.MarshalText(); err != nil || string(text) != canonical {
		t.Errorf("MarshalText com valor: %q, erro %v", text, err)
	}
	if text, err := absent.MarshalText(); err != nil || len(text) != 0 {
		t.Errorf("MarshalText sem valor: %q, erro %v, esperado vazio", text, err)
	}
	var fromText uuidv7.NullUUID
	if err := fromText.UnmarshalText([]byte("{" + canonical + "}")); err != nil || fromText != present {
		t.Errorf("UnmarshalText entre chaves: %+v, erro %v", fromText, err)
	}
	if err := fromText.UnmarshalText(nil); err != nil || fromText.Valid || !fromText.UUID.IsZero() {
		t.Errorf("UnmarshalText vazio: %+v, erro %v, esperado valor ausente", fromText, err)
	}
	// Entrada inválida não toca o receptor, como em UnmarshalJSON: o valor
	// que já estava presente continua presente e intacto.
	fromText = present
	if err := fromText.UnmarshalText([]byte("nao-e-um-uuid")); !errors.Is(err, uuidv7.ErrInvalidFormat) || fromText != present {
		t.Errorf("UnmarshalText inválido: %+v, erro %v, esperado ErrInvalidFormat sem alterar o receptor", fromText, err)
	}

	// Binário.
	if raw, err := present.MarshalBinary(); err != nil || !bytes.Equal(raw, reference[:]) {
		t.Errorf("MarshalBinary com valor: %x, erro %v", raw, err)
	}
	if raw, err := absent.MarshalBinary(); err != nil || len(raw) != 0 {
		t.Errorf("MarshalBinary sem valor: %x, erro %v, esperado vazio", raw, err)
	}
	var fromBinary uuidv7.NullUUID
	if err := fromBinary.UnmarshalBinary(reference[:]); err != nil || fromBinary != present {
		t.Errorf("UnmarshalBinary: %+v, erro %v", fromBinary, err)
	}
	if err := fromBinary.UnmarshalBinary([]byte{}); err != nil || fromBinary.Valid || !fromBinary.UUID.IsZero() {
		t.Errorf("UnmarshalBinary vazio: %+v, erro %v, esperado valor ausente", fromBinary, err)
	}
	fromBinary = present
	if err := fromBinary.UnmarshalBinary(reference[:15]); !errors.Is(err, uuidv7.ErrInvalidLength) || fromBinary != present {
		t.Errorf("UnmarshalBinary com 15 bytes: %+v, erro %v, esperado ErrInvalidLength sem alterar o receptor", fromBinary, err)
	}

	// Value com valor presente grava a string canônica.
	if value, err := present.Value(); err != nil || value != canonical {
		t.Errorf("Value com valor: %v, erro %v", value, err)
	}
}

// TestNullUUIDReceiverPolicyOnError trava a regra de receptor do tipo
// anulável em erro, incluindo a única exceção, para que uma "uniformização"
// futura tenha de passar por aqui.
//
// Os três desserializadores (JSON, texto e binário) preservam o receptor
// inteiro, seguindo a regra mestra de
// docs/05-conversao-e-analise.md seção 5. Scan é a
// exceção registrada: preserva o identificador e derruba o booleano,
// porque database/sql reaproveita o mesmo destino a cada linha e um
// chamador que ignore o erro leria o valor da linha anterior.
func TestNullUUIDReceiverPolicyOnError(t *testing.T) {
	reference := uuidv7.MustParse(canonical)
	present := uuidv7.NullUUID{UUID: reference, Valid: true}

	// Desserializadores: o receptor sai intacto.
	preservam := []struct {
		nome string
		call func(*uuidv7.NullUUID) error
	}{
		{"UnmarshalJSON", func(n *uuidv7.NullUUID) error { return n.UnmarshalJSON([]byte(`"nao-e-um-uuid"`)) }},
		{"UnmarshalJSON com 42", func(n *uuidv7.NullUUID) error { return n.UnmarshalJSON([]byte(`42`)) }},
		{"UnmarshalText", func(n *uuidv7.NullUUID) error { return n.UnmarshalText([]byte("nao-e-um-uuid")) }},
		{"UnmarshalText curto", func(n *uuidv7.NullUUID) error { return n.UnmarshalText([]byte("abc")) }},
		{"UnmarshalBinary", func(n *uuidv7.NullUUID) error { return n.UnmarshalBinary(reference[:15]) }},
	}
	for _, c := range preservam {
		n := present
		if err := c.call(&n); err == nil {
			t.Errorf("%s: esperado erro", c.nome)
		} else if n != present {
			t.Errorf("%s: deixou %+v, esperado o receptor intacto %+v", c.nome, n, present)
		}
	}

	// Scan: identificador intacto, booleano falso. É a exceção.
	for _, src := range []any{"nao-e-um-uuid", []byte("abc"), 42} {
		n := present
		if err := n.Scan(src); err == nil {
			t.Errorf("Scan(%#v): esperado erro", src)
		} else if n.Valid || n.UUID != reference {
			t.Errorf("Scan(%#v): deixou %+v, esperado Valid falso com o identificador intacto", src, n)
		}
	}

	// O anulável binário delega, portanto herda as duas regras.
	presenteBin := uuidv7.NullBinaryUUID{UUID: reference, Valid: true}
	nb := presenteBin
	if err := nb.UnmarshalText([]byte("nao-e-um-uuid")); err == nil || nb != presenteBin {
		t.Errorf("NullBinaryUUID.UnmarshalText inválido: %+v, erro %v, esperado o receptor intacto", nb, err)
	}
	nb = presenteBin
	if err := nb.Scan(42); err == nil || nb.Valid || nb.UUID != reference {
		t.Errorf("NullBinaryUUID.Scan(42): %+v, erro %v, esperado Valid falso com o identificador intacto", nb, err)
	}
}

// TestMustPropagatesError confere que Must entra em pânico com o próprio
// erro recebido, e não com uma mensagem nova, para que quem recupera o
// pânico consiga reconhecê-lo.
func TestMustPropagatesError(t *testing.T) {
	defer func() {
		recovered := recover()
		err, ok := recovered.(error)
		if !ok || !errors.Is(err, uuidv7.ErrInvalidFormat) {
			t.Errorf("Must entrou em pânico com %v, esperado o erro ErrInvalidFormat", recovered)
		}
	}()
	uuidv7.Must(uuidv7.Parse("nao-e-um-uuid"))
	t.Error("Must deveria ter entrado em pânico")
}

// TestErrorTaxonomy percorre a tabela de docs/05-conversao-e-analise.md
// seção 5: cada
// entrada produz o erro descrito, reconhecido pelo tipo e não só pela
// presença; o UUID devolvido é o nulo; o analisador estrito e Import
// devolvem exatamente o sentinela; a assimetria registrada do prefixo URN,
// que devolve o sentinela enquanto as chaves malformadas têm erro próprio,
// fica travada; e ErrNotV7 fica fora da família de formato.
func TestErrorTaxonomy(t *testing.T) {
	// classe reduz um erro à linha da tabela a que ele pertence.
	classe := func(err error) string {
		switch {
		case err == nil:
			return "nenhum"
		case errors.Is(err, uuidv7.ErrInvalidLength):
			return "comprimento"
		case errors.Is(err, uuidv7.ErrInvalidBrackets):
			return "chaves"
		case err == uuidv7.ErrInvalidFormat: //nolint:errorlint // o sentinela puro é o próprio contrato testado
			return "sentinela puro"
		case errors.Is(err, uuidv7.ErrInvalidFormat):
			return "formato embrulhado"
		}
		return "outro"
	}

	const compact = "0192f7c51a2b7c3d8e4faabbccddeeff"
	cases := []struct {
		name  string
		input string
		want  string // classe esperada do analisador permissivo
	}{
		{"comprimento errado", "abc", "comprimento"},
		{"comprimento 37", canonical + "0", "comprimento"},
		{"vazio", "", "comprimento"},
		{"chaves trocadas", "(" + canonical + ")", "chaves"},
		{"chave de fechamento ausente", "{" + canonical + "0", "chaves"},
		{"prefixo URN inválido", "urn:uiid:" + canonical, "sentinela puro"},
		{"prefixo URN de outro esquema", "urn:isbn:" + canonical, "sentinela puro"},
		{"dígito inválido na forma canônica", canonical[:35] + "g", "sentinela puro"},
		{"hífen fora de lugar", "0192f7c5-1a2b-7c3d-8e4faabbccddeeff-", "sentinela puro"},
		{"dígito inválido na forma crua", compact[:31] + "g", "sentinela puro"},
		{"dígito inválido entre chaves", "{" + canonical[:35] + "g}", "sentinela puro"},
		{"dígito inválido na URN", "urn:uuid:" + canonical[:35] + "g", "sentinela puro"},
	}
	reference := uuidv7.MustParse(canonical)
	for _, c := range cases {
		got, err := uuidv7.Parse(c.input)
		if classe(err) != c.want {
			t.Errorf("%s: Parse(%q) devolveu %v (%s), esperado %s", c.name, c.input, err, classe(err), c.want)
		}
		if !got.IsZero() {
			t.Errorf("%s: Parse devolveu %s junto do erro, esperado o UUID nulo", c.name, got)
		}
		if _, err := uuidv7.ParseBytes([]byte(c.input)); classe(err) != c.want {
			t.Errorf("%s: ParseBytes devolveu %v (%s), esperado %s", c.name, err, classe(err), c.want)
		}
		if err := uuidv7.Validate(c.input); classe(err) != c.want {
			t.Errorf("%s: Validate devolveu %v (%s), esperado %s", c.name, err, classe(err), c.want)
		}

		// A desserialização de texto classifica como Parse e não altera o
		// receptor.
		u := reference
		if err := u.UnmarshalText([]byte(c.input)); classe(err) != c.want || u != reference {
			t.Errorf("%s: UnmarshalText devolveu %v (%s) e deixou %s; esperado %s sem alterar o receptor",
				c.name, err, classe(err), u, c.want)
		}

		// O analisador estrito e Import devolvem sempre o sentinela puro.
		if _, err := uuidv7.FromString(c.input); classe(err) != "sentinela puro" {
			t.Errorf("%s: FromString devolveu %v (%s), esperado o sentinela puro", c.name, err, classe(err))
		}
		if tm, err := uuidv7.Import(c.input); classe(err) != "sentinela puro" || tm != (uuidv7.Time{}) {
			t.Errorf("%s: Import devolveu %+v com %v (%s), esperado estrutura zerada e o sentinela puro",
				c.name, tm, err, classe(err))
		}
	}

	// Construção e desserialização binária: comprimento inválido.
	if _, err := uuidv7.FromBytes(reference[:15]); classe(err) != "comprimento" {
		t.Errorf("FromBytes com 15 bytes: %v (%s), esperado comprimento inválido", err, classe(err))
	}
	u := reference
	if err := u.UnmarshalBinary(reference[:15]); classe(err) != "comprimento" || u != reference {
		t.Errorf("UnmarshalBinary com 15 bytes: %v (%s) e deixou %s; esperado comprimento inválido sem alterar o receptor",
			err, classe(err), u)
	}

	// JSON do tipo anulável: valor que não é string, ou sintaxe inválida,
	// devolve o sentinela puro; string recusada devolve o erro de Parse.
	for _, c := range []struct {
		name, input, want string
	}{
		{"número no lugar da string", `42`, "sentinela puro"},
		{"objeto no lugar da string", `{}`, "sentinela puro"},
		{"string sem fechar", `"abc`, "sentinela puro"},
		{"string curta", `"abc"`, "comprimento"},
		{"string com chaves trocadas", `"(` + canonical + `)"`, "chaves"},
		{"string com prefixo URN inválido", `"urn:uiid:` + canonical + `"`, "sentinela puro"},
		{"string com dígito inválido", `"0192f7c5-1a2b-7c3d-8e4f-aabbccddeefg"`, "sentinela puro"},
		// Com barra invertida a leitura passa pelo encoding/json antes de
		// Parse, e o erro específico de Parse precisa sobreviver ao desvio.
		{"string com escape e comprimento errado", `"\u0030bc"`, "comprimento"},
		{"string com escape e chaves trocadas", `"(\u0030` + canonical[1:] + `)"`, "chaves"},
		{"string com escape e dígito inválido", `"\u0030` + canonical[1:35] + `g"`, "sentinela puro"},
	} {
		n := uuidv7.NullUUID{UUID: reference, Valid: true}
		if err := n.UnmarshalJSON([]byte(c.input)); classe(err) != c.want || n.UUID != reference || !n.Valid {
			t.Errorf("NullUUID.UnmarshalJSON(%s): %v (%s) e deixou %+v; esperado %s sem alterar o receptor",
				c.name, err, classe(err), n, c.want)
		}
	}

	// Tipo não suportado, versão errada e fonte de entropia são famílias à
	// parte: não são erros de formato.
	if err := u.Scan(3.14); !errors.Is(err, uuidv7.ErrInvalidScanType) || classe(err) != "outro" {
		t.Errorf("Scan de float: %v (%s), esperado ErrInvalidScanType fora da família de formato", err, classe(err))
	}
	if classe(uuidv7.ErrEntropySource) != "outro" {
		t.Error("ErrEntropySource não deveria ser reconhecido como erro de formato")
	}
	if classe(uuidv7.ErrNotV7) != "outro" {
		t.Error("ErrNotV7 não deveria ser reconhecido como erro de formato")
	}

	// Versão errada: o texto é aceito pelos analisadores, e só a leitura de
	// tempo recusa, com ErrNotV7 puro e a estrutura zerada.
	v4 := rfc9562Vectors["A.3 versão 4"]
	if tm, err := uuidv7.Import(v4); err != uuidv7.ErrNotV7 || tm != (uuidv7.Time{}) { //nolint:errorlint // o sentinela puro é o próprio contrato testado
		t.Errorf("Import de UUIDv4: %+v com %v, esperado estrutura zerada e exatamente ErrNotV7", tm, err)
	}
	if _, err := uuidv7.ImportBinary(uuidv7.MustParse(v4)); err != uuidv7.ErrNotV7 { //nolint:errorlint // o sentinela puro é o próprio contrato testado
		t.Errorf("ImportBinary de UUIDv4: %v, esperado exatamente ErrNotV7", err)
	}
	if _, err := uuidv7.Parse(v4); err != nil {
		t.Errorf("Parse de UUIDv4: %v, esperado nenhum erro", err)
	}

	// A fonte de entropia que falha entra em pânico com ErrEntropySource,
	// para quem recupera o pânico reconhecer a causa.
	func() {
		defer func() {
			if p := recover(); p != uuidv7.ErrEntropySource { //nolint:errorlint // o valor do pânico é o próprio contrato testado
				t.Errorf("fonte esgotada: pânico com %v, esperado ErrEntropySource", p)
			}
		}()
		uuidv7.NewGeneratorWithReader(bytes.NewReader(nil)).Generate(uuidv7.Level1)
	}()
}
