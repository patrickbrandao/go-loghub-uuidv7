# 02. Instante, entropia e ordenação

Como o instante lido do relógio vira os campos de tempo do UUIDv7 sem
cair nas armadilhas de aritmética de relógio, quantas palavras de
entropia cada nível sorteia e de onde vem cada bit, que garantia de
ordenação a biblioteca dá (e qual não dá), e a arquitetura de entropia:
o gerador padrão, a política de falha rápida e as fontes criptográficas.

O layout dos campos está em [01-escopo-e-layout.md](01-escopo-e-layout.md)
§6. A construção a partir de um instante vindo por parâmetro, que tem
regras próprias de saturação, está em
[04-construcao-por-instante.md](04-construcao-por-instante.md).

---

## 1. Aritmética temporal segura

A decomposição do instante em milissegundos, microssegundos e
nanossegundos **DEVE** obedecer a estas regras:

1. **Prevenção de estouro em 2038 e 2262.** **NUNCA** leia o relógio como
   um único inteiro de 64 bits em nanossegundos (`UnixNano()`): esse
   valor transborda em **2262-04-11**. Obtenha o tempo em **duas partes
   separadas**: segundos Unix inteiros de 64 bits (`sec`) e nanossegundos
   residuais dentro do segundo (`0 <= nsec < 1.000.000.000`).
2. **Proteção contra instantes anteriores a 1970.** Com o relógio
   configurado antes de 1970-01-01 (`sec < 0`), a divisão e o módulo de
   várias linguagens produzem restos negativos e corrompem os bytes do
   UUID. **Regra**: se `sec < 0`, fixe `unix_ts_ms = 0`, `micro = 0` e
   `nano = 0`. Um relógio quebrado ou pré-época não pode gerar campos
   fora da faixa `0..999`.
   - **A verificação de sinal vem antes de qualquer conversão para tipo
     sem sinal.** Em aritmética sem sinal o valor negativo não existe:
     vira um número enorme, a guarda nunca dispara e o carimbo sai
     grande e errado, sem erro nenhum. É a mesma armadilha do item 11 de
     [07-armadilhas.md](07-armadilhas.md), pelo mesmo mecanismo. Em C e
     Rust, onde a aritmética sem sinal é o caminho natural, este é o
     ponto exato em que uma transcrição literal da fórmula do item 4
     perde a proteção.
   - **Forma equivalente aceita.** Decidir o piso sobre `unix_ts_ms` já
     calculado, em aritmética **com** sinal, é equivalente e conforme:
     como o item 1 garante `0 <= nsec < 1.000.000.000`, qualquer `sec`
     negativo produz `unix_ts_ms` negativo, e o piso dispara igual. As
     duas formas coexistem na implementação de referência, uma no
     caminho quente e outra no lado do parâmetro, e a diferença entre
     elas é de custo, não de resultado.
3. **Proteção contra instantes muito além da faixa.** A decomposição
   multiplica os segundos por mil. Vindo do relógio do sistema isso é
   inofensivo, mas vindo **por parâmetro** o tipo de data da linguagem
   comporta anos muito além dos 48 bits de milissegundos, e o produto
   **estoura o inteiro com sinal de 64 bits**, trocando de sinal: um
   instante remoto no futuro cai no piso da época e um remoto no passado
   vira carimbo enorme, e a guarda do item 2 não dispara porque o sinal
   já foi invertido. **Regra**: decida a saturação sobre os **segundos**,
   antes da multiplicação, **nas duas pontas**. É obrigatória em toda
   construção a partir de um instante
   ([04-construcao-por-instante.md](04-construcao-por-instante.md) §2).
4. **Decomposição pura**, em aritmética **com sinal**, convertida para os
   campos sem sinal **somente depois** do piso do item 2 e da saturação
   do item 3:
   - `unix_ts_ms = (sec * 1000) + (nsec / 1_000_000)`, gravado nos 48
     bits do campo apenas após as guardas;
   - `sub_ms = nsec % 1_000_000`;
   - `micro = sub_ms / 1000` (0..999; campo de 12 bits);
   - `nano = sub_ms % 1000` (0..999; campo de 10 bits).

Na implementação de referência a decomposição do caminho quente é uma
função pura (`splitUnixInstant`), testada de dentro do pacote; é o que
dispensa um relógio injetável ([10-decisoes.md](10-decisoes.md) §2).

## 2. O relógio do sistema

A geração pelo relógio lê o relógio de parede a cada chamada, em duas
partes (item 1 acima). O campo de 48 bits comporta datas até
10889-08-02; o caminho quente conserva o truncamento natural do
empacotamento acima disso, porque o relógio não chega perto do teto.

**Relógio atrasado.** Se o relógio for ajustado para trás (ajuste manual
ou salto de NTP), os UUIDs seguintes ficam lexicograficamente antes dos
anteriores até o relógio alcançar o instante antigo. A RFC 9562 permite
esse comportamento, e a biblioteca **não** tenta compensá-lo: isso
exigiria estado compartilhado entre threads no caminho quente (seção 4).

**Resolução do relógio.** O passo do relógio do host limita a precisão
que o Nível 3 grava de fato; ver [01-escopo-e-layout.md](01-escopo-e-layout.md)
§9.

---

## 3. Sorteio de entropia por nível

- **Níveis 2 e 3**: `rand_a` carrega os microssegundos e não é aleatório.
  A biblioteca **DEVE** sortear **exatamente uma palavra de 64 bits**
  (`r2`). Sortear uma segunda é desperdício de CPU e de entropia
  criptográfica do sistema.
- **Nível 1 (e níveis desconhecidos)**: sortear **duas palavras** (`r1` e
  `r2`), usando os 12 bits inferiores de `r1` para `rand_a`.

A economia é normativa e **DEVE** ser travada por testes de contagem
([08-casos-de-teste.md](08-casos-de-teste.md), caso 4). Ela importa para
quem fornece `crypto/rand` como fonte: cada palavra custa uma leitura.

**A correspondência entre palavras e campos também é normativa.** No
Nível 1, `r1` é a **primeira** palavra sorteada e `r2` a segunda, e
`rand_b` recebe os 62 bits inferiores de `r2`. Nos níveis 2 e 3 a única
palavra sorteada é `r2`: `rand_b` recebe os seus 62 bits inferiores no
Nível 2 e os 52 inferiores no Nível 3. O teste **DEVE** usar palavras
distintas, porque uma fonte constante não distingue `r1` de `r2`. No
gerador padrão, cuja fonte não é injetável, a independência das duas
palavras **DEVE** ser conferida estatisticamente: uma palavra repetida
faria `rand_a` coincidir sempre com os 12 bits inferiores de `rand_b` e
derrubaria o Nível 1 de 74 para 62 bits de entropia, sem sinal visível.

O mesmo consumo e a mesma correspondência valem para os nomes por versão
e por nível e para a geração por instante
([04-construcao-por-instante.md](04-construcao-por-instante.md) §4).

## 4. Ordenação e desempate, sem contador monotônico

A garantia de ordenação do UUIDv7 multinível é **exatamente esta, e não
mais que esta**:

- se o instante embutido de `B` for estritamente maior que o de `A`,
  então `B > A` tanto na comparação byte a byte quanto na comparação
  lexicográfica das strings canônicas;
- se `A` e `B` carregarem o **mesmo instante embutido**, a ordem entre
  eles é **aleatória**, decidida pelos bits de entropia. Não há contador,
  sequência nem qualquer desempate determinístico.

O instante embutido tem a resolução do nível: milissegundo no Nível 1,
microssegundo no Nível 2, nanossegundo no Nível 3. Como gerar um UUID
custa dezenas de nanossegundos, menos que o passo do relógio da maioria
dos hosts, **empates entre gerações consecutivas são o caso comum**:
universais no Nível 1 e frequentes nos demais.

**Regra normativa.** A implementação **NÃO DEVE** introduzir contador
monotônico, nem os métodos 1 ou 2 da RFC 9562 §6.2, no gerador padrão.
Ambos exigem estado compartilhado entre threads, e o custo sob
concorrência inviabiliza o objetivo de desempenho: um protótipo com
contador de 16 bits e estado atômico mediu +8,5% em série e **32 vezes**
pior em paralelo. A medição e a decisão estão em
[10-decisoes.md](10-decisoes.md) §1.

**Consequência para testes.** É proibido escrever teste de ordenação que
gere UUIDs em laço apertado e conte "regressões" contra um limite
tolerado: ele mede a resolução do relógio do host, não a biblioteca, e
falha de forma permanente em hosts com relógio de microssegundo. Teste a
invariante acima fazendo o instante avançar de verdade entre as gerações
([08-casos-de-teste.md](08-casos-de-teste.md), caso 6).

---

## 5. O gerador padrão

- O gerador é um objeto construído **uma vez**, no boot da aplicação, e
  compartilhado por todas as threads. As funções de pacote usam um
  gerador padrão interno, criado na carga do pacote, e não têm ponto de
  injeção de fonte ([10-decisoes.md](10-decisoes.md) §2).
- Para geração rápida (milhões de UUIDs por segundo), **não** use um lock
  global em torno de uma fonte compartilhada. Use uma fonte **local por
  thread**. Se a linguagem já oferecer uma no runtime, prefira-a: em Go,
  as funções de pacote de `math/rand/v2` leem de uma instância de ChaCha8
  por thread, semeada pelo sistema operacional, sem trava e sem estado a
  manter pela biblioteca.
- Se a plataforma não oferecer fonte por thread, mantenha um **pool de
  geradores locais**, cada um instanciado sob demanda e semeado **uma
  única vez** com 128 bits da fonte criptográfica forte do sistema
  operacional.
- Uma fonte de entropia fornecida pelo chamador (função que devolve 64
  bits, ou leitor) **DEVE** ser segura para uso concorrente: será chamada
  por várias threads ao mesmo tempo. Fornecer `nil` é erro de
  configuração (seção 6, caso 2).

A troca do pool de PRNGs pelo gerador do runtime mediu -34,7% em paralelo
e 2% a 6% em série; os números estão em
[09-testes-e-benchmark.md](09-testes-e-benchmark.md) §10.

## 6. Política de falha rápida, em três casos

Confundir os três casos leva a implementações que param o processo onde
não deveriam, ou que seguem onde não podem.

**Caso 1: falha da fonte durante a operação.** Se a fonte forte do
sistema falhar ao semear um gerador, ou se falhar a leitura de um gerador
construído sobre um leitor do chamador, a biblioteca **DEVE falhar alto e
imediatamente** (pânico, exceção, encerramento). **JAMAIS** recorra ao
relógio como fallback silencioso: semear vários geradores com o horário
atual produz sequências idênticas ou correlacionadas entre threads,
colisões maciças e previsibilidade total. O valor do pânico **DEVE** ser
o erro de fonte de entropia (`ErrEntropySource`), para que quem recupere
o pânico reconheça a causa. A geração propriamente dita nunca devolve
erro.

**Caso 2: configuração inválida explícita.** Uma função de construção que
receba fonte nula ou leitor nulo **DEVE** entrar em pânico na própria
construção. O erro é de configuração e precisa aparecer na carga do
programa, não na primeira geração, onde apareceria em produção como uma
falha de ponteiro nulo longe da causa.

**Caso 3: ausência de fonte por construção omitida.** Um gerador que
chegue a uma geração sem fonte, por ter sido montado fora das funções de
construção (valor zero embutido em outra estrutura, ou referência nula
usada como receptor), **DEVE** recorrer à fonte padrão do pacote em vez
de entrar em pânico. Aqui não há degradação: a fonte padrão é exatamente
a que a construção correta teria instalado. A tolerância é regra do tipo
gerador inteiro: vale para a geração pelo relógio nos três níveis, para
os nomes por versão e por nível e para a geração por instante.

A diferença entre os casos 2 e 3 é o momento e a intenção: quem pede uma
fonte inválida recebe o erro de imediato; quem simplesmente não construiu
o gerador recebe um resultado correto. Ver [10-decisoes.md](10-decisoes.md)
§3.

## 7. Fontes criptográficas dedicadas

- O gerador padrão **não promete força criptográfica**, mesmo quando a
  fonte do runtime resiste a predição. O ChaCha8 do Go é uma cifra de
  fluxo, mas a própria documentação do Go recomenda `crypto/rand` para
  uso sensível a segurança.
- Para casos que exigem imprevisibilidade (token de sessão, link privado,
  chave de recuperação), a biblioteca **DEVE** fornecer um gerador que
  use exclusivamente entropia criptográfica (`NewCryptoGenerator`) e um
  construtor sobre um leitor do chamador (`NewGeneratorWithReader`), além
  do construtor sobre uma função que devolve 64 bits
  (`NewGeneratorWith`). Para chave primária, identificador de registro e
  correlação de log, o gerador padrão é adequado.
- **Formação das palavras a partir do leitor (normativo).** Cada palavra
  de 64 bits é formada por **8 bytes em ordem de rede** (o primeiro byte
  lido é o mais significativo), e as palavras são lidas na ordem da
  seção 3: no Nível 1, os 8 primeiros bytes formam `r1` e os 8 seguintes,
  `r2`. Uma leitura **curta** sem erro, que o contrato de leitor da
  linguagem permite, **DEVE** ser completada com novas leituras até os 8
  bytes; só o fim dos dados ou um erro contam como falha, e aí vale o
  caso 1 da seção 6. Aceitar a leitura pela metade deixaria bytes zerados
  na palavra, em silêncio. A ordem é contrato porque é visível ao
  chamador: um leitor determinístico reproduz os mesmos identificadores
  ([10-decisoes.md](10-decisoes.md) §1).
- O gerador criptográfico **DEVE** ler da fonte criptográfica do sistema
  capturada na construção, e o teste **DEVE** provar a origem dos bits, e
  não só a validade do resultado: um gerador "criptográfico" que caísse
  na fonte padrão produziria UUIDv7 igualmente válidos
  ([08-casos-de-teste.md](08-casos-de-teste.md), caso 4).
- Mesmo com entropia criptográfica, **todo UUIDv7 expõe o instante de
  criação por construção**, com precisão de milissegundo no Nível 1 e
  até nanossegundo no Nível 3. É a função do formato, não um vazamento,
  e a documentação **DEVE** dizê-lo junto do gerador criptográfico.
