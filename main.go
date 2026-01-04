// BotIA - Bot WhatsApp com integração Gemini AI
// Este bot conecta ao WhatsApp via whatsmeow e processa mensagens privadas usando a API do Google Gemini
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/rs/zerolog"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite"
)

// Variáveis globais de configuração e estado
var (
	// logLevel define o nível de log (INFO ou DEBUG)
	logLevel = flag.String("loglevel", "", "Enable debug (INFO or DEBUG)")

	// logType define o formato de saída do log (console ou json)
	logType = flag.String("logtype", "console", "Type of log output (console or json)")

	// geminiAPIKey é a chave da API do Gemini (pode ser fornecida via flag ou variável de ambiente)
	geminiAPIKey = flag.String("geminikey", "", "Gemini API Key (opcional, pode usar GEMINI_API_KEY env var)")

	// geminiModel define qual modelo do Gemini será usado
	geminiModel = flag.String("geminimodel", "gemini-2.5-flash", "Modelo Gemini a usar")

	// tenorAPIKey é a chave da API do Tenor para GIFs
	tenorAPIKey = flag.String("tenorkey", "", "Tenor API Key para comandos de GIF (opcional)")

	// log é o logger zerolog configurado
	log zerolog.Logger

	// geminiClient é a instância do cliente Gemini (nil se não configurado)
	geminiClient *GeminiClient
)

// init inicializa o logger baseado nas flags de configuração
func init() {
	// Parse das flags de linha de comando
	flag.Parse()

	// Configurar logger baseado no tipo escolhido
	if *logType == "json" {
		// Formato JSON para logs estruturados
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		log = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		// Formato console colorido (padrão)
		output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		log = zerolog.New(output).With().Timestamp().Str("role", filepath.Base(os.Args[0])).Logger()
	}
}

// ChatMessage representa uma mensagem armazenada no banco de dados
type ChatMessage struct {
	ID          int       `json:"id"`
	UserJID     string    `json:"user_jid"`
	MessageType string    `json:"message_type"` // 'user' ou 'assistant'
	MessageText string    `json:"message_text"`
	Timestamp   time.Time `json:"timestamp"`
}

// ChatContext gerencia o armazenamento e recuperação do histórico de conversas
type ChatContext struct {
	db          *sql.DB
	maxMessages int
}

// BotClient representa o cliente do bot WhatsApp com suas dependências
type BotClient struct {
	WAClient       *whatsmeow.Client      // Cliente WhatsApp principal
	eventHandlerID uint32                 // ID do handler de eventos registrado
	geminiClient   *GeminiClient          // Cliente Gemini para processar mensagens (pode ser nil)
	chatContext    *ChatContext           // Gerenciador de contexto de conversa
	groupProcessor *GroupMessageProcessor // Processador de mensagens de grupo
}

// NewChatContext cria uma nova instância do gerenciador de contexto
func NewChatContext(db *sql.DB, maxMessages int) (*ChatContext, error) {
	ctx := &ChatContext{
		db:          db,
		maxMessages: maxMessages,
	}

	// Criar tabela se não existir
	err := ctx.initTable()
	if err != nil {
		return nil, fmt.Errorf("erro ao inicializar tabela chat_history: %w", err)
	}

	return ctx, nil
}

// initTable cria a tabela chat_history se ela não existir
func (c *ChatContext) initTable() error {
	// Criar tabela primeiro
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS chat_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_jid TEXT NOT NULL,
			message_type TEXT NOT NULL CHECK (message_type IN ('user', 'assistant')),
			message_text TEXT NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err := c.db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("erro ao criar tabela chat_history: %w", err)
	}

	// Criar índice separadamente
	createIndexQuery := `
		CREATE INDEX IF NOT EXISTS idx_user_jid_timestamp
		ON chat_history (user_jid, timestamp);
	`

	_, err = c.db.Exec(createIndexQuery)
	if err != nil {
		return fmt.Errorf("erro ao criar índice chat_history: %w", err)
	}

	// Criar tabela de histórico de piadas
	createJokesTableQuery := `
		CREATE TABLE IF NOT EXISTS jokes_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			joke_text TEXT NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err = c.db.Exec(createJokesTableQuery)
	if err != nil {
		return fmt.Errorf("erro ao criar tabela jokes_history: %w", err)
	}

	// Criar índice para piadas
	createJokesIndexQuery := `
		CREATE INDEX IF NOT EXISTS idx_jokes_timestamp
		ON jokes_history (timestamp);
	`

	_, err = c.db.Exec(createJokesIndexQuery)
	if err != nil {
		return fmt.Errorf("erro ao criar índice jokes_history: %w", err)
	}

	return nil
}

// SaveMessage salva uma mensagem no histórico
func (c *ChatContext) SaveMessage(ctx context.Context, userJID, messageType, messageText string) error {
	query := `
		INSERT INTO chat_history (user_jid, message_type, message_text, timestamp)
		VALUES (?, ?, ?, ?)
	`

	_, err := c.db.ExecContext(ctx, query, userJID, messageType, messageText, time.Now())
	if err != nil {
		return fmt.Errorf("erro ao salvar mensagem: %w", err)
	}

	return nil
}

// LoadMessages recupera as últimas mensagens de um usuário
func (c *ChatContext) LoadMessages(ctx context.Context, userJID string) ([]ChatMessage, error) {
	query := `
		SELECT id, user_jid, message_type, message_text, timestamp
		FROM chat_history
		WHERE user_jid = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := c.db.QueryContext(ctx, query, userJID, c.maxMessages)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar mensagens: %w", err)
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var msg ChatMessage
		err := rows.Scan(&msg.ID, &msg.UserJID, &msg.MessageType, &msg.MessageText, &msg.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler mensagem: %w", err)
		}
		messages = append(messages, msg)
	}

	// Inverter para ordem cronológica (mais antiga primeiro)
	// Verificar se há mensagens antes de inverter
	if len(messages) > 0 {
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
	}

	return messages, nil
}

// LoadGroupMessages recupera as últimas mensagens de um grupo
func (c *ChatContext) LoadGroupMessages(ctx context.Context, groupJID string, maxMessages int) ([]ChatMessage, error) {
	query := `
		SELECT id, user_jid, message_type, message_text, timestamp
		FROM chat_history
		WHERE user_jid = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := c.db.QueryContext(ctx, query, groupJID, maxMessages)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar mensagens do grupo: %w", err)
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var msg ChatMessage
		err := rows.Scan(&msg.ID, &msg.UserJID, &msg.MessageType, &msg.MessageText, &msg.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler mensagem do grupo: %w", err)
		}
		messages = append(messages, msg)
	}

	// Inverter para ordem cronológica (mais antiga primeiro)
	// Verificar se há mensagens antes de inverter
	if len(messages) > 0 {
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
	}

	return messages, nil
}

// GetContextSize retorna o número de mensagens no contexto
func (c *ChatContext) GetContextSize() int {
	return c.maxMessages
}

// CleanOldMessages remove mensagens antigas (mais de 30 dias) para manter o banco limpo
func (c *ChatContext) CleanOldMessages(ctx context.Context) error {
	query := `
		DELETE FROM chat_history
		WHERE timestamp < datetime('now', '-30 days')
	`

	result, err := c.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("erro ao limpar mensagens antigas: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Info().Int64("rowsDeleted", rowsAffected).Msg("Mensagens antigas removidas")
	}

	return nil
}

// SaveJoke salva uma piada no histórico
func (c *ChatContext) SaveJoke(ctx context.Context, jokeText string) error {
	query := `
		INSERT INTO jokes_history (joke_text, timestamp)
		VALUES (?, ?)
	`

	_, err := c.db.ExecContext(ctx, query, jokeText, time.Now())
	if err != nil {
		return fmt.Errorf("erro ao salvar piada: %w", err)
	}

	return nil
}

// LoadJokesHistory carrega as últimas piadas do histórico
func (c *ChatContext) LoadJokesHistory(ctx context.Context, maxJokes int) ([]string, error) {
	query := `
		SELECT joke_text
		FROM jokes_history
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := c.db.QueryContext(ctx, query, maxJokes)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar histórico de piadas: %w", err)
	}
	defer rows.Close()

	var jokes []string
	for rows.Next() {
		var joke string
		err := rows.Scan(&joke)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler piada: %w", err)
		}
		jokes = append(jokes, joke)
	}

	// Inverter para ordem cronológica (mais antiga primeiro)
	for i, j := 0, len(jokes)-1; i < j; i, j = i+1, j-1 {
		jokes[i], jokes[j] = jokes[j], jokes[i]
	}

	return jokes, nil
}

// FormatJokesHistory formata o histórico de piadas para o prompt
func FormatJokesHistory(jokes []string) string {
	if len(jokes) == 0 {
		return ""
	}

	var history strings.Builder
	history.WriteString("\n\nIMPORTANTE: As seguintes piadas já foram contadas anteriormente. NÃO repita nenhuma delas:\n\n")

	for i, joke := range jokes {
		history.WriteString(fmt.Sprintf("%d. %s\n", i+1, joke))
	}

	history.WriteString("\nGere uma piada NOVA e DIFERENTE das listadas acima.")

	return history.String()
}

// FormatConversationHistory formata o histórico de conversa para o prompt da IA
func FormatConversationHistory(messages []ChatMessage) string {
	if len(messages) == 0 {
		return "Nenhuma conversa anterior."
	}

	var history strings.Builder
	history.WriteString("Histórico da conversa:\n")

	for _, msg := range messages {
		var role string
		if msg.MessageType == "user" {
			role = "Usuário"
		} else {
			role = "DuckerIA"
		}

		history.WriteString(fmt.Sprintf("[%s] %s: %s\n",
			msg.Timestamp.Format("15:04"),
			role,
			msg.MessageText))
	}

	return history.String()
}

// eventHandler é o handler principal de eventos do WhatsApp
// Processa todos os eventos recebidos do WhatsApp e toma ações apropriadas
func (bot *BotClient) eventHandler(rawEvt interface{}) {
	switch evt := rawEvt.(type) {
	case *events.Connected:
		// Evento disparado quando o bot conecta com sucesso ao WhatsApp
		log.Info().Msg("Conectado ao WhatsApp!")

		// Enviar presença "disponível" quando conectar
		// Isso garante que o bot apareça como online
		if len(bot.WAClient.Store.PushName) > 0 {
			err := bot.WAClient.SendPresence(context.Background(), types.PresenceAvailable)
			if err != nil {
				log.Warn().Err(err).Msg("Falha ao enviar presença")
			} else {
				log.Info().Msg("Marcado como disponível")
			}
		}

	case *events.Message:
		// Evento disparado quando uma mensagem é recebida

		// Ignorar mensagens enviadas pelo próprio bot para evitar loops
		if evt.Info.Sender.User == bot.WAClient.Store.ID.User {
			log.Info().Str("sender", evt.Info.Sender.String()).Msg("Ignorando mensagem própria")
			return
		}

		// Tentar diferentes métodos para extrair o texto da mensagem
		msgText := evt.Message.GetConversation()

		// Verificar se é uma mensagem citada/resposta com comando !explique
		var quotedMessageText string
		if extended := evt.Message.GetExtendedTextMessage(); extended != nil {
			if extended.Text != nil {
				msgText = *extended.Text
			}
			// Verificar se há mensagem citada
			if extended.ContextInfo != nil && extended.ContextInfo.QuotedMessage != nil {
				// Extrair texto da mensagem citada - tentar diferentes métodos
				quotedMsg := extended.ContextInfo.QuotedMessage

				// Tentar obter de Conversation
				if quotedConv := quotedMsg.GetConversation(); quotedConv != "" {
					quotedMessageText = quotedConv
				} else if quotedExtended := quotedMsg.GetExtendedTextMessage(); quotedExtended != nil {
					// Tentar obter de ExtendedTextMessage
					if quotedExtended.Text != nil {
						quotedMessageText = *quotedExtended.Text
					}
				} else if quotedImage := quotedMsg.GetImageMessage(); quotedImage != nil {
					// Mensagem citada é uma imagem
					if quotedImage.Caption != nil {
						quotedMessageText = *quotedImage.Caption
					} else {
						quotedMessageText = "[Mensagem com imagem]"
					}
				} else if quotedVideo := quotedMsg.GetVideoMessage(); quotedVideo != nil {
					// Mensagem citada é um vídeo
					if quotedVideo.Caption != nil {
						quotedMessageText = *quotedVideo.Caption
					} else {
						quotedMessageText = "[Mensagem com vídeo]"
					}
				} else if quotedDoc := quotedMsg.GetDocumentMessage(); quotedDoc != nil {
					// Mensagem citada é um documento
					if quotedDoc.Caption != nil {
						quotedMessageText = *quotedDoc.Caption
					} else if quotedDoc.Title != nil {
						quotedMessageText = fmt.Sprintf("[Documento: %s]", *quotedDoc.Title)
					} else {
						quotedMessageText = "[Mensagem com documento]"
					}
				} else {
					quotedMessageText = "[Mensagem sem texto]"
				}
			}
		}

		// Se não conseguir com GetConversation, tentar outros métodos
		if msgText == "" {
			// Tentar obter texto de mensagem estendida
			if extended := evt.Message.GetExtendedTextMessage(); extended != nil && extended.Text != nil {
				msgText = *extended.Text
				log.Info().Str("source", "ExtendedTextMessage").Str("text", msgText).Msg("Texto extraído de ExtendedTextMessage")
			}
		}

		// Verificar se é comando !explique
		if strings.HasPrefix(strings.ToLower(msgText), "!explique") {
			if quotedMessageText != "" {
				log.Info().
					Str("quoted", quotedMessageText).
					Str("command", msgText).
					Msg("Comando !explique detectado com mensagem citada")

				// Processar comando !explique
				go bot.handleExpliqueCommand(context.Background(), evt, quotedMessageText)
				return
			} else {
				// Comando !explique sem mensagem citada
				errorMsg := "❌ Marque uma mensagem antes de usar !explique.\n\nComo usar:\n1. Marque/responda a mensagem que deseja explicar\n2. Digite: !explique"
				msg := &waProto.Message{
					Conversation: &errorMsg,
				}
				_, err := bot.WAClient.SendMessage(context.Background(), evt.Info.Chat, msg)
				if err != nil {
					log.Error().Err(err).Msg("Erro ao enviar mensagem de erro")
				}
				return
			}
		}

		// Ignorar mensagens vazias (provavelmente confirmações ou tipos especiais)
		if msgText == "" {
			log.Info().
				Str("id", evt.Info.ID).
				Msg("Ignorando mensagem vazia - provavelmente confirmação ou tipo especial")
			return
		}

		// Verificar se é mensagem de grupo
		if evt.Info.IsGroup {
			log.Info().
				Str("group", evt.Info.Chat.String()).
				Str("sender", evt.Info.Sender.String()).
				Str("message", msgText).
				Msg("Mensagem recebida de grupo - processando")

			// Verificar se é comando !explique em grupo
			if strings.HasPrefix(strings.ToLower(msgText), "!explique") && quotedMessageText != "" {
				go bot.handleExpliqueCommand(context.Background(), evt, quotedMessageText)
				return
			}

			// Processar mensagem de grupo
			go bot.groupProcessor.ProcessGroupMessage(context.Background(), evt, msgText)
			return
		}

		log.Info().
			Str("id", evt.Info.ID).
			Str("from", evt.Info.SourceString()).
			Bool("isGroup", evt.Info.IsGroup).
			Str("type", fmt.Sprintf("%T", evt.Message)).
			Str("message", msgText).
			Msg("EVENTO MESSAGE PRIVADA RECEBIDA - PROCESSANDO")

		errRead := bot.WAClient.MarkRead(context.Background(), []types.MessageID{evt.Info.ID}, time.Now(),
			evt.Info.Sender, evt.Info.Sender, types.ReceiptTypeRead)
		if errRead != nil {
			log.Error().Err(errRead).Msg("Erro ao marcar mensagem como lida")
		}

		// Processar mensagem privada com Gemini AI
		go bot.processPrivateMessage(context.Background(), evt, msgText)

	case *events.Receipt:
		// Evento disparado quando há confirmação de leitura ou entrega de mensagem
		if evt.Type == types.ReceiptTypeRead {
			// Mensagem foi lida pelo destinatário
			log.Info().
				Strs("messageIDs", evt.MessageIDs).
				Str("from", evt.SourceString()).
				Msg("Mensagem lida")
		} else if evt.Type == types.ReceiptTypeDelivered {
			// Mensagem foi entregue ao destinatário
			log.Info().
				Strs("messageIDs", evt.MessageIDs).
				Str("from", evt.SourceString()).
				Msg("Mensagem entregue")
		}

	case *events.Presence:
		// Evento disparado quando o status de presença de um usuário muda
		if evt.Unavailable {
			log.Info().Str("from", evt.From.String()).Msg("Usuário offline")
		} else {
			log.Info().Str("from", evt.From.String()).Msg("Usuário online")
		}

	case *events.LoggedOut:
		// Evento disparado quando o bot é desconectado do WhatsApp
		log.Info().Str("reason", evt.Reason.String()).Msg("Desconectado do WhatsApp")
		os.Exit(0)

	case *events.StreamReplaced:
		// Evento disparado quando a conexão é substituída (reconexão automática)
		log.Info().Msg("Stream substituído, reconectando...")

	default:
		// Eventos não tratados são logados em modo debug
		log.Debug().Str("event", fmt.Sprintf("%T", evt)).Msg("Evento não tratado")
	}
}

// processPrivateMessage processa mensagens privadas usando a API do Gemini
// Esta função é executada em uma goroutine separada para não bloquear outros eventos
//
// Parâmetros:
//   - ctx: Contexto para controle de cancelamento/timeout
//   - evt: Evento da mensagem recebida
//   - msgText: Texto da mensagem a ser processada
func (bot *BotClient) processPrivateMessage(ctx context.Context, evt *events.Message, msgText string) {
	// Verificar se o cliente Gemini está configurado
	if bot.geminiClient == nil {
		log.Warn().Msg("Gemini client não configurado, ignorando mensagem")

		// Informar ao usuário que o Gemini não está configurado
		errorMsg := "⚠️ Gemini não configurado. Configure a API key para usar esta funcionalidade."
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Sender, msg)
		if err != nil {
			log.Error().Err(err).Msg("Erro ao enviar mensagem de erro")
		}
		return
	}

	// Enviar feedback imediato ao usuário que a mensagem está sendo processada

	log.Info().Str("message", msgText).Str("from", evt.Info.Sender.String()).Msg("Processando mensagem com Gemini")

	// Enviar evento de "digitando"
	errTyping := bot.WAClient.SendChatPresence(context.Background(), evt.Info.Sender, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	if errTyping != nil {
		log.Warn().Err(errTyping).Msg("Erro ao enviar status de digitando")
	}

	// Carregar histórico da conversa para contextualizar a IA
	history, err := bot.chatContext.LoadMessages(ctx, evt.Info.Sender.String())
	if err != nil {
		log.Error().Err(err).Str("jid", evt.Info.Sender.String()).Msg("Erro ao carregar histórico de chat")
		// Continuar sem histórico se houver erro
		history = []ChatMessage{}
	}

	// Salvar mensagem do usuário no histórico
	err = bot.chatContext.SaveMessage(ctx, evt.Info.Sender.String(), "user", msgText)
	if err != nil {
		log.Error().Err(err).Str("jid", evt.Info.Sender.String()).Msg("Erro ao salvar mensagem do usuário")
	}

	// Criar prompt personalizado para o Gemini
	systemPrompt := `
Você é o DuckerIA, um assistente virtual da Hyper Ducker, empresa de tecnologia especializada em desenvolvimento de aplicativos web no Maranhão.

## Sua Identidade e Propósito

- Você se chama DuckerIA e representa a Hyper Ducker
- Você é um agent de conversação, não um agent de vendas
- Seu objetivo é conversar de forma descontraída com os clientes maranhenses
- Você responde perguntas de forma automática e amigável

## Informações da Empresa

**Nome:** Hyper Ducker
**Ramo:** Tecnologia - Desenvolvimento de aplicativos web
**Público:** Jovens
**Tipos de aplicativo:** Todos os tipos (e-commerce, sistemas internos, plataformas, etc.)
**Horário de funcionamento:** 07h às 19h
**Tempo de desenvolvimento:** Varia conforme o projeto

**IMPORTANTE:** A empresa atualmente não está vendendo serviços. Você apenas conversa e tira dúvidas.

## Tom e Estilo de Comunicação

- **Amigável e profissional** com um toque descontraído
- **Prestativo e direto** - responda de forma objetiva sem enrolação
- **Respostas simples e claras** - vá direto ao ponto sem repetir informações desnecessárias
- **Apresente-se apenas na primeira interação** - nas demais, seja natural e conversacional
- **Levemente informal, mas respeitoso** - use "você" predominantemente
- **Use expressões maranhenses com moderação:** visse, rapaz/moça (ocasionalmente), tranquilo, beleza (use de forma sutil e natural)
- **NÃO use emojis em nenhuma circunstância**
- **Seja profissional mas acessível** - equilibre cordialidade com objetividade

## Restrições Importantes

Você NÃO deve:
- Fornecer dados sensíveis de clientes
- Fazer promessas de desconto ou preços
- Realizar alterações de pedidos
- Transferir para atendimento humano (não há essa opção)
- Usar emojis

## Como Lidar com Situações Específicas

**Quando não souber uma informação:**
Seja honesto e direto. Exemplo: "Não tenho essa informação. Posso ajudar com algo mais?"

**Quando perguntarem sobre contratação/vendas:**
Informe de forma simples que no momento não estão comercializando, mas você pode esclarecer dúvidas sobre aplicativos web.

**Engajamento:**
Mantenha perguntas simples e diretas para continuar a conversa quando apropriado, sem forçar.

## Despedida

Quando a conversa terminar naturalmente, despeça-se com:
**"Team Hyper Ducker, agradecemos seu contato."**

Pode adicionar uma frase antes dessa se quiser ser mais caloroso, mas sempre finalize com essa frase.

## Exemplos de Interação

**Primeira interação:**
**Cliente:** "Olá"
**DuckerIA:** "Olá, tudo bem? Sou o DuckerIA da Hyper Ducker. Como posso ajudar?"

**Cumprimentos simples:**
**Cliente:** "Bom dia"
**DuckerIA:** "Bom dia! Tudo bem? Como posso ajudar?"

**Cliente:** "Oi"
**DuckerIA:** "Oi! Como posso ajudar?"

**Perguntas diretas:**
**Cliente:** "Vocês fazem aplicativo?"
**DuckerIA:** "Sim, fazemos aplicativos web de todos os tipos. Você tem algum projeto em mente?"

**Cliente:** "Quanto custa?"
**DuckerIA:** "No momento não estamos comercializando, mas posso tirar dúvidas sobre desenvolvimento. O que você gostaria de saber?"

**Cliente:** "Vocês têm Instagram?"
**DuckerIA:** "Ainda não temos redes sociais. Posso ajudar com algo mais?"

**Cliente:** "Quanto tempo demora?"
**DuckerIA:** "O tempo varia conforme a complexidade do projeto. Depende das funcionalidades que você precisa."

## Lembre-se

- Apresente-se apenas no primeiro contato da conversa
- Seja direto e objetivo nas respostas
- Não repita informações que já foram dadas
- Responda cumprimentos de forma simples e natural
- Use o toque maranhense de forma sutil
- Mantenha sempre o respeito e a simpatia
- Ajude o cliente de forma clara e sem enrolação`

	// Formatar histórico da conversa
	conversationHistory := FormatConversationHistory(history)

	// Combinar prompt do sistema, histórico e mensagem atual
	fullPrompt := fmt.Sprintf("%s\n\n%s\n\nMensagem atual do usuário: %s",
		systemPrompt, conversationHistory, msgText)

	// Gerar resposta usando a API do Gemini
	response, err := bot.geminiClient.GenerateContent(ctx, fullPrompt)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao gerar resposta com Gemini")

		// Informar erro ao usuário
		errorMsg := "❌ Erro ao processar sua solicitação. Tente novamente mais tarde."
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		bot.WAClient.SendMessage(ctx, evt.Info.Sender, msg)
		return
	}

	if len(response) > 4000 {
		response = response[:4000] + "\n\n... (resposta truncada)"
	}

	// Salvar resposta da IA no histórico antes de enviar
	err = bot.chatContext.SaveMessage(ctx, evt.Info.Sender.String(), "assistant", response)
	if err != nil {
		log.Error().Err(err).Str("jid", evt.Info.Sender.String()).Msg("Erro ao salvar resposta da IA")
		// Continuar mesmo com erro de salvamento
	}

	// Enviar resposta gerada pelo Gemini ao usuário
	responseMsg := fmt.Sprintf("🤖 %s", response)
	msg := &waProto.Message{
		Conversation: &responseMsg,
	}
	_, err = bot.WAClient.SendMessage(ctx, evt.Info.Sender, msg)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao enviar resposta")
	} else {
		log.Info().
			Int("contextSize", len(history)).
			Int("responseLength", len(response)).
			Msg("Resposta do Gemini enviada ao usuário")
	}

	// Enviar evento de "pausado"
	errTyping = bot.WAClient.SendChatPresence(context.Background(), evt.Info.Sender, types.ChatPresencePaused, types.ChatPresenceMediaText)
	if errTyping != nil {
		log.Warn().Err(errTyping).Msg("Erro ao enviar status de pausado")
	}

}

// handleExpliqueCommand processa o comando !explique para explicar mensagens citadas
func (bot *BotClient) handleExpliqueCommand(ctx context.Context, evt *events.Message, quotedMessageText string) {
	// Verificar se o cliente Gemini está configurado
	if bot.geminiClient == nil {
		errorMsg := "❌ Gemini não está configurado. Configure a API key para usar este comando."
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		if err != nil {
			log.Error().Err(err).Msg("Erro ao enviar mensagem de erro")
		}
		return
	}

	// Enviar evento de "digitando"
	errTyping := bot.WAClient.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	if errTyping != nil {
		log.Warn().Err(errTyping).Msg("Erro ao enviar status de digitando")
	}

	// Criar prompt para explicar a mensagem
	prompt := fmt.Sprintf(`Você é um assistente que explica mensagens de forma simples e clara.

Sua tarefa é explicar o que a seguinte mensagem quis dizer, de forma:
- Simples e direta
- Fácil de entender
- Objetiva (máximo 2-3 frases)
- Em português brasileiro
- Sem emojis
- Sem julgamentos ou opiniões, apenas explicação

Mensagem a ser explicada:
"%s"

Explique de forma simples o que essa mensagem quis dizer:`, quotedMessageText)

	log.Info().
		Str("quoted", quotedMessageText).
		Str("from", evt.Info.Sender.String()).
		Msg("Processando comando !explique")

	// Gerar explicação usando a API do Gemini
	explicacao, err := bot.geminiClient.GenerateContent(ctx, prompt)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao gerar explicação com Gemini")

		// Encerrar status de digitando
		bot.WAClient.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

		// Informar erro ao usuário
		errorMsg := "❌ Erro ao gerar explicação. Tente novamente mais tarde."
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		if err != nil {
			log.Error().Err(err).Msg("Erro ao enviar mensagem de erro")
		}
		return
	}

	// Limitar tamanho da explicação
	if len(explicacao) > 1000 {
		explicacao = explicacao[:1000] + "..."
	}

	// Encerrar status de digitando
	bot.WAClient.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

	// Enviar explicação gerada
	explicacaoMsg := fmt.Sprintf("💡 *Explicação:*\n\n%s", explicacao)
	msg := &waProto.Message{
		Conversation: &explicacaoMsg,
	}
	_, err = bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao enviar explicação")
	} else {
		log.Info().
			Int("length", len(explicacao)).
			Str("original", quotedMessageText).
			Msg("Explicação enviada com sucesso")
	}
}

// main é a função principal do programa
// Inicializa todos os componentes e mantém o bot rodando
func main() {
	log.Info().Str("loglevel", *logLevel).Str("logtype", *logType).Msg("Iniciando BotIA")

	// Inicializar cliente Gemini se API key fornecida
	// A API key pode vir de flag (-geminikey) ou variável de ambiente (GEMINI_API_KEY)
	if *geminiAPIKey != "" || os.Getenv("GEMINI_API_KEY") != "" {
		var err error
		geminiClient, err = NewGeminiClient(*geminiAPIKey)
		if err != nil {
			log.Fatal().Err(err).Msg("Erro ao inicializar cliente Gemini")
		}

		// Configurar modelo do Gemini se especificado via flag
		if *geminiModel != "" {
			geminiClient.SetModel(*geminiModel)
		}

		log.Info().
			Str("model", geminiClient.GetModel()).
			Msg("Cliente Gemini inicializado")
	} else {
		log.Warn().Msg("Gemini API key não fornecida. Funcionalidade do Gemini desabilitada.")
	}

	// Criar diretório para banco de dados SQLite
	// O banco armazena a sessão do WhatsApp para reconexão automática
	dbDirectory := "auth"
	_, err := os.Stat(dbDirectory)
	if os.IsNotExist(err) {
		// Criar diretório se não existir
		errDir := os.MkdirAll(dbDirectory, 0751)
		if errDir != nil {
			log.Fatal().Err(errDir).Msg("Não foi possível criar diretório auth")
		}
	}

	// Conectar ao banco de dados SQLite
	// O banco armazena credenciais e estado da sessão do WhatsApp
	var container *sqlstore.Container
	dbUri := "file:./auth/main.db?_pragma=foreign_keys(1)&cache=shared&mode=rwc&?_busy_timeout=20000"

	// Configurar logger do banco de dados se logLevel estiver ativo
	if *logLevel != "" {
		dbLog := waLog.Stdout("Database", *logLevel, true)
		container, err = sqlstore.New(context.Background(), "sqlite", dbUri, dbLog)
	} else {
		container, err = sqlstore.New(context.Background(), "sqlite", dbUri, nil)
	}

	if err != nil {
		log.Fatal().Err(err).Msg("Erro ao conectar ao banco de dados")
	}

	// Criar conexão separada para o contexto de chat
	chatDB, err := sql.Open("sqlite", dbUri)
	if err != nil {
		log.Fatal().Err(err).Msg("Erro ao abrir banco para contexto de chat")
	}

	// Inicializar gerenciador de contexto de chat
	chatContext, err := NewChatContext(chatDB, 100) // Máximo de 100 mensagens por conversa
	if err != nil {
		log.Fatal().Err(err).Msg("Erro ao inicializar contexto de chat")
	}

	// Inicializar processador de mensagens de grupo
	groupProcessor := NewGroupMessageProcessor(nil) // Será definido após criar o bot

	// Limpar mensagens antigas (opcional, roda em background)
	// go func() {
	// 	ticker := time.NewTicker(24 * time.Hour) // A cada 24 horas
	// 	defer ticker.Stop()

	// 	for {
	// 		select {
	// 		case <-ticker.C:
	// 			ctx := context.Background()
	// 			err := chatContext.CleanOldMessages(ctx)
	// 			if err != nil {
	// 				log.Warn().Err(err).Msg("Erro ao limpar mensagens antigas")
	// 			}
	// 		}
	// 	}
	// }()

	// Obter ou criar dispositivo WhatsApp
	// O dispositivo representa a sessão do WhatsApp
	var deviceStore *store.Device
	deviceStore, err = container.GetFirstDevice(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("Erro ao obter dispositivo")
	}

	// Se não houver dispositivo, criar um novo
	// Isso acontece na primeira execução
	if deviceStore == nil {
		log.Info().Msg("Nenhum dispositivo encontrado, criando novo")
		deviceStore = container.NewDevice()
	}

	// Configurar propriedades do dispositivo
	// Define como o bot aparece no WhatsApp
	store.DeviceProps.PlatformType = waProto.DeviceProps_CHROME.Enum()
	osName := "BotIA"
	store.DeviceProps.Os = &osName

	// Criar cliente WhatsApp
	// O cliente gerencia a conexão e comunicação com o WhatsApp
	var clientLog waLog.Logger
	if *logLevel != "" {
		clientLog = waLog.Stdout("Client", *logLevel, true)
	}

	client := whatsmeow.NewClient(deviceStore, clientLog)

	// Criar instância do bot com suas dependências
	bot := &BotClient{
		WAClient:       client,
		geminiClient:   geminiClient,
		chatContext:    chatContext,
		groupProcessor: groupProcessor,
	}

	// Configurar referência do bot no processador de grupos
	bot.groupProcessor.bot = bot

	// Registrar handler de eventos
	// Todos os eventos do WhatsApp serão processados por eventHandler
	bot.eventHandlerID = client.AddEventHandler(bot.eventHandler)

	// Verificar se já está autenticado
	// Se não houver ID no store, precisa fazer login pela primeira vez
	if client.Store.ID == nil {
		log.Info().Msg("Não autenticado, gerando QR Code...")

		// Obter canal de QR Code para autenticação
		qrChan, err := client.GetQRChannel(context.Background())
		if err != nil {
			log.Fatal().Err(err).Msg("Erro ao obter canal QR")
		}

		// Conectar ao WhatsApp
		err = client.Connect()
		if err != nil {
			log.Fatal().Err(err).Msg("Erro ao conectar")
		}

		// Processar eventos do QR Code
		// O QR Code precisa ser escaneado com o WhatsApp para autenticar
		for evt := range qrChan {
			if evt.Event == "code" {
				// Exibir QR Code no terminal
				if *logType != "json" {
					// Formato visual no console
					qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
					fmt.Println("\nQR Code:")
					fmt.Println(evt.Code)
				} else {
					// Formato JSON para logs estruturados
					log.Info().Str("qrcode", evt.Code).Msg("QR Code gerado")
				}
			} else if evt.Event == "timeout" {
				// QR Code expirou, precisa gerar novo
				log.Warn().Msg("QR Code expirado")
				os.Exit(1)
			} else if evt.Event == "success" {
				// Autenticação bem-sucedida
				log.Info().Msg("Pareamento bem-sucedido!")
			} else {
				// Outros eventos de login
				log.Info().Str("event", evt.Event).Msg("Evento de login")
			}
		}
	} else {
		// Já está autenticado (sessão salva no banco)
		// Apenas conectar novamente
		log.Info().Msg("Já autenticado, conectando...")
		err = client.Connect()
		if err != nil {
			log.Fatal().Err(err).Msg("Erro ao conectar")
		}
	}

	log.Info().Msg("BotIA está rodando! Pressione Ctrl+C para sair.")

	// Aguardar sinal de interrupção (Ctrl+C ou SIGTERM)
	// Isso mantém o programa rodando até ser interrompido
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	// Desconectar graciosamente ao receber sinal de interrupção
	log.Info().Msg("Desconectando...")
	client.Disconnect()
	log.Info().Msg("BotIA finalizado")
}
