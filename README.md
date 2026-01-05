# BotIA - Bot WhatsApp Simplificado

Versão simplificada do projeto wago, focada apenas na conexão e tratamento de eventos do WhatsApp, sem rotas HTTP/API.

## Características

- ✅ Conexão ao WhatsApp via whatsmeow
- ✅ Autenticação via QR Code
- ✅ Tratamento de eventos (mensagens, presença, recebimentos, etc.)
- ✅ Armazenamento de sessão em SQLite
- ✅ Logging configurável (console ou JSON)
- ❌ Sem rotas HTTP/API (versão simplificada)

## Pré-requisitos

- Go 1.24 ou superior
- Compilador C (para SQLite)

## Instalação

1. Clone ou baixe este repositório

2. Instale as dependências:
```bash
go mod download
```

## Uso

### Execução básica:
```bash
go run main.go
```

### Com opções de log:
```bash
# Log em modo DEBUG
go run main.go -loglevel=DEBUG

# Log em formato JSON
go run main.go -logtype=json

# Combinado
go run main.go -loglevel=DEBUG -logtype=json
```

### Com integração Gemini AI:
```bash
# Com Gemini AI habilitado (usando variável de ambiente)
export GEMINI_API_KEY=sua_api_key_aqui
go run main.go

# Ou passando a API key diretamente
go run main.go -geminikey=SUA_API_KEY

# Com modelo específico
go run main.go -geminikey=SUA_API_KEY -geminimodel=gemini-1.5-pro

# Todas as opções
go run main.go -loglevel=DEBUG -geminikey=SUA_API_KEY -geminimodel=gemini-2.5-flash
```

### Com sistema de comandos:
```bash
# Sistema de comandos ativo (GIFs locais)
go run main.go

# Completo - Gemini + Comandos + Debug
go run main.go -geminikey=SUA_API_KEY -loglevel=DEBUG
```

## Primeira Execução

1. Execute o projeto
2. Um QR Code será exibido no terminal
3. Escaneie o QR Code com o WhatsApp:
   - Abra o WhatsApp no seu celular
   - Vá em **Configurações > Aparelhos conectados > Conectar um aparelho**
   - Escaneie o QR Code exibido no terminal
4. Após conectar, o bot estará pronto para receber e enviar mensagens

## Funcionalidades

### Eventos Tratados

- **Connected**: Quando conecta ao WhatsApp
- **Message**: Mensagens recebidas
  - **Mensagens Privadas**: Processadas automaticamente com Gemini AI (se configurado)
  - **Mensagens em Grupo**: Sistema avançado de comandos e IA contextual
- **Receipt**: Confirmações de leitura e entrega
- **Presence**: Status online/offline de usuários
- **LoggedOut**: Quando desconecta

### Sistema de Comandos

O bot inclui um sistema de comandos especiais para grupos, iniciado com `!`:

#### Comandos Disponíveis

- **!tapa @usuario** - Dar um tapa virtual em alguém com GIF aleatório
- **!chute @usuario** - Dar um chute virtual em alguém com GIF aleatório
- **!voadora @usuario** - Dar uma voadora virtual em alguém com GIF aleatório
- **!beijo @usuario** - Dar um beijo virtual em alguém com GIF aleatório
- **!abraco @usuario** - Dar um abraço virtual em alguém com GIF aleatório
- **!piada** - Contar uma piada gerada por IA (requer Gemini configurado, evita repetições)
- **!cantada @usuario** - Gerar uma cantada para alguém usando IA (requer Gemini configurado)
- **!historia [tipo]** - Gerar uma história usando IA (ex: !historia terror, !historia comedia) (requer Gemini configurado)
- **!explique** - Explicar uma mensagem marcada (marque uma mensagem e digite !explique)
- **!autodestruicao [minutos]** - Pausar o bot por X minutos com countdown (padrão: 5 min, máximo: 60 min, só funciona em grupos)
- **!roletacasais** ou **!roleta** - Formar casais aleatórios com os membros do grupo (só funciona em grupos)
- **!help** ou **!ajuda** - Mostrar lista de comandos disponíveis

#### Como Usar
```bash
!tapa @amigo        # Dar um tapa no @amigo (menção clicável)
!chute @amigo       # Dar um chute no @amigo
!voadora @amigo     # Dar uma voadora no @amigo
!beijo @amigo       # Dar um beijo no @amigo
!abraco @amigo      # Dar um abraço no @amigo
!piada              # Contar uma piada gerada por IA
!cantada @amigo     # Gerar uma cantada para @amigo
!historia terror    # Gerar uma história de terror
!historia comedia   # Gerar uma história de comédia
!explique           # Marque uma mensagem e digite !explique
!autodestruicao 10  # Pausar o bot por 10 minutos com countdown (só em grupos)
!roletacasais       # Formar casais aleatórios com os membros do grupo (só em grupos)
!help              # Ver lista de comandos
```

#### Exemplos Práticos
```bash
João: !tapa @Maria
Bot: [Envia arquivo GIF animado]
     Legenda: 🤚 *João* deu um tapa em *@Maria*!

João: !beijo @Maria
Bot: [Envia arquivo GIF animado]
     Legenda: 💋 *João* deu um beijo em *@Maria*!

João: !piada
Bot: 😄 *Piada:*
     [Piada gerada pela IA do Gemini]

João: !cantada @Maria
Bot: 💕 *Cantada para @Maria:*
     [Cantada criativa gerada pela IA do Gemini]

João: !historia terror
Bot: 📖 *História de Terror:*
     [História de terror gerada pela IA do Gemini]

João: !historia comedia
Bot: 📖 *História de Comedia:*
     [História de comédia gerada pela IA do Gemini]

Maria: "A implementação do algoritmo de busca binária otimiza a complexidade temporal"
João: [Marca a mensagem] !explique
Bot: 💡 *Explicação:*
     [Explicação simples gerada pela IA]

João: @DuckerIA como funciona a busca binária?
Bot: 🤖 [Resposta gerada pela IA do Gemini explicando busca binária]

Maria: [Cita uma mensagem anterior do bot] você pode explicar melhor?
Bot: 🤖 [Resposta gerada pela IA do Gemini]

João: !autodestruicao 5
Bot: ⚠️ *AUTO-DESTRUIÇÃO ATIVADA*
     Bot será pausado por *5 minuto(s)*.
     Iniciando countdown de 5 segundos...
     💥 5
     💥 4
     💥 3
     💥 2
     💥 1
     💥 *Bot pausado!*
     Bot ficará inativo por um período.
     [5 minutos de silêncio...]
     ✅ *Bot reativado!*
     Auto-destruição concluída. Bot está funcionando normalmente novamente.

João: !roletacasais
Bot: 💕 *ROleta DOS CASAIS*
     💑 *Maria* e *Pedro*

# GIF é enviado como arquivo anexado do WhatsApp
# *@Maria* é uma menção clicável que notifica o usuário
# Usuários podem baixar e visualizar o GIF completo
# Piadas são geradas dinamicamente pela IA
# Sistema de histórico evita repetições
# !explique explica mensagens marcadas de forma simples
# Menções automáticas ativam respostas da IA sem comandos
# !autodestruicao pausa o bot temporariamente com countdown
# !roletacasais forma casais aleatórios com os membros do grupo
```

#### Comando !roletacasais
- ✅ **Formação aleatória de um casal** - Seleciona 2 membros aleatórios e forma um casal
- ✅ **Apenas em grupos** - Comando só funciona em grupos do WhatsApp
- ✅ **Exclui o bot** - O bot não participa da roleta
- ✅ **Aleatório a cada execução** - Cada vez que o comando é executado, forma um casal diferente
- ✅ **Nomes dos participantes** - Usa os nomes dos contatos quando disponíveis

**Como funciona:**
- Use `!roletacasais` ou `!roleta` em um grupo
- O bot obtém a lista de todos os membros do grupo (excluindo o bot)
- Seleciona aleatoriamente 2 membros diferentes
- Forma um único casal com esses 2 membros
- Envia a mensagem com o casal formado

**Requisitos:**
- Mínimo de 2 membros no grupo (além do bot)
- Bot deve ter permissões para ver a lista de membros

**Exemplo:**
```
João: !roletacasais
Bot: 💕 *ROleta DOS CASAIS*

     💑 *Maria* e *Pedro*

João: !roletacasais
Bot: 💕 *ROleta DOS CASAIS*

     💑 *Ana* e *Carlos*
```

#### Sistema de Auto-Destruição
- ✅ **Pausa temporária** - Pausa o bot por um período determinado (1-60 minutos)
- ✅ **Countdown de 5 segundos** - Countdown rápido com emoji de explosão antes da pausa
- ✅ **Apenas em grupos** - Comando só funciona em grupos do WhatsApp
- ✅ **Todas as funções pausadas** - Quando pausado, o bot ignora TUDO: comandos, mensagens, menções, etc.
- ✅ **Reativação automática** - Bot reativa automaticamente após o tempo determinado
- ✅ **Proteção contra duplicatas** - Não permite ativar auto-destruição se já estiver pausado
- ✅ **Silencioso durante pausa** - Não envia mensagens durante a pausa, apenas reativa no final

**Como funciona:**
- Use `!autodestruicao [minutos]` em um grupo (padrão: 5 minutos, máximo: 60 minutos)
- O bot faz um countdown de 5 segundos com emoji de explosão (💥)
- Após o countdown, o bot é pausado pelo tempo determinado
- Durante a pausa, o bot ignora COMPLETAMENTE todas as funções: comandos (!tapa, !piada, etc.), mensagens normais, menções, citações, etc.
- Não há mensagens durante a pausa
- Após o tempo determinado, o bot reativa automaticamente com uma mensagem de confirmação

**Exemplo:**
```
João: !autodestruicao 10
Bot: ⚠️ *AUTO-DESTRUIÇÃO ATIVADA*
     Bot será pausado por *10 minuto(s)*.
     Iniciando countdown de 5 segundos...
     💥 5
     💥 4
     💥 3
     💥 2
     💥 1
     💥 *Bot pausado!*
     Bot ficará inativo por um período.
     [10 minutos de silêncio...]
     ✅ *Bot reativado!*
     Auto-destruição concluída. Bot está funcionando normalmente novamente.
```

#### Sistema de Histórico de Piadas
- ✅ **Armazenamento persistente** - Piadas são salvas no banco SQLite
- ✅ **Evita repetições** - IA recebe histórico das últimas 50 piadas
- ✅ **Geração inteligente** - Gemini cria piadas novas e diferentes
- ✅ **Banco de dados** - Tabela `jokes_history` armazena todas as piadas
- ✅ **Limpeza automática** - Sistema pode ser expandido para limpar piadas antigas

#### Sistema de Menções e Respostas Automáticas em Grupos
- ✅ **Resposta automática a menções** - Quando mencionado (@bot), responde com IA
- ✅ **Resposta a citações** - Quando uma mensagem do bot é citada, responde com IA
- ✅ **Funciona sem comandos** - Não precisa usar "!" para ativar
- ✅ **Ignora RequireMention** - Menções sempre processam, mesmo com RequireMention ativo
- ✅ **Detecção inteligente** - Detecta menções em texto, imagens e vídeos
- ✅ **Múltiplos nomes** - Reconhece: ducker, duckeria, botia, bot

**Como funciona:**
- Mencione o bot em uma mensagem: `@DuckerIA como funciona isso?`
- Cite uma mensagem do bot e escreva algo
- O bot detecta automaticamente e responde usando a IA do Gemini

#### Prompt Exclusivo para Grupos
- ✅ **Direto e objetivo** - Respostas curtas e diretas ao ponto
- ✅ **Natural e descontraído** - Tom amigável mas sem enrolação
- ✅ **Linguagem natural** - Conversacional e acessível
- ✅ **Expressões maranhenses** - Usa ocasionalmente (visse, rapaz/moça, tranquilo, beleza)
- ✅ **Contexto do grupo** - Considera histórico de mensagens anteriores
- ✅ **Sem forçar tecnologia** - Não menciona tecnologia a menos que seja o assunto

**Características do prompt:**
- Respostas MUITO curtas e diretas (máximo 500 caracteres, idealmente 1-2 frases)
- Vai direto ao ponto, sem enrolação
- Não força assuntos de tecnologia
- Não tenta mudar o tema da conversa
- Responde apenas o que foi perguntado
- Mantém tom leve e natural
- Não usa emojis

**Exemplo de interação:**
```
João: @DuckerIA qual a melhor linguagem para iniciantes?
Bot: 🤖 Python é ideal para iniciantes, rapaz. É simples e tem uma comunidade grande.

Maria: O que você acha do tempo hoje?
Bot: 🤖 Tá quente demais, visse! Melhor ficar na sombra.
```

#### Comando !explique
- ✅ **Explicação inteligente** - Usa Gemini para explicar mensagens marcadas
- ✅ **Funciona em grupos e privado** - Disponível em todos os contextos
- ✅ **Explicações simples** - Respostas claras e objetivas (2-3 frases)
- ✅ **Sem julgamentos** - Apenas explicação factual
- ✅ **Fácil de usar** - Marque uma mensagem e digite !explique
- ✅ **Suporte a múltiplos tipos** - Funciona com texto, imagens, vídeos e documentos

**Como usar:**
1. Marque/responda a mensagem que deseja explicar (mantenha pressionado e selecione "Responder")
2. Digite: `!explique`
3. O bot explicará de forma simples o que a mensagem quis dizer

**Exemplo:**
```
Usuário A: "A implementação do algoritmo de busca binária otimiza a complexidade temporal"
Usuário B: [Marca a mensagem] !explique
Bot: 💡 *Explicação:*
     A busca binária é um método eficiente de encontrar algo em uma lista ordenada, 
     dividindo a busca pela metade a cada tentativa, tornando muito mais rápido 
     do que procurar item por item.
```

#### Comando !cantada
- ✅ **Cantadas geradas por IA** - Usa Gemini para criar cantadas criativas e engraçadas
- ✅ **Funciona em grupos** - Disponível para uso em grupos
- ✅ **Menção automática** - Menciona o usuário alvo de forma clicável
- ✅ **Cantadas adequadas** - Conteúdo apropriado para todos os públicos
- ✅ **Criativas e variadas** - Cada cantada é única e gerada dinamicamente
- ✅ **Fácil de usar** - Apenas digite !cantada @usuario

**Como usar:**
1. Digite: `!cantada @usuario`
2. O bot gerará uma cantada criativa usando IA
3. A cantada será enviada com menção ao usuário mencionado

**Exemplo:**
```
João: !cantada @Maria
Bot: 💕 *Cantada para @Maria:*
     Se você fosse um algoritmo, seria o mais eficiente do mundo, 
     porque você otimiza meu coração em tempo constante!

Maria: !cantada @João
Bot: 💕 *Cantada para @João:*
     Você não é um bug, você é uma feature que eu sempre quis ter no meu código!
```

#### Comando !historia
- ✅ **Histórias geradas por IA** - Usa Gemini para criar histórias criativas e envolventes
- ✅ **Múltiplos gêneros** - Suporta terror, comédia, romance, aventura, ficção científica, etc.
- ✅ **Histórias completas** - Começo, meio e fim (5-10 parágrafos)
- ✅ **Conteúdo adequado** - Apropriado para todos os públicos
- ✅ **Criativas e variadas** - Cada história é única e gerada dinamicamente
- ✅ **Fácil de usar** - Apenas digite !historia [tipo]

**Como usar:**
1. Digite: `!historia [tipo]`
2. O bot gerará uma história do gênero especificado
3. Se não especificar o tipo, usará "aventura" como padrão

**Tipos de história suportados:**
- `!historia terror` - História de terror e suspense
- `!historia comedia` - História de comédia
- `!historia romance` - História romântica
- `!historia aventura` - História de aventura
- `!historia ficcao` - História de ficção científica
- `!historia [qualquer tipo]` - O bot criará uma história do tipo especificado

**Exemplo:**
```
João: !historia terror
Bot: 📖 *História de Terror:*
     [História completa de terror gerada pela IA]

Maria: !historia comedia
Bot: 📖 *História de Comedia:*
     [História completa de comédia gerada pela IA]

João: !historia
Bot: 📖 *História de Aventura:*
     [História de aventura (padrão) gerada pela IA]
```

#### Características dos Comandos
- ✅ **Processamento prioritário** - Comandos têm prioridade sobre IA
- ✅ **GIFs locais reais** - Envia GIFs como arquivos anexados do WhatsApp
- ✅ **Upload automático** - Faz upload dos arquivos para o WhatsApp
- ✅ **Menções reais** - Menciona usuários alvo de forma clicável
- ✅ **Suporte completo a @usuario** - Menções funcionais no WhatsApp
- ✅ **Fallback elegante** - Se upload falhar, envia texto com menção
- ✅ **Múltiplas ações** - 5 comandos diferentes de interação

#### Arquivos Necessários
- **Pastas de GIFs:**
  - `static/gif/slap/` - GIFs de tapa
  - `static/gif/chute/` - GIFs de chute
  - `static/gif/voadora/` - GIFs de voadora
  - `static/gif/beijo/` - GIFs de beijo
  - `static/gif/abraco/` - GIFs de abraço
  - Adicione arquivos `.mp4` em cada pasta
  - O bot selecionará aleatoriamente um GIF para cada comando

### Integração com Gemini AI

Quando configurado com API key, o bot processa mensagens privadas usando a API do Google Gemini com **contexto de conversa persistente**:

1. **Recebe mensagem privada** → Armazena no histórico
2. **Carrega contexto** → Últimas 100 mensagens da conversa
3. **Gera resposta contextual** → O Gemini processa considerando o histórico
4. **Salva resposta** → Armazena no histórico para futuras referências
5. **Envia resposta** → Responde ao usuário no WhatsApp

**Características da Integração:**
- ✅ **Contexto persistente** - Histórico salvo em banco SQLite
- ✅ **Limite inteligente** - Até 100 mensagens por conversa
- ✅ **Limpeza automática** - Remove mensagens antigas para otimizar
- ✅ **Prompt personalizado** - Sistema do DuckerIA carregado dinamicamente

**Requisitos:**
- API Key do Gemini (obtenha em [Google AI Studio](https://aistudio.google.com/))
- Pode ser configurada via variável de ambiente `GEMINI_API_KEY` ou flag `-geminikey`

**Modelos disponíveis:**
- `gemini-2.5-flash` (padrão, rápido e eficiente)
- `gemini-1.5-pro` (mais poderoso)
- `gemini-1.5-flash` (equilíbrio entre velocidade e qualidade)
- `gemini-1.5-flash-8b` (versão leve)

### Personalização

Edite a função `eventHandler` em `main.go` para adicionar suas próprias funcionalidades e comandos.

Exemplo de resposta automática:
```go
if msgText == "ping" {
    // Enviar resposta
}
```

## Estrutura do Projeto

```
BotIA/
├── main.go          # Código principal do bot
├── bot.go           # Sistema de comandos e processamento de grupos
├── gemini.go        # Cliente para integração com Gemini AI
├── go.mod           # Dependências do projeto
├── go.sum           # Checksums das dependências
├── auth/            # Diretório de autenticação (criado automaticamente)
│   └── main.db      # Banco de dados SQLite
└── README.md        # Este arquivo
```

## Flags Disponíveis

- `-loglevel`: Nível de log (INFO ou DEBUG)
- `-logtype`: Tipo de saída de log (console ou json)
- `-geminikey`: API Key do Gemini (opcional, pode usar GEMINI_API_KEY env var)
- `-geminimodel`: Modelo Gemini a usar (padrão: gemini-2.5-flash)

## Diferenças do Projeto Wago

Esta versão simplificada **não inclui**:
- ❌ Rotas HTTP/API REST
- ❌ Webhooks
- ❌ WebSocket
- ❌ Múltiplos usuários/clientes
- ❌ Interface web
- ❌ Download automático de mídias

**Foco**: Apenas conexão e tratamento básico de eventos do WhatsApp.

## Aviso

O uso de bibliotecas não oficiais para interagir com o WhatsApp pode violar os Termos de Serviço da plataforma. Utilize com responsabilidade.

## Licença

Este projeto é apenas para fins educacionais.


