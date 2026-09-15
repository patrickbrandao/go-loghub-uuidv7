# 07. Catálogo de armadilhas evitadas

Guia anti-regressão: os doze defeitos reais encontrados e corrigidos ao
longo da história do projeto e da biblioteca de origem. Toda
reimplementação **DEVE** garantir proteção contra cada um deles. A
numeração é estável: código, testes e os demais arquivos desta pasta
citam as armadilhas pelo número.

| # | Armadilha histórica | Consequência | Solução obrigatória |
|---|:---|:---|:---|
| 1 | **Parser pulando hífens em laço** | Entrada maliciosa com hífen extra causava pânico e queda do processo por `index out of range` | Decodificar exclusivamente por tabela fixa de 16 posições (`hexOffsets`); [05-conversao-e-analise.md](05-conversao-e-analise.md) §3 |
| 2 | **Relógio anterior a 1970** | Módulo de números negativos corrompia `rand_a` e `rand_b` | Fixar piso em 0 para `unix_ts_ms`, microssegundos e nanossegundos se `sec < 0`, antes de qualquer conversão sem sinal; [02-instante-e-entropia.md](02-instante-e-entropia.md) §1 |
| 3 | **Inteiro de 64 bits para nanossegundos** | `UnixNano()` estoura em 2262-04-11 | Ler segundos e nanossegundos em duas partes separadas; [02-instante-e-entropia.md](02-instante-e-entropia.md) §1 |
| 4 | **Leitura de tempo cega quanto à versão** | A extração completa lia qualquer UUID como UUIDv7: um UUIDv4 vindo de fora devolvia uma data aleatória entre 1970 e 10889, sem erro | Conferir versão e variante antes de interpretar qualquer bit, e recusar com erro de versão próprio; [03-leitura-do-instante.md](03-leitura-do-instante.md) §2 |
| 5 | **Fallback de entropia no relógio** | Falha de `crypto/rand` degradava silenciosamente para o relógio, gerando colisões | Falhar imediatamente com pânico; proibido degradar em silêncio; [02-instante-e-entropia.md](02-instante-e-entropia.md) §6 |
| 6 | **Escrita em variável global em teste concorrente** | O detector de corrida (`-race`) disparava falso alerta em benchmarks | Usar `runtime.KeepAlive(u)` por goroutine em vez de escrever em variável compartilhada |
| 7 | **Recusa só pela versão** | A leitura por nível conferia o nibble de versão e ignorava a variante: um identificador com nibble `7` e variante `0b11`, que não é UUIDv7, era lido como tal | O predicado de validade exige as duas condições, e a varredura das 64 combinações de versão e variante o trava; [08-casos-de-teste.md](08-casos-de-teste.md), caso 7 |
| 8 | **Desperdício de entropia no v7** | Sortear duas palavras de 64 bits nos níveis 2 e 3 | Sortear apenas uma palavra nos níveis 2 e 3 (economia de 17% em concorrência); [02-instante-e-entropia.md](02-instante-e-entropia.md) §3 |
| 9 | **Serialização JSON como vetor** | `[1, 146, 247, ...]` em vez de `"0192f7c5-..."` quebrava a interoperabilidade | Implementar `MarshalText`/`MarshalBinary` canônicos; [06-serializacao-e-banco.md](06-serializacao-e-banco.md) §1 |
| 10 | **Tags sobrescritas com `-f`** | Quebrava a verificação de integridade no registro público (`sum.golang.org`) | Tags publicadas são estritamente imutáveis; nunca mover com `-f`; [11-release.md](11-release.md) §4 |
| 11 | **Estouro de `sec * 1000` com instante fora da faixa** | Com o instante vindo por parâmetro, o produto estoura o inteiro com sinal e troca de sinal: data remota no futuro cai no piso da época, data remota no passado vira carimbo enorme. A guarda de `sec < 0` não dispara, porque o sinal já foi invertido | Decidir a saturação sobre os **segundos**, antes da multiplicação; [02-instante-e-entropia.md](02-instante-e-entropia.md) §1 e [04-construcao-por-instante.md](04-construcao-por-instante.md) §2 |
| 12 | **Fronteira de intervalo truncando em vez de saturar** | O empacotamento por deslocamento descarta os bits acima de 48 de graça: a fronteira dá a volta e a consulta por faixa devolve as linhas erradas **em silêncio**. Zerar `micro` e `nano` na saturação tem o mesmo efeito na travessia da borda | Saturar nas duas pontas, levando `micro` e `nano` a 999 no teto; a fronteira nunca pode regredir quando o instante avança; [04-construcao-por-instante.md](04-construcao-por-instante.md) §2 |

**Além das doze.** A campanha de mutação de 2026-09-13 não encontrou
defeitos no código, mas mostrou 34 defeitos hipotéticos, injetados um a
um, passando por uma suíte com 100% de cobertura de instruções. Os
testes que os detectam estão descritos em
[08-casos-de-teste.md](08-casos-de-teste.md), e a lição está em
[09-testes-e-benchmark.md](09-testes-e-benchmark.md) §1: cobertura total
é necessária, não suficiente.
