// Exemplo 06: o UUID em JSON, texto, binário e gob, sem código extra,
// e a coluna anulável (NullUUID) nesses formatos.
//
//	go run ./skill/examples/06-json-e-gob
//
// Usa vetores fixos, então a saída é sempre a mesma.
package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

// Evento é uma estrutura como qualquer outra: o UUID viaja como string
// canônica em JSON porque o tipo implementa encoding.TextMarshaler.
type Evento struct {
	ID  uuidv7.UUID     `json:"id"`
	Pai uuidv7.NullUUID `json:"pai"` // coluna que aceita NULL: vira null em JSON
}

func main() {
	raiz := Evento{ID: uuidv7.MustParse("0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff")}
	filho := Evento{
		ID:  uuidv7.MustParse("0192f7c5-1a2b-7c3d-8e4f-aabbccddef00"),
		Pai: uuidv7.NullUUID{UUID: raiz.ID, Valid: true},
	}

	// JSON: string canônica entre aspas, nunca um vetor de 16 números.
	dados, err := json.Marshal([]Evento{raiz, filho})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(dados))

	// A leitura aceita qualquer forma de Parse, inclusive entre chaves ou
	// em maiúsculas, e null vira Valid=false.
	var lidos []Evento
	if err := json.Unmarshal([]byte(`[{"id":"{0192F7C5-1A2B-7C3D-8E4F-AABBCCDDEEFF}","pai":null}]`), &lidos); err != nil {
		panic(err)
	}
	fmt.Println("lido:", lidos[0].ID, "| pai presente?", lidos[0].Pai.Valid)

	// Texto e binário, pelas interfaces da biblioteca padrão.
	texto, _ := raiz.ID.MarshalText()
	binario, _ := raiz.ID.MarshalBinary()
	fmt.Printf("MarshalText: %s (%d bytes) | MarshalBinary: %x (%d bytes)\n", texto, len(texto), binario, len(binario))

	// Sem alocar, em um buffer reaproveitado: AppendTo para o texto,
	// AppendBinary para os 16 bytes.
	buf := make([]byte, 0, 64)
	buf = raiz.ID.AppendTo(buf)
	buf = append(buf, ' ')
	buf, _ = filho.ID.AppendBinary(buf)
	fmt.Printf("AppendTo + AppendBinary: %q + %x\n", buf[:36], buf[37:])

	// gob usa MarshalBinary: 16 bytes por UUID, nada quando não há valor.
	var rede bytes.Buffer
	if err := gob.NewEncoder(&rede).Encode(filho); err != nil {
		panic(err)
	}
	var recebido Evento
	if err := gob.NewDecoder(&rede).Decode(&recebido); err != nil {
		panic(err)
	}
	fmt.Println("gob ida e volta:", recebido == filho)

	// Entrada inválida devolve erro e deixa o receptor como estava.
	guardado := filho
	err = json.Unmarshal([]byte(`{"id":"abc","pai":"abc"}`), &guardado)
	fmt.Println("erro:", err, "| receptor intacto?", guardado == filho)
}
