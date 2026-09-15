# 08. Casos de teste obrigatórios

O que toda implementação conforme **DEVE** provar por teste, e os vetores
que servem de contrato. A numeração dos casos (1 a 19) é estável: código,
testes e os demais arquivos desta pasta citam os casos pelo número. Como
rodar a suíte da implementação de referência, e o que cada arquivo de
teste dela cobre, está em
[09-testes-e-benchmark.md](09-testes-e-benchmark.md).

**Cobertura total é necessária, não suficiente.** Em 2026-09-13 uma
campanha de mutação dirigida injetou 45 defeitos de uma linha, um por
vez, em pontos suspeitos de uma implementação com 100% de cobertura de
instruções: 34 passavam pela suíte inteira. Os casos abaixo incluem os
testes escritos para pegá-los. Ao acrescentar ou mudar comportamento,
pergunte qual defeito de uma linha ainda passaria e trave o valor exato:
uma fonte de entropia constante esconde qual palavra alimenta qual campo,
uma ida e volta esconde um analisador que aceita demais, uma verificação
de monotonicidade esconde onde a saturação começa, uma verificação de
validade esconde de onde os bits vieram, e um nome de fuso esconde
horário local num host configurado em UTC.

---

1. **Conformidade de versão e variante**
   ([01-escopo-e-layout.md](01-escopo-e-layout.md) §4, §6 e §8):
   - Validar que cada forma de gerar (os níveis 1 a 3, o nome por versão
     `GenerateV7`, os nomes por nível `GenerateV7Level1` a
     `GenerateV7Level3`, a geração por instante e o gerador
     criptográfico) define exatamente a versão 7 e a variante `0b10`, e
     que o predicado de validade aceita o resultado.
   - Validar que o nome por versão é o Nível 1: com entropia constante,
     `rand_a` e o topo de `rand_b` saem inteiros da fonte, sem campo de
     tempo sub-milissegundo, e o carimbo é o do relógio.
   - Validar que cada nome gera no nível que declara, e não em outro:
     com entropia constante em um, o identificador gerado pelo nome é
     lido de volta no nível declarado e regerado por instante nesse mesmo
     nível, e os 16 bytes **DEVEM** coincidir. A entropia em um deixa
     `rand_a` em `0xfff` e o topo de `rand_b` em `0x3ff`, valores que
     nenhum campo de tempo em 0..999 assume, então um nome que chamasse
     outro nível diverge em pelo menos um campo. Conferir também cada
     campo: `rand_a` em `0xfff` no Nível 1 e em 0..999 nos demais; topo
     de `rand_b` em `0x3ff` nos níveis 1 e 2 e em 0..999 no Nível 3.
   - Validar o nível de **todas** as formas de gerar, inclusive as
     funções de pacote, que usam o gerador padrão e não aceitam fonte
     injetada: a geração por nível, em binário e em texto, pelo relógio e
     por instante, nos três níveis e em níveis desconhecidos (que devem
     se comportar como o Nível 1), e os quatro nomes. Sem fonte
     injetável, a prova é a assinatura estatística dos bits livres, que
     decorre do layout: no Nível 1, `rand_a` passa de 999 em 3096 de cada
     4096 amostras, e nos níveis 2 e 3 nunca passa; no Nível 2, o topo de
     `rand_b` passa de 999 em 24 de cada 1024, e no Nível 3 nunca passa.
     Com 4.000 amostras por forma, confundir o Nível 2 com o 3 tem
     probabilidade abaixo de 10^-41. Sem este caso, uma função de pacote
     que ignorasse o nível pedido produziria UUIDv7 válidos e distintos,
     e passaria por todos os outros.
   - Validar que a leitura do nibble de versão e do código de variante
     devolve o valor dos bits para **qualquer** valor (versão de 0 a 15,
     variante de 0 a 3), com os demais bits do byte em zero e em um, e
     não só o `7` e o `0b10` dos identificadores gerados.

2. **Robustez do analisador contra mutações**
   ([05-conversao-e-analise.md](05-conversao-e-analise.md) §3 e §4):
   - Executar teste cobrindo **todas as 36 × 256 mutações de um único
     byte** sobre uma string canônica válida: nenhuma mutação pode
     causar pânico.
   - Sobre as mesmas mutações, exigir **aceitação exata**: o analisador
     estrito aceita a mutação se, e somente se, ela mantém um dígito
     hexadecimal numa posição de dígito ou o hífen numa posição de hífen,
     e devolve o valor calculado por um decodificador independente da
     implementação. Conferir só a ausência de pânico deixa passar um
     analisador que aceite `G` ou `:` como dígito.
   - Repetir a varredura, com o mesmo critério, nas **quatro formas** do
     analisador permissivo: canônica, entre chaves, URN e hexadecimal
     cru. As chaves só aceitam a si mesmas; o prefixo URN aceita as
     mesmas letras em qualquer caixa e só os dois-pontos. A ida e volta
     pela forma canônica não substitui este caso: uma entrada aceita
     indevidamente volta ao mesmo valor, e por isso o fuzzing que só
     confere a ida e volta não percebe um hífen ou um dois-pontos que
     deixaram de ser conferidos.
   - Submeter o analisador a campanhas de *fuzzing* contínuo. No
     analisador permissivo, o oráculo **DEVE** exigir que toda entrada
     aceita seja, sem distinção de caixa, exatamente uma das quatro
     formas escritas a partir do valor lido.
   - **O fuzzing não para no texto.** A aritmética temporal **DEVE**
     receber campanhas próprias, por um motivo diferente: no texto o
     risco é leitura fora dos limites, e aqui é saturação, estouro de
     sinal e resto negativo. Tabela de casos escolhidos à mão não varre
     faixa, e é nas duas metades do inteiro com sinal que o estouro da
     multiplicação por mil se manifesta. São dois alvos:
     - **Construção por instante**: recebe dois instantes arbitrários e
       exige, em todos os níveis mais um nível desconhecido, versão 7 e
       variante `0b10` nas duas fronteiras, fronteira inferior nunca
       acima da superior, o valor gerado sempre dentro das fronteiras do
       próprio instante, e monotonicidade quando o segundo instante não
       é anterior ao primeiro.
     - **Leitura de tempo**: recebe 16 bytes arbitrários e exige que as
       quatro leituras nunca entrem em pânico e concordem com o predicado
       de validade ao aceitar ou recusar; aceito o UUIDv7, que a
       extração cega devolva o carimbo exato e os campos crus dentro da
       faixa dos bits, que a leitura por nível some os campos
       sub-milissegundo exatamente quando cabem em 0 a 999, com os dois
       caindo juntos no Nível 3, e que o instante lido no Nível 3 regere
       pela fronteira inferior os mesmos campos de tempo.

3. **Bordas temporais extremas**
   ([02-instante-e-entropia.md](02-instante-e-entropia.md) §1):
   - Testar instantes com data anterior a 1970 (ex.: ano 1969 e ano 1800).
   - O caso pré-1970 **DEVE** usar um segundo negativo com fração
     positiva (por exemplo `sec = -1`, `nsec = 500.000.000`): é a entrada
     que uma transcrição da fórmula com conversão sem sinal antes da
     guarda transforma em carimbo enorme, e as duas formas de piso
     aceitas devem devolver a época.
   - Testar também o último nanossegundo antes da época (`sec = -1`,
     `nsec = 999.999.999`), em que `unix_ts_ms` vale exatamente `-1`: é a
     borda do piso, e um piso escrito como `unix_ts_ms < -1` passa no
     caso anterior e só falha aqui, com os campos sub-milissegundo em 999.
   - Testar instantes além do ano 2262 (ex.: ano 2300).
   - Validar viradas de segundo (`nsec = 999_999_999`) e viradas de
     milissegundo (`sub_ms = 999_999`).

4. **Contagem e fiação da entropia**
   ([02-instante-e-entropia.md](02-instante-e-entropia.md) §3, §5 e §7):
   - Com gerador de contagem determinística, verificar que Nível 2 e
     Nível 3 consomem 1 chamada; Nível 1 consome 2 chamadas. Os nomes
     consomem o mesmo que o nível que apelidam: `GenerateV7` e
     `GenerateV7Level1` consomem 2, `GenerateV7Level2` e
     `GenerateV7Level3` consomem 1.
   - **Fiação das palavras**: com uma fonte que devolva palavras
     **distintas** em sequência, exigir, pelo relógio e por instante, que
     no Nível 1 `rand_a` venha da primeira palavra e `rand_b` da segunda,
     e que nos níveis 2 e 3 `rand_b` venha da única palavra sorteada. A
     contagem não basta, e a entropia constante também não: as duas
     passam com `rand_a` tirado da palavra errada.
   - **Leitor do chamador**: com um leitor que entregue um byte por
     leitura, exigir os valores exatos que a ordem de rede e a ordem
     `r1`, `r2` produzem, o consumo de exatamente 8 bytes por palavra e o
     pânico com o erro de fonte quando os bytes acabam.
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

5. **Vetor externo da RFC 9562 (apêndice A.6)**
   ([03-leitura-do-instante.md](03-leitura-do-instante.md) §3 e §4):
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
     da faixa, e o descarte derruba os dois campos sub-milissegundo
     também no Nível 3, onde o topo de `rand_b` (396) caberia. O vetor
     exercita o descarte num UUIDv7 que a biblioteca não produziu.
   - Os vetores da RFC **DEVEM** ser conferidos com uma ferramenta
     independente antes de transcritos (a implementação de referência
     usou o módulo `uuid` do Python: versão e variante de cada um, o
     carimbo do A.6 decodificado, o instante do A.1 casado com o A.6, e
     A.2, A.4 e B.2 recomputados a partir das entradas por nome; o B.1
     ficou de fora porque as suas entradas não podem ser recomputadas).

6. **Ordenação temporal coerente**
   ([02-instante-e-entropia.md](02-instante-e-entropia.md) §4):
   - Testar que se o instante de B for estritamente superior ao de A, a
     comparação de strings e de bytes de B é estritamente maior que a de
     A.
   - Não contar regressões em laço apertado na mesma thread: se o tempo
     não avança na resolução do host, o desempate por entropia é
     aleatório.

7. **Recusa de UUIDs que não são v7**
   ([03-leitura-do-instante.md](03-leitura-do-instante.md) §1 e §2):
   - Sobre os mesmos 16 bytes de um UUIDv7 válido, varrer as **64
     combinações** de nibble de versão (0 a 15) e código de variante (0 a
     3) e exigir que o predicado de validade aceite só a combinação
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
     valores, que os analisadores estrito e permissivo os **aceitem**: a
     recusa é da leitura de tempo, não da análise.
   - Exigir que o erro de versão não seja reconhecido como erro de
     formato.

8. **Concorrência e ausência de corridas de dados**
   ([02-instante-e-entropia.md](02-instante-e-entropia.md) §5):
   - Gerar 1.000.000 de UUIDs divididos entre centenas de threads
     simultâneas sem nenhuma colisão e sem nenhum alerta no detector de
     corridas.

9. **Independência de ordem e de repetição**:
   - A suíte deve passar com repetição (`-count 3`) e com ordem
     embaralhada (`-shuffle on`). A biblioteca não tem estado global
     mutável além do gerador padrão, e nenhum teste pode depender da
     execução de outro: um teste que só passa numa ordem esconde um
     estado compartilhado que a especificação não prevê.

10. **Fronteiras de tempo**
    ([04-construcao-por-instante.md](04-construcao-por-instante.md) §2 e §3):
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
    - Conferir o estouro da multiplicação por mil, com instantes grandes
      o bastante para provocá-lo nas duas direções.
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
      entropia nula, que usa a mesma decomposição, produz a fronteira
      inferior de cada linha, e a leitura por nível de cada fronteira
      devolve o instante truncado à resolução do nível, ou o último
      instante representável nas linhas saturadas.

11. **Geração por instante explícito**
    ([04-construcao-por-instante.md](04-construcao-por-instante.md) §4):
    - **Ida e volta com a extração**: gerar para segundos, milissegundos,
      microssegundos e nanossegundos conhecidos e conferir que a leitura
      devolve exatamente esses campos, em cada nível que os grava. É o
      teste central, porque prova a simetria que motiva a operação.
    - **Não divergência com a geração pelo relógio**: com a mesma fonte
      de entropia constante, gerar pelo relógio, ler o instante embutido
      de volta e regerar para ele. Os 16 bytes **devem** ser idênticos. É
      a trava da duplicação deliberada do empacotamento, e ela falha se
      as duas cópias se separarem.
    - **Contenção pelas fronteiras**: o valor gerado para um instante cai
      sempre dentro das fronteiras daquele instante.
    - **Não determinismo**: muitas chamadas com o mesmo instante
      devolvem valores todos distintos, e com os campos de tempo iguais.
    - Consumo de entropia por nível igual ao do caso 4.
    - Bordas: pré-1970, saturação acima da faixa e níveis desconhecidos.

12. **Vetores dourados da extensão multinível**
    ([01-escopo-e-layout.md](01-escopo-e-layout.md) §6 e §7):
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
      duas fronteiras produzem, e é assim que se obtêm sem relógio:
      `MinAt` para os bits em zero, `MaxAt` para os bits em um.

    **Grupo A**: 2026-01-01T00:00:00.123456789Z (sec=1767225600, nsec=123456789)

    | Nível | Bits livres | 16 bytes | String canônica |
    |:---|:---|:---|:---|
    | 1 | zero | `019b76daa87b70008000000000000000` | `019b76da-a87b-7000-8000-000000000000` |
    | 1 | um | `019b76daa87b7fffbfffffffffffffff` | `019b76da-a87b-7fff-bfff-ffffffffffff` |
    | 2 | zero | `019b76daa87b71c88000000000000000` | `019b76da-a87b-71c8-8000-000000000000` |
    | 2 | um | `019b76daa87b71c8bfffffffffffffff` | `019b76da-a87b-71c8-bfff-ffffffffffff` |
    | 3 | zero | `019b76daa87b71c8b150000000000000` | `019b76da-a87b-71c8-b150-000000000000` |
    | 3 | um | `019b76daa87b71c8b15fffffffffffff` | `019b76da-a87b-71c8-b15f-ffffffffffff` |

    **Grupo B**: 2026-01-01T00:00:00.000000000Z (sec=1767225600, nsec=0)

    | Nível | Bits livres | 16 bytes | String canônica |
    |:---|:---|:---|:---|
    | 1 | zero | `019b76daa80070008000000000000000` | `019b76da-a800-7000-8000-000000000000` |
    | 1 | um | `019b76daa8007fffbfffffffffffffff` | `019b76da-a800-7fff-bfff-ffffffffffff` |
    | 2 | zero | `019b76daa80070008000000000000000` | `019b76da-a800-7000-8000-000000000000` |
    | 2 | um | `019b76daa8007000bfffffffffffffff` | `019b76da-a800-7000-bfff-ffffffffffff` |
    | 3 | zero | `019b76daa80070008000000000000000` | `019b76da-a800-7000-8000-000000000000` |
    | 3 | um | `019b76daa8007000800fffffffffffff` | `019b76da-a800-7000-800f-ffffffffffff` |

    **Grupo C**: 1970-01-01T00:00:00.000000000Z (sec=0, nsec=0)

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
      torna as fronteiras limites corretos.
    - **Grupo B contra o grupo A**: com o sub-milissegundo zerado, os
      níveis 2 e 3 devolvem `rand_a` em zero mesmo com os bits livres em
      um, enquanto o nível 1 devolve `0xfff`. É a prova de que os níveis
      2 e 3 **não** sorteiam `rand_a`, e pega erro de sinal e de
      deslocamento que um instante com todos os campos preenchidos
      esconderia.
    - **Grupo C**: a época Unix com os 48 bits de carimbo zerados. Um
      instante **anterior** a 1970 tem de produzir exatamente estes
      mesmos valores, pelo piso. Esse é o par que prova o piso, e por
      isso a suíte confere o grupo C duas vezes, uma com a época e outra
      com `1969-12-31T23:59:59.999999999Z`.

    Os valores publicados aqui foram calculados por uma implementação
    independente, escrita a partir das regras de layout, decomposição e
    fronteiras, e só então conferidos contra a implementação de
    referência. **Nunca preencha um vetor com a saída da própria
    implementação**: um vetor produzido por ela e conferido contra ela
    mesma não prova nada. O mesmo vale para os vetores do caso 4 (fonte
    em sequência e leitor) e para a tabela do caso 10.

13. **Extração cega de nível**
    ([03-leitura-do-instante.md](03-leitura-do-instante.md) §4):
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

14. **Descarte por faixa na leitura por nível**
    ([03-leitura-do-instante.md](03-leitura-do-instante.md) §3):
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

15. **Política de entropia em três casos**
    ([02-instante-e-entropia.md](02-instante-e-entropia.md) §6):
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

16. **Travas de alocação**
    ([01-escopo-e-layout.md](01-escopo-e-layout.md) §1):
    - O objetivo de zero alocação de heap no caminho quente é
      **verificável e obrigatório**, não aspiracional. Estas operações
      **DEVEM** ser livres de alocação, medidas em laço com contagem de
      alocações por iteração: geração binária pelo relógio nos três
      níveis, pelo nome por versão `GenerateV7` e pelos nomes por nível
      `GenerateV7Level1` a `GenerateV7Level3`; geração binária por
      instante; as duas fronteiras e o intervalo; análise estrita;
      análise permissiva nos quatro formatos, em texto e em bytes; as
      quatro leituras de tempo e o predicado de validade, **tanto
      aceitando quanto recusando**, porque a recusa não pode passar a
      montar um erro por chamada; escrita da forma canônica em buffer do
      chamador com capacidade sobrando; e escrita dos 16 bytes em buffer
      do chamador com capacidade sobrando.
    - **Exceção única.** A conversão que devolve uma string, pelo relógio
      ou por instante, pode alocar **exatamente uma vez**, porque o
      resultado é a alocação. Exigir zero aqui é impossível sem mudar a
      assinatura, e quem precisa de zero usa a escrita em buffer do
      chamador.
    - **Condição de medição.** A contagem de referência é obtida **sem**
      detector de corrida. O detector altera o código gerado e pode
      contar alocações que não existem em produção; a integração
      contínua da implementação de referência repete as travas em um
      passo próprio, sem o detector
      ([09-testes-e-benchmark.md](09-testes-e-benchmark.md) §2).
    - **Proibição.** Uma falha nestas travas **NÃO DEVE** ser resolvida
      elevando o limite tolerado. A trava existe para denunciar
      regressão de desempenho introduzida por refatoração, e relaxá-la
      remove a única defesa contra ela ([10-decisoes.md](10-decisoes.md)
      §1).
    - Em linguagens cujo tipo do identificador só existe no heap, o
      limite passa a ser uma alocação por operação, a do próprio
      resultado, e a divergência **DEVE** ser registrada junto da trava.

17. **Taxonomia de erros**
    ([05-conversao-e-analise.md](05-conversao-e-analise.md) §5):
    - Para cada linha da tabela, submeter a entrada descrita e exigir o
      erro descrito, reconhecido pelo tipo e não só pela presença.
      Entradas mínimas: comprimento errado (`abc`); 38 caracteres com as
      chaves trocadas por outro caractere; 45 caracteres com o prefixo
      `urn:uiid:`; dígito inválido na forma canônica; hífen fora de
      lugar; 15 bytes na construção binária e na desserialização
      binária; um número no lugar da string no JSON do tipo anulável; um
      tipo estranho na leitura de banco; um UUIDv4 bem formado na
      extração de tempo, binária e em texto.
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
      sentinela puro. Ao escrever a entrada de teste, confira os bytes
      gravados no arquivo: algumas ferramentas de edição decodificam o
      escape no próprio caractere, e o teste deixa de alcançar o caminho
      do escape em silêncio.
    - Exigir a recusa com erro de comprimento de **todo** comprimento de
      0 a 64 bytes diferente de 16 na construção binária e nas
      desserializações binárias dos quatro tipos, com o receptor intacto
      (nos anuláveis, o comprimento zero é ausência de valor). Testar só
      15 bytes deixa passar a aceitação silenciosa de 17 ou mais, com o
      excedente descartado.

18. **Ausência de valor por formato**
    ([06-serializacao-e-banco.md](06-serializacao-e-banco.md) §4):
    - Serializar o tipo anulável sem valor para banco, JSON, texto e
      binário e exigir, respectivamente, `NULL`, o literal `null`, a
      sequência vazia e a sequência vazia; desserializar a entrada vazia
      nos quatro e exigir ausência sem erro. Desserializar entrada
      inválida em JSON, texto e binário, a partir de um receptor que
      **já continha valor presente**, e exigir erro com o receptor
      inteiro intacto; fazer a mesma leitura recusada pela via de banco e
      exigir erro com o identificador intacto e o booleano falso, que é a
      exceção registrada.

19. **Coerência da API de apoio**
    ([03-leitura-do-instante.md](03-leitura-do-instante.md) §3 e §5,
    [05-conversao-e-analise.md](05-conversao-e-analise.md) §4,
    [06-serializacao-e-banco.md](06-serializacao-e-banco.md) §2 e §3):
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
