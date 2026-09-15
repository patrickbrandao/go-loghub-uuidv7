package tests

import (
	"bytes"
	crand "crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/iotest"

	"github.com/patrickbrandao/go-loghub-uuidv7"
)

// Este arquivo trava a fiação da entropia: de onde cada campo livre tira
// os seus bits. Os testes de layout usam fonte constante, que prova as
// máscaras mas não distingue uma palavra da outra, e uma campanha de
// mutação feita em 2026-09-13 mostrou que trocar r1 por r2, repetir a
// mesma palavra, ignorar a fonte do próprio gerador, aceitar leitura
// curta ou deixar de usar crypto/rand passava pela suíte inteira sem
// nenhuma falha. Os vetores exatos abaixo foram calculados por uma
// implementação independente, escrita a partir de
// docs/01-escopo-e-layout.md seção 6 e docs/02-instante-e-entropia.md
// seções 3 e 7, e não pela própria biblioteca.

// Duas palavras com todos os nibbles distintos: qualquer troca de
// palavra, de ordem de bytes ou de máscara muda o resultado.
const (
	firstWord  = 0x0123_4567_89AB_CDEF
	secondWord = 0xFEDC_BA98_7654_3210
)

// sequenceSource devolve, a cada chamada, a próxima palavra da lista,
// recomeçando do início ao fim dela. É segura para uso concorrente, como
// NewGeneratorWith exige, embora os testes a chamem de uma goroutine só.
func sequenceSource(words ...uint64) func() uint64 {
	var calls atomic.Uint64
	return func() uint64 {
		return words[(calls.Add(1)-1)%uint64(len(words))]
	}
}

// countingBytes devolve n bytes com os valores 1, 2, ..., n, de modo que
// cada posição lida de um leitor seja reconhecível no resultado.
func countingBytes(n int) []byte {
	raw := make([]byte, n)
	for i := range raw {
		raw[i] = byte(i + 1)
	}
	return raw
}

// panicValue executa f e devolve o valor do pânico, ou nulo se não houve
// pânico. Isola a chamada que deve falhar, para que um pânico anterior não
// se passe pelo esperado.
func panicValue(f func()) (recovered any) {
	defer func() { recovered = recover() }()
	f()
	return nil
}

// TestEntropyWordsFeedTheirFields trava a correspondência normativa de
// docs/02-instante-e-entropia.md seção 3: no Nível 1 e nos níveis
// desconhecidos, rand_a recebe os 12 bits baixos da primeira palavra
// sorteada (r1) e rand_b os 62 bits baixos da segunda (r2); nos níveis 2 e
// 3, a única palavra sorteada alimenta rand_b. Vale para a geração por
// instante e para a geração pelo relógio, que mantêm cópias separadas do
// empacotamento (docs/10-decisoes.md seção 2), e por isso as duas são
// conferidas.
func TestEntropyWordsFeedTheirFields(t *testing.T) {
	// Por instante, os 16 bytes inteiros são conhecidos.
	for _, c := range []struct {
		level uuidv7.Level
		want  string
	}{
		{uuidv7.Level1, "01a09076-bdfb-7def-bedc-ba9876543210"},
		{uuidv7.Level2, "01a09076-bdfb-71c8-8123-456789abcdef"},
		{uuidv7.Level3, "01a09076-bdfb-71c8-b153-456789abcdef"},
		{uuidv7.Level(9), "01a09076-bdfb-7def-bedc-ba9876543210"},
	} {
		g := uuidv7.NewGeneratorWith(sequenceSource(firstWord, secondWord))
		if got := g.GenerateAt(c.level, instanteConhecido).String(); got != c.want {
			t.Errorf("GenerateAt(nível %d): %s, esperado %s", c.level, got, c.want)
		}
	}

	// Pelo relógio, os campos de tempo variam e os de entropia não: confere
	// só os bytes que a entropia decide, em hexadecimal montado fora da
	// biblioteca.
	for _, c := range []struct {
		level      uuidv7.Level
		first, end int // bytes conferidos: [first, end)
		want       string
	}{
		{uuidv7.Level1, 6, 16, "7defbedcba9876543210"},
		{uuidv7.Level(9), 6, 16, "7defbedcba9876543210"},
		{uuidv7.Level2, 8, 16, "8123456789abcdef"},
		{uuidv7.Level3, 10, 16, "456789abcdef"},
	} {
		g := uuidv7.NewGeneratorWith(sequenceSource(firstWord, secondWord))
		u := g.Generate(c.level)
		if got := hex.EncodeToString(u[c.first:c.end]); got != c.want {
			t.Errorf("Generate(nível %d): bytes %d a %d = %s, esperado %s", c.level, c.first, c.end-1, got, c.want)
		}
		// No Nível 3 o nibble baixo do byte 9 também é entropia: os bits 51
		// a 48 de r2.
		if c.level == uuidv7.Level3 && u[9]&0x0F != 0x3 {
			t.Errorf("Generate(nível 3): nibble baixo do byte 9 = %#x, esperado 0x3", u[9]&0x0F)
		}
	}
}

// TestReaderGeneratorByteOrderAndShortReads trava o contrato de
// docs/02-instante-e-entropia.md seção 7 para o gerador sobre io.Reader:
// cada palavra é formada por 8 bytes em ordem de rede, o Nível 1 lê r1
// antes de r2, e uma leitura curta sem erro é completada, não aceita pela
// metade. O leitor entrega um byte por chamada, então uma implementação
// que confiasse numa única leitura ficaria com sete bytes zerados em cada
// palavra, em silêncio.
func TestReaderGeneratorByteOrderAndShortReads(t *testing.T) {
	// 24 bytes: 0x01 a 0x10 para o Nível 1, 0x11 a 0x18 para o Nível 3.
	g := uuidv7.NewGeneratorWithReader(iotest.OneByteReader(bytes.NewReader(countingBytes(24))))

	if got, want := g.GenerateAt(uuidv7.Level1, instanteConhecido).String(), "01a09076-bdfb-7708-890a-0b0c0d0e0f10"; got != want {
		t.Errorf("Nível 1 com os bytes 0x01 a 0x10: %s, esperado %s", got, want)
	}
	if got, want := g.GenerateAt(uuidv7.Level3, instanteConhecido).String(), "01a09076-bdfb-71c8-b152-131415161718"; got != want {
		t.Errorf("Nível 3 com os bytes 0x11 a 0x18: %s, esperado %s", got, want)
	}

	// Os 24 bytes acabaram exatamente: um consumo maior teria entrado em
	// pânico acima, e um menor deixaria bytes para esta geração, que tem de
	// falhar alto com o erro da fonte em vez de completar a palavra com zeros.
	if p := panicValue(func() { g.GenerateAt(uuidv7.Level2, instanteConhecido) }); p != uuidv7.ErrEntropySource { //nolint:errorlint // o valor do pânico é o próprio contrato testado
		t.Errorf("leitor esgotado: pânico com %v, esperado ErrEntropySource", p)
	}
}

// TestCryptoGeneratorReadsCryptoRandReader confere que NewCryptoGenerator
// tira a entropia de crypto/rand, e não do gerador padrão. O leitor global
// de crypto/rand é trocado por um leitor de bytes conhecidos só durante a
// construção: o gerador precisa capturá-lo ali, e os bits gerados depois,
// com o leitor original já restaurado, têm de ser exatamente os desse
// leitor. Uma fonte criptográfica trocada em silêncio pela do runtime é a
// degradação que docs/02-instante-e-entropia.md seção 6 proíbe, e nenhum
// outro teste a percebe, porque os dois geradores produzem UUIDv7
// igualmente válidos.
//
// Trocar uma variável global é seguro aqui porque nenhum teste do pacote
// usa t.Parallel. Um teste paralelo que leia crypto/rand.Reader tornaria
// esta troca uma corrida de dados.
func TestCryptoGeneratorReadsCryptoRandReader(t *testing.T) {
	g := func() *uuidv7.Generator {
		original := crand.Reader
		defer func() { crand.Reader = original }()
		crand.Reader = bytes.NewReader(countingBytes(16))
		return uuidv7.NewCryptoGenerator()
	}()

	if got, want := g.GenerateAt(uuidv7.Level1, instanteConhecido).String(), "01a09076-bdfb-7708-890a-0b0c0d0e0f10"; got != want {
		t.Errorf("NewCryptoGenerator não leu de crypto/rand.Reader: %s, esperado %s", got, want)
	}
}

// TestSuiteHasNoParallelTests trava a condição de que depende
// TestCryptoGeneratorReadsCryptoRandReader: nenhum arquivo de teste do
// pacote chama Parallel. Escrita só na documentação, a regra seria
// quebrada pelo primeiro teste paralelo acrescentado, e a troca de
// crypto/rand.Reader viraria uma corrida de dados que o detector de
// corrida só denunciaria às vezes.
func TestSuiteHasNoParallelTests(t *testing.T) {
	files, err := filepath.Glob("*_test.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("não foi possível listar os arquivos de teste do pacote: %v", err)
	}
	// Montado em duas partes para que este arquivo não acuse a si mesmo.
	// b.RunParallel, dos benchmarks, não casa: não roda junto dos testes.
	call := []byte("." + "Parallel(")
	for _, name := range files {
		src, err := os.ReadFile(name) //nolint:gosec // lê só os arquivos de teste do próprio pacote, listados por Glob
		if err != nil {
			t.Fatalf("lendo %s: %v", name, err)
		}
		if bytes.Contains(src, call) {
			t.Errorf("%s chama Parallel; a suíte precisa rodar em série (ver TestCryptoGeneratorReadsCryptoRandReader)", name)
		}
	}
}

// TestTextFormsUseTheGeneratorEntropy confere que as formas em texto usam
// a fonte do próprio gerador, como as binárias. Com entropia constante, os
// bits livres da string são os da fonte; um método que recorresse ao
// gerador padrão devolveria bits aleatórios, válidos e indistinguíveis por
// qualquer outro teste, e num gerador criptográfico trocaria crypto/rand
// pelo runtime sem aviso.
func TestTextFormsUseTheGeneratorEntropy(t *testing.T) {
	for _, c := range []struct {
		name   string
		source uint64
		level  uuidv7.Level
		suffix string // o fim da string, que só a entropia decide
	}{
		{"entropia nula, nível 1", 0, uuidv7.Level1, "-7000-8000-000000000000"},
		{"entropia em um, nível 1", ^uint64(0), uuidv7.Level1, "-7fff-bfff-ffffffffffff"},
		{"entropia nula, nível 2", 0, uuidv7.Level2, "-8000-000000000000"},
		{"entropia em um, nível 2", ^uint64(0), uuidv7.Level2, "-bfff-ffffffffffff"},
		{"entropia nula, nível 3", 0, uuidv7.Level3, "0-000000000000"},
		{"entropia em um, nível 3", ^uint64(0), uuidv7.Level3, "f-ffffffffffff"},
	} {
		g := uuidv7.NewGeneratorWith(constantSource(c.source))
		if s := g.GenerateString(c.level); !strings.HasSuffix(s, c.suffix) {
			t.Errorf("GenerateString, %s: %s não termina em %s", c.name, s, c.suffix)
		}
		if s := g.GenerateAtString(c.level, instanteConhecido); !strings.HasSuffix(s, c.suffix) {
			t.Errorf("GenerateAtString, %s: %s não termina em %s", c.name, s, c.suffix)
		}
	}
}

// TestDefaultGeneratorDrawsIndependentWords confere que o gerador padrão
// sorteia duas palavras independentes no Nível 1. A contagem de sorteios
// de TestEntropyDrawsPerLevel só vale para NewGeneratorWith, onde a fonte é
// injetável; o gerador padrão lê do runtime, e uma palavra repetida faria
// rand_a sempre igual aos 12 bits baixos de rand_b, derrubando o Nível 1 de
// 74 para 62 bits de entropia sem nenhum sinal visível.
//
// Com palavras independentes a coincidência ocorre com probabilidade
// 1/4096 por amostra: em 20.000 amostras a média é menos de 5, e passar do
// limite de 1/64 das amostras por azar tem probabilidade abaixo de
// 10^-400, pela cota de Chernoff.
func TestDefaultGeneratorDrawsIndependentWords(t *testing.T) {
	const samples = 20_000
	g := uuidv7.NewGenerator()
	for _, c := range []struct {
		name string
		gen  func() uuidv7.UUID
	}{
		{"NewGenerator().Generate", func() uuidv7.UUID { return g.Generate(uuidv7.Level1) }},
		{"NewGenerator().GenerateAt", func() uuidv7.UUID { return g.GenerateAt(uuidv7.Level1, instanteConhecido) }},
		{"Generate do pacote", func() uuidv7.UUID { return uuidv7.Generate(uuidv7.Level1) }},
	} {
		matches := 0
		for i := 0; i < samples; i++ {
			u := c.gen()
			randA := uint16(u[6]&0x0F)<<8 | uint16(u[7])
			lowB := uint16(u[14]&0x0F)<<8 | uint16(u[15])
			if randA == lowB {
				matches++
			}
		}
		if matches > samples/64 {
			t.Errorf("%s: rand_a coincidiu com os 12 bits baixos de rand_b em %d de %d amostras; esperado perto de %d",
				c.name, matches, samples, samples/4096)
		}
	}
}
