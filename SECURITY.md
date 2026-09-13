# Política de segurança

## Versões suportadas

Só a última versão etiquetada recebe correções. Ao relatar um problema,
confirme antes que ele se reproduz na versão mais recente listada em
`https://github.com/patrickbrandao/go-loghub-uuidv7/tags`.

| Versão          | Suportada |
| --------------- | --------- |
| última tag      | sim       |
| tags anteriores | não       |

## Como relatar uma vulnerabilidade

Não abra uma issue pública para um problema de segurança. Use o relato
privado do GitHub, disponível na aba "Security" do repositório, opção
"Report a vulnerability":

`https://github.com/patrickbrandao/go-loghub-uuidv7/security/advisories/new`

Inclua a versão afetada, os passos para reproduzir e, se possível, uma
avaliação do impacto. A resposta inicial chega em até sete dias; a
correção, quando confirmada, é publicada como versão nova com a entrada
correspondente no `CHANGELOG.md`, e o relato é creditado no aviso, salvo
pedido em contrário.

## O que a biblioteca garante e o que não garante

Esta biblioteca gera identificadores para registros, chaves primárias e
correlação de logs. Os pontos abaixo são propriedades do projeto, já
documentadas no `README.md` e em `docs/DEPLOY-FULL.md`, e não são
considerados vulnerabilidades:

- **O gerador padrão não serve para segredos.** `NewGenerator` e as
  funções de pacote `Generate`, `GenerateString`, `GenerateAt`,
  `GenerateV7` e `GenerateV7Level1` a `GenerateV7Level3` leem do gerador
  do runtime do Go (ChaCha8 por thread). É resistente a predição, mas a
  documentação do Go recomenda `crypto/rand` para uso sensível a
  segurança e a biblioteca não promete força criptográfica nessa fonte.
  Para identificadores que precisem ser inadivinháveis (token de sessão,
  link privado, chave de recuperação), use `NewCryptoGenerator` ou
  `NewGeneratorWithReader` com `crypto/rand`.
- **Todo UUIDv7 expõe o instante de criação**, com precisão de
  milissegundo no Nível 1 e até nanossegundo no Nível 3. Isso é a
  função do formato, não um vazamento.
- **A análise de texto confere só a forma.** `FromString`, `Parse`,
  `Scan` e os desserializadores aceitam um UUID bem formado de qualquer
  versão; só as leituras de tempo o recusam, com `ErrNotV7`. Quem
  precisa garantir que a entrada é UUIDv7 chama `IsValid`.

Relatos bem-vindos: pânico ou leitura fora dos limites a partir de
entrada externa nos analisadores (`FromString`, `Parse`, `Scan`,
`UnmarshalJSON` e afins) ou nas leituras de tempo (`Import`,
`ImportBinary`, `Timestamp`, `TimestampWithLevel`), leitura de tempo que
aceite um UUID que não é UUIDv7, repetição de identificador em condições
que a documentação promete únicas, corrida de dados no uso concorrente
documentado, e qualquer desvio do layout de bits da RFC 9562.
