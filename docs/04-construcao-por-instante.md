# 04. Construção a partir de um instante

As operações que recebem um instante por parâmetro e devolvem um
UUIDv7: as **fronteiras** de um instante (`MinAt`, `MaxAt`, `RangeAt`),
para consulta por intervalo pelo índice da própria chave primária, e a
**geração por instante** (`GenerateAt`, `GenerateAtString`), para
reprocessar histórico, semear dados de teste e importar registros
antigos preservando a ordenação da chave.

É a operação **inversa** da leitura de
[03-leitura-do-instante.md](03-leitura-do-instante.md): ali se lê o tempo
de um identificador, aqui se constroem identificadores para um tempo.

---

## 1. Regras comuns

- Toda construção a partir de um instante recebe o instante **por
  parâmetro**, é função separada da geração pelo relógio, e a geração
  **nunca** a chama. O caminho quente não ganha desvio nem indireção por
  causa dela ([10-decisoes.md](10-decisoes.md) §1 e §3).
- Versão 7 e variante `0b10` são gravadas sempre, qualquer que seja o
  conteúdo dos bits livres.
- Níveis desconhecidos são tratados como Nível 1, como na geração.
- O resultado só vale para identificadores gravados no **mesmo nível**
  (seção 3).
- Nenhuma delas devolve erro. Nenhum gerador desta biblioteca devolve
  erro, e a construção por instante não abre exceção.
- Todas são livres de alocação, exceto a forma em texto, que aloca a
  string devolvida ([08-casos-de-teste.md](08-casos-de-teste.md), caso 16).

## 2. Decomposição com saturação nas duas pontas

O instante é decomposto pelas fórmulas de
[02-instante-e-entropia.md](02-instante-e-entropia.md) §1, com uma
diferença obrigatória em relação ao caminho quente, que só tem piso: a
construção a partir de um instante **DEVE** saturar nas duas pontas da
faixa representável.

- Instante anterior a `1970-01-01T00:00:00Z`: resultado igual ao da
  própria época, com os campos abaixo do milissegundo zerados. É o mesmo
  comportamento da geração pelo relógio, e a coerência é obrigatória.
- Instante posterior a `10889-08-02T05:31:50.655999999Z`, o último que
  cabe em 48 bits de milissegundos: resultado igual ao desse instante,
  **com `micro` e `nano` em 999**. Zerá-los faria o resultado regredir ao
  cruzar a borda, quebrando a monotonicidade.

**A saturação é decidida sobre os segundos, antes da multiplicação por
mil, nas duas direções.** Como o instante vem por parâmetro e não do
relógio, ele pode estar longe o bastante para estourar o inteiro com
sinal (item 3 de [02-instante-e-entropia.md](02-instante-e-entropia.md)
§1); o estouro troca o sinal e a guarda de piso deixa de disparar.

**Truncar é proibido.** Descartar os bits acima de 48, como o
empacotamento por deslocamento faria de graça, deixaria a fronteira dar a
volta e a consulta por faixa devolveria as linhas erradas em silêncio. A
propriedade a preservar é que a fronteira nunca regride quando o instante
avança. Em compensação, dois instantes distintos fora da faixa devolvem o
mesmo valor: um intervalo inteiramente fora dela é vazio.

**Divergência consciente da RFC.** A RFC 9562 §6.1 manda, ao truncar um
carimbo, manter os bits **menos** significativos, o que faz um instante
posterior a 10889 dar a volta para perto de 1970. A construção a partir
de um instante diverge dessa regra de propósito; a geração pelo relógio
conserva o truncamento natural do empacotamento, que nunca é alcançado
antes daquele ano ([10-decisoes.md](10-decisoes.md) §3).

**A saturação entra exatamente no primeiro instante acima da faixa**, nem
um segundo nem um milissegundo antes: em `10889-08-02T05:31:50Z` o
carimbo ainda é `0xfffffffffd70`, e em `10889-08-02T05:31:50.655456789Z`
os campos abaixo do milissegundo ainda são os do instante. Saturar cedo
demais preserva a ordem e passa por qualquer teste que só confira
monotonicidade; os valores exatos dessa borda estão no caso 10 de
[08-casos-de-teste.md](08-casos-de-teste.md).

## 3. Fronteiras: `MinAt`, `MaxAt` e `RangeAt`

O motivo prático de adotar UUIDv7 como chave primária é responder a uma
janela de tempo com o índice da própria chave, sem coluna nem índice de
carimbo temporal. Para isso a implementação **DEVE** oferecer as duas
fronteiras de um instante, e elas dependem do nível.

Sejam `ms`, `micro` e `nano` os campos da decomposição da seção 2 para o
instante `t`. Define-se um valor de preenchimento `fill`: **todos os
bits em zero** para a fronteira inferior e **todos os bits em um** para
a superior. A fronteira é montada exatamente como a geração de
[01-escopo-e-layout.md](01-escopo-e-layout.md) §6, trocando a entropia
por `fill`:

| Nível    | `rand_a` (12 bits) | `rand_b[61:52]` | `rand_b[51:0]` |
|:---------|:-------------------|:----------------|:---------------|
| Nível 1  | `fill`             | `fill`          | `fill`         |
| Nível 2  | `micro`            | `fill`          | `fill`         |
| Nível 3  | `micro`            | `nano`          | `fill`         |

- **`MinAt(nível, t)`**: o menor UUIDv7 que a biblioteca poderia gerar
  naquele instante e naquele nível, com os bits livres em zero.
- **`MaxAt(nível, t)`**: o maior, com os bits livres em um. É o limite
  superior **fechado** do instante.
- **`RangeAt(nível, from, to)`**: o par de um intervalo **semiaberto**
  `[from, to)`, com `lo = MinAt(nível, from)` e `hi = MinAt(nível, to)`,
  correspondendo diretamente a `WHERE id >= lo AND id < hi`. Não reordena
  os argumentos: `to` anterior a `from` produz um intervalo vazio, e é
  isso que a comparação vai refletir. Para um intervalo fechado nas duas
  pontas, use `MinAt` e `MaxAt` diretamente.

**Versão e variante são preservadas nas duas fronteiras**, e é isso que
as torna limites corretos. O byte 6 recebe `0x70 | (rand_a >> 8)` e o
byte 8 recebe `0x80 | (rand_b >> 56)`, de modo que a fronteira superior
do Nível 1 termina em `0x7F` no byte 6 e `0xBF` no byte 8, não em `0xFF`.
Como **todo** UUIDv7 válido tem o nibble de versão em `7` e o byte 8 na
faixa `0x80..0xBF`, e a comparação é byte a byte a partir do mais
significativo, as duas fronteiras contêm todos os valores geráveis
naquele instante e naquele nível.

**Regra normativa: a precisão da fronteira é a do nível.** No Nível 1 a
faixa delimita o milissegundo inteiro; no Nível 2, o microssegundo; no
Nível 3, o nanossegundo. No Nível 1, `RangeAt(Nível 1, inicio, fim)`
exclui o milissegundo inteiro de `fim`; se a janela precisar terminar
dentro daquele milissegundo, o nível não tem resolução para isso.

**Regra normativa: fronteiras de níveis distintos não compõem.** Os bits
abaixo do milissegundo significam coisas diferentes em cada nível, então
uma fronteira calculada para um nível só delimita identificadores
gravados naquele mesmo nível. Uma fronteira superior de Nível 3 fica
abaixo de parte dos identificadores de Nível 1 do mesmo instante, porque
nela `rand_a` vale os microssegundos reais (0 a 999) enquanto no Nível 1
é aleatório (0 a 4095). A implementação **DEVE** documentar isso de forma
destacada: **é o erro mais provável do chamador, e ele não dá mensagem
nenhuma: a consulta devolve linhas a menos.** Escolha o nível ao criar a
tabela e não o mude. Se já houver dados misturados, a consulta por faixa
precisa usar a fronteira do nível mais permissivo, o Nível 1, nas duas
pontas, o que devolve linhas a mais e exige filtro adicional.

A prova da fronteira é uma rajada gerada entre dois instantes lidos do
relógio caindo inteira dentro das fronteiras desses instantes, e não a
inspeção do layout ([08-casos-de-teste.md](08-casos-de-teste.md), caso
10).

## 4. Geração por instante: `GenerateAt` e `GenerateAtString`

A biblioteca **DEVE** oferecer a geração de um UUIDv7 para um instante
informado pelo chamador, no lugar do instante atual. Sem ela, quem
reprocessa um histórico, semeia dados de teste ou importa registros
antigos preservando a ordenação da chave monta os 16 bytes à mão, e é
justamente o cálculo por nível que ele erra. Gerar pelo relógio na
importação daria a todos os registros antigos o carimbo do momento da
importação, e a ordenação da chave passaria a refletir a ordem de
importação, não a do histórico.

**Sem estado a preservar.** A geração pelo relógio não tem estado
compartilhado nem piso de relógio: a unicidade vem inteiramente dos bits
de entropia. Aceitar um instante arbitrário não fura invariante nenhuma,
e a operação não precisa de trava.

**Regra normativa: os bits livres são sorteados.** Preenchidos os campos
de tempo do nível, os bits restantes recebem entropia: 74 no Nível 1, 62
no Nível 2 e 52 no Nível 3. Duas chamadas com o mesmo instante **DEVEM**
devolver identificadores diferentes, com os campos de tempo iguais. É um
gerador, não um construtor determinístico: a forma determinística de um
instante já existe na seção 3, e expor uma segunda com o verbo "gerar"
convidaria ao pior mal-entendido possível, o de usar como identificador
único algo que colide na primeira repetição de instante.

**A entropia DEVE vir do mesmo gerador do resto da biblioteca**, para que
uma fonte criptográfica configurada pelo chamador continue valendo aqui;
por isso as duas formas existem também como métodos do gerador. O consumo
por nível é o de [02-instante-e-entropia.md](02-instante-e-entropia.md)
§3: uma palavra de 64 bits nos níveis 2 e 3, duas no Nível 1 e nos níveis
desconhecidos, com a mesma correspondência entre palavras e campos.

**Regra normativa: mesma decomposição das fronteiras.** O instante é
decomposto pela regra da seção 2, com saturação nas duas pontas, e não
pela decomposição do caminho quente, que só tem piso. Os motivos são os
mesmos: o instante vem por parâmetro e pode estourar a multiplicação por
mil, e um carimbo que dá a volta destrói a ordenação que é a razão de
existir do UUIDv7.

**Regra normativa: um só empacotamento.** O empacotamento dos 16 bytes
**DEVE** existir em um único lugar, compartilhado pelas fronteiras e pela
geração por instante, parametrizado pelos bits livres: constantes em um
caso, sorteados no outro. A geração pelo relógio pode manter cópia
própria, pela regra de custo do caminho quente, mas as duas construções a
partir de instante **não** podem divergir uma da outra. Duas cópias
dessa aritmética divergindo é um defeito que só aparece em produção, no
nível menos usado. A divergência entre a cópia do caminho quente e a
compartilhada é travada por teste: gerar pelo relógio, ler o instante
embutido de volta e regerar para ele exigindo bytes idênticos
([08-casos-de-teste.md](08-casos-de-teste.md), caso 11;
[10-decisoes.md](10-decisoes.md) §2).

**Consequência para a unicidade.** Como o instante deixa de vir do
relógio, nada impede o chamador de gerar em volume para um único
instante, e aí a margem passa a ser só a dos bits livres. No Nível 3 são
52 bits, o que põe a probabilidade de colisão na casa de um em dois
elevado a 26 gerações **para o mesmo nanossegundo**. É folgado na prática
e **DEVE** estar documentado, porque a geração pelo relógio nunca expõe o
chamador a essa escolha.

## 5. Contrato

| Operação | Devolve | Bits livres |
|:---|:---|:---|
| `MinAt(nível, instante) UUID` | o menor UUIDv7 do instante no nível | zero |
| `MaxAt(nível, instante) UUID` | o maior; limite superior fechado | um |
| `RangeAt(nível, from, to) (lo, hi UUID)` | o par do intervalo semiaberto `[from, to)` | zero nas duas pontas |
| `GenerateAt(nível, instante) UUID` | um UUIDv7 do instante | sorteados |
| `GenerateAtString(nível, instante) string` | o mesmo, na forma canônica | sorteados |

`GenerateAt` e `GenerateAtString` existem como funções de pacote (sobre o
gerador padrão) e como métodos do gerador. As fronteiras são funções de
pacote apenas: não sorteiam nada. Nenhuma das cinco é método de um UUID.

A forma é `GenerateAt(nível, instante)`, o mesmo par que `Generate(nível)`
e `MinAt(nível, instante)` já usam, e não uma família de quatro aridades
mapeando o nível no número de argumentos ([10-decisoes.md](10-decisoes.md)
§3). Os nomes `MinAt`/`MaxAt` foram escolhidos sobre `FloorAt`/`CeilAt`
e `LowerBound`/`UpperBound` por serem curtos e dizerem o extremo.
