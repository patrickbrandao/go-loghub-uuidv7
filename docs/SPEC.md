# Especificação de Desenvolvimento — Biblioteca UUIDv7 Multinível (RFC 9562)

> Documento de implementação **agnóstico de linguagem**. Descreve, sem
> depender de bibliotecas externas ou de recursos exclusivos do Go, tudo
> o que é necessário para construir a biblioteca do zero em qualquer
> linguagem (Go, Rust, C, C++, Java, C#, Python, etc.).
>
> Um desenvolvedor ou modelo de IA deve conseguir produzir uma
> implementação completa, interoperável, de alto desempenho e
> estritamente correta seguindo apenas este texto — **sem cometer
> nenhum dos erros históricos já encontrados e corrigidos neste projeto**.

---

## 1. Objetivo e Escopo da Biblioteca

Construir uma biblioteca leve, de altíssimo desempenho (dezenas de
nanossegundos por identificador, zero alocações de heap no caminho
quente) e segura para concorrência pesada, dedicada **exclusivamente ao
UUIDv7 da RFC 9562**, com extensão de precisão temporal sub-milissegundo
nos dois sentidos: gerar o identificador e ler de volta o instante
gravado nele.

1. **Geração do UUIDv7 Multinível**:
   - **Nível 1**: UUIDv7 padrão RFC 9562 com carimbo de milissegundos
     Unix e 74 bits de entropia. Existe também pelo nome por versão
     (`GenerateV7`) e pelo nome por nível (`GenerateV7Level1`), que são
     o mesmo (seção 3.1).
   - **Nível 2**: UUIDv7 com carimbo de milissegundos e
     **microssegundos** embutidos em `rand_a` (62 bits de entropia).
     Existe também pelo nome por nível (`GenerateV7Level2`).
   - **Nível 3**: UUIDv7 com carimbo de milissegundos,
     **microssegundos** em `rand_a` e **nanossegundos** no topo de
     `rand_b` (52 bits de entropia). Existe também pelo nome por nível
     (`GenerateV7Level3`).
2. **Leitura do Instante (o sentido inverso)**:
   - Duas leituras de tempo do UUIDv7 com políticas opostas, ambas
     normativas: a leitura por nível, que devolve um instante e descarta
     o que não pode ser tempo, e a extração completa, que devolve os
     campos crus sem julgar a origem dos bits (seção 7).
   - As duas **recusam** qualquer UUID que não seja de versão 7 com a
     variante da RFC, em vez de interpretar como instante bits que não
     foram gravados como tal (seção 4).
3. **Conversão e Análise (Parsing) Segura**:
   - Analisador estrito: formato canônico `8-4-4-4-12`.
   - Analisador permissivo: 4 formatos aceitos (canônico com hífens, sem
     hífens com 32 dígitos, entre chaves `{...}`, e prefixo URN
     `urn:uuid:...`).
   - Algoritmo de parsing imune a pânicos e leituras fora dos limites
     (out-of-bounds).
   - A análise é de **forma**: ela não confere a versão (seção 4).
4. **Construção a Partir de um Instante Explícito**:
   - É a operação **inversa** da leitura do item 2: ali se lê o tempo de
     um identificador, aqui se constroem identificadores para um tempo.
     Toda ela recebe o instante por parâmetro, e nenhuma parte dela é
     chamada pela geração pelo relógio.
   - **Fronteiras** (consulta por intervalo): o menor e o maior UUIDv7
     que a biblioteca poderia gerar naquele instante e naquele nível,
     para que uma janela de tempo seja respondida pelo índice da própria
     chave primária, sem coluna nem índice de carimbo temporal.
   - **Geração por instante**: um UUIDv7 daquele instante com os bits
     livres sorteados, para reprocessar histórico, semear dados de teste
     e importar registros antigos preservando a ordenação da chave.
5. **Serialização e Integração**:
   - Serialização de texto e JSON como string canônica entre aspas.
   - Suporte a colunas de banco de dados textuais e binárias de 16 bytes,
     com e sem `NULL`: o tipo simples, o anulável (`NullUUID`), o de
     escrita binária (`BinaryUUID`) e o anulável binário
     (`NullBinaryUUID`). A seção 8 especifica os quatro.
   - Ordenação lexicográfica e o valor especial `Nil`.

**Fora do escopo, por decisão (seção 11.3).** As demais versões da RFC
9562 (1, 2, 3, 4, 5, 6 e 8) não são geradas nem interpretadas, e com elas
saem o estado de relógio gregoriano, a sequência de relógio, o
identificador de nó, os espaços de nomes e a camada de compatibilidade de
nomes com outros pacotes de UUID. A biblioteca de origem deste projeto,
`go-loghub-uuid`, cobre essas versões.

---

## 2. Conceitos Gerais e Layout de Bits

Um UUID é composto exatamente por **128 bits** = **16 bytes**, dispostos
em ordem de rede (*big-endian*, byte 0 mais significativo).

### 2.1 Campos Fixos (RFC 9562)

Em todo UUIDv7, dois campos são invioláveis:

- **Versão (`ver`, 4 bits)**: Ocupa o nibble mais alto do **byte 6**
  (bits 48 a 51 do UUID). O valor é `7`. A RFC define versões de `1` a
  `8`; esta biblioteca só gera a `7` e só lê tempo da `7` (seção 4).
- **Variante (`var`, 2 bits)**: Ocupa os 2 bits mais significativos do
  **byte 8** (bits 64 e 65 do UUID). O padrão RFC 9562 exige variante
  `10` em binário (`0b10` nos bits superiores, correspondendo à máscara
  `0x80` ou nibble alto `8`, `9`, `A` ou `B`).

### 2.2 Representação Canônica em Texto

Consiste em **36 caracteres** hexadecimais minúsculos agrupados como
**`8-4-4-4-12`** com hifens nas posições 8, 13, 18 e 23 (índices
iniciados em zero). No UUIDv7 os cinco grupos se distribuem assim pelos
campos da seção 3.1:

| Grupo | Dígitos | Bits | Campos |
|:---|:---|:---|:---|
| 1 | 8 | 32 | `unix_ts_ms[47:16]` |
| 2 | 4 | 16 | `unix_ts_ms[15:0]` |
| 3 | 4 | 16 | `ver` (o primeiro dígito, sempre `7`) e `rand_a` (os três seguintes) |
| 4 | 4 | 16 | `var` e `rand_b[61:48]` (o primeiro dígito fica em `8`, `9`, `a` ou `b`) |
| 5 | 12 | 48 | `rand_b[47:0]` |

Exemplo, com o nibble de versão em `7` no início do terceiro grupo e o
nibble `8` da variante no início do quarto:

```text
0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff
```

---

## 3. Especificação Detalhada do UUIDv7 Multinível

O UUIDv7 dedica os 48 bits iniciais ao carimbo Unix em milissegundos,
garantindo ordenação temporal natural por comparação byte a byte.

### 3.1 Distribuição de Bits por Nível

| Campo         | Tamanho | Posição (bits) | Bytes   | Nível 1 (RFC)        | Nível 2 (+us)          | Nível 3 (+us +ns)      |
|:--------------|:--------|:---------------|:--------|:---------------------|:-----------------------|:-----------------------|
| `unix_ts_ms`  | 48 bits | 0–47           | 0..5    | ms Unix (UTC)        | ms Unix (UTC)          | ms Unix (UTC)          |
| `ver`         | 4 bits  | 48–51          | 6 (alto)| `0x7`                | `0x7`                  | `0x7`                  |
| `rand_a`      | 12 bits | 52–63          | 6(b)..7 | aleatório            | microssegundos (0..999)| microssegundos (0..999)|
| `var`         | 2 bits  | 64–65          | 8 (alto)| `0b10`               | `0b10`                 | `0b10`                 |
| `rand_b`      | 62 bits | 66–127         | 8(b)..15| aleatório (62 bits)  | aleatório (62 bits)    | 10b ns (0..999) + 52b  |

**Codificação decimal, e não a do Método 3 da RFC.** Os microssegundos e
os nanossegundos são contagens decimais de 0 a 999, gravadas em binário
nos seus campos, e **não** a fração do milissegundo multiplicada por 4096
que o Método 3 da RFC 9562 §6.2 descreve. O resultado continua sendo um
UUIDv7 válido e cronologicamente ordenável, mas um leitor de terceiros que
siga o Método 3 interpreta errado o sub-milissegundo (456 µs viram cerca
de 111 µs), e o Nível 3 usa 22 bits de tempo, além dos 12 que o método
prevê. A codificação decimal é o que permite à leitura por nível
reconhecer ruído pela faixa (seção 7). A escolha é deliberada e está
registrada na seção 11.3.

**Nome por versão e nome por nível.** O Nível 1 **DEVE** existir também
sob o nome da versão (`GenerateV7`), e cada um dos três
níveis **DEVE** existir sob um nome que diga o nível (`GenerateV7Level1`,
`GenerateV7Level2`, `GenerateV7Level3`), todos como método do gerador e
como função de pacote, sem parâmetro de nível e sem forma em texto.
`GenerateV7` e `GenerateV7Level1` são o mesmo. Cada nome é um apelido
exato da geração no nível correspondente: mesmos bytes, mesmo consumo de
entropia (seção 3.3) e mesma trava de alocação (seção 10, caso 16). O
nome por versão designa o UUIDv7 da RFC, sem a extensão de precisão
desta biblioteca; os nomes por nível existem para
que o nível, que é contrato de toda a coluna (seção 3.5: fronteiras de
níveis distintos não compõem), fique legível no ponto da chamada em vez
de numa constante. A constante continua sendo a única forma de dizer o
nível **como valor**, que é o que a geração por nível, a construção por
instante e a leitura por nível recebem; os nomes são grafias fixas das
três constantes na geração pelo relógio, e nada mais. Os apelidos **NÃO
DEVEM** acrescentar custo ao caminho quente: a implementação permanece
na geração por nível, e os nomes apenas a chamam. Um nome que gere em
nível diferente do que declara é defeito, travado pelo caso 1 da seção
10. Ver a seção 11.3.

### 3.2 Aritmética Temporal Segura (Evitando Armadilhas de Relógio)

A decomposição do instante em milissegundos, microssegundos e
nanossegundos deve obedecer a regras estritas de robustez:

1. **Prevenção de estouro em 2038 e 2262**:
   - **NUNCA** leia o relógio como um único número inteiro de
     nanossegundos de 64 bits (`UnixNano()`). Esse valor satura e
     transborda em **2262-04-11**.
   - Obtenha o tempo lendo **duas partes separadas**: segundos Unix
     inteiros de 64 bits (`sec`) e nanossegundos residuais dentro do
     segundo atual (`0 <= nsec < 1.000.000.000`).
2. **Proteção contra instantes anteriores a 1970 (Época Unix)**:
   - Se o relógio do sistema estiver configurado para uma data anterior a
     1970-01-01 (`sec < 0`), a aritmética de divisão e módulo padrão de
     várias linguagens produz restos negativos, corrompendo os bytes do
     UUID.
   - **Regra**: se `sec < 0`, fixe `unix_ts_ms = 0`, `micro = 0` e
     `nano = 0`. Um relógio quebrado ou pré-época não pode gerar campos
     fora da faixa `0..999`.
   - **A verificação de sinal vem antes de qualquer conversão para tipo
     sem sinal.** Em aritmética sem sinal o valor negativo não existe:
     ele vira um número enorme, a guarda nunca dispara e o carimbo sai
     grande e errado, sem erro nenhum. É a mesma armadilha do item 11 da
     seção 9, pelo mesmo mecanismo. Em linguagens onde a aritmética sem
     sinal é o caminho natural, como C e Rust, este é o ponto exato em
     que uma transcrição literal da fórmula do item 4 perde a proteção.
   - **Forma equivalente aceita.** Decidir o piso sobre `unix_ts_ms` já
     calculado, em aritmética **com** sinal, é equivalente e também
     conforme: como o item 1 garante `0 <= nsec < 1.000.000.000`,
     qualquer `sec` negativo produz `unix_ts_ms` negativo, e o piso
     dispara igual. As duas formas coexistem na implementação de
     referência, uma no caminho quente e outra no lado do parâmetro
     (seção 3.5), e a diferença entre elas é de custo, não de resultado.
3. **Proteção contra instantes muito além da faixa do UUIDv7**:
   - A decomposição multiplica os segundos por mil. Quando o instante vem
     do relógio do sistema isso é inofensivo, mas quando vem **por
     parâmetro** (seção 3.5) o tipo de data da linguagem costuma comportar
     anos muito além dos 48 bits de milissegundos do UUIDv7, e o produto
     **estoura o inteiro com sinal de 64 bits**.
   - O estouro dá a volta trocando o sinal, e o resultado é pior que um
     valor grande errado: um instante remoto no **futuro** vira negativo e
     cai no piso da época, e um remoto no **passado** vira positivo e
     produz um carimbo enorme. Nos dois casos a regra do item 2 deixa de
     disparar, porque o sinal já foi invertido.
   - **Regra**: decida a saturação sobre os **segundos**, antes da
     multiplicação, **nas duas pontas**: o piso do item 2 e o teto deste
     item são a mesma guarda, aplicada aos dois sinais. Ver a seção 3.5,
     onde essa saturação é obrigatória.
4. **Decomposição pura**, em aritmética **com sinal**, convertida para
   os campos sem sinal **somente depois** de o piso do item 2 e a
   saturação do item 3 terem sido aplicados:
   - `unix_ts_ms = (sec * 1000) + (nsec / 1_000_000)`, gravado nos 48
     bits do campo apenas após as guardas acima.
   - `sub_ms = nsec % 1_000_000`
   - `micro = sub_ms / 1000` (faixa 0..999, cabe em 10 bits; campo tem 12 bits)
   - `nano = sub_ms % 1000` (faixa 0..999, cabe em 10 bits; campo tem 10 bits)

### 3.3 Sorteio de Entropia Otimizado e Normativo

- **Nível 2 e Nível 3**: o campo `rand_a` carrega os microssegundos
  (não é aleatório). Portanto, a biblioteca **DEVE sortear exatamente uma
  única palavra de 64 bits aleatórios (`r2`)**. O sorteio de uma segunda
  palavra é desperdício de CPU e de entropia criptográfica do sistema.
- **Nível 1 (e níveis desconhecidos)**: sortear **duas palavras de 64 bits
  (`r1` e `r2`)**, usando os 12 bits inferiores de `r1` para preencher
  `rand_a`.
- Essa economia é normativa e deve ser travada por testes de contagem.
- **A correspondência entre palavras e campos também é normativa.** No
  Nível 1, `r1` é a **primeira** palavra sorteada e `r2` a segunda, e
  `rand_b` recebe os 62 bits inferiores de `r2`. Nos níveis 2 e 3, a
  única palavra sorteada é `r2`: `rand_b` recebe os seus 62 bits
  inferiores no Nível 2 e os 52 inferiores no Nível 3. O teste **DEVE**
  usar palavras distintas, porque uma fonte constante não distingue `r1`
  de `r2`. No gerador padrão, cuja fonte não é injetável, a independência
  das duas palavras **DEVE** ser conferida estatisticamente: uma palavra
  repetida faria `rand_a` coincidir sempre com os 12 bits inferiores de
  `rand_b` e derrubaria o Nível 1 de 74 para 62 bits de entropia, sem
  nenhum sinal visível (seção 10, caso 4).

### 3.4 Ordenação e Desempate (Sem Contador Monotônico)

A garantia de ordenação do UUIDv7 multinível é **exatamente esta, e não
mais que esta**:

- Se o instante embutido de `B` for estritamente maior que o de `A`,
  então `B > A` tanto na comparação byte a byte quanto na comparação
  lexicográfica das strings canônicas.
- Se `A` e `B` carregarem o **mesmo instante embutido**, a ordem entre
  eles é **aleatória**, decidida pelos bits de entropia. Não há contador,
  nem sequência, nem qualquer outro desempate determinístico.

O instante embutido tem a resolução do nível: milissegundo no Nível 1,
microssegundo no Nível 2, nanossegundo no Nível 3. Como gerar um UUID
custa dezenas de nanossegundos — menos que o passo do relógio da maioria
dos hosts —, **empates entre gerações consecutivas são o caso comum**,
não a exceção: são universais no Nível 1 e frequentes nos demais.

**Regra normativa**: a implementação **NÃO DEVE** introduzir contador
monotônico, nem os métodos 1 ou 2 da RFC 9562 §6.2, no gerador padrão.
Ambos exigem estado compartilhado entre threads, e o custo sob
concorrência inviabiliza o objetivo de desempenho da seção 1. O
raciocínio completo e a medição estão na seção 11.

**Consequência para testes**: é proibido escrever teste de ordenação que
gere UUIDs em laço apertado e conte "regressões" contra um limite
tolerado. Esse teste mede a resolução do relógio do host, não a
biblioteca, e falha de forma permanente em hosts com relógio de
microssegundo. Teste a invariante acima fazendo o instante avançar de
verdade entre as gerações.

---

### 3.5 Fronteiras de Tempo para Consulta por Intervalo

O motivo prático de adotar UUIDv7 como chave primária é responder a uma
janela de tempo com o índice da própria chave, sem coluna nem índice de
carimbo temporal. Para isso a implementação **DEVE** oferecer as duas
fronteiras de um instante, e elas dependem do nível.

Sejam `ms`, `micro` e `nano` os campos produzidos pela decomposição da
seção 3.2 aplicada ao instante `t`. Define-se um valor de preenchimento
`fill`: **todos os bits em zero** para a fronteira inferior e **todos os
bits em um** para a superior. A fronteira é então montada exatamente
como a geração da seção 3.1, trocando a entropia por `fill`:

| Nível    | `rand_a` (12 bits) | `rand_b[61:52]` | `rand_b[51:0]` |
|----------|--------------------|-----------------|----------------|
| `Level1` | `fill`             | `fill`          | `fill`         |
| `Level2` | `micro`            | `fill`          | `fill`         |
| `Level3` | `micro`            | `nano`          | `fill`         |

Níveis desconhecidos são tratados como `Level1`, como na geração.

**Versão e variante são preservadas nas duas fronteiras**, e é isso que
as torna limites corretos. O byte 6 recebe `0x70 | (rand_a >> 8)` e o
byte 8 recebe `0x80 | (rand_b >> 56)`, de modo que a fronteira superior
do Nível 1 termina em `0x7F` no byte 6 e `0xBF` no byte 8, não em `0xFF`.
Como **todo** UUIDv7 válido tem o nibble de versão em `7` e o byte 8 na
faixa `0x80..0xBF`, e como a comparação é byte a byte a partir do mais
significativo, as duas fronteiras de fato contêm todos os valores
geráveis naquele instante e naquele nível.

**Regra normativa — a precisão da fronteira é a do nível.** No Nível 1 a
faixa delimita o milissegundo inteiro; no Nível 2, o microssegundo; no
Nível 3, o nanossegundo.

**Regra normativa — fronteiras de níveis distintos não compõem.** Os
bits abaixo do milissegundo significam coisas diferentes em cada nível,
então uma fronteira calculada para um nível só delimita identificadores
gravados naquele mesmo nível. Uma fronteira superior de Nível 3 fica
abaixo de parte dos identificadores de Nível 1 do mesmo instante, porque
nela `rand_a` vale os microssegundos reais (0 a 999) enquanto no Nível 1
é aleatório (0 a 4095). A implementação **DEVE** documentar isso de
forma destacada: é o erro mais provável do chamador, e ele se manifesta
como linhas faltando, sem erro nenhum.

**Regra normativa — saturação nas duas pontas.** Diferentemente da
geração, que só tem piso na época, as fronteiras **DEVEM** saturar
também no teto:

- Instante anterior a `1970-01-01T00:00:00Z`: resultado igual ao da
  própria época, com os campos abaixo do milissegundo zerados. É o mesmo
  comportamento da seção 3.2, e a coerência com a geração é obrigatória.
- Instante posterior a `10889-08-02T05:31:50.655999999Z`, o último que
  cabe em 48 bits de milissegundos: resultado igual ao desse instante,
  **com `micro` e `nano` em 999**. Zerá-los faria a fronteira regredir ao
  cruzar a borda, quebrando a monotonicidade.

Truncar os bits excedentes, como o empacotamento por deslocamento faria
naturalmente, é **proibido**: a fronteira daria a volta e a consulta
passaria a devolver as linhas erradas em silêncio. A propriedade a
preservar é que a fronteira nunca regride quando o instante avança.

**Divergência consciente da RFC.** A RFC 9562 §6.1 manda, ao truncar um
carimbo, manter os bits **menos** significativos, o que faz um instante
posterior a 10889 dar a volta para perto de 1970. A construção a partir
de um instante diverge dessa regra de propósito, pelo motivo acima; a
geração pelo relógio conserva o truncamento natural do empacotamento,
que nunca é alcançado antes daquele ano. Ver a seção 11.3.

A saturação entra **exatamente** no primeiro instante acima da faixa, nem
um segundo nem um milissegundo antes: em `10889-08-02T05:31:50Z` o
carimbo ainda é `0xfffffffffd70`, e em
`10889-08-02T05:31:50.655456789Z` os campos abaixo do milissegundo ainda
são os do instante. Saturar cedo demais preserva a ordem e passa por
qualquer teste que só confira monotonicidade, por isso os valores exatos
dessa borda estão no caso 10 da seção 10.

**Cuidado de implementação.** É aqui que a regra 3 da seção 3.2 passa a
valer: como o instante vem por parâmetro e não do relógio do sistema, ele
pode estar longe o bastante para estourar a multiplicação por mil. A
saturação **DEVE** ser decidida sobre os segundos, antes dela, nas duas
direções.

**Intervalo semiaberto.** A conveniência que devolve o par de um
intervalo `[from, to)` usa a fronteira **inferior** nas duas pontas:
`lo = MinAt(nível, from)` e `hi = MinAt(nível, to)`. Isso corresponde
diretamente a `WHERE id >= lo AND id < hi`. Ela não reordena os
argumentos: `to` anterior a `from` produz um intervalo vazio.

**Esta funcionalidade não toca o caminho quente.** As fronteiras recebem
o instante por parâmetro, são funções de pacote separadas e a geração
nunca as chama. Ver a seção 11.3.

---

### 3.6 Geração a Partir de um Instante Explícito

A biblioteca **DEVE** oferecer a geração de um UUIDv7 para um instante
informado pelo chamador, no lugar do instante atual. É o que fecha a
assimetria com a extração da seção 7: sem ela, quem reprocessa um
histórico, semeia dados de teste ou importa registros antigos
preservando a ordenação da chave monta os 16 bytes à mão.

**Sem estado a preservar.** A geração pelo relógio do UUIDv7 não tem
estado compartilhado nem piso de relógio: a unicidade vem inteiramente
dos bits de entropia. Por isso aceitar um instante arbitrário do chamador
não fura invariante nenhuma, e a operação não precisa de trava.

**Regra normativa — os bits livres são sorteados.** Preenchidos os campos
de tempo do nível, os bits restantes recebem entropia: 74 no Nível 1, 62
no Nível 2 e 52 no Nível 3. Duas chamadas com o mesmo instante **DEVEM**
devolver identificadores diferentes. É um gerador, não um construtor
determinístico: a forma determinística de um instante já existe na seção
3.5, e expor uma segunda com o verbo "gerar" convidaria ao pior
mal-entendido possível, o de usar como identificador único algo que
colide na primeira repetição de instante.

A entropia **DEVE** vir do mesmo gerador do resto da biblioteca, para
que uma fonte criptográfica configurada pelo chamador continue valendo
aqui. O consumo por nível é o mesmo da seção 3.3: uma palavra de 64 bits
nos níveis 2 e 3, duas no Nível 1 e nos níveis desconhecidos.

**Regra normativa — mesma decomposição das fronteiras.** O instante é
decomposto pela regra da seção 3.5, com saturação nas duas pontas, e não
pela decomposição do caminho quente, que só tem piso. Os dois motivos
são os mesmos: o instante vem por parâmetro e pode estourar a
multiplicação por mil, e um carimbo que dá a volta destrói a ordenação
que é a razão de existir do UUIDv7.

**Regra normativa — um só empacotamento.** O empacotamento dos 16 bytes
**DEVE** existir em um único lugar, compartilhado pelas fronteiras da
seção 3.5 e pela geração desta seção, parametrizado pelos bits livres:
constantes em um caso, sorteados no outro. A geração pelo relógio pode
manter cópia própria, pela regra de custo da seção 11.1, mas as duas
construções a partir de instante **não** podem divergir uma da outra.
Duas cópias dessa aritmética divergindo é um defeito que só aparece em
produção, no nível menos usado.

**Consequência para a unicidade.** Como o instante deixa de vir do
relógio, nada impede o chamador de gerar em volume para um único
instante, e aí a margem passa a ser só a dos bits livres. No Nível 3 são
52 bits, o que põe a probabilidade de colisão na casa de um em dois
elevado a 26 gerações **para o mesmo nanossegundo**. É folgado na
prática e **DEVE** estar documentado, porque a geração pelo relógio
nunca expõe o chamador a essa escolha.

A forma da API, a aridade e a nomenclatura estão na seção 11.3.

---

## 4. Versão Única: Recusa das Demais Versões na Leitura de Tempo

A biblioteca gera somente UUIDv7, mas **recebe** identificadores de fora:
texto de uma requisição, coluna de banco, JSON de outro sistema. Nada
impede que um deles seja de outra versão, e um UUIDv4 lido como UUIDv7
devolve um instante sem sentido, sem erro nenhum: os 48 bits altos de um
UUIDv4 são aleatórios e caem em qualquer data entre 1970 e 10889. Esta
seção fixa onde essa recusa acontece.

**Regra normativa — o predicado de validade é o de UUIDv7.** A
biblioteca **DEVE** oferecer um predicado de validade (`IsValid`)
verdadeiro **se e somente se** o nibble de versão for `7` **e** os dois
bits de variante forem `0b10`. As duas condições são obrigatórias: um
identificador com nibble `7` e variante `0b11` (a reservada à Microsoft)
tem o mesmo layout aparente e não é UUIDv7. O UUID nulo é **recusado**,
porque não carrega versão nem variante; a verificação dele é feita pelo
predicado próprio (`IsZero`). O UUID com todos os bits em um também é
recusado, pela mesma razão.

**Regra normativa — as leituras de tempo recusam o que o predicado
recusa.** As quatro leituras da seção 7 **DEVEM** conferir o predicado
antes de interpretar qualquer bit:

| Leitura | Recusa |
|:---|:---|
| Extração completa a partir dos 16 bytes (`ImportBinary`) | erro de versão (`ErrNotV7`), com a estrutura zerada |
| Extração completa a partir do texto (`Import`) | erro de formato puro se o texto for recusado pelo analisador estrito; erro de versão, com a estrutura zerada, se for aceito e o predicado recusar |
| Instante em milissegundo (`Timestamp`) | booleano falso, com o instante zero |
| Instante por nível (`TimestampWithLevel`) | booleano falso, com o instante zero, em qualquer nível |

A recusa é por **forma**, não por origem: não há como saber se um UUIDv7
foi gerado por esta biblioteca, nem em qual nível. Um UUIDv7 padrão de
outro gerador é aceito e lido como qualquer outro, com as políticas de
nível da seção 7.

**Regra normativa — a análise continua sendo só de forma.** Os
analisadores das seções 6.2 e 6.3, a desserialização de texto, de binário
e de JSON e a leitura de banco da seção 8 **NÃO DEVEM** recusar um UUID
bem formado de outra versão. Guardar, transportar e comparar um
identificador não depende da versão, e recusar na análise impediria até
de registrar em log o valor que chegou errado. Quem precisa exigir a
versão logo na entrada chama o predicado depois da análise.

**Regra normativa — o erro de versão é família própria.** O erro de
versão **NÃO DEVE** embrulhar o sentinela de formato da seção 6.4: o
texto foi aceito, e a recusa é sobre o conteúdo. Quem trata só erro de
formato continua tratando só texto malformado.

**Custo.** A verificação é uma leitura de dois campos e duas
comparações, e a recusa devolve um erro pré-alocado: a extração completa
continua sem alocação (seção 10, caso 16). Na implementação de
referência a extração binária passou de cerca de 1,7 ns para 2,3 ns por
chamada, fora do caminho quente de geração.

---

## 5. Arquitetura de Entropia e Política de Falha Rápida (Fail-Fast)

### 5.1 O Gerador Padrão (Alta Concorrência e Zero Contenção)

- Para geração rápida (milhões de UUIDs/s), não utilize um lock global em
  torno de uma fonte compartilhada.
- Use uma fonte **local por thread**. Se a linguagem já oferecer uma no
  runtime, prefira-a: em Go, as funções de pacote de `math/rand/v2` leem
  de uma instância de ChaCha8 por thread, semeada pelo sistema
  operacional, sem trava e sem estado a manter pela biblioteca.
- Se a plataforma não oferecer uma fonte por thread, mantenha um **pool
  de geradores locais**, cada um instanciado sob demanda e semeado **uma
  única vez** com 128 bits obtidos da fonte criptográfica forte do
  sistema operacional.

### 5.2 Política Anti-Degradação Silenciosa (Evitando Falha Grave de Segurança)

A política tem **três casos distintos**, e confundi-los leva a
implementações que param o processo onde não deveriam, ou que seguem
onde não podem. O vocabulário de "falhar alto" vale para o primeiro; os
outros dois são erros de configuração do chamador, e recebem tratamento
próprio.

**Caso 1 — falha da fonte durante a operação.** Se a fonte forte de
entropia do sistema operacional falhar ao semear um gerador, ou se falhar
a leitura de um gerador construído sobre um leitor do chamador:
  - **A biblioteca DEVE falhar alto e imediatamente (pânico / exceção /
    encerramento)**.
  - **JAMAIS recorra ao relógio do sistema como fallback silencioso**.
    Semear múltiplos geradores com o horário atual produz sequências
    idênticas ou correlacionadas entre threads, levando a colisões
    maciças de UUIDs e previsibilidade total de chaves e identificadores.
  - O valor do pânico **DEVE** ser o erro de fonte de entropia (seção
    6.4), para que quem recupere o pânico reconheça a causa. A geração
    propriamente dita nunca devolve erro.

**Caso 2 — configuração inválida explícita.** Uma função de construção
que receba fonte nula, ou leitor nulo, **DEVE** entrar em pânico na
própria construção. O erro é de configuração e precisa aparecer na
carga do programa, não na primeira geração, onde apareceria em produção
como uma falha de ponteiro nulo longe da causa.

**Caso 3 — ausência de fonte por construção omitida.** Um gerador que
chegue a uma geração sem fonte, por ter sido montado fora das funções de
construção (valor zero embutido em outra estrutura, ou referência nula
usada como receptor), **DEVE** recorrer à fonte padrão do pacote em vez
de entrar em pânico. Aqui não há degradação: a fonte padrão é exatamente
a que a construção correta teria instalado, então a imprevisibilidade
dos bits não muda. O caso 1 não se aplica, porque nenhuma fonte falhou.
A tolerância é regra do tipo gerador inteiro, sem exceção por forma de
gerar: vale para a geração pelo relógio nos três níveis, para os nomes
por versão e por nível (seção 3.1) e para a geração por instante (seção
3.6).

A diferença entre os casos 2 e 3 é o momento e a intenção: quem pede uma
fonte inválida recebe o erro de imediato; quem simplesmente não construiu
o gerador recebe um resultado correto. Ver a seção 11.3.

### 5.3 Fontes Criptográficas Dedicadas

- O gerador padrão não promete força criptográfica, mesmo quando a fonte
  do runtime é resistente a predição.
- Para casos que exigem imprevisibilidade (tokens de sessão, links
  secretos), a biblioteca deve fornecer um gerador explícito que utiliza
  exclusivamente entropia criptográfica (`NewCryptoGenerator`), e um
  construtor sobre um leitor do chamador (`NewGeneratorWithReader`).
- **Regra normativa — formação das palavras a partir do leitor.** Cada
  palavra de 64 bits é formada por **8 bytes em ordem de rede**
  (*big-endian*, o primeiro byte lido é o mais significativo), e as
  palavras são lidas na ordem da seção 3.3: no Nível 1, os 8 primeiros
  bytes formam `r1` e os 8 seguintes, `r2`. Uma leitura **curta** sem
  erro, que o contrato de leitor da linguagem permite, **DEVE** ser
  completada com novas leituras até os 8 bytes; só o fim dos dados ou um
  erro contam como falha, e aí vale o caso 1 da seção 5.2. Aceitar a
  leitura pela metade deixaria bytes zerados na palavra, em silêncio. A
  ordem é contrato porque é visível ao chamador: um leitor determinístico
  reproduz os mesmos identificadores (seção 11.1).
- O gerador criptográfico **DEVE** ler da fonte criptográfica do sistema
  capturada na construção, e o teste **DEVE** provar a origem dos bits, e
  não só a validade do resultado: um gerador "criptográfico" que caísse
  na fonte padrão produziria UUIDv7 igualmente válidos (seção 10, caso 4).
- Mesmo com entropia criptográfica, todo UUIDv7 expõe o instante de
  criação por construção. A documentação **DEVE** dizer isso junto do
  gerador criptográfico.

---

## 6. Algoritmos de Conversão e Parsing Seguro

### 6.1 Formatação (Binário → String Canônica)

- Utilizar uma tabela de caracteres hexadecimais minúsculos
  `"0123456789abcdef"`.
- Gravar diretamente em um buffer fixo de 36 caracteres, inserindo os
  hifens nos índices 8, 13, 18 e 23 sem alocações intermediárias.

### 6.2 Análise Estrita (String Canônica → Binário)

- Validar comprimento exato de 36 caracteres.
- Validar se `s[8] == '-'`, `s[13] == '-'`, `s[18] == '-'` e
  `s[23] == '-'`.
- **REGRA CRÍTICA DE SEGURANÇA (Eliminação de Defeito Crítico de Pânico)**:
  - **NUNCA** decodifique a string percorrendo caractere por caractere e
    avançando um índice quando encontrar um hífen. Se a entrada contiver
    hifens extras em posições inesperadas, o laço desalinha e tenta ler
    `s[36]`, resultando em pânico de estouro de array (*index out of
    range*) e derrubando o processo da aplicação.
  - **DECODIFIQUE SEMPRE via tabela de deslocamentos fixos conhecidos**:
    `hexOffsets = [16]int{0, 2, 4, 6, 9, 11, 14, 16, 19, 21, 24, 26, 28, 30, 32, 34}`
  - Para cada byte `i` de 0 a 15, decodifique os dois caracteres
    hexadecimais localizados em `s[hexOffsets[i]]` e
    `s[hexOffsets[i]+1]`.
  - Se qualquer caractere não for hexadecimal válido (0..9, a..f, A..F),
    retorne erro de formato e devolva o UUID zerado.

### 6.3 Analisador Permissivo (4 Formatos)

A função `Parse` deve aceitar quatro formatos distintos (maiúsculas ou
minúsculas):

1. **Canônico com hífens** (36 caracteres): decodificado conforme 6.2.
2. **Sem hífens** (32 caracteres): decodificado diretamente a cada 2
   caracteres (`i * 2`).
3. **Entre chaves** (38 caracteres): deve iniciar com `{` e terminar com
   `}`, contendo os 36 caracteres canônicos internamente.
4. **Prefixo URN** (45 caracteres): deve iniciar com `urn:uuid:`
   (insensível a maiúsculas), seguido dos 36 caracteres canônicos.
- Qualquer outro comprimento deve ser rejeitado imediatamente como
  comprimento inválido.

### 6.4 Taxonomia de Erros

A biblioteca **DEVE** expor um erro sentinela de formato e erros mais
específicos que o **embrulham**, de modo que a verificação pelo
sentinela (`errors.Is` em Go, ou o equivalente da linguagem) continue
verdadeira para qualquer um deles. Quem trata só a presença de erro não
precisa conhecer a tabela; quem decide pelo tipo, precisa, e é para esse
chamador que ela existe. A tabela é normativa: duas implementações
conformes devolvem o mesmo erro para a mesma entrada.

| Operação | Entrada | Erro devolvido |
|:---|:---|:---|
| Analisador estrito (6.2) e a extração de tempo em texto (seção 7) | qualquer recusa de texto, inclusive comprimento | sentinela de formato, **puro** |
| Analisador permissivo (6.3) | comprimento fora de 32, 36, 38 e 45 | comprimento inválido |
| Analisador permissivo (6.3) | 38 caracteres sem `{` na primeira posição ou sem `}` na última | chaves inválidas |
| Analisador permissivo (6.3) | 45 caracteres sem o prefixo `urn:uuid:` | sentinela de formato |
| Analisador permissivo (6.3) | dígito não hexadecimal, ou hífen fora das posições 8, 13, 18 e 23 | sentinela de formato |
| Construção a partir de bytes crus | tamanho diferente de 16 | comprimento inválido |
| Desserialização binária | tamanho diferente de 16 | comprimento inválido |
| Desserialização de texto | as mesmas entradas do analisador permissivo | os mesmos erros do analisador permissivo |
| Desserialização JSON do tipo anulável | valor que não é string nem `null`, ou sintaxe JSON inválida | sentinela de formato |
| Desserialização JSON do tipo anulável | string cujo conteúdo o analisador permissivo recusa | os erros do analisador permissivo |
| Leitura de valor de banco | texto, ou bytes em tamanho diferente de 16, que o analisador permissivo recusa | os erros do analisador permissivo |
| Leitura de valor de banco | tipo que não é texto, bytes nem nulo | tipo não suportado |
| Extração de tempo, binária ou em texto (seção 7) | UUID aceito cuja versão não é 7 ou cuja variante não é `0b10`, inclusive o nulo (seção 4) | erro de versão, **puro** |
| Fonte de entropia do chamador (leitor) | leitura que falhou durante a geração | pânico com o erro de fonte de entropia como valor (seção 5.2, caso 1) |
| Construção do gerador sobre fonte ou leitor | fonte nula ou leitor nulo | pânico na construção (seção 5.2, caso 2) |

Todo erro de comprimento e de chaves **DEVE** embrulhar o sentinela de
formato. O erro de tipo não suportado, o de versão e o de fonte de
entropia são famílias à parte e **NÃO** embrulham o sentinela: não são
recusas de texto. A desserialização JSON do tipo simples delega ao codificador da
linguagem, que devolve o erro dele para valores que não são string e o
erro do analisador permissivo para strings recusadas.

**Regra normativa — o analisador estrito não embrulha.** Ele devolve o
sentinela puro em todos os casos, inclusive comprimento errado, para que
a comparação por igualdade direta valha (seção 11.3). O erro de
comprimento embrulhado é reconhecido pela verificação da linguagem
através de qualquer camada de embrulho, sem predicado próprio.

**Assimetria registrada.** No analisador permissivo, as chaves
malformadas têm erro próprio e o prefixo URN inválido **não** tem: ele
devolve o sentinela puro, na mesma classe do dígito inválido. As duas
situações são análogas e a assimetria não tem motivo técnico; veio com a
primeira versão do analisador permissivo, na biblioteca de origem, e é
mantida para que as duas bibliotecas classifiquem a mesma entrada do
mesmo modo: criar o erro de prefixo mudaria o valor devolvido para uma
entrada que lá recebe o sentinela puro. Uma
reimplementação **DEVE** reproduzi-la, para que os dois analisadores
classifiquem a mesma entrada do mesmo modo. A decisão e o que
justificaria revê-la estão na seção 11.3.

Em todos os casos de recusa o valor devolvido **DEVE** ser o UUID zerado,
e nenhum byte parcialmente decodificado pode vazar; nas desserializações
com receptor, o receptor **NÃO DEVE** ser alterado. A regra vale para os
dois tipos: no anulável ela alcança **as duas** componentes, o
identificador e o booleano de presença, que **NÃO DEVEM** mudar quando a
entrada é recusada.

**Exceção única — a leitura de valor de banco do tipo anulável.** Ali o
identificador continua intacto, mas o booleano **DEVE** cair para falso.
Não é inconsistência: a interface de leitura de banco recebe um destino
**reaproveitado a cada linha**, e um chamador que ignore o erro leria o
valor da linha anterior como se fosse o da linha que falhou. Derrubar o
booleano transforma esse descuido em ausência de valor, e não em dado
errado. As desserializações não têm destino reaproveitado, e por isso não
têm a exceção. A decisão está na seção 11.3.

Os dois auxiliares que entram em pânico em vez de devolver erro, um para
constantes do próprio código (`MustParse`) e um para encadear com
funções que devolvem par de valores (`Must`), não fazem parte da tabela.
O segundo propaga o próprio erro recebido, para que quem recupere o
pânico o reconheça pelo sentinela.

---

## 7. Inspeção e Contrato de API

A biblioteca deve disponibilizar operações de consulta. As quatro
leituras de tempo abaixo começam pela recusa da seção 4: nenhuma delas
interpreta bits de um UUID que o predicado de validade recusa.

- **`Version() byte`**: retorna o nibble de versão, lido como está. Vale
  `7` em todo UUID gerado pela biblioteca; um UUID de fora pode trazer
  qualquer valor de 0 a 15.
- **`Variant() byte`**: retorna os 2 bits altos do byte 8, lidos como
  estão. Vale `0b10` (2), a variante da RFC 9562, em todo UUID gerado.
- **`IsValid() bool`**: o predicado de validade da seção 4, verdadeiro
  somente para versão 7 com variante `0b10`.
- **`Timestamp() (time.Time, bool)`**: o instante do carimbo de 48 bits,
  em UTC e com resolução de milissegundo. É o mesmo que a leitura por
  nível no Nível 1. Devolve falso, com o instante zero, se o predicado
  recusar.
- **`TimestampWithLevel(Level) (time.Time, bool)`**: devolve o instante
  de um UUIDv7 somando ao milissegundo a precisão sub-milissegundo
  gravada pelo nível informado. O nível **não** é dedutível do
  identificador, e por isso vem por parâmetro. Devolve falso, com o
  instante zero, se o predicado recusar; no Nível 1 e em níveis
  desconhecidos devolve apenas o milissegundo.

  **Regra normativa — descarte por faixa.** A condição de aproveitamento
  é sobre o **valor lido**, não sobre o nível pedido. Se um campo
  sub-milissegundo estiver fora da faixa de 0 a 999, ele denuncia
  entropia em vez de tempo, e a implementação **DEVE** descartar a
  precisão sub-milissegundo, devolvendo apenas o milissegundo do
  carimbo de 48 bits. Informar o nível errado degrada para o
  milissegundo, que é um resultado utilizável; não devolve erro nem
  lixo.

  **Regra normativa — no Nível 3 os dois campos caem juntos.** Um
  `rand_a` fora da faixa prova que o topo de `rand_b` também é ruído,
  porque os dois vêm da mesma geração. A implementação **NÃO DEVE**
  aproveitar `nano` quando `micro` foi descartado, mesmo que `nano`
  caiba em 0 a 999 por acaso, o que ocorre em cerca de 98% dos casos
  (1000 valores válidos em 1024 possíveis). O caso simétrico vale
  igualmente: `nano` fora da faixa descarta também `micro`. No Nível 2
  o `nano` não é lido, e só o `micro` decide.

  Esta política é o **oposto** da extração completa descrita a seguir,
  que entrega os bits sem julgar a faixa. A divergência é deliberada:
  uma operação entrega um instante, a outra entrega os bits. Ver a
  seção 11.3.
- **`ImportBinary(UUID) (Time, erro)`** e **`Import(texto) (Time, erro)`**:
  extração completa dos campos de tempo de um UUIDv7, devolvendo uma
  estrutura com quatro campos. A forma binária devolve o erro de versão
  com a estrutura zerada se o predicado recusar. A forma em texto analisa
  a string pelo analisador **estrito** da seção 6.2, devolve o sentinela
  puro com a estrutura zerada se o texto for recusado, e delega à forma
  binária, que pode então devolver o erro de versão.
  - `Seconds`: segundos Unix, por divisão inteira do carimbo de 48 bits
    por mil.
  - `Milliseconds`: o resto dessa divisão, sempre na faixa 0 a 999. Os
    dois campos saem do mesmo carimbo, e o recorte entre segundo e
    fração é decisão de contrato, não consequência do layout.
  - `Microseconds`: os 12 bits de `rand_a`, lidos como estão.
  - `Nanoseconds`: os 10 bits altos de `rand_b`, lidos como estão.

  **Regra normativa — a extração é cega quanto ao nível.** Aceito o
  UUIDv7, ela **DEVE** sempre interpretar `rand_a` como microssegundos e
  o topo de `rand_b` como nanossegundos, e **NÃO DEVE** validar faixa nem
  descartar campo algum. A cegueira é quanto ao **nível**, não quanto à
  versão: a versão já foi conferida pela seção 4. A operação não recebe o nível, e ele não é dedutível do
  identificador: em um UUIDv7 de Nível 1 esses campos carregam entropia,
  e a extração devolve os bits lidos sem julgar a origem.
  Consequentemente `Microseconds` vai de 0 a 4095 e `Nanoseconds` de 0 a
  1023 quando a origem é aleatória, e é o chamador, que sabe o nível,
  quem decide o que aproveitar. É a operação inversa da geração por
  instante da seção 3.6, e a estrutura devolvida **NÃO** passa pelo
  descarte da leitura por nível acima.
- **`IsZero() bool`**: verdadeiro somente para o UUID nulo.
- **`Compare(outro) int`**: comparação byte a byte, devolvendo `-1`, `0`
  ou `1`, com a assinatura que a ordenação da biblioteca padrão aceita
  diretamente (seção 8).
- **`Bytes() []byte`**: cópia dos 16 bytes em ordem de rede. Existe
  separado do acesso direto ao vetor porque este último devolve uma
  referência ao próprio valor, e porque em linguagens onde o tipo tem
  formatação própria (como `fmt.Stringer` em Go) os verbos hexadecimais
  formatam o texto, não os bytes.
- **`AppendTo(dst) dst`**: escreve os 36 bytes da forma canônica no fim
  do buffer do chamador e devolve o buffer estendido, **sem alocar**
  quando houver capacidade. É o caminho previsto para serializar grandes
  volumes; a conversão que devolve string aloca a cada chamada.
- **`AppendBinary(dst) dst`**: o equivalente binário, escrevendo os 16
  bytes em ordem de rede no fim do buffer do chamador, também sem alocar
  quando houver capacidade. O conteúdo é idêntico ao da serialização
  binária, portanto acrescentá-lo **não** muda formato de dados gravado.

  **Regra normativa — os dois anexadores andam juntos.** Uma
  implementação que ofereça um **DEVE** oferecer o outro. A assimetria
  não tem justificativa técnica: se o motivo de existir o anexador de
  texto é serializar em volume sem alocar, o mesmo motivo vale para os
  bytes, e quem grava em coluna binária é justamente quem grava em
  volume. Em linguagens que definam interfaces de anexação em texto e em
  binário, como o Go a partir da 1.24, satisfazer só a primeira deixa o
  tipo pela metade em todo consumidor genérico que prefira anexar a
  alocar.

  O lado binário não precisa de uma segunda forma sem erro, ao contrário
  do de texto: a formatação canônica é um cálculo, os 16 bytes não são.
A biblioteca deve disponibilizar também as operações de **derivação**
da seção 3.5, que são o sentido inverso das acima: recebem um instante
e um nível e devolvem o identificador que os delimita. Diferentemente
de todas as operações desta seção, elas não são métodos de um UUID.

- **`MinAt(Level, instante) UUID`**: o menor UUIDv7 que a biblioteca
  poderia gerar naquele instante e naquele nível, com os bits livres de
  entropia em zero.
- **`MaxAt(Level, instante) UUID`**: o maior, com os bits livres em um.
  É o limite superior **fechado** do instante.
- **`RangeAt(Level, from, to) (lo, hi)`**: o par de um intervalo
  **semiaberto** `[from, to)`, com `lo = MinAt(nível, from)` e
  `hi = MinAt(nível, to)`, para `id >= lo AND id < hi`.
- **`GenerateAt(Level, instante) UUID`** e
  **`GenerateAtString(Level, instante) string`** (seção 3.6): um UUIDv7
  daquele instante com os bits livres **sorteados**. Duas chamadas com o
  mesmo instante devolvem valores diferentes. Existem também como
  métodos do gerador, para que uma fonte de entropia configurada pelo
  chamador continue valendo.

As cinco preservam versão e variante, decompõem o instante com saturação
nas duas pontas da faixa representável e valem apenas para
identificadores gravados no **mesmo nível**. As regras normativas estão
nas seções 3.5 e 3.6.

---

## 8. Serialização, Banco de Dados e Valores Especiais

1. **Serialização em Texto e JSON**:
   - Um UUID serializado em JSON **DEVE ser formatado como string
     canônica entre aspas** (`"0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff"`).
   - **NUNCA** serialize como um vetor/array de 16 números inteiros.
2. **Integração com Banco de Dados**:
   - A **leitura DEVE aceitar as duas formas**: string canônica em
     qualquer formato aceito pelo analisador permissivo, e 16 bytes
     crus. Texto vazio e fatia vazia equivalem a ausência de valor, sem
     erro. Como a análise, a leitura confere só a forma (seção 4).
   - A **escrita padrão DEVE ser a string canônica**, e esse formato é
     estável: trocá-lo deixaria duas representações na mesma coluna e as
     linhas antigas parariam de casar com as consultas.
   - A escrita binária **DEVE existir como tipo distinto**, escolhido por
     conversão no ponto da consulta, e nunca por configuração global. Ela
     entrega os 16 bytes em ordem de rede, sem rotacionar campos: a
     rotação que algumas receitas de banco sugerem para melhorar a
     localidade do índice é desnecessária no UUIDv7, que já nasce
     ordenado, e produziria um valor ilegível para outras ferramentas.
   - O tipo de escrita binária **DEVE implementar as mesmas interfaces de
     serialização** que o tipo padrão (`MarshalText`, `UnmarshalText`,
     `MarshalBinary`, `UnmarshalBinary`), delegando para a implementação
     do tipo base. Sem esses métodos, `encoding/json` serializaria o
     valor como vetor de 16 números inteiros em vez da string canônica,
     violando a regra 1 desta seção.
   - Fornecer um tipo `NullUUID` contendo o UUID e um booleano `Valid`
     para campos de tabela que permitem valor `NULL`, e o equivalente
     para a escrita binária. O equivalente binário anulável **DEVE
     implementar as mesmas interfaces de serialização** que o anulável
     padrão (`MarshalJSON`, `UnmarshalJSON`, `MarshalText`,
     `UnmarshalText`, `MarshalBinary`, `UnmarshalBinary`), para que a
     tabela normativa de ausência abaixo se aplique igualmente a ambos.
   - **Ausência de valor e UUID nulo são valores distintos** e **NÃO
     DEVEM** colapsar um no outro. Com o booleano falso a escrita produz
     `NULL`; dezesseis bytes zerados só saem com o booleano verdadeiro e
     o UUID igual a `Nil`.
   - A representação da **ausência** no tipo anulável depende do formato
     de destino, e não se deduz de uma regra só: cada formato usa a
     convenção própria de "nada aqui".

     | Destino | Ausência produz | Leitura de entrada vazia |
     |:---|:---|:---|
     | Banco de dados | `NULL` | ausência, sem erro |
     | JSON | o literal `null`, sem aspas | ausência, sem erro |
     | Texto | sequência vazia (zero bytes) | ausência, sem erro |
     | Binário | sequência vazia (zero bytes) | ausência, sem erro |

     Na leitura, a entrada vazia em qualquer dos quatro produz ausência
     de valor, sem erro, coerente com a regra de texto vazio acima. Um
     destino que espere a sequência vazia e receba o literal `null`, ou
     o contrário, falha na desserialização, e é por isso que a tabela é
     normativa. Entrada inválida devolve erro e, nas desserializações,
     **preserva o receptor inteiro**, identificador e booleano, conforme
     a seção 6.4; só a leitura de valor de banco derruba o booleano, pela
     exceção registrada ali.
3. **Valor Especial e Ordenação**:
   - `Nil`: todos os 16 bytes em zero (`00000000-0000-0000-0000-000000000000`).
     É o valor devolvido junto de toda recusa de análise e o que a
     leitura de banco grava para ausência. Não é UUIDv7 (seção 4).
   - `Compare(a, b)`: comparação byte a byte em ordem lexicográfica
     retornando `-1`, `0` ou `1`. Para UUIDv7 coincide com a ordem
     cronológica na resolução do nível (seção 3.4). A assinatura
     **DEVE** servir diretamente à função de ordenação da biblioteca
     padrão da linguagem, quando ela aceitar uma função de comparação,
     para que ordenar uma lista não exija tipo de lista próprio.

---

## 9. Catálogo de Armadilhas Evitadas (Guia Anti-Regressão)

Toda reimplementação deve garantir proteção contra estes 12 defeitos
reais:

| # | Armadilha Histórica | Consequência | Solução Obrigatória |
|---|:---|:---|:---|
| 1 | **Parser pulando hífens em laço** | Entrada maliciosa com hífen extra causava pânico e crash por `index out of range` | Decodificar exclusivamente por tabela fixa de 16 posições (`hexOffsets`) |
| 2 | **Relógio anterior a 1970** | Módulo de números negativos corrompia `rand_a` e `rand_b` | Fixar piso em 0 para `unix_ts_ms`, microssegundos e nanossegundos se `sec < 0` |
| 3 | **Inteiro de 64 bits para nanos** | `UnixNano()` estoura em 2262-04-11 | Ler segundos e nanossegundos em duas partes separadas |
| 4 | **Leitura de tempo cega quanto à versão** | A extração completa lia qualquer UUID como UUIDv7: um UUIDv4 vindo de fora devolvia uma data aleatória entre 1970 e 10889, sem erro | Conferir versão e variante antes de interpretar qualquer bit, e recusar com erro de versão próprio (seção 4) |
| 5 | **Fallback de entropia no relógio** | Falha de `crypto/rand` degradava silenciosamente para o relógio, gerando colisões | Falhar imediatamente com pânico (fail-fast); proibido degradar em silêncio |
| 6 | **Escrita em sink global em teste concorrente** | Detector de corrida (`-race`) disparava falso alerta em benchmarks | Usar `runtime.KeepAlive(u)` por goroutine em vez de escrever em variável compartilhada |
| 7 | **Recusa só pela versão** | A leitura por nível conferia o nibble de versão e ignorava a variante: um identificador com nibble `7` e variante `0b11`, que não é UUIDv7, era lido como tal | O predicado de validade exige as duas condições, e a varredura das 64 combinações de versão e variante o trava (seção 10, caso 7) |
| 8 | **Desperdício de entropia no v7** | Sortear duas palavras de 64 bits nos Níveis 2 e 3 | Sortear apenas 1 palavra nos Níveis 2 e 3 (economia de 17% em concorrência) |
| 9 | **Serialização JSON como array** | `[1, 146, 247, ...]` em vez de `"0192f7c5-..."` quebrava interoperabilidade | Implementar `MarshalText`/`MarshalBinary` canônicos |
| 10 | **Tags sobrescritas com `-f`** | Quebrava a verificação de integridade no registro público (`sum.golang.org`) | Tags publicadas são estritamente imutáveis; nunca mover com `-f` |
| 11 | **Estouro de `sec * 1000` com instante fora da faixa** | Com o instante vindo por parâmetro, o produto estoura o inteiro com sinal e troca de sinal: data remota no futuro cai no piso da época, data remota no passado vira carimbo enorme. A guarda de `sec < 0` não dispara, porque o sinal já foi invertido | Decidir a saturação sobre os **segundos**, antes da multiplicação (seções 3.2 e 3.5) |
| 12 | **Fronteira de intervalo truncando em vez de saturar** | O empacotamento por deslocamento descarta os bits acima de 48 de graça: a fronteira dá a volta e a consulta por faixa devolve as linhas erradas **em silêncio**, sem erro nenhum. Zerar `micro` e `nano` na saturação tem o mesmo efeito na travessia da borda | Saturar nas duas pontas, levando `micro` e `nano` a 999 no teto; a fronteira nunca pode regredir quando o instante avança (seção 3.5) |

---

## 10. Casos de Teste Obrigatórios para Validação

1. **Conformidade de Versão e Variante**:
   - Validar que cada forma de gerar (os Níveis 1 a 3, o nome por versão
     `GenerateV7`, os nomes por nível `GenerateV7Level1`,
     `GenerateV7Level2` e `GenerateV7Level3`, a geração por instante e o
     gerador criptográfico) define exatamente a versão 7 e a variante
     `0b10`, e que o predicado de validade da seção 4 aceita o resultado.
   - Validar que o nome por versão é o Nível 1: com entropia constante,
     `rand_a` e o topo de `rand_b` saem inteiros da fonte, sem campo de
     tempo sub-milissegundo, e o carimbo é o do relógio.
   - Validar que cada nome gera no nível que declara, e não em outro:
     com entropia constante em um, o identificador gerado pelo nome é
     lido de volta no nível declarado (seção 7) e regerado por instante
     nesse mesmo nível (seção 3.6), e os 16 bytes **DEVEM** coincidir.
     A entropia em um deixa `rand_a` em `0xfff` e o topo de `rand_b` em
     `0x3ff`, valores que nenhum campo de tempo em 0..999 assume, então
     um nome que chamasse outro nível diverge em pelo menos um campo.
     Conferir também cada campo: `rand_a` em `0xfff` no Nível 1 e em
     0..999 nos demais; topo de `rand_b` em `0x3ff` nos níveis 1 e 2 e
     em 0..999 no Nível 3.
   - Validar o nível de **todas** as formas de gerar, inclusive as
     funções de pacote, que usam o gerador padrão e não aceitam fonte
     injetada: a geração por nível, em binário e em texto, pelo relógio e
     por instante, nos três níveis e em níveis desconhecidos (que devem
     se comportar como o Nível 1), e os quatro nomes. Sem fonte
     injetável, a prova é a assinatura estatística dos bits livres, que
     decorre da seção 3.1: no Nível 1, `rand_a` passa de 999 em 3096 de
     cada 4096 amostras, e nos níveis 2 e 3 nunca passa; no Nível 2, o
     topo de `rand_b` passa de 999 em 24 de cada 1024, e no Nível 3 nunca
     passa. Com 4.000 amostras por forma, confundir o Nível 2 com o 3 tem
     probabilidade abaixo de 10^-41. Sem este caso, uma função de pacote
     que ignorasse o nível pedido produziria UUIDv7 válidos e distintos,
     e passaria por todos os outros.
   - Validar que a leitura do nibble de versão e do código de variante
     devolve o valor dos bits para **qualquer** valor (versão de 0 a 15,
     variante de 0 a 3), com os demais bits do byte em zero e em um, e
     não só o `7` e o `0b10` dos identificadores gerados.
2. **Robustez do Analisador contra Mutações**:
   - Executar teste cobrindo **todas as 36 × 256 mutações de um único byte**
     sobre uma string canônica válida: nenhuma mutação pode causar pânico.
   - Sobre as mesmas mutações, exigir **aceitação exata**: o analisador
     estrito aceita a mutação se, e somente se, ela mantém um dígito
     hexadecimal numa posição de dígito ou o hífen numa posição de hífen,
     e devolve o valor calculado por um decodificador independente da
     implementação. Conferir só a ausência de pânico deixa passar um
     analisador que aceite `G` ou `:` como dígito.
   - Repetir a varredura, com o mesmo critério, nas **quatro formas** do
     analisador permissivo (seção 6.3): canônica, entre chaves, URN e
     hexadecimal cru. As chaves só aceitam a si mesmas; o prefixo URN
     aceita as mesmas letras em qualquer caixa e só os dois-pontos. A ida
     e volta pela forma canônica não substitui este caso: uma entrada
     aceita indevidamente volta ao mesmo valor, e por isso o fuzzing que
     só confere a ida e volta não percebe um hífen ou um dois-pontos que
     deixaram de ser conferidos.
   - Submeter o analisador a campanhas de *fuzzing* contínuo. No
     analisador permissivo, o oráculo **DEVE** exigir que toda entrada
     aceita seja, sem distinção de caixa, exatamente uma das quatro formas
     escritas a partir do valor lido; a ida e volta sozinha não basta,
     pelo motivo do item anterior.
   - **O fuzzing não para no texto.** A aritmética temporal **DEVE**
     receber campanhas próprias, e por um motivo diferente: no texto o
     risco é leitura fora dos limites, e aqui é saturação, estouro de
     sinal e resto negativo. Tabela de casos escolhidos à mão não varre
     faixa, e é precisamente nas duas metades do inteiro com sinal que o
     estouro da multiplicação por mil (seção 3.2, item 3) se manifesta.
     São dois alvos:
     - **Construção por instante**: recebe dois instantes arbitrários e
       exige, em todos os níveis mais um nível desconhecido, versão 7 e
       variante `0b10` nas duas fronteiras, fronteira inferior nunca
       acima da superior, o valor gerado sempre dentro das fronteiras do
       próprio instante, e monotonicidade quando o segundo instante não
       é anterior ao primeiro.
     - **Leitura de tempo**: recebe 16 bytes arbitrários e exige que as
       quatro leituras da seção 7 nunca entrem em pânico e concordem com
       o predicado de validade ao aceitar ou recusar (seção 4); aceito o
       UUIDv7, que a extração cega devolva o carimbo exato e os campos
       crus dentro da faixa dos bits, que a leitura por nível some os
       campos sub-milissegundo exatamente quando cabem em 0 a 999, com
       os dois caindo juntos no Nível 3, e que o instante lido no Nível 3
       regere pela fronteira inferior os mesmos campos de tempo.
3. **Bordas Temporais Extremas**:
   - Testar instantes com data anterior a 1970 (ex.: ano 1969 e ano 1800).
   - O caso pré-1970 **DEVE** usar um segundo negativo com fração
     positiva (por exemplo `sec = -1`, `nsec = 500.000.000`): é a entrada
     que uma transcrição da fórmula com conversão sem sinal antes da
     guarda (seção 3.2, item 2) transforma em carimbo enorme, e as duas
     formas de piso aceitas devem devolver a época.
   - Testar também o último nanossegundo antes da época (`sec = -1`,
     `nsec = 999.999.999`), em que `unix_ts_ms` vale exatamente `-1`: é a
     borda do piso, e um piso escrito como `unix_ts_ms < -1` passa no
     caso anterior e só falha aqui, com os campos sub-milissegundo em 999.
   - Testar instantes além do ano 2262 (ex.: ano 2300).
   - Validar viradas de segundo (`nsec = 999_999_999`) e viradas de
     milissegundo (`sub_ms = 999_999`).
4. **Contagem de Sorteios de Entropia**:
   - Com gerador de contagem determinística, verificar que Nível 2 e
     Nível 3 consomem 1 chamada; Nível 1 consome 2 chamadas. Os nomes
     consomem o mesmo que o nível que apelidam: `GenerateV7` e
     `GenerateV7Level1` consomem 2, `GenerateV7Level2` e
     `GenerateV7Level3` consomem 1.
   - **Fiação das palavras** (seção 3.3): com uma fonte que devolva
     palavras **distintas** em sequência, exigir, pelo relógio e por
     instante, que no Nível 1 `rand_a` venha da primeira palavra e
     `rand_b` da segunda, e que nos níveis 2 e 3 `rand_b` venha da única
     palavra sorteada. A contagem não basta, e a entropia constante
     também não: as duas passam com `rand_a` tirado da palavra errada.
   - **Leitor do chamador** (seção 5.3): com um leitor que entregue um
     byte por leitura, exigir os valores exatos que a ordem de rede e a
     ordem `r1`, `r2` produzem, o consumo de exatamente 8 bytes por
     palavra e o pânico com o erro de fonte quando os bytes acabam.
   - **Formas em texto**: com entropia constante, exigir que a geração em
     texto, pelo relógio e por instante, tenha os bits livres da fonte do
     **próprio** gerador, e não os do gerador padrão.
   - **Gerador criptográfico**: exigir que os bits venham da fonte
     criptográfica capturada na construção. Em Go, a prova é trocar
     `crypto/rand.Reader` por um leitor de bytes conhecidos só durante a
     construção e exigir esses bytes no identificador gerado depois.
   - **Gerador padrão**: exigir, em volume, que o Nível 1 sorteie duas
     palavras independentes: `rand_a` coincide com os 12 bits inferiores
     de `rand_b` em cerca de uma de cada 4096 amostras, e não em todas.
5. **Vetor Externo da RFC 9562 (Apêndice A.6)**:
   - A RFC publica um único exemplo de UUIDv7, e ele é o único vetor de
     leitura de tempo calculado fora deste projeto:
     `017f22e2-79b0-7cc3-98c4-dc0c0c07398f`, com `unix_ts_ms` =
     `0x017F22E279B0` (2022-02-22T19:22:22Z), `rand_a` = `0xCC3` e
     `rand_b` = `0x18C4DC0C0C07398F`.
   - Exigir da extração completa `Seconds` = 1.645.557.742,
     `Milliseconds` = 0, `Microseconds` = 3267 e `Nanoseconds` = 396, na
     forma binária e na forma em texto, esta também com o vetor em
     maiúsculas, como aparece na RFC.
   - Exigir do instante em milissegundo e da leitura por nível, nos três
     níveis, exatamente 2022-02-22T19:22:22Z: `rand_a` = 3267 está fora
     da faixa, e o descarte da seção 7 derruba os dois campos
     sub-milissegundo também no Nível 3, onde o topo de `rand_b` (396)
     caberia. O vetor exercita o descarte num UUIDv7 que a biblioteca
     não produziu.
6. **Ordenação Temporal Coerente**:
   - Testar que se o instante de B for estritamente superior ao instante de
     A, a comparação de strings e de bytes de B é estritamente maior que a
     de A.
   - Não contar regressões em laço apertado na mesma thread: se o tempo não
     avança na resolução do host, o desempate por entropia é aleatório.
7. **Recusa de UUIDs que Não São v7 (seção 4)**:
   - Sobre os mesmos 16 bytes de um UUIDv7 válido, varrer as **64
     combinações** de nibble de versão (0 a 15) e código de variante (0
     a 3) e exigir que o predicado de validade aceite só a combinação
     versão 7 com variante 2, e que as quatro leituras de tempo
     concordem com ele: extração binária e em texto devolvendo
     **exatamente** o erro de versão com a estrutura zerada, e as
     leituras de instante devolvendo falso com o instante zero, em todos
     os níveis mais um desconhecido. A varredura é exaustiva de
     propósito: uma implementação que confira só a versão, ou só a
     variante, passa num teste com um único UUIDv4 e falha aqui.
   - Repetir a recusa sobre UUIDs reais: os exemplos da RFC 9562 das
     versões 1 (A.1), 3 (A.2), 4 (A.3), 5 (A.4), 6 (A.5) e 8 (B.2), o
     UUID nulo e o UUID com todos os bits em um. Exigir, para os mesmos
     valores, que os analisadores estrito e permissivo os **aceitem**:
     a recusa é da leitura de tempo, não da análise.
   - Exigir que o erro de versão não seja reconhecido como erro de
     formato.
8. **Concorrência e Ausência de Corridas de Dados**:
   - Gerar 1.000.000 de UUIDs divididos entre centenas de threads
     simultâneas sem nenhuma colisão e sem nenhum alerta no detector de
     corridas (*race detector*).
9. **Independência de Ordem e de Repetição**:
   - A suíte deve passar com repetição (`-count 3`) e com ordem
     embaralhada (`-shuffle on`). A biblioteca não tem estado global
     mutável além do gerador padrão, e nenhum teste pode depender da
     execução de outro: um teste que só passa numa ordem esconde um
     estado compartilhado que a especificação não prevê.
10. **Fronteiras de Tempo (seção 3.5)**:
    - Gerar uma rajada entre dois instantes lidos do relógio e conferir
      que **toda** ela cai dentro das fronteiras desses instantes, nos
      três níveis. É o teste que prova a fronteira, e não a inspeção do
      layout de bits.
    - Conferir versão 7 e variante `0b10` nas duas fronteiras, nos três
      níveis, inclusive nos instantes saturados.
    - Conferir a ordem: a fronteira inferior nunca passa da superior no
      mesmo instante, e instantes separados pela resolução do nível
      produzem faixas disjuntas e em ordem.
    - Conferir a monotonicidade sobre uma lista de instantes que inclua
      as duas pontas saturadas e a travessia da borda superior. Zerar
      `micro` e `nano` na saturação **deve** fazer este teste falhar.
    - Conferir o estouro da multiplicação por mil descrito na seção 3.5,
      com instantes grandes o bastante para provocá-lo nas duas direções.
    - Conferir a ida e volta: ler a fronteira com a leitura por nível
      devolve o instante de origem, truncado à resolução do nível.
    - Travar o layout com vetores fixos, calculados fora da
      implementação.
    - Travar os **valores exatos na borda superior**, porque a
      monotonicidade não mostra onde a saturação começa: saturar desde o
      início do último segundo representável, ou já no último
      milissegundo, preserva a ordem e passa pelos casos acima. A
      fronteira inferior (bits livres em zero) de cada instante:

      | Instante | Nível 1 | Nível 2 | Nível 3 |
      |:---|:---|:---|:---|
      | `10889-08-02T05:31:50Z` | `ffffffff-fd70-7000-8000-000000000000` | `ffffffff-fd70-7000-8000-000000000000` | `ffffffff-fd70-7000-8000-000000000000` |
      | `10889-08-02T05:31:50.655Z` | `ffffffff-ffff-7000-8000-000000000000` | `ffffffff-ffff-7000-8000-000000000000` | `ffffffff-ffff-7000-8000-000000000000` |
      | `10889-08-02T05:31:50.655456789Z` | `ffffffff-ffff-7000-8000-000000000000` | `ffffffff-ffff-71c8-8000-000000000000` | `ffffffff-ffff-71c8-b150-000000000000` |
      | `10889-08-02T05:31:50.656Z` e posteriores | `ffffffff-ffff-7000-8000-000000000000` | `ffffffff-ffff-73e7-8000-000000000000` | `ffffffff-ffff-73e7-be70-000000000000` |

      A fronteira superior saturada do Nível 3 é
      `ffffffff-ffff-73e7-be7f-ffffffffffff`. A geração por instante com
      entropia nula, que usa a mesma decomposição (seção 3.6), produz a
      fronteira inferior de cada linha, e a leitura por nível de cada
      fronteira devolve o instante truncado à resolução do nível, ou o
      último instante representável nas linhas saturadas.
11. **Geração por Instante Explícito (seção 3.6)**:
    - **Ida e volta com a extração**: gerar para segundos, milissegundos,
      microssegundos e nanossegundos conhecidos e conferir que a leitura
      devolve exatamente esses campos, em cada nível que os grava. É o
      teste central, porque prova a simetria que motiva a operação.
    - **Não divergência com a geração pelo relógio**: com a mesma fonte
      de entropia constante, gerar pelo relógio, ler o instante embutido
      de volta e regerar para ele. Os 16 bytes **devem** ser idênticos. É
      esta a trava da duplicação deliberada da seção 11.2, e ela falha se
      as duas cópias do empacotamento se separarem.
    - **Contenção pelas fronteiras**: o valor gerado para um instante cai
      sempre dentro das fronteiras daquele instante (seção 3.5).
    - **Não determinismo**: muitas chamadas com o mesmo instante
      devolvem valores todos distintos, e com os campos de tempo iguais.
    - Consumo de entropia por nível igual ao da seção 3.3.
    - Bordas: pré-1970, saturação acima da faixa e níveis desconhecidos.
12. **Vetores Dourados da Extensão Multinível**:
    - A RFC 9562 publica um exemplo de UUIDv7 padrão (caso 5), mas a
      extensão multinível é deste projeto e não tem vetor publicado em
      lugar nenhum. Sem eles, uma reimplementação em outra linguagem não
      tem contra o que se conferir justamente na parte que não é padrão,
      onde é mais fácil errar.
    - Os vetores abaixo **são contrato**. Uma implementação que produza
      outra coisa para as mesmas entradas está errada. Mudá-los é
      mudança de formato de dados, não ajuste de teste.
    - As entradas são um instante e o valor dos **bits livres de
      entropia**, todos em zero ou todos em um. São exatamente o que as
      duas fronteiras da seção 3.5 produzem, e é assim que se obtêm sem
      relógio: `MinAt` para os bits em zero, `MaxAt` para os bits em um.

    **Grupo A** — 2026-01-01T00:00:00.123456789Z  (sec=1767225600, nsec=123456789)

    | Nível | Bits livres | 16 bytes | String canônica |
    |:---|:---|:---|:---|
    | 1 | zero | `019b76daa87b70008000000000000000` | `019b76da-a87b-7000-8000-000000000000` |
    | 1 | um | `019b76daa87b7fffbfffffffffffffff` | `019b76da-a87b-7fff-bfff-ffffffffffff` |
    | 2 | zero | `019b76daa87b71c88000000000000000` | `019b76da-a87b-71c8-8000-000000000000` |
    | 2 | um | `019b76daa87b71c8bfffffffffffffff` | `019b76da-a87b-71c8-bfff-ffffffffffff` |
    | 3 | zero | `019b76daa87b71c8b150000000000000` | `019b76da-a87b-71c8-b150-000000000000` |
    | 3 | um | `019b76daa87b71c8b15fffffffffffff` | `019b76da-a87b-71c8-b15f-ffffffffffff` |

    **Grupo B** — 2026-01-01T00:00:00.000000000Z  (sec=1767225600, nsec=0)

    | Nível | Bits livres | 16 bytes | String canônica |
    |:---|:---|:---|:---|
    | 1 | zero | `019b76daa80070008000000000000000` | `019b76da-a800-7000-8000-000000000000` |
    | 1 | um | `019b76daa8007fffbfffffffffffffff` | `019b76da-a800-7fff-bfff-ffffffffffff` |
    | 2 | zero | `019b76daa80070008000000000000000` | `019b76da-a800-7000-8000-000000000000` |
    | 2 | um | `019b76daa8007000bfffffffffffffff` | `019b76da-a800-7000-bfff-ffffffffffff` |
    | 3 | zero | `019b76daa80070008000000000000000` | `019b76da-a800-7000-8000-000000000000` |
    | 3 | um | `019b76daa8007000800fffffffffffff` | `019b76da-a800-7000-800f-ffffffffffff` |

    **Grupo C** — 1970-01-01T00:00:00.000000000Z  (sec=0, nsec=0)

    | Nível | Bits livres | 16 bytes | String canônica |
    |:---|:---|:---|:---|
    | 1 | zero | `00000000000070008000000000000000` | `00000000-0000-7000-8000-000000000000` |
    | 1 | um | `0000000000007fffbfffffffffffffff` | `00000000-0000-7fff-bfff-ffffffffffff` |
    | 2 | zero | `00000000000070008000000000000000` | `00000000-0000-7000-8000-000000000000` |
    | 2 | um | `0000000000007000bfffffffffffffff` | `00000000-0000-7000-bfff-ffffffffffff` |
    | 3 | zero | `00000000000070008000000000000000` | `00000000-0000-7000-8000-000000000000` |
    | 3 | um | `0000000000007000800fffffffffffff` | `00000000-0000-7000-800f-ffffffffffff` |

    **O que cada coisa prova.**

    - **Grupo A**, os três níveis: os microssegundos 456 aparecem como
      `1c8` em `rand_a` nos níveis 2 e 3, e não no nível 1, onde o campo
      é entropia. É a prova da posição dos microssegundos.
    - **Grupo A**, nível 3: os nanossegundos 789 (`0x315`, dez bits)
      aparecem repartidos entre os seis bits baixos do byte 8 (`0x31`,
      somados à variante dão `0xb1`) e o nibble alto do byte 9 (`0x5`).
      É a prova da posição dos nanossegundos nos dez bits altos de
      `rand_b`.
    - **Todas as linhas com bits livres em um**: o byte 6 nunca passa de
      `0x7f` e o byte 8 fica sempre na faixa `0x80..0xbf`. É a prova de
      que a versão e a variante sobrevivem ao preenchimento, e é o que
      torna as fronteiras da seção 3.5 limites corretos.
    - **Grupo B contra o grupo A**: com o sub-milissegundo zerado, os
      níveis 2 e 3 devolvem `rand_a` em zero mesmo com os bits livres em
      um, enquanto o nível 1 devolve `0xfff`. É a prova de que os níveis
      2 e 3 **não** sorteiam `rand_a`, e pega erro de sinal e de
      deslocamento que um instante com todos os campos preenchidos
      esconderia.
    - **Grupo C**: a época Unix com os 48 bits de carimbo zerados. Um
      instante **anterior** a 1970 tem de produzir exatamente estes
      mesmos valores, pelo piso da seção 3.2. Esse é o par que prova o
      piso, e por isso a suíte confere o grupo C duas vezes, uma com a
      época e outra com `1969-12-31T23:59:59.999999999Z`.

    Os valores publicados aqui foram calculados por uma implementação
    independente, escrita a partir das regras das seções 3.1, 3.2 e 3.5,
    e só então conferidos contra a implementação de referência. Um vetor
    produzido pela própria implementação e conferido contra ela mesma
    não prova nada.
13. **Extração Cega de Nível (seção 7)**:
    - Fixar um vetor com os campos sub-milissegundo **fora** da faixa de
      0 a 999 e exigir que a extração completa os devolva crus. Um vetor
      com valores dentro da faixa não distingue as duas políticas de
      leitura e por isso não prova nada aqui.
    - Vetor: os 16 bytes `0192f7c51a2b7c3d8e4faabbccddeeff`, string
      `0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff`, produzem carimbo
      `0x0192f7c51a2b` = 1.730.733.742.635 ms, portanto `Seconds` =
      1.730.733.742 e `Milliseconds` = 635; `Microseconds` = `0xc3d` =
      3133 e `Nanoseconds` = `0xe4` = 228, os dois fora da faixa de
      tempo real.
    - Conferir que a forma em texto concorda com a binária, campo a
      campo, e que uma string recusada devolve o sentinela puro com a
      estrutura zerada.
    - Conferir, em volume, as faixas por nível: gerado em Nível 2 ou 3, o
      campo de microssegundos fica em 0 a 999; gerado em Nível 3, o de
      nanossegundos também; gerado em Nível 1, os dois **excedem** 999 em
      alguma amostra, o que prova que a extração não filtra.
14. **Descarte por Faixa na Leitura por Nível (seção 7)**:
    - Montar um UUIDv7 com `rand_a` acima de 999 e exigir que a leitura
      em Nível 2 devolva o instante truncado ao milissegundo.
    - Montar um UUIDv7 com `rand_a` acima de 999 e `nano` dentro de 0 a
      999 e exigir que a leitura em Nível 3 descarte **os dois** campos.
      É este o caso que distingue as duas regras: uma implementação que
      valide os campos separadamente passa no anterior e falha neste.
    - Montar o caso simétrico, `micro` válido e `nano` acima de 999, e
      exigir que o Nível 3 devolva só o milissegundo enquanto o Nível 2
      soma os microssegundos, porque não lê `nano`.
    - Montar o caso inteiramente válido e exigir a soma dos dois.
    - Conferir que a extração cega, sobre os mesmos bytes, devolve os
      valores crus. As duas leituras divergindo é o comportamento
      correto.
15. **Política de Entropia em Três Casos (seção 5.2)**:
    - Fonte nula ou leitor nulo entregue a uma função de construção
      **DEVE** entrar em pânico na construção, antes de qualquer
      geração.
    - Um gerador de valor zero, e uma referência nula usada como
      receptor, **DEVEM** produzir identificadores válidos e distintos
      de zero em todas as gerações do tipo: pelo relógio nos três
      níveis, pelo nome por versão `GenerateV7`, pelos nomes por nível
      `GenerateV7Level1` a `GenerateV7Level3` e por instante, em binário
      e em texto. Nenhuma pode derrubar o processo.
    - Uma leitura que falhe em um gerador construído sobre um leitor
      **DEVE** entrar em pânico com o erro de fonte de entropia como
      valor; um leitor esgotado é o jeito de provocá-lo.
16. **Travas de Alocação (seção 1)**:
    - O objetivo de zero alocação de heap no caminho quente é
      **verificável e obrigatório**, não aspiracional. Estas operações
      **DEVEM** ser livres de alocação, medidas em laço com contagem de
      alocações por iteração: geração binária pelo relógio nos três
      níveis, pelo nome por versão `GenerateV7` e pelos nomes por nível
      `GenerateV7Level1` a `GenerateV7Level3`; geração binária por
      instante (seção 3.6); as duas fronteiras e o intervalo (seção
      3.5); análise estrita (seção 6.2); análise permissiva nos quatro
      formatos, em texto e em bytes (seção 6.3); as quatro leituras de
      tempo (seção 7) e o predicado de validade, **tanto aceitando quanto
      recusando** (seção 4), porque a recusa não pode passar a montar um
      erro por chamada; escrita da forma canônica em buffer do chamador
      com capacidade sobrando; e **escrita dos 16 bytes em buffer do
      chamador com capacidade sobrando**.
    - **Exceção única.** A conversão que devolve uma string, pelo relógio
      ou por instante, pode alocar **exatamente uma vez**, porque o
      resultado é a alocação. Exigir zero aqui é impossível sem mudar a
      assinatura, e quem precisa de zero usa a escrita em buffer do
      chamador, que está na lista acima.
    - **Condição de medição.** A contagem de referência é obtida **sem**
      detector de corrida. O detector altera o código gerado e pode
      contar alocações que não existem em produção; a integração
      contínua da implementação de referência repete as travas em um
      passo próprio, sem o detector, por esse motivo.
    - **Proibição.** Uma falha nestas travas **NÃO DEVE** ser resolvida
      elevando o limite tolerado. A trava existe para denunciar
      regressão de desempenho introduzida por refatoração, e relaxá-la
      remove a única defesa contra ela. Ver a seção 11.1.
    - Em linguagens cujo tipo do identificador só existe no heap, o
      limite passa a ser uma alocação por operação, a do próprio
      resultado, e a divergência **DEVE** ser registrada junto da trava.
17. **Taxonomia de Erros (seção 6.4)**:
    - Para cada linha da tabela da seção 6.4, submeter a entrada
      descrita e exigir o erro descrito, reconhecido pelo tipo e não só
      pela presença. Entradas mínimas: comprimento errado (`abc`); 38
      caracteres com as chaves trocadas por outro caractere; 45
      caracteres com o prefixo `urn:uiid:`; dígito inválido na forma
      canônica; hífen fora de lugar; 15 bytes na construção binária e na
      desserialização binária; um número no lugar da string no JSON do
      tipo anulável; um tipo estranho na leitura de banco; um UUIDv4 bem
      formado na extração de tempo, binária e em texto.
    - Exigir que todo erro de comprimento e de chaves seja reconhecível
      pelo sentinela de formato, e que o analisador estrito e a extração
      de tempo em texto devolvam exatamente o sentinela, por igualdade
      direta, para todas essas entradas.
    - Exigir que o prefixo URN inválido **não** seja classificado como
      comprimento inválido nem como chaves inválidas: é a assimetria
      registrada, e o teste a trava.
    - Exigir o UUID zerado junto de todo erro, e o receptor inalterado
      nas desserializações. Exigir que o erro de tipo não suportado, o de
      versão e o de fonte de entropia **não** sejam reconhecidos como
      erro de formato, e que o UUIDv4 que a extração de tempo recusa seja
      aceito pelo analisador permissivo.
    - No JSON do tipo anulável, repetir a string curta, a de chaves
      trocadas e a de dígito inválido **com uma sequência de escape**
      JSON no conteúdo. Com escape, a leitura passa pelo decodificador de
      JSON da linguagem antes da análise, e o erro específico da análise
      (comprimento, chaves) **DEVE** sobreviver ao desvio, e não virar o
      sentinela puro.
    - Exigir a recusa com erro de comprimento de **todo** comprimento de
      0 a 64 bytes diferente de 16 na construção binária e nas
      desserializações binárias dos quatro tipos, com o receptor intacto
      (nos anuláveis, o comprimento zero é ausência de valor, pela seção
      8). Testar só 15 bytes deixa passar a aceitação silenciosa de 17 ou
      mais, com o excedente descartado.
18. **Ausência de Valor por Formato (seção 8)**:
    - Serializar o tipo anulável sem valor para banco, JSON, texto e
      binário e exigir, respectivamente, `NULL`, o literal `null`, a
      sequência vazia e a sequência vazia; desserializar a entrada vazia
      nos quatro e exigir ausência sem erro. Desserializar entrada
      inválida em JSON, texto e binário, a partir de um receptor que
      **já continha valor presente**, e exigir erro com o receptor
      inteiro intacto; fazer a mesma leitura recusada pela via de banco e
      exigir erro com o identificador intacto e o booleano falso, que é a
      exceção da seção 6.4.
19. **Coerência da API de Apoio (seções 6.3, 7 e 8)**:
    - A leitura de banco dos quatro tipos e a análise que entra em pânico
      **DEVEM** aceitar as quatro formas do analisador permissivo, em
      maiúsculas e minúsculas, e a leitura de banco também com o texto
      em sequência de bytes. Testar só a forma canônica deixa passar
      qualquer restrição a ela.
    - A comparação **DEVE** distinguir cada uma das 16 posições sozinha,
      com os demais bytes em zero e em um, e com `0x7f` contra `0x80`,
      que também denuncia comparação com sinal. O predicado do UUID nulo
      **DEVE** recusar um valor com um único byte diferente de zero, em
      cada posição, e a fronteira inferior de Nível 1 na época, que
      começa e termina em zero mas é UUIDv7.
    - As duas leituras de instante **DEVEM** devolver UTC em todos os
      níveis, nos desconhecidos e no caminho do descarte por faixa. A
      conferência é pela identidade do fuso UTC da linguagem, e não pelo
      nome: numa máquina configurada em UTC o fuso local também se chama
      "UTC", e um instante em horário local passaria despercebido
      justamente na integração contínua.
    - A serialização binária nativa da linguagem que usa a interface
      binária (em Go, `encoding/gob`) **DEVE** fazer a ida e volta dos
      quatro tipos dentro de uma estrutura: com valor, sem valor e com o
      UUID nulo presente.

---

## 11. Decisões de Projeto Firmadas (Não Reabrir)

Esta seção existe para **encerrar** discussões, não para abri-las. Cada
item abaixo foi avaliado, medido quando cabia, e decidido. Uma auditoria
ou revisão que encontre um destes pontos **não deve abrir tarefa pedindo
a mudança** apenas por reconhecer o padrão: a decisão já é a resposta.

Reabrir um item exige **argumento novo**, e a coluna "o que justificaria
rever" diz qual. Preferência documentada, simetria de código ou "o outro
pacote faz diferente" não são argumento novo.

### 11.1 Entropia e desempenho

| Decisão | Data | Motivo | O que justificaria rever |
|:---|:---|:---|:---|
| **A fonte de entropia padrão é o gerador por thread do runtime**, não um pool mantido pela biblioteca nem `crypto/rand`. | 2026-09-11 | O pool custava o par `Get`/`Put`, era esvaziado pelo GC (pagando duas leituras de `crypto/rand` a cada recriação) e descartava itens sob o detector de corrida. A troca mediu -34,7% em paralelo. `crypto/rand` em toda geração custa multiplicado, e existe explicitamente como gerador dedicado. | O runtime da linguagem deixar de oferecer fonte por thread, ou medição mostrando regressão. |
| **Não há contador monotônico** no gerador padrão nem como construtor opcional. | 2026-09-11 | Protótipo com contador de 16 bits e estado atômico: +8,5% em série e **32 vezes** pior em paralelo (7,40 ns para 233,6 ns em 8 núcleos). Exige ainda decidir layout por nível, política de estouro e conviver com a leitura cega de nível na importação, que leria o contador como tempo. Ver 3.4. | Uma construção que dê ordem estrita **sem** estado compartilhado entre threads. |
| **O caminho quente não ganha desvio, indireção nem alocação** para acomodar funcionalidade nova. | permanente | Dezenas de nanossegundos por identificador é o objetivo declarado na seção 1. Uma chamada indireta a mais pode impedir a embutição e custar 1 a 2 ns em um caminho de 40 ns. | Medição antes e depois mostrando custo nulo. |
| **As travas de alocação são requisito verificável da seção 10 (caso 16), não só propriedade desta implementação.** | 2026-09-12 | O objetivo de zero alocação estava declarado na seção 1 sem nenhum caso de teste que o cobrasse. A lista de operações, a exceção única da conversão para texto e a condição de medir sem o detector de corrida viviam só nas instruções de manutenção, que não são especificação. A alternativa, tratar desempenho como propriedade da implementação e não do formato, foi recusada: uma reimplementação poderia alocar em toda geração e se dizer conforme. | Uma linguagem-alvo em que o limite de uma alocação por operação seja comprovadamente inatingível, caso em que a divergência é registrada junto da trava, e não a trava removida. |
| **O gerador sobre leitor forma cada palavra com 8 bytes em ordem de rede, completa as leituras curtas e lê `r1` antes de `r2`; a correspondência é contrato (seção 5.3).** | 2026-09-13 | Era detalhe de implementação, sem teste: uma campanha de mutação mostrou que trocar a ordem dos bytes, ou aceitar a leitura pela metade, passava pela suíte inteira. É a única construção em que o mapeamento de bytes em bits é visível ao chamador, e quem usa um leitor determinístico para reproduzir identificadores depende dele. A leitura curta sem erro é permitida pelo contrato de leitor da linguagem, e aceitá-la deixaria bytes zerados na palavra, a degradação silenciosa que a seção 5.2 proíbe. | Nada previsto: mudar a ordem muda os identificadores de quem já usa leitor determinístico. |

### 11.2 Testabilidade

| Decisão | Data | Motivo | O que justificaria rever |
|:---|:---|:---|:---|
| **O relógio não é injetável.** A geração lê o relógio do sistema diretamente. | 2026-09-11 | A motivação original era testar bordas de relógio; isso foi resolvido isolando a decomposição do instante (3.2) em função pura, testada de dentro do pacote. O layout de bits está travado por testes de entropia fixa. Sobrava apenas o vetor dourado de ponta a ponta, que não paga um campo de função no caminho quente. | Necessidade de teste que a função pura de decomposição comprovadamente não cobre. |
| **A duplicação entre a formatação canônica do caminho quente e a dos serializadores é deliberada.** | permanente | A conversão para texto é caminho quente e não deve pagar uma chamada de função por causa dos serializadores. As duas cópias são pequenas e travadas pelos mesmos testes. | Compilador que comprovadamente embuta a chamada sem custo. |
| **O layout de bytes é escrito duas vezes, e só duas: uma no caminho quente e uma compartilhada por tudo que constrói a partir de um instante (seções 3.5 e 3.6).** | 2026-09-11 | O empacotamento compartilhado recebe os bits livres por parâmetro: constantes na fronteira, sorteados na geração por instante. Fundi-lo com a geração pelo relógio poria uma chamada ou um desvio no caminho quente, que a decisão 11.1 proíbe. O limite é esse: **uma** cópia privada, no caminho quente, e nenhuma outra duplicação tolerada. A divergência entre as duas é travada por teste, que gera pelo relógio, lê o instante de volta e regera para ele exigindo bytes idênticos. | Compilador que comprovadamente embuta a chamada sem custo, medido antes e depois. |
| **O gerador padrão continua sem ponto de injeção; as formas de gerar que o usam são provadas por assinatura estatística dos bits livres.** | 2026-09-13 | As funções de pacote não recebem fonte, e uma campanha de mutação mostrou que todas podiam ignorar o nível pedido, e os nomes, chamar o nível errado, sem nenhuma falha: continuavam produzindo UUIDv7 válidos e distintos. Um ponto de injeção no gerador padrão resolveria o teste criando estado global mutável, que quebraria a independência de ordem da suíte (seção 10, caso 9) e daria a qualquer código do processo o poder de trocar a entropia de todos. A assinatura pela faixa dos campos (seção 10, caso 1) distingue os três níveis com probabilidade de falso alarme abaixo de 10^-41 em 4.000 amostras, a um custo de milissegundos, e a mesma técnica prova a independência das duas palavras do gerador padrão (seção 3.3). | Um nível novo cuja assinatura não se distinga pela faixa dos campos. |
| **A origem dos bits do gerador criptográfico é provada trocando a fonte criptográfica global do sistema só durante a construção; por isso a suíte não usa testes paralelos.** | 2026-09-13 | Validar o resultado não prova a origem: um gerador criptográfico que caísse na fonte padrão produziria UUIDv7 igualmente válidos, e a campanha de mutação mostrou exatamente isso. Provar sem trocar a fonte exigiria um gancho de teste no código de produção, ou API pública nova para um uso que não é de produção. A troca é desfeita antes de qualquer geração; um teste paralelo que lesse a fonte ao mesmo tempo seria corrida de dados, e por isso um teste confere que nenhum arquivo de teste da suíte declara execução paralela. | Um meio, na versão mínima suportada da linguagem, de tornar a fonte criptográfica determinística só dentro de um teste, sem variável global. |

### 11.3 Contrato público

| Decisão | Data | Motivo | O que justificaria rever |
|:---|:---|:---|:---|
| **A biblioteca trata só UUIDv7.** As versões 1, 2, 3, 4, 5, 6 e 8 foram removidas, com o estado de relógio gregoriano, a sequência de relógio, o nó, os espaços de nomes, a camada de compatibilidade de nomes com `github.com/google/uuid` e a suíte comparativa com aquele pacote. | 2026-09-13 | Este projeto nasce de `go-loghub-uuid` v0.6.0 para apresentar uma biblioteca com um único propósito: gerar UUIDv7 nos níveis 1 a 3 e converter UUIDv7 em instante nas mesmas precisões. Tudo que não está diretamente ligado a gerar ou a processar UUIDv7 foi descartado. As versões removidas continuam disponíveis na biblioteca de origem, que segue existindo; mantê-las aqui dobraria a superfície a especificar, testar e versionar sem servir ao propósito. O que ficou de apoio — análise permissiva, serialização, integração com banco, fronteiras e geração por instante — ficou porque serve para receber, guardar e consultar UUIDv7. | Nada previsto: quem precisa das demais versões usa `go-loghub-uuid`. |
| **As leituras de tempo recusam UUID que não é de versão 7 com a variante da RFC**; a extração passou a devolver erro (`ImportBinary(UUID) (Time, error)`), e o erro de versão (`ErrNotV7`) é família própria, fora da de formato. Os analisadores continuam só de forma. | 2026-09-13 | Na biblioteca de origem a extração era cega também quanto à versão: um UUIDv4 vindo de fora devolvia uma data aleatória, sem erro (seção 9, item 4), e a leitura por nível conferia a versão mas não a variante (item 7). Numa biblioteca só de v7 o erro silencioso não tem justificativa. A recusa fica na leitura de tempo, e não na análise, porque guardar, transportar e registrar em log um identificador não depende da versão; quem quer exigir a versão na entrada chama o predicado. O erro de versão não embrulha o de formato porque o texto foi aceito. O custo medido foi de cerca de 0,6 ns por extração binária (de 1,7 para 2,3 ns), fora do caminho quente de geração e sem alocação; a verificação em uma comparação só foi medida e saiu mais lenta que as duas comparações legíveis. | Um caso real de leitura em volume de UUIDv7 já validados em que 0,6 ns por chamada pese, e ainda assim a solução seria uma leitura separada, sem mudar esta. |
| **O predicado de validade reconhece UUIDv7** — versão 7 e variante `0b10` —, e recusa o nulo e o UUID com todos os bits em um. | 2026-09-13 | Na biblioteca de origem ele aceitava as versões 1 a 8 e os valores especiais nulo e máximo, pela RFC 9562 seções 5.9 e 5.10. Numa biblioteca só de v7, um predicado que dissesse "válido" para um UUIDv4 enquanto a leitura de tempo o recusa seria contraditório, e o predicado é exatamente a condição da seção 4. O nulo continua reconhecível pelo predicado próprio. | Mudança na própria RFC. |
| **`Max`, `IsMax`, o tipo de lista `UUIDs`, `VersionString`, `VariantString` e `IsInvalidLengthError` foram removidos.** | 2026-09-13 | Nenhum está ligado a gerar ou processar UUIDv7. `Max` é valor especial da RFC para faixa genérica, e a faixa de UUIDv7 é dada por `MaxAt`. `UUIDs` e o predicado de comprimento vieram da paridade de nomes com o pacote do Google, removida; `errors.Is(err, ErrInvalidLength)` e `slices.SortFunc(lista, UUID.Compare)` cobrem os dois em uma linha. As descrições de versão e variante descreviam códigos que a biblioteca não trata mais. | Um uso de UUIDv7 que comprovadamente precise de um deles e não seja coberto pela linha equivalente da biblioteca padrão. |
| **O pacote chama-se `uuidv7`**, no módulo `github.com/patrickbrandao/go-loghub-uuidv7`. | 2026-09-13 | O nome do pacote diz o que ele gera e é o que aparece em cada chamada: `uuidv7.Generate(uuidv7.Level3)`. O caminho do módulo acompanha o repositório. | Colisão comprovada de nome de pacote em uso real que o apelido de importação não resolva. |
| **O analisador estrito devolve o erro sentinela puro**, enquanto o permissivo devolve erros embrulhados e específicos. | 2026-09-13 | Herdado da biblioteca de origem (v0.3.0), onde código existente compara o erro do analisador estrito por igualdade direta. Mantido para que o código que migra de lá continue correto sem revisão, e porque a distinção é útil: quem quer só o sentinela usa o estrito. | Uma versão maior que revise a taxonomia inteira de uma vez. |
| **O versionamento permanece em `v0.x`** até a superfície pública assentar. | 2026-09-13 | O projeto acaba de nascer com a superfície reduzida e com mudanças de assinatura em relação à biblioteca de origem. Um compromisso de estabilidade só faz sentido depois de uso real. | Uso em produção estabilizado, mais revisão da superfície pública inteira e política de compatibilidade publicada. |
| **A construção a partir de um instante — `MinAt`, `MaxAt`, `RangeAt` e `GenerateAt` — recebe o instante por parâmetro, e isso não reabre a decisão 11.2.** | 2026-09-11 | O que a 11.2 recusou foi um campo de função de relógio dentro do `Generator`, no caminho quente, como costura de teste. Aqui o instante é parâmetro de funções separadas, a geração nunca as chama e o caminho quente não ganha desvio nem indireção, então a regra da 11.1 continua satisfeita. Sem elas, a consulta por intervalo — o argumento central para adotar UUIDv7 como chave primária — exige que o chamador monte os 16 bytes à mão, e é justamente o cálculo por nível que ele erra. Os nomes `MinAt`/`MaxAt` foram escolhidos sobre `FloorAt`/`CeilAt` e `LowerBound`/`UpperBound` por serem curtos e dizerem o extremo. `RangeAt` devolve intervalo **semiaberto**, que é a forma do SQL que motiva a função. | Uma proposta de fazer a geração chamar estas funções, ou de mover o instante para dentro do `Generator`, que aí sim seria a 11.2. |
| **As fronteiras saturam nas duas pontas da faixa representável**, inclusive levando `micro` e `nano` a 999 no teto. | 2026-09-11 | Uma fronteira é predicado de consulta: o que a torna correta é nunca regredir quando o instante avança. Truncar os bits excedentes, como o empacotamento por deslocamento faria de graça, deixaria a fronteira dar a volta e a consulta devolveria as linhas erradas em silêncio. Saturar no teto é a escolha simétrica ao piso na época que a seção 3.2 já faz embaixo. Zerar `micro` e `nano` na saturação quebraria a monotonicidade na travessia da borda, e há teste para isso. | Nada previsto: a alternativa é aceitar resposta errada em silêncio. |
| **A saturação acima de 48 bits diverge, de propósito, do truncamento da RFC 9562 §6.1**, que manda manter os bits menos significativos de um carimbo grande demais. | 2026-09-13 | A regra da RFC faz um instante posterior a 10889-08-02 dar a volta para perto de 1970: numa fronteira ou numa geração por instante, isso é o erro silencioso da decisão acima, agora com o texto da RFC a favor dele. A divergência vale só para a construção a partir de um instante (seções 3.5 e 3.6); a geração pelo relógio conserva o truncamento natural do empacotamento, e o relógio não chega ao teto antes daquele ano. O ponto exato em que a saturação entra é travado por vetores (seção 10, caso 10). | Uma revisão da RFC que trate de instantes além da faixa de 48 bits. |
| **Os bits livres de `GenerateAt` são sorteados, não zerados.** | 2026-09-11 | O verbo pedido é gerar, e um gerador que devolve o mesmo valor para o mesmo instante colide na primeira repetição. A forma determinística de um instante já existe, e é a seção 3.5: expor uma segunda com nome de gerador convidaria ao mal-entendido mais caro possível. A entropia vem do mesmo gerador do resto da biblioteca, por isso as funções também existem como métodos. | Nada previsto: a alternativa determinística já está coberta por `MinAt`. |
| **A forma é `GenerateAt(nível, instante)`, e não uma família de quatro aridades.** | 2026-09-11 | A proposta original mapeava a aridade no nível: quatro funções por número de argumentos, cada uma com variante em texto, em função de pacote e em método, somando dezesseis símbolos novos. A forma escolhida usa o mesmo par nível-instante que `Generate(nível)` e `MinAt(nível, instante)` já usam, custa quatro símbolos e deixa uma única maneira de dizer nível na biblioteca inteira. | Uso real mostrando que a forma posicional por campos de tempo é necessária, e não só conveniente. |
| **Não há tipo de lista de UUIDs, nem implementação da interface de ordenação clássica.** | 2026-09-13 | A biblioteca padrão do Go ordena com uma função de comparação desde a 1.21, e o `go.mod` já está em 1.22: `slices.SortFunc(lista, UUID.Compare)` resolve em uma linha, sem alocação, reusando o `Compare` que já existe e já é testado, e a documentação de `Compare` diz isso. Um tipo próprio acrescentaria símbolos exportados para oferecer um caminho mais verboso que o que o chamador já tem. | Uma interface de terceiros que exija a interface de ordenação clássica e não aceite função de comparação. |
| **A escrita em banco continua sendo texto por padrão, e a forma binária é um tipo à parte.** | 2026-09-11 | Trocar o formato de `Value` quebraria em silêncio quem já tem texto gravado: a mesma coluna passaria a ter duas representações e nenhuma consulta acharia as linhas antigas. Tornar o formato configurável é pior ainda, porque estado global mudaria o comportamento de bibliotecas de terceiros no mesmo processo. O tipo à parte deixa a escolha explícita no ponto da consulta, sem efeito sobre quem não usa. A leitura sempre aceitou as duas formas e continua aceitando. | Nada previsto: unificar os dois caminhos é a quebra que a decisão evita. |
| **A extração completa (`ImportBinary`/`Import`) é cega quanto ao nível e a leitura por nível (`TimestampWithLevel`) descarta por faixa; as duas políticas são opostas de propósito e não serão unificadas.** | 2026-09-12 | Uma operação entrega os bits, a outra entrega um instante. Unificar tiraria do chamador a única leitura que devolve `rand_a` e o topo de `rand_b` crus, quebraria o vetor do caso 13 da seção 10 e mudaria o resultado público das duas. As duas conferem a versão antes (decisão acima); a cegueira é só quanto ao nível. | Uma terceira operação que receba o nível e devolva os campos crus com um indicador de validade, sem alterar as duas existentes. |
| **Os campos abaixo do milissegundo são contagens decimais de 0 a 999, e não a fração binária do Método 3 da RFC 9562 §6.2**; o Nível 3 usa 22 bits de tempo, além dos 12 que o método prevê. | 2026-09-13 | A codificação decimal é o que dá sentido às duas leituras da seção 7: a extração devolve os campos direto como microssegundos e nanossegundos, e a leitura por nível reconhece ruído porque um valor acima de 999 não pode ser tempo. Com a fração binária, os 4096 valores de `rand_a` seriam todos válidos, e nenhum valor denunciaria um identificador de Nível 1 lido como Nível 2. O Método 3 é opcional na RFC, que recomenda tratar o identificador como opaco, e o resultado continua sendo UUIDv7 válido e ordenável. A seção 5.7 da RFC admite preencher os 74 bits com subcampos para ordenar dentro do milissegundo, na ordem fração de até 12 bits, contador e aleatório; os 10 bits de nanossegundos do Nível 3 ocupam os bits altos de `rand_b`, o lugar que os métodos 1 e 2 dão ao contador, com a mesma função de ordenar, mas fora da letra da RFC, que não prevê tempo ali. O custo é de interoperabilidade: um leitor de terceiros que siga o Método 3 lê errado o sub-milissegundo (456 µs viram cerca de 111 µs), e a resolução de `rand_a` fica no microssegundo, e não em cerca de 244 ns. A codificação vem da biblioteca de origem e está nos vetores dourados (seção 10, caso 12): mudá-la é mudança de formato de dados. | Um nível novo com a fração binária, acrescentado sem alterar os três existentes; nunca a troca da codificação de um nível publicado. |
| **O gerador sem fonte de entropia (valor zero, ou referência nula) recorre à fonte padrão do pacote; fonte nula ou leitor nulo na construção entram em pânico.** | 2026-09-12 | A regra crítica da seção 5.2 manda falhar alto quando a **fonte falha**, e o valor zero era a única exceção não escrita: um auditor que a comparasse com a regra tinha fundamento textual para propor o pânico. Os três casos foram separados na seção 5.2. No valor zero não há degradação, porque a fonte padrão é a mesma que a construção correta instalaria; derrubar o processo em produção por um campo não inicializado penalizaria o chamador por um erro que não afeta a qualidade dos bits. A tolerância vale para o tipo inteiro e é travada por teste na geração pelo relógio, nos nomes e na geração por instante. | Uma versão maior que aceite quebra de compatibilidade e um caso real em que o silêncio tenha escondido um erro de configuração. |
| **O prefixo URN inválido devolve o sentinela de formato puro, e não um erro próprio, ao contrário das chaves malformadas.** | 2026-09-12 | A assimetria não tem motivo técnico: veio com a primeira versão do analisador permissivo e nunca foi registrada. Criar o erro de prefixo é acréscimo de símbolo público e muda o valor devolvido para uma entrada que hoje recebe o sentinela puro, também na biblioteca de origem; quem verifica pelo sentinela não quebraria, quem compara por igualdade direta, sim. A escolha foi documentar a assimetria na seção 6.4 e travá-la no caso 17 da seção 10. | Um chamador real que precise distinguir prefixo inválido de dígito inválido, ou a próxima versão maior, quando a taxonomia inteira for revista de uma vez. |
| **Os dois anexadores em buffer do chamador, o de texto e o de binário, andam juntos; o binário não ganha uma segunda forma sem erro.** | 2026-09-12 | O motivo de existir o anexador de texto é serializar em volume sem alocar, e quem grava em coluna binária é justamente quem grava em volume. Em linguagens com as duas interfaces de anexação, satisfazer só a de texto deixa o tipo pela metade em consumidor genérico. O lado binário fica com uma única forma, a da interface, porque `append(dst, u[:]...)` é literalmente o corpo dela: `AppendTo` existe no texto porque a formatação canônica é um cálculo, e os bytes não são. O conteúdo coincide com a serialização binária, então nada de gravado muda. | Uma terceira interface de anexação na biblioteca padrão da linguagem. |
| **O UUIDv7 existe também pelo nome: por versão, `GenerateV7`, e por nível, `GenerateV7Level1`, `GenerateV7Level2` e `GenerateV7Level3`, como métodos do gerador e funções de pacote, sem parâmetro de nível e sem forma em texto.** | 2026-09-13 | Criados na biblioteca de origem em 2026-09-12 para acompanhar os nomes por versão das demais; aquele motivo saiu com as outras versões, e a permanência foi decidida de novo na criação deste projeto, por motivos que não dependem delas. O nome por versão designa o UUIDv7 padrão da RFC, sem a extensão de precisão, e é o que procura quem não conhece os níveis. Os nomes por nível deixam o nível, que é contrato de toda a coluna (seção 3.5), legível no ponto da chamada em vez de numa constante; `GenerateV7Level1` repete `GenerateV7` de propósito, para a família ser completa. Os apelidos são embutidos pelo compilador e não tocam o caminho quente, porque `Generate` continua sendo a única implementação; o benchmark de cada nome iguala o do nível. A constante continua sendo a única forma de dizer o nível **como valor**. Isto não reabre a decisão contra a família de aridades de `GenerateAt`: lá o nível ia na contagem de argumentos, aqui está escrito no nome, sem forma em texto nem variante por instante. | Uma forma em texto, só se for acrescentada aos quatro nomes de uma vez; um parâmetro de nível em `GenerateV7`, nunca, porque aí o nome deixaria de designar o UUIDv7 da RFC; um quarto nome por nível, só com um quarto nível. |
| **A recusa de uma desserialização não altera o receptor; a leitura de valor de banco do tipo anulável é a única exceção, e derruba o booleano de presença.** | 2026-09-12 | A regra de 6.4 é a convenção da linguagem e a única defensável nos desserializadores: uma entrada recusada não é informação sobre o valor que o receptor já tinha. A exceção do banco tem motivo próprio e não é uniformizável: `database/sql` reaproveita o mesmo destino a cada linha, então quem ignore o erro leria o valor da linha anterior como se fosse o da que falhou; derrubar o booleano transforma o descuido em ausência de valor, e não em dado errado. | Uma linguagem cuja interface de leitura de banco não reaproveite o destino entre linhas retiraria o motivo da exceção. |
| **O tipo de escrita binária é adaptador de gravação, não um segundo tipo de identificador: ele não ganha os anexadores em buffer nem os métodos de inspeção e comparação.** | 2026-09-12 | A regra dos anexadores da seção 7 diz que **quem oferece um deve oferecer o outro**, e o adaptador não oferece nenhum dos dois, de modo que a cumpre; a paridade exigida na seção 8 enumera exatamente os quatro serializadores, com motivo declarado, e o adaptador os tem todos. O tipo anulável simples está na mesma posição. A forma prevista é a conversão no ponto da consulta, `BinaryUUID(u)`, e ampliar a superfície pública criaria um segundo tipo a especificar, testar e versionar em troca de evitar uma conversão que já é explícita de propósito. | Uma interface da biblioteca padrão da linguagem que o adaptador precise satisfazer **como adaptador de gravação**, e não por semelhança com o tipo base. |
| **`Nil` é variável exportada, sem função de acesso e sem defesa interna contra alteração; o contrato é a documentação pedir que não seja alterada.** | 2026-09-13 | A linguagem não tem constante de vetor, e a convenção dela para variável exportada que não deve mudar é dizer isso na documentação. Blindar os predicados contra uma cópia privada não protege: a biblioteca atribui o nulo em cada recusa de análise e de leitura de banco, e a comparação que o chamador escreve contra a variável fica fora do alcance de qualquer implementação. Um acessor seria um segundo nome para o mesmo valor sem proteger quem usa o primeiro. Alterar a variável é defeito de quem altera. | Uma linguagem-alvo com constante de vetor, em que o valor imutável sai de graça, ou um caso real de corrupção vindo de um uso comum, e não de atribuição deliberada. |

### 11.4 Como registrar uma decisão nova

Toda decisão de projeto — inclusive a recusa de uma proposta — entra
**nesta seção** e no histórico de mudanças, com a data e o motivo. Uma
proposta recusada sem registro volta na auditoria seguinte, e o custo de
reavaliá-la é pago de novo.
