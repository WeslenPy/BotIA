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
- **!explique** - Explicar uma mensagem marcada (marque uma mensagem e digite !explique)
- **!help** ou **!ajuda** - Mostrar lista de comandos disponíveis

#### Como Usar
```bash
!tapa @amigo        # Dar um tapa no @amigo (menção clicável)
!chute @amigo       # Dar um chute no @amigo
!voadora @amigo     # Dar uma voadora no @amigo
!beijo @amigo       # Dar um beijo no @amigo
!abraco @amigo      # Dar um abraço no @amigo
!piada              # Contar uma piada gerada por IA
!explique           # Marque uma mensagem e digite !explique
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

Maria: "A implementação do algoritmo de busca binária otimiza a complexidade temporal"
João: [Marca a mensagem] !explique
Bot: 💡 *Explicação:*
     [Explicação simples gerada pela IA]

# GIF é enviado como arquivo anexado do WhatsApp
# *@Maria* é uma menção clicável que notifica o usuário
# Usuários podem baixar e visualizar o GIF completo
# Piadas são geradas dinamicamente pela IA
# Sistema de histórico evita repetições
# !explique explica mensagens marcadas de forma simples
```

#### Sistema de Histórico de Piadas
- ✅ **Armazenamento persistente** - Piadas são salvas no banco SQLite
- ✅ **Evita repetições** - IA recebe histórico das últimas 50 piadas
- ✅ **Geração inteligente** - Gemini cria piadas novas e diferentes
- ✅ **Banco de dados** - Tabela `jokes_history` armazena todas as piadas
- ✅ **Limpeza automática** - Sistema pode ser expandido para limpar piadas antigas

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


