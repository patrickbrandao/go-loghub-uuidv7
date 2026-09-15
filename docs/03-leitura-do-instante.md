# 03. Leitura do instante e inspeção

O sentido inverso da geração: o predicado que reconhece um UUIDv7, a
recusa de tudo o que não é UUIDv7 nas leituras de tempo, as duas
políticas de leitura (a que devolve um instante e descarta o que não pode
ser tempo, e a que devolve os campos crus sem julgar), e as operações de
inspeção e os valores especiais do tipo.

A leitura recebe identificadores de fora: texto de uma requisição, coluna
de banco, JSON de outro sistema. Nada impede que um deles seja de outra
versão, e um UUIDv4 lido como UUIDv7 devolve um instante sem sentido, sem
erro nenhum: os 48 bits altos de um UUIDv4 são aleatórios e caem em
qualquer data entre 1970 e 10889. Este arquivo fixa onde a recusa
acontece.

---

## 1. O predicado de validade

A biblioteca **DEVE** oferecer um predicado de validade (`IsValid`)
verdadeiro **se e somente se** o nibble de versão for `7` **e** os dois
bits de variante forem `0b10`. As duas condições são obrigatórias: um
identificador com nibble `7` e variante `0b11` (a reservada à Microsoft)
tem o mesmo layout aparente e não é UUIDv7. O UUID nulo é **recusado**,
porque não carrega versão nem variante; a verificação dele é feita pelo
predicado próprio (`IsZero`, seção 5). O UUID com todos os bits em um
também é recusado, pela mesma razão.

A verificação é de **forma**, não de origem: não há como saber se um
UUIDv7 foi gerado por esta biblioteca, nem em qual nível. Um UUIDv7
padrão de outro gerador é aceito e lido como qualquer outro, com as
políticas de nível das seções 3 e 4.

Na implementação de referência a forma legível `u[6]>>4 == 7 &&
u[8]>>6 == 0b10` foi mantida: a verificação em uma comparação só foi
medida e saiu mais lenta ([10-decisoes.md](10-decisoes.md) §3).

## 2. As leituras recusam o que o predicado recusa

As quatro leituras de tempo **DEVEM** conferir o predicado antes de
interpretar qualquer bit:

| Leitura | Recusa |
|:---|:---|
| Extração completa a partir dos 16 bytes (`ImportBinary`) | erro de versão (`ErrNotV7`), com a estrutura zerada |
| Extração completa a partir do texto (`Import`) | erro de formato puro se o texto for recusado pelo analisador estrito; erro de versão, com a estrutura zerada, se for aceito e o predicado recusar |
| Instante em milissegundo (`Timestamp`) | booleano falso, com o instante zero |
| Instante por nível (`TimestampWithLevel`) | booleano falso, com o instante zero, em qualquer nível |

**A análise continua sendo só de forma.** Os analisadores estrito e
permissivo, a desserialização de texto, de binário e de JSON e a leitura
de banco **NÃO DEVEM** recusar um UUID bem formado de outra versão.
Guardar, transportar e comparar um identificador não depende da versão,
e recusar na análise impediria até de registrar em log o valor que chegou
errado. Quem precisa exigir a versão logo na entrada chama o predicado
depois da análise.

**O erro de versão é família própria.** `ErrNotV7` **NÃO DEVE** embrulhar
o sentinela de formato ([05-conversao-e-analise.md](05-conversao-e-analise.md)
§5): o texto foi aceito, e a recusa é sobre o conteúdo. Quem trata só
erro de formato continua tratando só texto malformado.

**Custo.** A verificação é uma leitura de dois campos e duas comparações,
e a recusa devolve um erro pré-alocado: a extração continua sem alocação,
aceitando e recusando ([08-casos-de-teste.md](08-casos-de-teste.md), caso
16). Na implementação de referência a extração binária passou de cerca
de 1,7 ns para 2,2 ns por chamada, fora do caminho quente de geração.

A varredura das 64 combinações de versão e variante, e a recusa dos
exemplos reais da RFC das outras versões, são o caso 7 de
[08-casos-de-teste.md](08-casos-de-teste.md): uma implementação que
confira só a versão, ou só a variante, passa num teste com um único
UUIDv4 e falha lá.

---

## 3. Instante por nível: `Timestamp` e `TimestampWithLevel`

- **`Timestamp() (instante, bool)`**: o instante do carimbo de 48 bits,
  em UTC e com resolução de milissegundo. É o mesmo que a leitura por
  nível no Nível 1. Devolve falso, com o instante zero, se o predicado
  recusar.
- **`TimestampWithLevel(nível) (instante, bool)`**: o instante somando ao
  milissegundo a precisão sub-milissegundo gravada pelo nível informado.
  O nível **não** é dedutível do identificador, e por isso vem por
  parâmetro: informe o nível com que a coluna foi gravada. Devolve falso,
  com o instante zero, se o predicado recusar; no Nível 1 e em níveis
  desconhecidos devolve apenas o milissegundo.

**Regra normativa: descarte por faixa.** A condição de aproveitamento é
sobre o **valor lido**, não sobre o nível pedido. Se um campo
sub-milissegundo estiver fora da faixa de 0 a 999, ele denuncia entropia
em vez de tempo, e a implementação **DEVE** descartar a precisão
sub-milissegundo, devolvendo apenas o milissegundo do carimbo. Informar o
nível errado degrada para o milissegundo, que é um resultado utilizável;
não devolve erro nem uma data errada.

**Regra normativa: no Nível 3 os dois campos caem juntos.** Um `rand_a`
fora da faixa prova que o topo de `rand_b` também é ruído, porque os dois
vêm da mesma geração. A implementação **NÃO DEVE** aproveitar `nano`
quando `micro` foi descartado, mesmo que `nano` caiba em 0 a 999 por
acaso, o que ocorre em cerca de 98% dos casos (1000 valores válidos em
1024 possíveis). O caso simétrico vale igualmente: `nano` fora da faixa
descarta também `micro`. No Nível 2 o `nano` não é lido, e só o `micro`
decide.

**As duas leituras DEVEM devolver UTC** em todos os níveis, nos
desconhecidos e no caminho do descarte. O teste confere a identidade do
fuso, e não o nome ([08-casos-de-teste.md](08-casos-de-teste.md), caso
19).

Esta política é o **oposto** da extração completa da seção 4, que
entrega os bits sem julgar a faixa. A divergência é deliberada: uma
operação entrega um instante, a outra entrega os bits
([10-decisoes.md](10-decisoes.md) §3).

## 4. Extração completa: `ImportBinary` e `Import`

Extração dos campos de tempo de um UUIDv7 numa estrutura com quatro
campos (`Time`):

| Campo | Origem | Faixa |
|:---|:---|:---|
| `Seconds` | divisão inteira do carimbo de 48 bits por mil | segundos Unix |
| `Milliseconds` | o resto dessa divisão | sempre 0 a 999 |
| `Microseconds` | os 12 bits de `rand_a`, lidos como estão | 0 a 999 se gerado em Nível 2 ou 3; 0 a 4095 se os bits forem aleatórios |
| `Nanoseconds` | os 10 bits altos de `rand_b`, lidos como estão | 0 a 999 se gerado em Nível 3; 0 a 1023 se os bits forem aleatórios |

`Seconds` e `Milliseconds` saem do mesmo carimbo, e o recorte entre
segundo e fração é decisão de contrato, não consequência do layout.

- **`ImportBinary(UUID) (Time, erro)`**: devolve o erro de versão com a
  estrutura zerada se o predicado recusar.
- **`Import(texto) (Time, erro)`**: analisa o texto pelo analisador
  **estrito** ([05-conversao-e-analise.md](05-conversao-e-analise.md)
  §3), devolve o sentinela de formato **puro** com a estrutura zerada se
  o texto for recusado, e delega à forma binária, que pode então devolver
  o erro de versão.

**Regra normativa: a extração é cega quanto ao nível.** Aceito o UUIDv7,
ela **DEVE** sempre interpretar `rand_a` como microssegundos e o topo de
`rand_b` como nanossegundos, e **NÃO DEVE** validar faixa nem descartar
campo algum. A cegueira é quanto ao **nível**, não quanto à versão, que
já foi conferida na seção 2. A operação não recebe o nível, e ele não é
dedutível: em um UUIDv7 de Nível 1 esses campos carregam entropia, e a
extração devolve os bits lidos sem julgar a origem. É o chamador, que
sabe o nível, quem decide o que aproveitar. Não valide esses campos
contra 0..999 sem saber o nível de origem.

É a operação inversa da geração por instante
([04-construcao-por-instante.md](04-construcao-por-instante.md) §4), e a
estrutura devolvida **NÃO** passa pelo descarte da seção 3. O vetor que
distingue as duas políticas é o caso 13 de
[08-casos-de-teste.md](08-casos-de-teste.md); o único vetor de leitura de
tempo calculado fora deste projeto, o exemplo de UUIDv7 do apêndice A.6
da RFC 9562, é o caso 5.

---

## 5. Inspeção e valores

Operações que leem o que está no identificador sem interpretar tempo, e
os valores especiais do tipo. Nenhuma delas recusa nada: leem os bits
como estão.

- **`Version() byte`**: o nibble de versão. Vale `7` em todo UUID gerado
  pela biblioteca; um UUID de fora pode trazer qualquer valor de 0 a 15.
- **`Variant() byte`**: os 2 bits altos do byte 8. Vale `0b10` (2), a
  variante da RFC 9562, em todo UUID gerado. Os dois leitores **DEVEM**
  devolver o valor dos bits para qualquer combinação, e não só para o `7`
  e o `0b10` ([08-casos-de-teste.md](08-casos-de-teste.md), caso 1).
- **`IsValid() bool`**: o predicado da seção 1.
- **`IsZero() bool`**: verdadeiro somente para o UUID nulo. **DEVE**
  recusar um valor com um único byte diferente de zero, em cada posição.
- **`Nil`**: o UUID com os 16 bytes em zero
  (`00000000-0000-0000-0000-000000000000`). É o valor devolvido junto de
  toda recusa de análise e o que a leitura de banco grava para ausência
  de valor. Não é UUIDv7: o predicado o recusa. Na implementação de
  referência é uma variável exportada, sem função de acesso e sem defesa
  contra alteração; o contrato é a documentação pedir que não seja
  alterada ([10-decisoes.md](10-decisoes.md) §3).
- **`Compare(outro) int`**: comparação byte a byte em ordem
  lexicográfica, devolvendo `-1`, `0` ou `1`. Para UUIDv7 coincide com a
  ordem cronológica na resolução do nível; dentro do mesmo instante
  embutido a ordem é decidida pelos bits aleatórios
  ([02-instante-e-entropia.md](02-instante-e-entropia.md) §4). A
  assinatura **DEVE** servir diretamente à função de ordenação da
  biblioteca padrão da linguagem quando ela aceitar uma função de
  comparação (`slices.SortFunc(lista, UUID.Compare)` em Go), para que
  ordenar uma lista não exija tipo de lista próprio. A comparação
  **DEVE** distinguir cada uma das 16 posições sozinha, inclusive `0x7f`
  contra `0x80`, que denuncia comparação com sinal
  ([08-casos-de-teste.md](08-casos-de-teste.md), caso 19). Para simples
  igualdade, o tipo é um vetor de bytes e aceita o operador de igualdade.
- **`Bytes() []byte`**: **cópia** dos 16 bytes em ordem de rede. Existe
  separada do acesso direto ao vetor (`u[:]`, mais barato, mas apoiado no
  próprio valor) porque o destino pode guardar a referência, e porque em
  linguagens onde o tipo tem formatação própria (`fmt.Stringer` em Go)
  os verbos hexadecimais formatam o texto, não os bytes:
  `fmt.Printf("%x", u)` imprime 72 caracteres, e `fmt.Printf("%x",
  u.Bytes())` imprime os 32 dígitos esperados.
- **`URN() string`**: a string canônica prefixada por `urn:uuid:` (RFC
  8141). O analisador permissivo aceita esta forma de volta.

A ordem lexicográfica das strings canônicas coincide com a de `Compare`
sobre os bytes; ordenar pelo texto dá o mesmo resultado.
