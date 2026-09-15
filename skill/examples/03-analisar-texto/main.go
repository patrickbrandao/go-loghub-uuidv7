// Exemplo 03: converter texto em UUID com o analisador estrito e com o
// permissivo, e tratar cada erro pelo tipo.
//
//	go run ./skill/examples/03-analisar-texto
//
// Usa vetores fixos, então a saída é sempre a mesma.
package main

import (
	"errors"
	"fmt"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

func main() {
	const canonico = "0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff"

	// FromString aceita só a forma canônica 8-4-4-4-12 (maiúsculas ou
	// minúsculas) e devolve exatamente ErrInvalidFormat para o resto:
	// a comparação com == faz parte do contrato.
	u, err := uuidv7.FromString("0192F7C5-1A2B-7C3D-8E4F-AABBCCDDEEFF")
	fmt.Println("FromString:", u, err)
	_, err = uuidv7.FromString("0192f7c51a2b7c3d8e4faabbccddeeff")
	fmt.Println("FromString sem hífens:", err == uuidv7.ErrInvalidFormat) //nolint:errorlint // FromString devolve o sentinela puro; a comparação direta faz parte do contrato

	// Parse aceita quatro formas; todas produzem o mesmo valor.
	for _, entrada := range []string{
		canonico,
		"{0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff}",
		"urn:uuid:0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff",
		"0192f7c51a2b7c3d8e4faabbccddeeff",
	} {
		p, err := uuidv7.Parse(entrada)
		fmt.Printf("Parse(%-47q) = %s %v\n", entrada, p, err)
	}

	// ParseBytes faz o mesmo a partir de []byte, sem alocar; Validate só
	// valida; FromBytes recebe os 16 bytes crus.
	pb, _ := uuidv7.ParseBytes([]byte(canonico))
	fb, _ := uuidv7.FromBytes(pb.Bytes())
	fmt.Println("ParseBytes == FromBytes(Bytes()):", pb == fb, "| Validate:", uuidv7.Validate(canonico))

	// Os erros de Parse são específicos e todos embrulham ErrInvalidFormat.
	// Quem só quer saber se falhou usa errors.Is com o sentinela; quem
	// decide pelo tipo usa os específicos.
	for _, entrada := range []string{
		"abc",                                    // comprimento errado
		"(0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff)", // 38 caracteres sem as chaves
		"urn:uiid:0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff", // prefixo errado
		"0192f7c5-1a2b-7c3d-8e4f-aabbccddeefg",          // dígito inválido
	} {
		p, err := uuidv7.Parse(entrada)
		fmt.Printf("%-47q -> %v | formato=%t comprimento=%t chaves=%t | Nil=%t\n",
			entrada, err,
			errors.Is(err, uuidv7.ErrInvalidFormat),
			errors.Is(err, uuidv7.ErrInvalidLength),
			errors.Is(err, uuidv7.ErrInvalidBrackets),
			p.IsZero())
	}

	// A análise confere só a forma: um UUID de outra versão passa. Para
	// exigir UUIDv7 na entrada, confira IsValid depois de Parse.
	entrada := "919108f7-52d1-4320-9bac-f847db4148a8" // UUIDv4
	p, err := uuidv7.Parse(entrada)
	if err != nil {
		panic(err)
	}
	if !p.IsValid() {
		fmt.Println("bem formado, mas não é UUIDv7:", p, "->", uuidv7.ErrNotV7)
	}

	// MustParse é para constantes do próprio código; Must encadeia com
	// funções que devolvem par de valores. Os dois entram em pânico.
	fixo := uuidv7.MustParse(canonico)
	outro := uuidv7.Must(uuidv7.FromString(canonico))
	fmt.Println("MustParse == Must(FromString):", fixo == outro)
	fmt.Println("URN:", fixo.URN())
}
