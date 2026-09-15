// Exemplo 07: como o UUID entra e sai de database/sql, em coluna de
// texto e em coluna binária de 16 bytes, com e sem NULL.
//
//	go run ./skill/examples/07-banco-de-dados
//
// Não há driver aqui (a biblioteca não tem dependências): o exemplo
// chama Scan e Value diretamente, que é exatamente o que database/sql
// faz ao ler uma linha e ao passar um parâmetro de consulta. A consulta
// real fica em comentários. Usa vetores fixos, então a saída é sempre a
// mesma.
package main

import (
	"fmt"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

func main() {
	id := uuidv7.MustParse("0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff")

	// ESCRITA EM COLUNA DE TEXTO (uuid no PostgreSQL, CHAR(36) em outros):
	//   db.Exec("INSERT INTO eventos (id, pai) VALUES ($1, $2)", id, pai)
	// Value entrega a string canônica. Esse formato não muda.
	v, _ := id.Value()
	fmt.Printf("UUID.Value():        %T %v\n", v, v)

	// ESCRITA EM COLUNA BINÁRIA (BINARY(16) no MySQL/MariaDB, BLOB no SQLite):
	//   db.Exec("INSERT INTO eventos (id) VALUES (?)", uuidv7.BinaryUUID(id))
	// A conversão é no ponto da consulta; os 16 bytes saem em ordem de
	// rede, sem rotação de campos.
	bv, _ := uuidv7.BinaryUUID(id).Value()
	fmt.Printf("BinaryUUID.Value():  %T %x\n", bv, bv)

	// COLUNA QUE ACEITA NULL: NullUUID (texto) e NullBinaryUUID (binário).
	// Valid=false grava NULL. Para gravar o UUID nulo de fato, é preciso
	// Valid=true com UUID == Nil: ausência e UUID nulo são valores
	// distintos, e uma coluna que os misture não os distingue mais.
	semPai := uuidv7.NullUUID{}
	comPai := uuidv7.NullBinaryUUID{UUID: id, Valid: true}
	nuloDeFato := uuidv7.NullBinaryUUID{UUID: uuidv7.Nil, Valid: true}
	nv, _ := semPai.Value()
	cv, _ := comPai.Value()
	zv, _ := nuloDeFato.Value()
	fmt.Printf("NullUUID{}.Value():             %v\n", nv)
	fmt.Printf("NullBinaryUUID{id}.Value():     %x\n", cv)
	fmt.Printf("NullBinaryUUID{Nil,true}.Value(): %x\n", zv)

	// LEITURA:
	//   err := db.QueryRow("SELECT id, pai FROM eventos WHERE id = $1", id).Scan(&r.ID, &r.Pai)
	// Scan aceita texto em qualquer forma de Parse, 16 bytes crus e NULL.
	// Texto vazio equivale a NULL (colunas DEFAULT '' não falham).
	var lido uuidv7.UUID
	for _, src := range []any{
		"0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff",
		"{0192F7C5-1A2B-7C3D-8E4F-AABBCCDDEEFF}",
		id.Bytes(), // 16 bytes crus, como uma coluna BINARY(16) devolve
		"",
		nil,
	} {
		if err := lido.Scan(src); err != nil {
			panic(err)
		}
		fmt.Printf("Scan(%-42v) -> %s\n", descreve(src), lido)
	}

	// A coluna anulável: NULL e vazio deixam Valid=false, sem erro.
	var pai uuidv7.NullUUID
	_ = pai.Scan(nil)
	fmt.Println("NullUUID.Scan(nil):  Valid =", pai.Valid)
	_ = pai.Scan(id.Bytes())
	fmt.Println("NullUUID.Scan(16 B): Valid =", pai.Valid, "UUID =", pai.UUID)

	// Um valor que não é UUID devolve erro; o tipo simples fica intacto,
	// e o anulável derruba Valid, porque database/sql reaproveita o
	// destino entre linhas.
	err := pai.Scan("não é um uuid")
	fmt.Println("Scan de lixo:", err, "| Valid =", pai.Valid, "| UUID intacto =", pai.UUID == id)
	err = lido.Scan(42)
	fmt.Println("Scan de int:", err)

	// A leitura confere só a forma: para exigir UUIDv7 na entrada, use
	// IsValid depois do Scan. E a consulta por intervalo (exemplo 04)
	// funciona igual na coluna binária:
	//   lo, hi := uuidv7.RangeAt(nivel, inicio, fim)
	//   db.Query("... WHERE id >= ? AND id < ?", uuidv7.BinaryUUID(lo), uuidv7.BinaryUUID(hi))
	fmt.Println("lido.IsValid():", lido.IsValid())
}

// descreve mostra o valor de entrada de forma legível no relatório.
func descreve(src any) string {
	switch v := src.(type) {
	case nil:
		return "nil"
	case string:
		return fmt.Sprintf("%q", v)
	case []byte:
		return fmt.Sprintf("[]byte %x", v)
	}
	return fmt.Sprint(src)
}
