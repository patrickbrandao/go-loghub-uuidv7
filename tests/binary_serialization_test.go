package tests

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"testing"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

// ---------- BinaryUUID ----------

// TestBinaryUUIDMarshalTextIsCanonical confere que MarshalText devolve a
// string canônica de 36 bytes, exatamente como UUID.
func TestBinaryUUIDMarshalTextIsCanonical(t *testing.T) {
	u := uuidv7.MustParse(amostraCanonica)
	b := uuidv7.BinaryUUID(u)

	text, err := b.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText devolveu erro: %v", err)
	}
	if string(text) != amostraCanonica {
		t.Errorf("MarshalText devolveu %q, esperado %q", text, amostraCanonica)
	}
}

// TestBinaryUUIDUnmarshalTextRoundTrip prova a ida e volta pelo texto.
func TestBinaryUUIDUnmarshalTextRoundTrip(t *testing.T) {
	original := uuidv7.BinaryUUID(uuidv7.MustParse(amostraCanonica))

	text, err := original.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText devolveu erro: %v", err)
	}

	var lido uuidv7.BinaryUUID
	if err := lido.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText devolveu erro: %v", err)
	}
	if lido != original {
		t.Errorf("ida e volta devolveu %s, esperado %s", lido, original)
	}
}

// TestBinaryUUIDMarshalBinaryIs16Bytes confere que MarshalBinary devolve
// os 16 bytes crus.
func TestBinaryUUIDMarshalBinaryIs16Bytes(t *testing.T) {
	u := uuidv7.MustParse(amostraCanonica)
	b := uuidv7.BinaryUUID(u)

	bin, err := b.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary devolveu erro: %v", err)
	}
	if len(bin) != 16 {
		t.Fatalf("MarshalBinary devolveu %d bytes, esperado 16", len(bin))
	}
	if !bytes.Equal(bin, u.Bytes()) {
		t.Errorf("MarshalBinary devolveu %x, esperado %x", bin, u.Bytes())
	}
}

// TestBinaryUUIDUnmarshalBinaryRoundTrip prova a ida e volta pelo binário.
func TestBinaryUUIDUnmarshalBinaryRoundTrip(t *testing.T) {
	original := uuidv7.BinaryUUID(uuidv7.MustParse(amostraCanonica))

	bin, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary devolveu erro: %v", err)
	}

	var lido uuidv7.BinaryUUID
	if err := lido.UnmarshalBinary(bin); err != nil {
		t.Fatalf("UnmarshalBinary devolveu erro: %v", err)
	}
	if lido != original {
		t.Errorf("ida e volta devolveu %s, esperado %s", lido, original)
	}
}

// TestBinaryUUIDJSONIsCanonicalString é o teste mais importante: sem
// MarshalText, encoding/json gravaria um vetor de 16 números. Com o
// método presente, deve gravar a string canônica entre aspas.
func TestBinaryUUIDJSONIsCanonicalString(t *testing.T) {
	u := uuidv7.MustParse(amostraCanonica)
	b := uuidv7.BinaryUUID(u)

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("json.Marshal devolveu erro: %v", err)
	}

	esperado := `"` + amostraCanonica + `"`
	if string(data) != esperado {
		t.Errorf("json.Marshal devolveu %s, esperado %s", data, esperado)
	}

	// Ida e volta pelo JSON.
	var lido uuidv7.BinaryUUID
	if err := json.Unmarshal(data, &lido); err != nil {
		t.Fatalf("json.Unmarshal devolveu erro: %v", err)
	}
	if lido != b {
		t.Errorf("ida e volta JSON devolveu %s, esperado %s", lido, b)
	}
}

// ---------- NullBinaryUUID — JSON ----------

// TestNullBinaryUUIDJSONWithValue confere que um valor presente sai como
// string canônica entre aspas, sem o envelope {"UUID":"...","Valid":true}.
func TestNullBinaryUUIDJSONWithValue(t *testing.T) {
	n := uuidv7.NullBinaryUUID{UUID: uuidv7.MustParse(amostraCanonica), Valid: true}

	data, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("json.Marshal devolveu erro: %v", err)
	}

	esperado := `"` + amostraCanonica + `"`
	if string(data) != esperado {
		t.Errorf("json.Marshal devolveu %s, esperado %s", data, esperado)
	}
}

// TestNullBinaryUUIDJSONAbsent confere que ausência produz o literal
// null, conforme a tabela normativa de docs/06-serializacao-e-banco.md
// seção 4.
func TestNullBinaryUUIDJSONAbsent(t *testing.T) {
	n := uuidv7.NullBinaryUUID{Valid: false}

	data, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("json.Marshal devolveu erro: %v", err)
	}

	if string(data) != "null" {
		t.Errorf("json.Marshal de ausente devolveu %s, esperado null", data)
	}
}

// TestNullBinaryUUIDUnmarshalJSONNull confere que ler null produz
// Valid=false.
func TestNullBinaryUUIDUnmarshalJSONNull(t *testing.T) {
	var n uuidv7.NullBinaryUUID
	if err := json.Unmarshal([]byte("null"), &n); err != nil {
		t.Fatalf("json.Unmarshal devolveu erro: %v", err)
	}
	if n.Valid {
		t.Error("após ler null, Valid deveria ser falso")
	}
	if n.UUID != uuidv7.Nil {
		t.Errorf("após ler null, UUID deveria ser Nil, obteve %s", n.UUID)
	}
}

// TestNullBinaryUUIDUnmarshalJSONValue confere que ler uma string
// canônica produz Valid=true com o UUID correto.
func TestNullBinaryUUIDUnmarshalJSONValue(t *testing.T) {
	entrada := `"` + amostraCanonica + `"`
	var n uuidv7.NullBinaryUUID
	if err := json.Unmarshal([]byte(entrada), &n); err != nil {
		t.Fatalf("json.Unmarshal devolveu erro: %v", err)
	}
	if !n.Valid {
		t.Error("após ler string, Valid deveria ser verdadeiro")
	}
	if n.UUID != uuidv7.MustParse(amostraCanonica) {
		t.Errorf("UUID lido %s, esperado %s", n.UUID, amostraCanonica)
	}
}

// ---------- NullBinaryUUID — Texto ----------

// TestNullBinaryUUIDMarshalTextWithValue confere a string canônica.
func TestNullBinaryUUIDMarshalTextWithValue(t *testing.T) {
	n := uuidv7.NullBinaryUUID{UUID: uuidv7.MustParse(amostraCanonica), Valid: true}

	text, err := n.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText devolveu erro: %v", err)
	}
	if string(text) != amostraCanonica {
		t.Errorf("MarshalText devolveu %q, esperado %q", text, amostraCanonica)
	}
}

// TestNullBinaryUUIDMarshalTextAbsent confere sequência vazia.
func TestNullBinaryUUIDMarshalTextAbsent(t *testing.T) {
	n := uuidv7.NullBinaryUUID{Valid: false}

	text, err := n.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText devolveu erro: %v", err)
	}
	if len(text) != 0 {
		t.Errorf("MarshalText de ausente devolveu %q, esperado vazio", text)
	}
}

// TestNullBinaryUUIDUnmarshalTextEmpty confere que entrada vazia produz
// ausência.
func TestNullBinaryUUIDUnmarshalTextEmpty(t *testing.T) {
	var n uuidv7.NullBinaryUUID
	if err := n.UnmarshalText([]byte{}); err != nil {
		t.Fatalf("UnmarshalText devolveu erro: %v", err)
	}
	if n.Valid {
		t.Error("após ler vazio, Valid deveria ser falso")
	}
}

// TestNullBinaryUUIDUnmarshalTextValue confere ida e volta pelo texto.
func TestNullBinaryUUIDUnmarshalTextValue(t *testing.T) {
	var n uuidv7.NullBinaryUUID
	if err := n.UnmarshalText([]byte(amostraCanonica)); err != nil {
		t.Fatalf("UnmarshalText devolveu erro: %v", err)
	}
	if !n.Valid {
		t.Error("após ler texto, Valid deveria ser verdadeiro")
	}
	if n.UUID != uuidv7.MustParse(amostraCanonica) {
		t.Errorf("UUID lido %s, esperado %s", n.UUID, amostraCanonica)
	}
}

// ---------- NullBinaryUUID — Binário ----------

// TestNullBinaryUUIDMarshalBinaryWithValue confere os 16 bytes.
func TestNullBinaryUUIDMarshalBinaryWithValue(t *testing.T) {
	u := uuidv7.MustParse(amostraCanonica)
	n := uuidv7.NullBinaryUUID{UUID: u, Valid: true}

	bin, err := n.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary devolveu erro: %v", err)
	}
	if len(bin) != 16 {
		t.Fatalf("MarshalBinary devolveu %d bytes, esperado 16", len(bin))
	}
	if !bytes.Equal(bin, u.Bytes()) {
		t.Errorf("MarshalBinary devolveu %x, esperado %x", bin, u.Bytes())
	}
}

// TestNullBinaryUUIDMarshalBinaryAbsent confere sequência vazia.
func TestNullBinaryUUIDMarshalBinaryAbsent(t *testing.T) {
	n := uuidv7.NullBinaryUUID{Valid: false}

	bin, err := n.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary devolveu erro: %v", err)
	}
	if len(bin) != 0 {
		t.Errorf("MarshalBinary de ausente devolveu %d bytes, esperado 0", len(bin))
	}
}

// TestNullBinaryUUIDUnmarshalBinaryEmpty confere que entrada vazia
// produz ausência.
func TestNullBinaryUUIDUnmarshalBinaryEmpty(t *testing.T) {
	var n uuidv7.NullBinaryUUID
	if err := n.UnmarshalBinary([]byte{}); err != nil {
		t.Fatalf("UnmarshalBinary devolveu erro: %v", err)
	}
	if n.Valid {
		t.Error("após ler vazio, Valid deveria ser falso")
	}
}

// TestNullBinaryUUIDUnmarshalBinaryValue confere ida e volta pelo
// binário.
func TestNullBinaryUUIDUnmarshalBinaryValue(t *testing.T) {
	u := uuidv7.MustParse(amostraCanonica)
	original := uuidv7.NullBinaryUUID{UUID: u, Valid: true}

	bin, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary devolveu erro: %v", err)
	}

	var lido uuidv7.NullBinaryUUID
	if err := lido.UnmarshalBinary(bin); err != nil {
		t.Fatalf("UnmarshalBinary devolveu erro: %v", err)
	}
	if !lido.Valid {
		t.Error("após ler binário, Valid deveria ser verdadeiro")
	}
	if lido.UUID != original.UUID {
		t.Errorf("UUID lido %s, esperado %s", lido.UUID, original.UUID)
	}
}

// ---------- JSON em struct ----------

// TestNullBinaryUUIDInStructJSON confere o caso de uso real: o tipo
// dentro de uma struct serializada como JSON. Sem os métodos, o
// encoding/json produziria {"campo":{"UUID":"...","Valid":true}}.
func TestNullBinaryUUIDInStructJSON(t *testing.T) {
	type registro struct {
		ID uuidv7.NullBinaryUUID `json:"id"`
	}

	t.Run("presente", func(t *testing.T) {
		r := registro{ID: uuidv7.NullBinaryUUID{UUID: uuidv7.MustParse(amostraCanonica), Valid: true}}
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("json.Marshal devolveu erro: %v", err)
		}
		esperado := `{"id":"` + amostraCanonica + `"}`
		if string(data) != esperado {
			t.Errorf("json.Marshal devolveu %s, esperado %s", data, esperado)
		}
	})

	t.Run("ausente", func(t *testing.T) {
		r := registro{ID: uuidv7.NullBinaryUUID{Valid: false}}
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("json.Marshal devolveu erro: %v", err)
		}
		esperado := `{"id":null}`
		if string(data) != esperado {
			t.Errorf("json.Marshal devolveu %s, esperado %s", data, esperado)
		}
	})

	t.Run("roundtrip", func(t *testing.T) {
		original := registro{ID: uuidv7.NullBinaryUUID{UUID: uuidv7.MustParse(amostraCanonica), Valid: true}}
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("json.Marshal devolveu erro: %v", err)
		}
		var lido registro
		if err := json.Unmarshal(data, &lido); err != nil {
			t.Fatalf("json.Unmarshal devolveu erro: %v", err)
		}
		if lido.ID.UUID != original.ID.UUID || lido.ID.Valid != original.ID.Valid {
			t.Errorf("ida e volta devolveu %+v, esperado %+v", lido.ID, original.ID)
		}
	})
}

// ---------- Comprimento e encoding/gob ----------

// TestBinaryDecodersRejectEveryOtherLength confere que a construção e as
// desserializações binárias recusam todo comprimento diferente de 16, de 0
// a 64 bytes, com ErrInvalidLength e sem alterar o receptor. Os testes
// anteriores usavam só 10 e 15 bytes, e uma campanha de mutação mostrou que
// aceitar 17 ou mais em silêncio, descartando o excedente, passava pela
// suíte. Nos tipos anuláveis a entrada vazia é ausência de valor
// (docs/06-serializacao-e-banco.md seção 4) e fica fora da recusa.
func TestBinaryDecodersRejectEveryOtherLength(t *testing.T) {
	reference := uuidv7.MustParse(amostraCanonica)
	for n := 0; n <= 64; n++ {
		if n == 16 {
			continue
		}
		data := bytes.Repeat([]byte{0xA5}, n)

		if got, err := uuidv7.FromBytes(data); !errors.Is(err, uuidv7.ErrInvalidLength) || got != uuidv7.Nil {
			t.Errorf("FromBytes com %d bytes: %s, erro %v; esperado ErrInvalidLength com o UUID nulo", n, got, err)
		}
		u := reference
		if err := u.UnmarshalBinary(data); !errors.Is(err, uuidv7.ErrInvalidLength) || u != reference {
			t.Errorf("UUID.UnmarshalBinary com %d bytes: erro %v, receptor %s", n, err, u)
		}
		b := uuidv7.BinaryUUID(reference)
		if err := b.UnmarshalBinary(data); !errors.Is(err, uuidv7.ErrInvalidLength) || uuidv7.UUID(b) != reference {
			t.Errorf("BinaryUUID.UnmarshalBinary com %d bytes: erro %v, receptor %s", n, err, b)
		}
		if n == 0 {
			continue
		}
		present := uuidv7.NullUUID{UUID: reference, Valid: true}
		nu := present
		if err := nu.UnmarshalBinary(data); !errors.Is(err, uuidv7.ErrInvalidLength) || nu != present {
			t.Errorf("NullUUID.UnmarshalBinary com %d bytes: erro %v, receptor %+v", n, err, nu)
		}
		nb := uuidv7.NullBinaryUUID(present)
		if err := nb.UnmarshalBinary(data); !errors.Is(err, uuidv7.ErrInvalidLength) || nb != uuidv7.NullBinaryUUID(present) {
			t.Errorf("NullBinaryUUID.UnmarshalBinary com %d bytes: erro %v, receptor %+v", n, err, nb)
		}
	}
}

// TestGobRoundTrip confere a ida e volta por encoding/gob, que usa
// MarshalBinary e UnmarshalBinary, dos quatro tipos dentro de uma
// estrutura: com valor, sem valor e com o UUID nulo presente. A
// documentação de uso afirma esse caminho, e nenhum teste o exercitava. A
// leitura é sempre sobre um valor novo, porque o gob omite os campos com
// valor zero e não os limpa num destino reaproveitado.
func TestGobRoundTrip(t *testing.T) {
	type record struct {
		ID             uuidv7.UUID
		Parent         uuidv7.NullUUID
		Binary         uuidv7.BinaryUUID
		NullableBinary uuidv7.NullBinaryUUID
	}
	reference := uuidv7.MustParse(amostraCanonica)
	cases := map[string]record{
		"com valor": {
			ID:             reference,
			Parent:         uuidv7.NullUUID{UUID: reference, Valid: true},
			Binary:         uuidv7.BinaryUUID(reference),
			NullableBinary: uuidv7.NullBinaryUUID{UUID: reference, Valid: true},
		},
		"sem valor":          {ID: reference},
		"UUID nulo presente": {Parent: uuidv7.NullUUID{Valid: true}, NullableBinary: uuidv7.NullBinaryUUID{Valid: true}},
	}
	for name, original := range cases {
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(original); err != nil {
			t.Fatalf("%s: gob.Encode devolveu erro: %v", name, err)
		}
		var decoded record
		if err := gob.NewDecoder(&buf).Decode(&decoded); err != nil {
			t.Fatalf("%s: gob.Decode devolveu erro: %v", name, err)
		}
		if decoded != original {
			t.Errorf("%s: ida e volta devolveu %+v, esperado %+v", name, decoded, original)
		}
	}
}
