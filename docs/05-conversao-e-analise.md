# 05. Conversão e análise de texto

Os algoritmos que levam os 16 bytes à forma canônica e de volta: a
formatação, a escrita em buffer do chamador, o analisador estrito, o
analisador permissivo em quatro formatos e a taxonomia de erros que os
dois devolvem. Todos são imunes a pânico e a leitura fora dos limites, e
nenhum aloca além do resultado.

A análise é de **forma**: um UUID bem formado de qualquer versão é
aceito. A recusa de tudo o que não é UUIDv7 fica nas leituras de tempo
([03-leitura-do-instante.md](03-leitura-do-instante.md) §2).

---

## 1. Formatação canônica

- Utilizar uma tabela de caracteres hexadecimais minúsculos
  `"0123456789abcdef"`.
- Gravar diretamente em um buffer fixo de 36 caracteres, inserindo os
  hífens nos índices 8, 13, 18 e 23, sem alocações intermediárias. A
  conversão que devolve string (`String()`) aloca exatamente uma vez, a
  do próprio resultado.
- `BinaryToString(u)` é apelido explícito de `String()`.

Na implementação de referência a formatação do caminho quente e a dos
serializadores são a mesma lógica escrita duas vezes, de propósito: a
conversão para texto não deve pagar uma chamada de função por causa dos
serializadores ([10-decisoes.md](10-decisoes.md) §2).

**Armadilha de formatação.** Como o tipo satisfaz `fmt.Stringer`, os
verbos `%x` e `%X` formatam esta string (72 caracteres), e não os 16
bytes; ver `Bytes` em [03-leitura-do-instante.md](03-leitura-do-instante.md)
§5.

## 2. Escrita em buffer do chamador

- **`AppendTo(dst) dst`**: escreve os 36 bytes da forma canônica no fim
  do buffer do chamador e devolve o buffer estendido, **sem alocar**
  quando houver capacidade. Passar `nil` é válido e aloca os 36 bytes. É
  o caminho previsto para serializar grandes volumes (uma linha de log,
  um corpo JSON montado à mão): reaproveite o buffer com
  `buf = u.AppendTo(buf[:0])`. Medido na implementação de referência
  (Apple M2): cerca de 18,8 ns e zero alocações, contra cerca de 26 ns e
  uma alocação de 48 bytes de `String()`
  ([09-testes-e-benchmark.md](09-testes-e-benchmark.md) §10).
- **`AppendText(dst) (dst, erro)`**: `AppendTo` com a assinatura da
  interface de anexação de texto da linguagem (`encoding.TextAppender`,
  Go 1.24); o erro é sempre nulo. A interface não é referenciada no
  pacote, então o método compila nas versões anteriores e o `go.mod`
  permanece em 1.22.
- **`AppendBinary(dst) (dst, erro)`**: o equivalente binário, escrevendo
  os 16 bytes em ordem de rede no fim do buffer, também sem alocar quando
  houver capacidade (`encoding.BinaryAppender`, Go 1.24). O conteúdo é
  idêntico ao da serialização binária de
  [06-serializacao-e-banco.md](06-serializacao-e-banco.md) §2, portanto
  acrescentá-lo não muda formato de dados gravado.

**Regra normativa: os dois anexadores andam juntos.** Uma implementação
que ofereça um **DEVE** oferecer o outro. Se o motivo de existir o
anexador de texto é serializar em volume sem alocar, o mesmo motivo vale
para os bytes, e quem grava em coluna binária é justamente quem grava em
volume. Em linguagens com as duas interfaces de anexação, satisfazer só a
de texto deixa o tipo pela metade em todo consumidor genérico que prefira
anexar a alocar.

O lado binário não ganha uma segunda forma sem erro, ao contrário do de
texto: a formatação canônica é um cálculo, os 16 bytes não são;
`append(dst, u[:]...)` é literalmente o corpo do método
([10-decisoes.md](10-decisoes.md) §3).

---

## 3. Análise estrita: `FromString`

Aceita apenas a forma canônica `8-4-4-4-12`, em maiúsculas ou
minúsculas. `StringToBinary(s)` é apelido explícito.

- Validar comprimento exato de 36 caracteres.
- Validar `s[8] == '-'`, `s[13] == '-'`, `s[18] == '-'` e `s[23] == '-'`.
- **REGRA CRÍTICA DE SEGURANÇA.** **NUNCA** decodifique a string
  percorrendo caractere por caractere e avançando um índice ao encontrar
  um hífen. Com hífens extras em posições inesperadas o laço desalinha e
  tenta ler `s[36]`: pânico de estouro de vetor e queda do processo
  (armadilha 1 de [07-armadilhas.md](07-armadilhas.md)).
  **DECODIFIQUE SEMPRE via tabela de deslocamentos fixos**:

  ```text
  hexOffsets = [16]int{0, 2, 4, 6, 9, 11, 14, 16, 19, 21, 24, 26, 28, 30, 32, 34}
  ```

  Para cada byte `i` de 0 a 15, decodifique os dois caracteres em
  `s[hexOffsets[i]]` e `s[hexOffsets[i]+1]`. Um hífen fora do lugar cai
  numa posição de dígito e é recusado por não ser hexadecimal.
- Se qualquer caractere não for hexadecimal válido (`0..9`, `a..f`,
  `A..F`), devolver erro de formato e o UUID zerado: nenhum byte
  parcialmente decodificado pode vazar.

**Regra normativa: o analisador estrito não embrulha.** Devolve o
sentinela de formato **puro** (`ErrInvalidFormat`) em todos os casos,
inclusive comprimento errado, para que a comparação por igualdade direta
(`err == ErrInvalidFormat`) valha. É herança da biblioteca de origem,
mantida de propósito ([10-decisoes.md](10-decisoes.md) §3).

A extração de tempo em texto ([03-leitura-do-instante.md](03-leitura-do-instante.md)
§4) usa este analisador, e devolve o mesmo sentinela puro.

## 4. Analisador permissivo: `Parse` e companhia

`Parse` **DEVE** aceitar quatro formatos, em maiúsculas ou minúsculas:

1. **Canônico com hífens** (36 caracteres): decodificado como na seção 3.
2. **Sem hífens** (32 caracteres): decodificado a cada 2 caracteres
   (`i * 2`).
3. **Entre chaves** (38 caracteres): inicia com `{` e termina com `}`,
   contendo os 36 caracteres canônicos.
4. **Prefixo URN** (45 caracteres): inicia com `urn:uuid:` (insensível a
   maiúsculas), seguido dos 36 caracteres canônicos.

Qualquer outro comprimento é rejeitado imediatamente como comprimento
inválido. Em caso de erro o UUID devolvido é sempre `Nil`.

Complementos, todos com o mesmo critério de forma:

- **`ParseBytes([]byte)`**: o mesmo a partir de uma sequência de bytes,
  sem converter para string e sem alocar (na implementação de referência
  o analisador é genérico sobre `string` e `[]byte` por isso).
- **`Validate(string) erro`**: apenas valida, devolvendo só o erro.
- **`FromBytes([]byte) (UUID, erro)`**: constrói a partir de exatamente
  16 bytes em ordem de rede; outro tamanho devolve comprimento inválido.
- **`MustParse(string) UUID`**: entra em pânico se a string for inválida.
  Destina-se a constantes do próprio código; nunca a entrada externa.
- **`Must(UUID, erro) UUID`**: devolve o UUID ou entra em pânico com o
  próprio erro recebido, para encadear com funções que devolvem par de
  valores; quem recupera o pânico reconhece o erro pelo sentinela.

**Robustez obrigatória.** O analisador estrito e o permissivo, nas quatro
formas, **DEVEM** sobreviver a todas as 36 × 256 mutações de um byte
sobre uma entrada válida sem pânico, e aceitar **exatamente** as mutações
que mantêm a forma, com o valor conferido por um decodificador
independente. As campanhas de fuzzing exigem que toda entrada aceita pelo
permissivo seja, sem distinção de caixa, uma das quatro formas do valor
lido; a ida e volta sozinha não percebe um hífen ou um dois-pontos que
deixaram de ser conferidos ([08-casos-de-teste.md](08-casos-de-teste.md),
caso 2).

---

## 5. Taxonomia de erros

A biblioteca **DEVE** expor um erro sentinela de formato e erros mais
específicos que o **embrulham**, de modo que a verificação pelo sentinela
(`errors.Is` em Go, ou o equivalente da linguagem) continue verdadeira
para qualquer um deles. Quem trata só a presença de erro não precisa
conhecer a tabela; quem decide pelo tipo, precisa. A tabela é normativa:
duas implementações conformes devolvem o mesmo erro para a mesma entrada.

| Operação | Entrada | Erro devolvido |
|:---|:---|:---|
| Analisador estrito (seção 3) e a extração de tempo em texto | qualquer recusa de texto, inclusive comprimento | sentinela de formato, **puro** (`ErrInvalidFormat`) |
| Analisador permissivo (seção 4) | comprimento fora de 32, 36, 38 e 45 | comprimento inválido (`ErrInvalidLength`) |
| Analisador permissivo | 38 caracteres sem `{` na primeira posição ou sem `}` na última | chaves inválidas (`ErrInvalidBrackets`) |
| Analisador permissivo | 45 caracteres sem o prefixo `urn:uuid:` | sentinela de formato |
| Analisador permissivo | dígito não hexadecimal, ou hífen fora das posições 8, 13, 18 e 23 | sentinela de formato |
| Construção a partir de bytes crus (`FromBytes`) | tamanho diferente de 16 | comprimento inválido |
| Desserialização binária | tamanho diferente de 16 | comprimento inválido |
| Desserialização de texto | as mesmas entradas do analisador permissivo | os mesmos erros do analisador permissivo |
| Desserialização JSON do tipo anulável | valor que não é string nem `null`, ou sintaxe JSON inválida | sentinela de formato |
| Desserialização JSON do tipo anulável | string cujo conteúdo o analisador permissivo recusa | os erros do analisador permissivo |
| Leitura de valor de banco | texto, ou bytes em tamanho diferente de 16, que o analisador permissivo recusa | os erros do analisador permissivo |
| Leitura de valor de banco | tipo que não é texto, bytes nem nulo | tipo não suportado (`ErrInvalidScanType`) |
| Extração de tempo, binária ou em texto | UUID aceito cuja versão não é 7 ou cuja variante não é `0b10`, inclusive o nulo | erro de versão, **puro** (`ErrNotV7`) |
| Fonte de entropia do chamador (leitor) | leitura que falhou durante a geração | pânico com o erro de fonte de entropia (`ErrEntropySource`) como valor |
| Construção do gerador sobre fonte ou leitor | fonte nula ou leitor nulo | pânico na construção |

Todo erro de comprimento e de chaves **DEVE** embrulhar o sentinela de
formato. O erro de tipo não suportado, o de versão e o de fonte de
entropia são famílias à parte e **NÃO** embrulham o sentinela: não são
recusas de texto. A desserialização JSON do tipo simples delega ao
codificador da linguagem, que devolve o erro dele para valores que não
são string e o erro do analisador permissivo para strings recusadas.

**Assimetria registrada.** No analisador permissivo, as chaves
malformadas têm erro próprio e o prefixo URN inválido **não** tem: ele
devolve o sentinela puro, na mesma classe do dígito inválido. As duas
situações são análogas e a assimetria não tem motivo técnico; veio com a
primeira versão do analisador permissivo, na biblioteca de origem, e é
mantida para que as duas bibliotecas classifiquem a mesma entrada do
mesmo modo. Uma reimplementação **DEVE** reproduzi-la
([10-decisoes.md](10-decisoes.md) §3).

**Receptor intacto.** Em todos os casos de recusa o valor devolvido
**DEVE** ser o UUID zerado, e nenhum byte parcialmente decodificado pode
vazar; nas desserializações com receptor, o receptor **NÃO DEVE** ser
alterado. No tipo anulável a regra alcança **as duas** componentes, o
identificador e o booleano de presença.

**Exceção única: a leitura de valor de banco do tipo anulável.** Ali o
identificador continua intacto, mas o booleano **DEVE** cair para falso.
A interface de leitura de banco recebe um destino **reaproveitado a cada
linha**, e um chamador que ignore o erro leria o valor da linha anterior
como se fosse o da linha que falhou; derrubar o booleano transforma esse
descuido em ausência de valor, e não em dado errado. As desserializações
não têm destino reaproveitado, e por isso não têm a exceção
([06-serializacao-e-banco.md](06-serializacao-e-banco.md) §4;
[10-decisoes.md](10-decisoes.md) §3).

Os dois auxiliares que entram em pânico em vez de devolver erro,
`MustParse` e `Must`, não fazem parte da tabela.

A verificação de cada linha, com as entradas mínimas, é o caso 17 de
[08-casos-de-teste.md](08-casos-de-teste.md).
