# 01. Escopo e layout de bits

Objetivo da biblioteca, o que ela cobre e o que ficou de fora, a anatomia
do UUIDv7 da RFC 9562 e a distribuição dos bits em cada um dos três
níveis de precisão temporal.

Este arquivo e os que o seguem formam a especificação **agnóstica de
linguagem**: descrevem, sem depender de bibliotecas externas ou de
recursos exclusivos do Go, tudo o que é necessário para construir a
biblioteca do zero em qualquer linguagem e obter uma implementação
interoperável, de alto desempenho e estritamente correta, sem repetir
os defeitos históricos catalogados em
[07-armadilhas.md](07-armadilhas.md). Os nomes citados (`Generate`,
`MinAt`, `ErrNotV7`) são os da implementação de referência em Go e
servem para identificar cada operação; uma reimplementação pode
adaptá-los à convenção da sua linguagem.

As palavras **DEVE** e **NÃO DEVE** marcam regras normativas. Tudo o que
não as carrega é explicação, exemplo ou registro de medição.

---

## 1. Objetivo

Construir uma biblioteca leve, de altíssimo desempenho (dezenas de
nanossegundos por identificador, zero alocações de heap no caminho
quente) e segura para concorrência pesada, dedicada **exclusivamente ao
UUIDv7 da RFC 9562**, com extensão de precisão temporal sub-milissegundo
nos dois sentidos: gerar o identificador com o instante embutido e ler
de volta, do identificador, o instante gravado nele.

A biblioteca não tem dependências: só a biblioteca padrão da linguagem.

## 2. O que a biblioteca cobre

1. **Geração do UUIDv7 multinível**, pelo relógio do sistema:
   - **Nível 1**: o UUIDv7 padrão da RFC 9562, com carimbo Unix em
     milissegundos e 74 bits de entropia.
   - **Nível 2**: carimbo em milissegundos e os **microssegundos** do
     instante gravados em `rand_a`; 62 bits de entropia.
   - **Nível 3**: carimbo em milissegundos, microssegundos em `rand_a` e
     os **nanossegundos** nos 10 bits altos de `rand_b`; 52 bits de
     entropia.
   Os três existem também pelo nome, sem argumento de nível (seção 8).
   A aritmética do instante, o sorteio da entropia e a ordenação estão
   em [02-instante-e-entropia.md](02-instante-e-entropia.md).
2. **Leitura do instante**, o sentido inverso: duas leituras com
   políticas opostas, ambas normativas, e ambas recusando qualquer UUID
   que não seja de versão 7 com a variante da RFC. Ver
   [03-leitura-do-instante.md](03-leitura-do-instante.md).
3. **Construção a partir de um instante explícito**: as fronteiras de um
   instante, para consulta por intervalo pelo índice da própria chave
   primária, e a geração para um instante informado, para reprocessar
   histórico preservando a ordenação. Ver
   [04-construcao-por-instante.md](04-construcao-por-instante.md).
4. **Conversão e análise segura**: formatação canônica, analisador
   estrito, analisador permissivo em quatro formatos, imunes a pânico e
   a leitura fora dos limites, com uma taxonomia de erros normativa. A
   análise é de forma e não confere a versão. Ver
   [05-conversao-e-analise.md](05-conversao-e-analise.md).
5. **Serialização e integração**: texto e JSON como string canônica,
   binário de 16 bytes, colunas de banco textuais e binárias, com e sem
   `NULL`. Ver [06-serializacao-e-banco.md](06-serializacao-e-banco.md).

## 3. Fora do escopo

As demais versões da RFC 9562 (1, 2, 3, 4, 5, 6 e 8) não são geradas nem
interpretadas, e com elas ficam de fora o estado de relógio gregoriano, a
sequência de relógio, o identificador de nó, os espaços de nomes e
qualquer camada de compatibilidade de nomes com outros pacotes de UUID.
A biblioteca de origem deste projeto, `go-loghub-uuid`, cobre essas
versões. O que ficou de apoio (análise permissiva, serialização,
integração com banco, fronteiras e geração por instante) ficou porque
serve para receber, guardar e consultar UUIDv7. A decisão está em
[10-decisoes.md](10-decisoes.md) §3.

---

## 4. Anatomia do UUID

Um UUID tem exatamente **128 bits**, ou **16 bytes**, dispostos em ordem
de rede (*big-endian*: o byte 0 é o mais significativo). Em todo UUIDv7
dois campos são invioláveis:

- **Versão (`ver`, 4 bits)**: o nibble alto do **byte 6** (bits 48 a 51
  do UUID). Vale `7`. A RFC define versões de `1` a `8`; esta biblioteca
  só gera a `7` e só lê tempo da `7`.
- **Variante (`var`, 2 bits)**: os 2 bits mais significativos do **byte
  8** (bits 64 e 65). A RFC 9562 exige `0b10`, o que deixa o byte 8 na
  faixa `0x80..0xBF` e o seu nibble alto em `8`, `9`, `a` ou `b`.

Layout dos 16 bytes:

```text
byte:  0    1    2    3    4    5    6    7    8    9   10   11   12   13   14   15
      [-------- unix_ts_ms (48) --------][V|aa][ aa ][v|bb][----------- rand_b -----------]
                                          7  ^             10 ^
                                          versão            variante
```

- bytes 0..5: `unix_ts_ms`, milissegundos desde a época Unix;
- byte 6: nibble alto `0x7` (versão), nibble baixo = topo de `rand_a`;
- byte 7: o resto de `rand_a` (12 bits no total);
- byte 8: 2 bits altos `10` (variante), 6 bits baixos = topo de `rand_b`;
- bytes 9..15: o resto de `rand_b` (62 bits no total).

## 5. Forma canônica em texto

**36 caracteres** hexadecimais minúsculos agrupados como **`8-4-4-4-12`**,
com hífens nas posições 8, 13, 18 e 23 (índices a partir de zero). No
UUIDv7 os cinco grupos se distribuem assim pelos campos da seção 6:

| Grupo | Dígitos | Bits | Campos |
|:---|:---|:---|:---|
| 1 | 8 | 32 | `unix_ts_ms[47:16]` |
| 2 | 4 | 16 | `unix_ts_ms[15:0]` |
| 3 | 4 | 16 | `ver` (o primeiro dígito, sempre `7`) e `rand_a` (os três seguintes) |
| 4 | 4 | 16 | `var` e `rand_b[61:48]` (o primeiro dígito fica em `8`, `9`, `a` ou `b`) |
| 5 | 12 | 48 | `rand_b[47:0]` |

Exemplo, com o `7` da versão no início do terceiro grupo e o `8` da
variante no início do quarto:

```text
0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff
```

A ordem lexicográfica das strings canônicas acompanha a ordem dos bytes,
e portanto a ordem cronológica (seção 6). Os algoritmos de formatação e
de análise estão em [05-conversao-e-analise.md](05-conversao-e-analise.md).

---

## 6. Distribuição de bits por nível

O UUIDv7 dedica os 48 bits iniciais ao carimbo Unix em milissegundos, o
que dá ordenação temporal natural por comparação byte a byte. Os 74 bits
restantes são, na RFC, aleatórios (`rand_a` com 12 bits e `rand_b` com
62). Esta biblioteca sobrescreve parte deles com precisão
sub-milissegundo, **sempre preservando a versão e a variante**, de modo
que o resultado é um UUIDv7 válido em qualquer nível:

| Campo         | Tamanho | Posição (bits) | Bytes    | Nível 1 (RFC)       | Nível 2 (+us)           | Nível 3 (+us +ns)       |
|:--------------|:--------|:---------------|:---------|:--------------------|:------------------------|:------------------------|
| `unix_ts_ms`  | 48 bits | 0 a 47         | 0..5     | ms Unix (UTC)       | ms Unix (UTC)           | ms Unix (UTC)           |
| `ver`         | 4 bits  | 48 a 51        | 6 (alto) | `0x7`               | `0x7`                   | `0x7`                   |
| `rand_a`      | 12 bits | 52 a 63        | 6 (baixo)..7 | aleatório       | microssegundos (0..999) | microssegundos (0..999) |
| `var`         | 2 bits  | 64 a 65        | 8 (alto) | `0b10`              | `0b10`                  | `0b10`                  |
| `rand_b`      | 62 bits | 66 a 127       | 8 (baixo)..15 | aleatório (62 bits) | aleatório (62 bits) | 10 bits de ns (0..999) + 52 bits aleatórios |

Microssegundos e nanossegundos são sempre a fração do campo acima, de 0
a 999: `micro` cabe em 10 bits e ocupa os 12 de `rand_a`; `nano` cabe em
10 bits e ocupa exatamente `rand_b[61:52]`. Como os bits de precisão
ficam logo depois dos milissegundos, a ordem lexicográfica continua
cronológica em todos os níveis, na resolução do nível (ver
[02-instante-e-entropia.md](02-instante-e-entropia.md) §4 para o
desempate dentro do mesmo instante).

A entropia restante é de **74 bits** no Nível 1, **62** no Nível 2 e
**52** no Nível 3. Níveis desconhecidos **DEVEM** ser tratados como
Nível 1, em toda forma de gerar e de ler.

O layout é escrito em código exatamente duas vezes: uma no caminho
quente da geração pelo relógio e uma no empacotamento compartilhado por
tudo que constrói a partir de um instante. O limite é esse, e está
registrado em [10-decisoes.md](10-decisoes.md) §2.

## 7. Codificação decimal do sub-milissegundo

Os microssegundos e os nanossegundos são **contagens decimais de 0 a
999**, gravadas em binário nos seus campos, e **não** a fração do
milissegundo multiplicada por 4096 que o Método 3 da RFC 9562 §6.2
descreve. O resultado continua sendo um UUIDv7 válido e
cronologicamente ordenável, mas:

- um leitor de terceiros que siga o Método 3 interpreta errado o
  sub-milissegundo (456 µs viram cerca de 111 µs), e a resolução de
  `rand_a` fica no microssegundo, e não em cerca de 244 ns;
- o Nível 3 usa 22 bits de tempo, além dos 12 que o método prevê.

Qualquer biblioteca lê o milissegundo destes identificadores; o
sub-milissegundo só esta biblioteca, ou uma implementação desta
especificação, lê corretamente. A codificação decimal é o que permite à
leitura por nível reconhecer ruído pela faixa
([03-leitura-do-instante.md](03-leitura-do-instante.md) §3): um valor
acima de 999 não pode ser tempo. A escolha é deliberada e está registrada
em [10-decisoes.md](10-decisoes.md) §3; mudá-la seria mudança de formato
de dados, travada pelos vetores dourados de
[08-casos-de-teste.md](08-casos-de-teste.md), caso 12.

## 8. Nomes por versão e por nível

O Nível 1 **DEVE** existir também sob o nome da versão (`GenerateV7`), e
cada um dos três níveis **DEVE** existir sob um nome que diga o nível
(`GenerateV7Level1`, `GenerateV7Level2`, `GenerateV7Level3`), todos como
método do gerador e como função de pacote, sem parâmetro de nível e sem
forma em texto. `GenerateV7` e `GenerateV7Level1` são o mesmo.

Cada nome é um apelido exato da geração no nível correspondente: mesmos
bytes, mesmo consumo de entropia
([02-instante-e-entropia.md](02-instante-e-entropia.md) §3) e mesma trava
de alocação ([08-casos-de-teste.md](08-casos-de-teste.md), caso 16). O
nome por versão designa o UUIDv7 da RFC, sem a extensão de precisão, e é
o que procura quem não conhece os níveis. Os nomes por nível existem para
que o nível, que é contrato de toda a coluna (fronteiras de níveis
distintos não compõem: [04-construcao-por-instante.md](04-construcao-por-instante.md)
§3), fique legível no ponto da chamada em vez de numa constante. A
constante continua sendo a única forma de dizer o nível **como valor**,
que é o que a geração por nível, a construção por instante e a leitura
por nível recebem.

Os apelidos **NÃO DEVEM** acrescentar custo ao caminho quente: a
implementação permanece na geração por nível, e os nomes apenas a chamam.
Um nome que gere em nível diferente do que declara é defeito, travado
pelo caso 1 de [08-casos-de-teste.md](08-casos-de-teste.md). A decisão
está em [10-decisoes.md](10-decisoes.md) §3.

## 9. Escolha do nível

| Nível    | Precisão embutida        | Entropia restante | Uso típico                                 |
|:---------|:-------------------------|:------------------|:-------------------------------------------|
| Nível 1  | milissegundos            | 74 bits           | UUIDv7 padrão; máxima compatibilidade      |
| Nível 2  | + microssegundos         | 62 bits           | ordenação mais fina dentro do mesmo ms     |
| Nível 3  | + micro e nanossegundos  | 52 bits           | logs e eventos de altíssima frequência     |

Quanto maior o nível, mais bits de tempo e menos bits aleatórios. Em
todos, a colisão é desprezível para volumes normais; uma aplicação que
dependa criticamente de unicidade entre máquinas combina o UUID com um
identificador de origem fora dele.

O nível é **contrato de toda a coluna**: as fronteiras de consulta e a
leitura por nível precisam do mesmo nível com que os identificadores
foram gravados, e ele não é dedutível do UUID. Escolha o nível ao criar a
tabela e não o mude.

**Resolução do relógio do host.** O Nível 3 só grava nanossegundos reais
se o relógio do sistema os fornecer. Em hosts cujo relógio tem resolução
de microssegundo, como o macOS, o campo de nanossegundos sai sempre zero:
são dez bits de entropia trocados por nada, sem ganho de ordenação em
relação ao Nível 2. O teste de diagnóstico `TestTieRateReport` informa
quantos instantes distintos o relógio do host oferece
([09-testes-e-benchmark.md](09-testes-e-benchmark.md) §1); use-o para
escolher o nível.
