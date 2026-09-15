# Erros e pânicos: como tratar cada um

Leia este arquivo ao escrever o tratamento de erro de código que analisa
texto, lê tempo de um UUID, lê do banco ou configura a fonte de entropia.

## Os erros exportados

| Erro | Quem devolve | Embrulha `ErrInvalidFormat`? | Como reconhecer |
|:---|:---|:---|:---|
| `ErrInvalidFormat` | `FromString`, `StringToBinary`, `Import` (texto recusado), `Parse` (dígito inválido, hífen fora de lugar, prefixo `urn:uuid:` errado), `NullUUID.UnmarshalJSON` (valor que não é string nem `null`, ou JSON inválido) | é o sentinela | `errors.Is(err, uuidv7.ErrInvalidFormat)`; de `FromString` e `Import` também `err == uuidv7.ErrInvalidFormat` |
| `ErrInvalidLength` | `Parse`, `ParseBytes`, `Validate`, `UnmarshalText`, `Scan` com texto: comprimento fora de 32, 36, 38 e 45; `FromBytes` e `UnmarshalBinary`: tamanho diferente de 16 | sim | `errors.Is(err, uuidv7.ErrInvalidLength)` |
| `ErrInvalidBrackets` | `Parse` e derivados: 38 caracteres sem `{` no início ou `}` no fim | sim | `errors.Is(err, uuidv7.ErrInvalidBrackets)` |
| `ErrNotV7` | `Import`, `ImportBinary`: UUID bem formado que não é de versão 7 com a variante da RFC (inclusive `Nil`) | **não** | `errors.Is(err, uuidv7.ErrNotV7)` |
| `ErrInvalidScanType` | `Scan` dos quatro tipos: valor que não é `string`, `[]byte` nem `nil` | **não** | `errors.Is(err, uuidv7.ErrInvalidScanType)` |
| `ErrEntropySource` | valor de **pânico** quando a leitura do `io.Reader` de `NewGeneratorWithReader` (e `NewCryptoGenerator`) falha durante a geração | **não** | `recover()` e `errors.Is(v.(error), uuidv7.ErrEntropySource)` |

`Timestamp` e `TimestampWithLevel` não devolvem erro: devolvem `false`
com o instante zero para um UUID que não é v7.

## Padrão recomendado na entrada de uma API

```go
u, err := uuidv7.Parse(entrada)
if err != nil {
	// texto malformado: 400 Bad Request, com err na mensagem
	return err
}
if !u.IsValid() {
	// bem formado, mas não é UUIDv7: a decisão é sua (400, ou aceitar e
	// só não ler o tempo dele)
	return uuidv7.ErrNotV7
}
```

Decidindo pelo tipo, quando a mensagem ao usuário depende dele:

```go
switch {
case errors.Is(err, uuidv7.ErrInvalidLength):
	// comprimento errado
case errors.Is(err, uuidv7.ErrInvalidBrackets):
	// chaves malformadas
case errors.Is(err, uuidv7.ErrInvalidFormat):
	// qualquer outra recusa de texto (dígito inválido, hífen fora de lugar, prefixo)
}
```

## Ao ler o tempo

```go
t, err := uuidv7.Import(texto)
switch {
case errors.Is(err, uuidv7.ErrInvalidFormat):
	// o texto não é um UUID canônico (Import usa o analisador estrito)
case errors.Is(err, uuidv7.ErrNotV7):
	// é um UUID, mas de outra versão: não carrega instante
case err != nil:
	// não acontece hoje; deixe o caso para não engolir um erro futuro
}
```

`ErrNotV7` não é erro de formato: `errors.Is(err, uuidv7.ErrInvalidFormat)`
é falso para ele. Um `switch` que só trate o formato deixa passar o
UUIDv4.

## Ao ler do banco

- Em caso de erro, `UUID.Scan` deixa o receptor como estava; `NullUUID.Scan`
  e `NullBinaryUUID.Scan` deixam o UUID como estava mas derrubam `Valid`
  para `false`, porque `database/sql` reaproveita o destino entre linhas
  e um erro ignorado leria a linha anterior como se fosse a atual.
- `NULL`, `""` e `[]byte{}` não são erro: gravam `Nil` (e `Valid=false`
  nos anuláveis).
- `ErrInvalidScanType` indica um driver que entregou um tipo inesperado
  (um inteiro, por exemplo); é defeito de mapeamento, não de dado.

## Pânicos

| Pânico | Quando | O que fazer |
|:---|:---|:---|
| `MustParse` com string inválida | só deve ser usado com constantes do código | corrija a constante; nunca use com entrada externa |
| `Must(u, err)` com `err != nil` | propaga o próprio `err` como valor do pânico | use só onde um pânico é aceitável |
| `NewGeneratorWith(nil)`, `NewGeneratorWithReader(nil)` | na construção, no boot | erro de configuração: corrija a chamada |
| `ErrEntropySource` | leitura do `io.Reader` falhou durante a geração | a fonte de entropia está quebrada; a biblioteca nunca degrada em silêncio para um gerador previsível |

Nenhuma função de geração devolve erro. `Generate`, `GenerateString`,
`GenerateAt`, `GenerateAtString`, `GenerateV7*`, `MinAt`, `MaxAt` e
`RangeAt` sempre devolvem um UUIDv7 válido; instantes fora da faixa
representável saturam em vez de falhar.
