package uuidv7

// Nil é o UUID de valor zero, com os 128 bits em zero. É o valor devolvido
// junto de todo erro de análise e o que a leitura de banco grava para
// ausência de valor. Não é um UUIDv7: IsValid devolve falso para ele. Não
// altere o valor desta variável.
var Nil UUID

// IsZero informa se o UUID é o valor nulo.
func (u UUID) IsZero() bool { return u == Nil }

// Bytes devolve uma cópia dos 16 bytes em ordem de rede. Alterar o
// resultado não afeta o UUID de origem.
//
// Equivale a MarshalBinary sem o erro sempre nulo. Dentro do pacote do
// chamador, u[:] é mais barato e não copia, mas devolve um slice apoiado
// no próprio valor: use Bytes quando o destino guardar a referência.
//
// Serve também para contornar uma armadilha de formatação: como UUID
// satisfaz fmt.Stringer, %x sobre um UUID formata a string canônica, não
// os bytes. fmt.Printf("%x", u.Bytes()) imprime os 32 dígitos esperados.
func (u UUID) Bytes() []byte {
	out := make([]byte, 16)
	copy(out, u[:])
	return out
}

// IsValid informa se o UUID é um UUIDv7: número de versão 7 e variante da
// RFC 9562 (0b10). É a mesma condição que as leituras de tempo exigem, e
// um UUID que falhe aqui é recusado por Import e ImportBinary com
// ErrNotV7 e por Timestamp e TimestampWithLevel com falso.
//
// A verificação é de forma, não de origem: não há como saber se o UUID
// foi gerado por esta biblioteca, nem em qual nível. Nil é recusado,
// porque não carrega versão nem variante; use IsZero para reconhecê-lo.
func (u UUID) IsValid() bool {
	return u[6]>>4 == 7 && u[8]>>6 == 0b10
}

// Compare ordena dois UUIDs pela sequência de bytes, devolvendo -1, 0 ou
// 1. Para UUIDv7 a ordem coincide com a ordem cronológica, na resolução
// do nível gravado; dentro do mesmo instante embutido a ordem é decidida
// pelos bits aleatórios.
//
// A assinatura serve diretamente à biblioteca padrão:
// slices.SortFunc(lista, UUID.Compare) ordena uma lista sem alocar.
//
// Para simples igualdade não é preciso chamar nada: UUID é um vetor de
// bytes e aceita o operador de igualdade diretamente.
func (u UUID) Compare(other UUID) int {
	for i := 0; i < 16; i++ {
		switch {
		case u[i] < other[i]:
			return -1
		case u[i] > other[i]:
			return 1
		}
	}
	return 0
}

// URN devolve a representação como URN da RFC 8141, ou seja, a string
// canônica prefixada por "urn:uuid:". Parse aceita esta forma de volta.
func (u UUID) URN() string {
	var buf [45]byte
	copy(buf[:9], "urn:uuid:")
	encodeHex(buf[9:], u)
	return string(buf[:])
}
