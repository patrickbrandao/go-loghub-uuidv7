# Índice da especificação

Especificação da biblioteca `go-loghub-uuidv7`: UUIDv7 (RFC 9562) com
três níveis de precisão temporal, nos dois sentidos. Os arquivos são
numerados na ordem de leitura, e cada assunto vive em um único arquivo:
quando outro precisa dele, aponta em vez de repetir.

Quem quer **usar** a biblioteca em Go começa pela skill em
[../skill/SKILL.md](../skill/SKILL.md), que tem as instruções de uso e
os programas de exemplo; esta pasta é o contrato que a skill e o código
seguem.

## Convenções

- **DEVE** e **NÃO DEVE** marcam regras normativas; o resto é
  explicação, exemplo ou medição.
- Os arquivos 01 a 08 e 10 são agnósticos de linguagem: descrevem o que
  uma implementação em qualquer linguagem precisa fazer. Os nomes citados
  (`Generate`, `MinAt`, `ErrNotV7`) são os da implementação de referência
  em Go. Os arquivos 09 e 11 são guias de execução desta implementação.
- Uma referência a outro arquivo tem a forma `NN-nome.md §n`, onde `n` é
  o número da seção de segundo nível daquele arquivo. Os **casos de
  teste** (1 a 19) e as **armadilhas** (1 a 12) têm numeração estável e
  são citados pelo número, em código, testes e documentos.
- Toda decisão de projeto, inclusive a recusa de uma proposta, é
  registrada em [10-decisoes.md](10-decisoes.md) e no `CHANGELOG.md`.

## Arquivos

| Arquivo | O que cobre |
|:---|:---|
| [01-escopo-e-layout.md](01-escopo-e-layout.md) | Objetivo e escopo; o que ficou de fora; anatomia dos 128 bits, campos fixos de versão e variante, forma canônica em texto; a distribuição de bits em cada nível; a codificação decimal do sub-milissegundo; os nomes por versão e por nível; como escolher o nível. |
| [02-instante-e-entropia.md](02-instante-e-entropia.md) | Aritmética temporal segura (2038, 2262, pré-1970, estouro por parâmetro); comportamento com o relógio do sistema; sorteio de entropia por nível e fiação das palavras; ordenação e desempate sem contador monotônico; o gerador padrão; a política de falha rápida em três casos; as fontes criptográficas e o contrato do leitor. |
| [03-leitura-do-instante.md](03-leitura-do-instante.md) | O predicado de validade; a recusa de tudo o que não é UUIDv7 nas quatro leituras de tempo, enquanto a análise segue só de forma; a leitura por nível com descarte por faixa; a extração completa, cega quanto ao nível; inspeção e valores: `Version`, `Variant`, `IsZero`, `Nil`, `Compare`, `Bytes`, `URN`. |
| [04-construcao-por-instante.md](04-construcao-por-instante.md) | Saturação nas duas pontas da faixa representável; as fronteiras `MinAt`, `MaxAt` e `RangeAt` para consulta por intervalo, com a regra de que níveis distintos não compõem; a geração para um instante informado, `GenerateAt`; o empacotamento único; o contrato das cinco operações. |
| [05-conversao-e-analise.md](05-conversao-e-analise.md) | Formatação canônica; escrita em buffer do chamador (`AppendTo`, `AppendText`, `AppendBinary`); o analisador estrito e a regra crítica contra pânico; o analisador permissivo em quatro formatos e os auxiliares; a taxonomia normativa de erros. |
| [06-serializacao-e-banco.md](06-serializacao-e-banco.md) | Texto e JSON como string canônica; binário e `gob`; leitura e escrita em banco de dados; a escrita binária como tipo distinto (`BinaryUUID`); os tipos anuláveis e a representação da ausência em cada destino; `NULL` contra UUID nulo. |
| [07-armadilhas.md](07-armadilhas.md) | O catálogo das doze armadilhas históricas, com a consequência de cada uma e a solução obrigatória. |
| [08-casos-de-teste.md](08-casos-de-teste.md) | Os dezenove casos de teste obrigatórios, com os vetores que são contrato: o exemplo A.6 da RFC 9562, os vetores dourados da extensão multinível e os valores exatos no teto de 48 bits. |
| [09-testes-e-benchmark.md](09-testes-e-benchmark.md) | Como rodar a suíte, o detector de corrida, as travas de alocação, o fuzzing, os exemplos, o linter, a cobertura e a integração contínua; os benchmarks, a geração em massa e os resultados de referência. |
| [10-decisoes.md](10-decisoes.md) | O registro de decisões firmadas, com data, motivo e o que justificaria rever cada uma: entropia e desempenho, testabilidade, contrato público. Leia antes de propor qualquer mudança de projeto. |
| [11-release.md](11-release.md) | O procedimento de publicação de uma versão: pré-requisitos, tag e release, verificação no proxy de módulos, e por que tags nunca são movidas. |

## Caminhos de leitura

- **Vou usar a biblioteca em Go**: [../skill/SKILL.md](../skill/SKILL.md).
- **Quero entender o formato**: 01, depois 02.
- **Vou ler o instante de identificadores que recebo**: 03; para exigir
  UUIDv7 na entrada, 03 §1 e §2.
- **Vou consultar por intervalo ou importar histórico**: 04.
- **Vou reimplementar em outra linguagem**: 01 a 08 na ordem, com 10 ao
  lado; os vetores de 08 são o critério de aceitação.
- **Vou propor uma mudança, ou auditar a biblioteca**: 10 primeiro, depois
  07 e 08.
- **Vou mexer no código**: 09 para verificar, 08 para saber o que travar,
  `CLAUDE.md` na raiz para as regras de manutenção.
- **Vou publicar uma versão**: 11.

## Mapa da API

Onde cada símbolo público está especificado. As assinaturas em Go estão
em `STARTHERE.md` §3, na raiz, e em [../skill/references/api.md](../skill/references/api.md).

| Símbolos | Arquivo |
|:---|:---|
| `Level`, `Level1`, `Level2`, `Level3`, `UUID` | 01 §6 |
| `GenerateV7`, `GenerateV7Level1`, `GenerateV7Level2`, `GenerateV7Level3` | 01 §8 |
| `Generator`, `NewGenerator`, `NewGeneratorWith`, `Generate`, `GenerateString` | 02 §3, §5 e §6 |
| `NewCryptoGenerator`, `NewGeneratorWithReader`, `ErrEntropySource` | 02 §6 e §7 |
| `IsValid`, `ErrNotV7` | 03 §1 e §2 |
| `Timestamp`, `TimestampWithLevel` | 03 §3 |
| `Import`, `ImportBinary`, `Time` | 03 §4 |
| `Version`, `Variant`, `IsZero`, `Nil`, `Compare`, `Bytes`, `URN` | 03 §5 |
| `MinAt`, `MaxAt`, `RangeAt` | 04 §3 |
| `GenerateAt`, `GenerateAtString` | 04 §4 |
| `String`, `BinaryToString` | 05 §1 |
| `AppendTo`, `AppendText`, `AppendBinary` | 05 §2 |
| `FromString`, `StringToBinary`, `ErrInvalidFormat` | 05 §3 |
| `Parse`, `ParseBytes`, `Validate`, `FromBytes`, `MustParse`, `Must` | 05 §4 |
| `ErrInvalidLength`, `ErrInvalidBrackets`, `ErrInvalidScanType` | 05 §5 |
| `MarshalText`, `UnmarshalText`, `MarshalBinary`, `UnmarshalBinary` | 06 §1 e §2 |
| `Scan`, `Value`, `BinaryUUID` | 06 §3 |
| `NullUUID`, `NullBinaryUUID` | 06 §4 |

## Correspondência com a especificação anterior

Até 2026-09-14 a pasta tinha cinco arquivos: `SPEC.md` (a especificação
em onze seções), `DEPLOY-FAST.md` e `DEPLOY-FULL.md` (guias de uso),
`TEST-AND-BENCHMARK.md` e `RELEASE.md`. As entradas do `CHANGELOG.md`
anteriores a essa data citam as seções antigas; esta tabela traduz.

| Antes | Agora |
|:---|:---|
| `SPEC.md` §1, §2 e §3.1 | 01 |
| `SPEC.md` §3.2, §3.3, §3.4 e §5 | 02 |
| `SPEC.md` §4 e §7 (leituras, inspeção) | 03 |
| `SPEC.md` §3.5, §3.6 e §7 (derivação) | 04 |
| `SPEC.md` §6 | 05 |
| `SPEC.md` §8 | 06 |
| `SPEC.md` §9 | 07 (mesma numeração das armadilhas) |
| `SPEC.md` §10 | 08 (mesma numeração dos casos) |
| `SPEC.md` §11.1, §11.2, §11.3 e §11.4 | 10 §1, §2, §3 e §4 |
| `TEST-AND-BENCHMARK.md` | 09 |
| `RELEASE.md` | 11 |
| `DEPLOY-FAST.md` e `DEPLOY-FULL.md` | [../skill/SKILL.md](../skill/SKILL.md) e `skill/references/` |
