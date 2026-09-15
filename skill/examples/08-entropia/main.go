// Exemplo 08: escolher a fonte de entropia: o gerador padrão, o
// criptográfico, um leitor do chamador e uma função própria; reproduzir
// identificadores com um leitor determinístico; e o que acontece quando
// a fonte falha.
//
//	go run ./skill/examples/08-entropia
package main

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

func main() {
	// O gerador padrão lê do ChaCha8 do runtime do Go: sem trava, rápido,
	// adequado para chave primária, identificador de registro e
	// correlação de log. Não serve para segredo: a recomendação do Go
	// para uso sensível é crypto/rand, e todo UUIDv7 expõe o instante de
	// criação de qualquer forma.
	padrao := uuidv7.NewGenerator()
	fmt.Println("padrão:        ", padrao.GenerateString(uuidv7.Level3))

	// Para token de sessão, link privado ou chave de recuperação: entropia
	// inteiramente de crypto/rand. Mais lento, imprevisível.
	cripto := uuidv7.NewCryptoGenerator()
	fmt.Println("criptográfico: ", cripto.GenerateString(uuidv7.Level3))

	// O mesmo, a partir de qualquer io.Reader seguro para concorrência.
	// Cada palavra de 64 bits são 8 bytes em ordem de rede; o Nível 1
	// consome duas palavras, os níveis 2 e 3 uma.
	leitor := uuidv7.NewGeneratorWithReader(crand.Reader)
	fmt.Println("com leitor:    ", leitor.GenerateString(uuidv7.Level1))

	// Ou a partir de uma função que devolve 64 bits. Ela PRECISA ser
	// segura para uso concorrente; crypto/rand.Read é.
	proprio := uuidv7.NewGeneratorWith(func() uint64 {
		var b [8]byte
		if _, err := crand.Read(b[:]); err != nil {
			panic(err)
		}
		return binary.BigEndian.Uint64(b[:])
	})
	fmt.Println("função própria:", proprio.GenerateString(uuidv7.Level2))

	// Em testes, um leitor determinístico mais GenerateAt reproduz o mesmo
	// identificador sempre: o instante é o parâmetro e os bits livres vêm
	// dos bytes conhecidos. Nível 1 consome 16 bytes (r1 = rand_a, r2 =
	// rand_b), Nível 3 consome 8 (só rand_b).
	quando := time.Date(2026, 1, 1, 0, 0, 0, 123_456_789, time.UTC)
	fixo := uuidv7.NewGeneratorWithReader(bytes.NewReader([]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
	}))
	fmt.Println("determinístico:", fixo.GenerateAt(uuidv7.Level1, quando))
	// rand_a = 12 bits baixos de r1 (0x708), rand_b = 62 bits baixos de r2.

	// Entropia constante em zero dá a fronteira inferior do instante, a
	// mesma que MinAt devolve; é uma forma de ver onde cada campo fica.
	zero := uuidv7.NewGeneratorWith(func() uint64 { return 0 })
	fmt.Println("entropia zero: ", zero.GenerateAt(uuidv7.Level3, quando), "== MinAt:",
		zero.GenerateAt(uuidv7.Level3, quando) == uuidv7.MinAt(uuidv7.Level3, quando))

	// Quando a fonte falha durante a geração, a biblioteca entra em pânico
	// com ErrEntropySource como valor: uma fonte quebrada nunca degrada em
	// silêncio para um gerador previsível. Um leitor esgotado provoca isso.
	esgotado := uuidv7.NewGeneratorWithReader(bytes.NewReader([]byte{0x01, 0x02}))
	func() {
		defer func() {
			r := recover()
			err, _ := r.(error)
			fmt.Println("fonte esgotada -> pânico com ErrEntropySource:", errors.Is(err, uuidv7.ErrEntropySource))
		}()
		esgotado.Generate(uuidv7.Level3)
	}()

	// Fonte nula é erro de configuração e falha na construção, no boot.
	func() {
		defer func() { fmt.Println("fonte nula -> pânico na construção:", recover() != nil) }()
		uuidv7.NewGeneratorWith(nil)
	}()
}
