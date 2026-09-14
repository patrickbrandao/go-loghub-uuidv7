package uuidv7

// Testes internos da decomposição do instante. Este é um dos dois arquivos
// de teste autorizados na raiz, e cobre o que não pode ser exercitado de
// fora do pacote: as bordas de splitUnixInstant (antes de 1970, ano 2300,
// viradas de milissegundo e de segundo), porque a geração lê o relógio do
// sistema e ele não é injetável (docs/SPEC.md seção 11.2).

import (
	"testing"
	"time"
)

func TestSplitUnixInstantFloorsPreEpoch(t *testing.T) {
	cases := []struct {
		name      string
		sec, nsec int64
	}{
		// 1969-12-31T23:59:59.5Z: Unix() devolve -1 e Nanosecond() 500_000_000.
		{"meio do último segundo antes da época", -1, 500_000_000},
		// 1969-12-31T23:59:59.999999999Z: o carimbo em milissegundos vale
		// exatamente -1, a borda do piso. Um piso escrito como ms < -1 passa no
		// caso anterior e falha só aqui, com micro e nano em 999.
		{"último nanossegundo antes da época", -1, 999_999_999},
		{"início do último segundo antes da época", -1, 0},
		// O valor zero de time.Time: ano 1, muito antes da época.
		{"valor zero de time.Time", -62_135_596_800, 0},
	}
	for _, c := range cases {
		ms, micro, nano := splitUnixInstant(c.sec, c.nsec)
		if ms != 0 || micro != 0 || nano != 0 {
			t.Errorf("pré-época, %s: ms=%d micro=%d nano=%d, esperado 0/0/0", c.name, ms, micro, nano)
		}
	}
}

func TestSplitUnixInstantYear2300FitsIn48Bits(t *testing.T) {
	at := time.Date(2300, 1, 1, 0, 0, 0, 0, time.UTC)
	ms, _, _ := splitUnixInstant(at.Unix(), 0)
	if ms != 10_413_792_000_000 {
		t.Fatalf("2300-01-01: ms=%d, esperado 10413792000000", ms)
	}
	if ms >= 1<<48 {
		t.Fatalf("2300-01-01: ms=%d não cabe em 48 bits", ms)
	}
}

func TestSplitUnixInstantRollovers(t *testing.T) {
	cases := []struct {
		sec, nsec int64
		ms        int64
		micro     uint16
		nano      uint16
	}{
		{0, 0, 0, 0, 0},
		{0, 999_999, 0, 999, 999},
		{0, 1_000_000, 1, 0, 0},
		{1, 999_999_999, 1_999, 999, 999},
		{2, 0, 2_000, 0, 0},
	}
	for _, c := range cases {
		ms, micro, nano := splitUnixInstant(c.sec, c.nsec)
		if ms != c.ms || micro != c.micro || nano != c.nano {
			t.Errorf("sec=%d nsec=%d: obtido %d/%d/%d, esperado %d/%d/%d",
				c.sec, c.nsec, ms, micro, nano, c.ms, c.micro, c.nano)
		}
	}
}
