# 06. Serialização e banco de dados

Como o identificador viaja em texto, JSON e binário, como entra e sai de
um banco de dados em coluna textual ou binária de 16 bytes, e como os
tipos anuláveis representam a ausência de valor em cada destino.

Toda leitura descrita aqui confere **só a forma**, como os analisadores
de [05-conversao-e-analise.md](05-conversao-e-analise.md): um UUID bem
formado de outra versão é aceito, e a recusa fica nas leituras de tempo
([03-leitura-do-instante.md](03-leitura-do-instante.md) §2). Os erros
devolvidos são os da taxonomia de
[05-conversao-e-analise.md](05-conversao-e-analise.md) §5.

---

## 1. Texto e JSON

- Um UUID serializado em JSON **DEVE ser formatado como string canônica
  entre aspas** (`"0192f7c5-1a2b-7c3d-8e4f-aabbccddeeff"`). **NUNCA**
  serialize como um vetor de 16 números inteiros (armadilha 9 de
  [07-armadilhas.md](07-armadilhas.md)).
- Em Go isso é obtido pelas interfaces `encoding.TextMarshaler` e
  `encoding.TextUnmarshaler`: `MarshalText` devolve os 36 bytes
  canônicos, e é ela que faz `encoding/json` gravar o UUID como string.
  Sem ela o pacote gravaria uma lista de 16 números, porque o tipo é um
  vetor de bytes.
- `UnmarshalText` aceita os mesmos quatro formatos do analisador
  permissivo. Em caso de erro o receptor não é alterado.

## 2. Binário

- `MarshalBinary` devolve os 16 bytes em ordem de rede; `UnmarshalBinary`
  lê exatamente 16 bytes e devolve comprimento inválido para qualquer
  outro tamanho, com o receptor intacto. A recusa **DEVE** valer para
  **todo** comprimento diferente de 16, e não só para 15: testar um
  comprimento deixa passar a aceitação silenciosa de 17 ou mais com o
  excedente descartado ([08-casos-de-teste.md](08-casos-de-teste.md),
  caso 17).
- A serialização binária nativa da linguagem (`encoding/gob` em Go) usa
  `MarshalBinary` e grava os 16 bytes; a ida e volta dos quatro tipos
  dentro de uma estrutura é o caso 19 de
  [08-casos-de-teste.md](08-casos-de-teste.md).
- `AppendBinary` escreve os mesmos 16 bytes em buffer do chamador
  ([05-conversao-e-analise.md](05-conversao-e-analise.md) §2).

---

## 3. Banco de dados

**Leitura (`Scan`).** A leitura **DEVE aceitar as duas formas**:

| Valor vindo do banco | Resultado |
|:---|:---|
| `NULL` | o UUID nulo, sem erro |
| texto, em qualquer formato do analisador permissivo | o valor lido |
| texto vazio | o UUID nulo, sem erro: equivale a ausência |
| 16 bytes crus | o valor binário |
| sequência de bytes vazia | o UUID nulo, sem erro |
| bytes em outro tamanho | tratados como texto, pelo analisador permissivo |
| qualquer outro tipo | tipo não suportado (`ErrInvalidScanType`) |

Texto vazio equivale a ausência de valor porque colunas de texto cujo
valor padrão é a string vazia (`DEFAULT ''`) não devem falhar na leitura.
Em caso de erro o receptor não é alterado.

**Escrita (`Value`).** A escrita padrão **DEVE ser a string canônica**,
e esse formato é estável: trocá-lo deixaria duas representações na mesma
coluna, e as linhas antigas parariam de casar com as consultas. Tornar o
formato configurável seria pior, porque estado global mudaria o
comportamento de bibliotecas de terceiros no mesmo processo
([10-decisoes.md](10-decisoes.md) §3).

**Escrita binária como tipo distinto.** A escrita de 16 bytes crus
**DEVE existir como tipo distinto** (`BinaryUUID`), escolhido por
conversão no ponto da consulta (`BinaryUUID(u)`), nunca por configuração
global:

- Entrega os 16 bytes em ordem de rede, a mesma de `MarshalBinary`,
  **sem rotacionar campos**: a rotação que algumas receitas de MySQL
  sugerem para melhorar a localidade do índice é desnecessária no
  UUIDv7, que já nasce ordenado, e produziria um valor ilegível para
  outras ferramentas.
- **DEVE implementar as mesmas interfaces de serialização** que o tipo
  padrão (`MarshalText`, `UnmarshalText`, `MarshalBinary`,
  `UnmarshalBinary`), delegando para a implementação do tipo base. Sem
  esses métodos, `encoding/json` serializaria o valor como vetor de 16
  números, violando a seção 1. Tem também a forma canônica em `String`,
  para aparecer legível em mensagens de log.
- A leitura delega à do tipo base e continua aceitando texto e binário:
  ler por `BinaryUUID` não exige que a coluna seja binária. O tipo existe
  para a escrita.
- `Value` aloca a fatia devolvida, porque o driver pode reter o valor
  depois do retorno. É uma alocação por parâmetro de consulta, não por
  identificador gerado.
- É um **adaptador de gravação**, não um segundo tipo de identificador:
  não ganha os anexadores em buffer nem os métodos de inspeção e
  comparação ([10-decisoes.md](10-decisoes.md) §3).

Vale a pena onde não há tipo nativo de UUID: `BINARY(16)` no MySQL e
MariaDB, `BLOB` no SQLite. Trinta e seis bytes de texto contra dezesseis
de binário é mais que o dobro por linha, replicado em todo índice
secundário que referencie a chave. No PostgreSQL o tipo `uuid` é nativo,
o driver converte o texto e não há ganho. A consulta por intervalo
funciona igual na coluna binária, com as fronteiras convertidas do mesmo
jeito.

## 4. Tipos anuláveis e ausência de valor

A biblioteca **DEVE** fornecer um tipo `NullUUID` contendo o UUID e um
booleano `Valid`, para colunas que permitem `NULL`, e o equivalente para
a escrita binária (`NullBinaryUUID`). O anulável binário **DEVE
implementar as mesmas interfaces de serialização** que o anulável padrão
(`MarshalJSON`, `UnmarshalJSON`, `MarshalText`, `UnmarshalText`,
`MarshalBinary`, `UnmarshalBinary`), para que a tabela abaixo se aplique
igualmente a ambos.

**Ausência de valor e UUID nulo são valores distintos** e **NÃO DEVEM**
colapsar um no outro. Com o booleano falso a escrita produz `NULL`;
dezesseis bytes zerados (ou a string canônica do nulo) só saem com o
booleano verdadeiro e o UUID igual a `Nil`. Uma coluna que misture os
dois casos não consegue mais distinguir "não havia valor" de "o valor era
o UUID nulo".

A representação da **ausência** depende do destino, e não se deduz de uma
regra só: cada formato usa a convenção própria de "nada aqui".

| Destino | Ausência produz | Leitura de entrada vazia |
|:---|:---|:---|
| Banco de dados | `NULL` | ausência, sem erro |
| JSON | o literal `null`, sem aspas | ausência, sem erro |
| Texto | sequência vazia (zero bytes) | ausência, sem erro |
| Binário | sequência vazia (zero bytes) | ausência, sem erro |

Na leitura, a entrada vazia em qualquer dos quatro produz ausência sem
erro, coerente com a regra de texto vazio da seção 3. Um destino que
espere a sequência vazia e receba o literal `null`, ou o contrário, falha
na desserialização, e é por isso que a tabela é normativa.

**Entrada inválida** devolve erro e, nas desserializações, **preserva o
receptor inteiro**, identificador e booleano: o booleano de um valor que
já estava presente continua verdadeiro. Só a leitura de valor de banco
derruba o booleano, pela exceção registrada em
[05-conversao-e-analise.md](05-conversao-e-analise.md) §5.

**JSON do tipo anulável.** O caso comum, uma string sem sequências de
escape, é lido direto dos bytes, sem alocar. Se a string contiver escapes
JSON (`0` no lugar de `0`), a decodificação é delegada ao
decodificador da linguagem, que os interpreta como faria para o tipo
simples; isso custa uma alocação e só acontece nesse caso raro. Erros de
sintaxe JSON viram o sentinela de formato, mas o erro específico do
analisador permissivo (comprimento, chaves) **DEVE** sobreviver ao
desvio ([08-casos-de-teste.md](08-casos-de-teste.md), caso 17).

A serialização de cada tipo sem valor para os quatro destinos, e a
leitura da entrada vazia e da entrada inválida, são o caso 18 de
[08-casos-de-teste.md](08-casos-de-teste.md).

## 5. Valor nulo e ordenação

`Nil` é o valor devolvido junto de toda recusa de análise e o que a
leitura de banco grava para ausência; não é UUIDv7. `Compare` ordena
byte a byte e serve à ordenação da biblioteca padrão. Os dois estão
especificados em [03-leitura-do-instante.md](03-leitura-do-instante.md)
§5.
