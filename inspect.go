package uuidv7

import "time"

// Timestamp devolve o instante de criação embutido no UUIDv7, em UTC, com
// resolução de milissegundo. É o mesmo que TimestampWithLevel(Level1).
//
// Devolve falso se o UUID não for de versão 7 com a variante da RFC 9562
// (ver IsValid).
//
// Para recuperar também os microssegundos e nanossegundos gravados pelos
// níveis 2 e 3 desta biblioteca, use TimestampWithLevel.
func (u UUID) Timestamp() (time.Time, bool) {
	if !u.IsValid() {
		return time.Time{}, false
	}
	return time.UnixMilli(milliFromV7(u)).UTC(), true
}

// TimestampWithLevel devolve o instante de criação de um UUIDv7 em UTC,
// reconstruindo a precisão sub-milissegundo gravada pelo nível informado.
//
// O nível não pode ser deduzido do UUID, por isso precisa ser informado
// pelo chamador. Se algum campo lido estiver fora da faixa 0 a 999 — o
// que denuncia bits aleatórios, e não tempo — todos os campos
// sub-milissegundo são descartados e devolve-se apenas o milissegundo.
// No nível 3 os dois campos vêm da mesma geração: um rand_a fora da
// faixa prova que o topo de rand_b também é ruído, mesmo que caiba em
// 0 a 999 por acaso. No Nível 1 e em níveis desconhecidos devolve apenas
// o milissegundo.
//
// Devolve falso se o UUID não for de versão 7 com a variante da RFC 9562
// (ver IsValid).
func (u UUID) TimestampWithLevel(level Level) (time.Time, bool) {
	if !u.IsValid() {
		return time.Time{}, false
	}
	t := time.UnixMilli(milliFromV7(u)).UTC()

	micro := int(uint16(u[6]&0x0F)<<8 | uint16(u[7]))
	nano := int(uint16(u[8]&0x3F)<<4 | uint16(u[9])>>4)

	switch level {
	case Level2:
		if micro <= 999 {
			t = t.Add(time.Duration(micro) * time.Microsecond)
		}
	case Level3:
		if micro <= 999 && nano <= 999 {
			t = t.Add(time.Duration(micro)*time.Microsecond + time.Duration(nano)*time.Nanosecond)
		}
	}
	return t, true
}

// milliFromV7 lê os 48 bits de milissegundos de um UUIDv7.
func milliFromV7(u UUID) int64 {
	return int64(u[0])<<40 |
		int64(u[1])<<32 |
		int64(u[2])<<24 |
		int64(u[3])<<16 |
		int64(u[4])<<8 |
		int64(u[5])
}
