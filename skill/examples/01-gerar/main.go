// Exemplo 01: gerar UUIDv7 nos três níveis, pelas funções de pacote e
// por um Generator criado no boot.
//
//	go run ./skill/examples/01-gerar
//
// A saída muda a cada execução, porque o instante e os bits livres
// mudam. O que não muda: o comprimento (36), a versão (7) e a variante
// (2), e o nível que cada forma grava.
package main

import (
	"fmt"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

// Em um serviço, o gerador é criado uma única vez e compartilhado por
// todas as goroutines. Uma variável de pacote é o lugar natural.
var gen = uuidv7.NewGenerator()

func main() {
	// A forma mais curta: as funções de pacote usam um gerador padrão
	// interno, já pronto e seguro para concorrência.
	fmt.Println("Nível 1 (milissegundos):     ", uuidv7.GenerateString(uuidv7.Level1))
	fmt.Println("Nível 2 (+ microssegundos):  ", uuidv7.GenerateString(uuidv7.Level2))
	fmt.Println("Nível 3 (+ nanossegundos):   ", uuidv7.GenerateString(uuidv7.Level3))

	// Binário: um [16]byte, sem alocação. String() converte quando for
	// preciso; para gravar em volume sem alocar, ver o exemplo 09.
	u := gen.Generate(uuidv7.Level3)
	fmt.Printf("binário: %x -> texto: %s\n", u.Bytes(), u.String())
	fmt.Println("versão:", u.Version(), "variante:", u.Variant(), "válido:", u.IsValid())

	// Os níveis também existem pelo nome, sem o argumento. GenerateV7 é
	// o UUIDv7 padrão da RFC 9562, o mesmo que Generate(Level1).
	fmt.Println("GenerateV7:       ", uuidv7.GenerateV7())
	fmt.Println("GenerateV7Level2: ", gen.GenerateV7Level2())
	fmt.Println("GenerateV7Level3: ", gen.GenerateV7Level3())

	// Dois UUIDs gerados em sequência ficam em ordem cronológica quando o
	// instante embutido avança; dentro do mesmo instante a ordem é
	// aleatória. Compare devolve -1, 0 ou 1.
	a := gen.Generate(uuidv7.Level3)
	b := gen.Generate(uuidv7.Level3)
	fmt.Println("a.Compare(b):", a.Compare(b), "(0 nunca acontece: os bits livres diferem)")
}
