package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// CommandHandler gerencia comandos especiais
type CommandHandler struct {
	// Sem dependências externas - usa arquivos locais
}

// GroupMessageProcessor processa mensagens provenientes de grupos
type GroupMessageProcessor struct {
	bot            *BotClient
	groupRules     map[string]*GroupRules // Regras específicas por grupo
	commandHandler *CommandHandler
}

// NewCommandHandler cria um novo gerenciador de comandos
func NewCommandHandler() *CommandHandler {
	return &CommandHandler{}
}

// ProcessCommand processa um comando especial
func (ch *CommandHandler) ProcessCommand(ctx context.Context, command string, args []string, evt *events.Message, bot *BotClient) error {
	switch strings.ToLower(command) {
	case "tapa":
		return ch.handleTapaCommand(ctx, args, evt, bot)
	case "chute":
		return ch.handleChuteCommand(ctx, args, evt, bot)
	case "voadora":
		return ch.handleVoadoraCommand(ctx, args, evt, bot)
	case "beijo":
		return ch.handleBeijoCommand(ctx, args, evt, bot)
	case "abraco", "abraço":
		return ch.handleAbracoCommand(ctx, args, evt, bot)
	case "piada":
		return ch.handlePiadaCommand(ctx, evt, bot)
	case "help", "ajuda":
		return ch.handleHelpCommand(ctx, evt, bot)
	default:
		// Comando não reconhecido
		return nil
	}
}

// handleActionCommand processa comandos de ação genéricos (tapa, chute, etc.)
func (ch *CommandHandler) handleActionCommand(ctx context.Context, args []string, evt *events.Message, bot *BotClient, folder, actionName, emoji, commandUsage string) error {
	if len(args) == 0 {
		// Sem menção, enviar mensagem de erro
		errorMsg := fmt.Sprintf("❌ Use: %s\nExemplo: %s @johndoe", commandUsage, commandUsage)
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		return err
	}

	// Extrair menção do primeiro argumento
	targetMention := args[0]
	var targetJID string
	var targetName string

	// Verificar se é uma menção válida (@usuario)
	if strings.HasPrefix(targetMention, "@") {
		targetJID, targetName = ch.extractMentionInfo(targetMention, evt)
	} else {
		// Se não é uma menção com @, usar como nome simples
		targetName = targetMention
	}

	// Buscar GIF aleatório na pasta específica
	gifPath, err := ch.searchLocalGIF(folder)
	if err != nil {
		log.Warn().Err(err).Str("folder", folder).Msg("Erro ao buscar GIF local")
		// Fallback: enviar mensagem de texto com menção (se houver JID)
		caption := fmt.Sprintf("%s *%s* %s *%s*!", emoji, evt.Info.PushName, actionName, targetName)
		if targetJID != "" {
			caption = fmt.Sprintf("%s *%s* %s *@%s*!", emoji, evt.Info.PushName, actionName, targetName)
			return ch.sendMentionMessage(ctx, caption, targetJID, evt, bot)
		} else {
			msg := &waProto.Message{
				Conversation: &caption,
			}
			_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
			return err
		}
	}

	// Enviar GIF com menção (se houver JID)
	caption := fmt.Sprintf("%s *%s* %s *%s*!", emoji, evt.Info.PushName, actionName, targetName)
	if targetJID != "" {
		caption = fmt.Sprintf("%s *%s* %s *@%s*!", emoji, evt.Info.PushName, actionName, targetName)
		return ch.sendGIFMessageWithMention(ctx, gifPath, caption, targetJID, evt, bot)
	} else {
		return ch.sendGIFMessage(ctx, gifPath, caption, evt, bot)
	}
}

// handleTapaCommand processa o comando !tapa
func (ch *CommandHandler) handleTapaCommand(ctx context.Context, args []string, evt *events.Message, bot *BotClient) error {
	return ch.handleActionCommand(ctx, args, evt, bot, "slap", "deu um tapa em", "🤚", "!tapa @usuario")
}

// handleChuteCommand processa o comando !chute
func (ch *CommandHandler) handleChuteCommand(ctx context.Context, args []string, evt *events.Message, bot *BotClient) error {
	return ch.handleActionCommand(ctx, args, evt, bot, "kick", "deu um chute em", "🦵", "!chute @usuario")
}

// handleVoadoraCommand processa o comando !voadora
func (ch *CommandHandler) handleVoadoraCommand(ctx context.Context, args []string, evt *events.Message, bot *BotClient) error {
	return ch.handleActionCommand(ctx, args, evt, bot, "flying", "deu uma voadora em", "💥", "!voadora @usuario")
}

// handleBeijoCommand processa o comando !beijo
func (ch *CommandHandler) handleBeijoCommand(ctx context.Context, args []string, evt *events.Message, bot *BotClient) error {
	return ch.handleActionCommand(ctx, args, evt, bot, "kiss", "deu um beijo em", "💋", "!beijo @usuario")
}

// handleAbracoCommand processa o comando !abraco
func (ch *CommandHandler) handleAbracoCommand(ctx context.Context, args []string, evt *events.Message, bot *BotClient) error {
	return ch.handleActionCommand(ctx, args, evt, bot, "hug", "deu um abraço em", "🤗", "!abraco @usuario")
}

// handlePiadaCommand processa o comando !piada
func (ch *CommandHandler) handlePiadaCommand(ctx context.Context, evt *events.Message, bot *BotClient) error {
	// Verificar se o cliente Gemini está configurado
	if bot.geminiClient == nil {
		errorMsg := "❌ Gemini não está configurado. Configure a API key para usar este comando."
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		return err
	}

	// Enviar evento de "digitando"
	errTyping := bot.WAClient.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	if errTyping != nil {
		log.Warn().Err(errTyping).Msg("Erro ao enviar status de digitando")
	}

	// Carregar histórico de piadas anteriores
	jokesHistory, err := bot.chatContext.LoadJokesHistory(ctx, 50) // Últimas 50 piadas
	if err != nil {
		log.Warn().Err(err).Msg("Erro ao carregar histórico de piadas, continuando sem histórico")
		jokesHistory = []string{}
	}

	// Formatar histórico de piadas
	historyText := FormatJokesHistory(jokesHistory)

	// Criar prompt para gerar piada
	basePrompt := `Você é um comediante descontraído. Conte uma piada curta e engraçada em português brasileiro.

Requisitos:
- A piada deve ser curta (máximo 3-4 frases)
- Deve ser engraçada e adequada para todos os públicos
- Use linguagem natural e descontraída
- Pode ser uma piada de qualquer tipo (trocadilho, situação, etc.)
- NÃO use emojis
- Responda APENAS com a piada, sem explicações ou comentários adicionais`

	// Combinar prompt base com histórico
	prompt := basePrompt + historyText + "\n\nConte a piada agora:"

	log.Info().
		Int("historySize", len(jokesHistory)).
		Msg("Gerando piada com Gemini (com histórico)")

	// Gerar piada usando a API do Gemini
	piada, err := bot.geminiClient.GenerateContent(ctx, prompt)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao gerar piada com Gemini")

		// Encerrar status de digitando
		bot.WAClient.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

		// Informar erro ao usuário
		errorMsg := "❌ Erro ao gerar piada. Tente novamente mais tarde."
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		return err
	}

	// Limitar tamanho da piada
	if len(piada) > 500 {
		piada = piada[:500] + "..."
	}

	// Salvar piada no histórico antes de enviar
	err = bot.chatContext.SaveJoke(ctx, piada)
	if err != nil {
		log.Warn().Err(err).Msg("Erro ao salvar piada no histórico, mas continuando")
		// Não retornar erro aqui, pois a piada já foi gerada
	}

	// Encerrar status de digitando
	bot.WAClient.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

	// Enviar piada gerada
	piadaMsg := fmt.Sprintf("😄 *Piada:*\n\n%s", piada)
	msg := &waProto.Message{
		Conversation: &piadaMsg,
	}
	_, err = bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao enviar piada")
		return err
	}

	log.Info().
		Int("length", len(piada)).
		Int("historySize", len(jokesHistory)).
		Msg("Piada enviada e salva no histórico com sucesso")

	return nil
}

// extractMentionInfo extrai informações de menção de uma mensagem
func (ch *CommandHandler) extractMentionInfo(mentionText string, evt *events.Message) (string, string) {
	// Verificar se há informações de contexto da mensagem
	if evt.Message.GetExtendedTextMessage() == nil || evt.Message.GetExtendedTextMessage().GetContextInfo() == nil {
		// Sem informações de contexto, retornar apenas o nome
		return "", strings.TrimPrefix(mentionText, "@")
	}

	contextInfo := evt.Message.GetExtendedTextMessage().GetContextInfo()
	mentionedJIDs := contextInfo.GetMentionedJID()

	if len(mentionedJIDs) == 0 {
		// Sem JIDs mencionados, retornar apenas o nome
		return "", strings.TrimPrefix(mentionText, "@")
	}

	// Para simplificar, pegar o primeiro JID mencionado
	// Em uma implementação mais robusta, mapearia o JID correto baseado na posição da menção
	targetJID := mentionedJIDs[0]
	targetName := strings.TrimPrefix(mentionText, "@")

	return targetJID, targetName
}

// handleHelpCommand mostra a lista de comandos disponíveis
func (ch *CommandHandler) handleHelpCommand(ctx context.Context, evt *events.Message, bot *BotClient) error {
	helpMsg := `*🤖 Comandos Disponíveis:*

• *!tapa @usuario* - Dar um tapa virtual em alguém com GIF
• *!chute @usuario* - Dar um chute virtual em alguém com GIF
• *!voadora @usuario* - Dar uma voadora virtual em alguém com GIF
• *!beijo @usuario* - Dar um beijo virtual em alguém com GIF
• *!abraco @usuario* - Dar um abraço virtual em alguém com GIF
• *!piada* - Contar uma piada gerada por IA
• *!explique* - Explicar uma mensagem marcada (marque uma mensagem e digite !explique)
• *!help* ou *!ajuda* - Mostrar esta lista de comandos

_Exemplos:_
• !tapa @amigo
• !chute @amigo
• !beijo @amigo
• !abraco @amigo
• !piada
• Marque uma mensagem e digite: !explique
• !help`

	msg := &waProto.Message{
		Conversation: &helpMsg,
	}
	_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
	return err
}

// searchLocalGIF busca um GIF aleatório em uma pasta específica
func (ch *CommandHandler) searchLocalGIF(folder string) (string, error) {
	// Caminho para a pasta de GIFs
	gifDir := filepath.Join("static/gif", folder)

	// Listar arquivos na pasta
	files, err := ioutil.ReadDir(gifDir)
	if err != nil {
		return "", fmt.Errorf("erro ao ler diretório de GIFs: %w", err)
	}

	// Filtrar apenas arquivos .mp4
	var gifFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".mp4") {
			gifFiles = append(gifFiles, file.Name())
		}
	}

	// Verificar se há GIFs disponíveis
	if len(gifFiles) == 0 {
		return "", fmt.Errorf("nenhum arquivo GIF encontrado na pasta %s", gifDir)
	}

	// Selecionar GIF aleatório
	randomIndex := rand.Intn(len(gifFiles))
	selectedGIF := gifFiles[randomIndex]

	// Retornar caminho completo do arquivo
	return filepath.Join(gifDir, selectedGIF), nil
}

// searchLocalSlapGIF busca um GIF aleatório na pasta local de slaps
func (ch *CommandHandler) searchLocalSlapGIF() (string, error) {
	return ch.searchLocalGIF("slap")
}

// sendMentionMessage envia uma mensagem de texto com menção
func (ch *CommandHandler) sendMentionMessage(ctx context.Context, text, targetJID string, evt *events.Message, bot *BotClient) error {
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: &text,
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: []string{targetJID},
			},
		},
	}

	_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
	return err
}

// sendGIFMessage envia uma mensagem com GIF local
func (ch *CommandHandler) sendGIFMessage(ctx context.Context, gifPath, caption string, evt *events.Message, bot *BotClient) error {
	// Ler o arquivo GIF
	gifData, err := ioutil.ReadFile(gifPath)
	if err != nil {
		log.Warn().Err(err).Str("path", gifPath).Msg("Erro ao ler arquivo GIF")
		// Fallback: enviar apenas texto
		message := fmt.Sprintf("%s\n\n[GIF indisponível]", caption)
		msg := &waProto.Message{
			Conversation: &message,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		return err
	}

	filename := filepath.Base(gifPath)

	// Tentar upload do arquivo
	log.Info().
		Str("gif", filename).
		Int("size", len(gifData)).
		Msg("Iniciando upload do GIF")

	uploadResp, err := bot.WAClient.Upload(ctx, gifData, whatsmeow.MediaVideo)
	if err != nil {
		log.Error().Err(err).Str("path", gifPath).Int("size", len(gifData)).Msg("Erro ao fazer upload do GIF")
		// Fallback: enviar apenas texto
		message := fmt.Sprintf("%s\n\n[GIF: %s]", caption, filename)
		msg := &waProto.Message{
			Conversation: &message,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		return err
	}

	log.Info().
		Str("gif", filename).
		Str("url", uploadResp.URL).
		Uint64("fileLength", uploadResp.FileLength).
		Msg("Upload do GIF concluído com sucesso")

	// Criar mensagem com o GIF anexado como vídeo (GIFs são enviados como VideoMessage com GifPlayback=true)
	msg := &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			Mimetype:      proto.String("video/mp4"),
			FileLength:    proto.Uint64(uploadResp.FileLength),
			MediaKey:      uploadResp.MediaKey,
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			GifPlayback:   proto.Bool(true),
			Width:         proto.Uint32(500), // Largura padrão (necessário)
			Height:        proto.Uint32(500), // Altura padrão (necessário)
		},
	}

	// Tentar enviar o GIF primeiro
	log.Info().
		Str("gif", filename).
		Str("chat", evt.Info.Chat.String()).
		Msg("Enviando GIF como VideoMessage")

	_, err = bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
	if err != nil {
		log.Error().
			Err(err).
			Str("gif", filename).
			Str("url", uploadResp.URL).
			Str("directPath", uploadResp.DirectPath).
			Uint64("fileLength", uploadResp.FileLength).
			Msg("Erro detalhado ao enviar GIF")
		// Fallback: tentar enviar apenas a mensagem de texto
		textMsg := &waProto.Message{
			Conversation: &caption,
		}
		_, fallbackErr := bot.WAClient.SendMessage(ctx, evt.Info.Chat, textMsg)
		if fallbackErr != nil {
			log.Error().Err(fallbackErr).Msg("Erro ao enviar mensagem de fallback")
		}
		return err
	}

	log.Info().
		Str("gif", filename).
		Msg("GIF enviado com sucesso como VideoMessage")

	// Enviar a mensagem de texto separadamente após o GIF
	if caption != "" {
		textMsg := &waProto.Message{
			Conversation: &caption,
		}
		_, err = bot.WAClient.SendMessage(ctx, evt.Info.Chat, textMsg)
		if err != nil {
			log.Warn().Err(err).Msg("Erro ao enviar legenda do GIF, mas GIF foi enviado")
			// Não retornar erro aqui, pois o GIF já foi enviado
		}
	}

	log.Info().
		Str("gif", filename).
		Int("size", len(gifData)).
		Msg("GIF enviado com sucesso")

	return nil
}

// sendGIFMessageWithMention envia uma mensagem com GIF local e menção
func (ch *CommandHandler) sendGIFMessageWithMention(ctx context.Context, gifPath, caption, targetJID string, evt *events.Message, bot *BotClient) error {
	// Ler o arquivo GIF
	gifData, err := ioutil.ReadFile(gifPath)
	if err != nil {
		log.Warn().Err(err).Str("path", gifPath).Msg("Erro ao ler arquivo GIF")
		// Fallback: enviar apenas texto com menção
		return ch.sendMentionMessage(ctx, fmt.Sprintf("%s\n\n[GIF indisponível]", caption), targetJID, evt, bot)
	}

	filename := filepath.Base(gifPath)

	// Tentar upload do arquivo
	log.Info().
		Str("gif", filename).
		Int("size", len(gifData)).
		Str("mentioned", targetJID).
		Msg("Iniciando upload do GIF com menção")

	uploadResp, err := bot.WAClient.Upload(ctx, gifData, whatsmeow.MediaVideo)
	if err != nil {
		log.Error().Err(err).Str("path", gifPath).Int("size", len(gifData)).Msg("Erro ao fazer upload do GIF")
		// Fallback: enviar apenas texto com menção
		return ch.sendMentionMessage(ctx, fmt.Sprintf("%s\n\n[GIF indisponível]", caption), targetJID, evt, bot)
	}

	log.Info().
		Str("gif", filename).
		Str("url", uploadResp.URL).
		Uint64("fileLength", uploadResp.FileLength).
		Msg("Upload do GIF concluído com sucesso")

	// Criar mensagem com o GIF anexado como vídeo (GIFs são enviados como VideoMessage com GifPlayback=true)
	msg := &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			Mimetype:      proto.String("video/mp4"),
			FileLength:    proto.Uint64(uploadResp.FileLength),
			MediaKey:      uploadResp.MediaKey,
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			GifPlayback:   proto.Bool(true),
			Width:         proto.Uint32(500), // Largura padrão (necessário)
			Height:        proto.Uint32(500), // Altura padrão (necessário)
		},
	}

	// Enviar o GIF primeiro
	log.Info().
		Str("gif", filename).
		Str("chat", evt.Info.Chat.String()).
		Str("mentioned", targetJID).
		Msg("Enviando GIF como VideoMessage com menção")

	_, err = bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
	if err != nil {
		log.Error().
			Err(err).
			Str("gif", filename).
			Str("url", uploadResp.URL).
			Str("directPath", uploadResp.DirectPath).
			Uint64("fileLength", uploadResp.FileLength).
			Str("mentioned", targetJID).
			Msg("Erro detalhado ao enviar GIF com menção")
		// Fallback: tentar enviar apenas a mensagem com menção
		return ch.sendMentionMessage(ctx, caption, targetJID, evt, bot)
	}

	log.Info().
		Str("gif", filename).
		Str("mentioned", targetJID).
		Msg("GIF enviado com sucesso como VideoMessage")

	// Enviar a mensagem com menção separadamente após o GIF
	if caption != "" {
		if targetJID != "" {
			// Enviar mensagem com menção
			err = ch.sendMentionMessage(ctx, caption, targetJID, evt, bot)
			if err != nil {
				log.Warn().Err(err).Msg("Erro ao enviar legenda com menção do GIF, mas GIF foi enviado")
				// Não retornar erro aqui, pois o GIF já foi enviado
			}
		} else {
			// Enviar mensagem simples sem menção
			textMsg := &waProto.Message{
				Conversation: &caption,
			}
			_, err = bot.WAClient.SendMessage(ctx, evt.Info.Chat, textMsg)
			if err != nil {
				log.Warn().Err(err).Msg("Erro ao enviar legenda do GIF, mas GIF foi enviado")
				// Não retornar erro aqui, pois o GIF já foi enviado
			}
		}
	}

	log.Info().
		Str("gif", filename).
		Int("size", len(gifData)).
		Str("mentioned", targetJID).
		Msg("GIF com menção enviado com sucesso via sendGIFMessageWithMention")

	return nil
}

// GroupRules define regras específicas para cada grupo
type GroupRules struct {
	GroupJID         string    `json:"group_jid"`
	AllowedUsers     []string  `json:"allowed_users"`     // Usuários autorizados (JIDs)
	BlockedUsers     []string  `json:"blocked_users"`     // Usuários bloqueados
	EnableAI         bool      `json:"enable_ai"`         // Se IA está habilitada para o grupo
	MaxMessages      int       `json:"max_messages"`      // Máximo de mensagens no contexto
	RequireMention   bool      `json:"require_mention"`   // Se requer menção para responder
	CustomPrompt     string    `json:"custom_prompt"`     // Prompt personalizado para o grupo
	ResponseCooldown int       `json:"response_cooldown"` // Cooldown entre respostas (segundos)
	LastResponse     time.Time `json:"last_response"`     // Última resposta enviada
}

// NewGroupMessageProcessor cria um novo processador de mensagens de grupo
func NewGroupMessageProcessor(bot *BotClient) *GroupMessageProcessor {
	return &GroupMessageProcessor{
		bot:            bot,
		groupRules:     make(map[string]*GroupRules),
		commandHandler: NewCommandHandler(),
	}
}

// ProcessGroupMessage processa uma mensagem recebida de um grupo
func (gmp *GroupMessageProcessor) ProcessGroupMessage(ctx context.Context, evt *events.Message, msgText string) error {
	groupJID := evt.Info.Chat.String()

	// Verificar se existem regras para este grupo
	rules := gmp.getGroupRules(groupJID)

	// Verificar se é um comando
	if strings.HasPrefix(msgText, "!") {
		return gmp.processCommand(ctx, evt, msgText, rules)
	}

	// Verificar permissões do usuário
	if !gmp.isUserAllowed(evt.Info.Sender.String(), rules) {
		log.Info().
			Str("group", groupJID).
			Str("user", evt.Info.Sender.String()).
			Msg("Usuário não autorizado tentou interagir com o grupo")
		return nil
	}

	// Verificar cooldown
	if !gmp.canRespond(rules) {
		log.Info().
			Str("group", groupJID).
			Msg("Cooldown ativo, ignorando mensagem")
		return nil
	}

	// Verificar se IA está habilitada para o grupo
	if !rules.EnableAI {
		log.Info().
			Str("group", groupJID).
			Msg("IA desabilitada para este grupo")
		return nil
	}

	// Verificar se o bot foi mencionado ou se a mensagem é uma resposta ao bot
	botMentioned := gmp.isMentioned(evt, msgText)
	botQuoted := gmp.isBotQuoted(evt)

	// Se o bot foi mencionado ou citado, SEMPRE processar com IA (ignora RequireMention)
	if botMentioned || botQuoted {
		log.Info().
			Str("group", groupJID).
			Str("user", evt.Info.Sender.String()).
			Str("message", msgText).
			Bool("mentioned", botMentioned).
			Bool("quoted", botQuoted).
			Msg("Bot mencionado ou citado, processando com IA")
		
		return gmp.processWithAI(ctx, evt, msgText, rules)
	}

	// Se RequireMention está ativo e bot não foi mencionado, ignorar
	if rules.RequireMention {
		log.Info().
			Str("group", groupJID).
			Msg("Mensagem não menciona o bot e RequireMention está ativo, ignorando")
		return nil
	}

	// Se não requer menção, processar normalmente com IA
	log.Info().
		Str("group", groupJID).
		Str("user", evt.Info.Sender.String()).
		Msg("Processando mensagem com IA (RequireMention desativado)")
	
	return gmp.processWithAI(ctx, evt, msgText, rules)
}

// processCommand processa comandos especiais
func (gmp *GroupMessageProcessor) processCommand(ctx context.Context, evt *events.Message, msgText string, rules *GroupRules) error {
	// Parsear comando e argumentos
	parts := strings.Fields(msgText)
	if len(parts) == 0 {
		return nil
	}

	command := strings.TrimPrefix(parts[0], "!")
	args := parts[1:]

	log.Info().
		Str("command", command).
		Strs("args", args).
		Str("group", evt.Info.Chat.String()).
		Str("user", evt.Info.Sender.String()).
		Msg("Comando recebido")

	// Processar comando
	return gmp.commandHandler.ProcessCommand(ctx, command, args, evt, gmp.bot)
}

// getGroupRules obtém ou cria regras padrão para um grupo
func (gmp *GroupMessageProcessor) getGroupRules(groupJID string) *GroupRules {
	if rules, exists := gmp.groupRules[groupJID]; exists {
		return rules
	}

	// Criar regras padrão
	defaultRules := &GroupRules{
		GroupJID:         groupJID,
		AllowedUsers:     []string{}, // Vazio = todos permitidos
		BlockedUsers:     []string{},
		EnableAI:         true, // IA habilitada por padrão
		MaxMessages:      50,   // Menos mensagens que chats privados
		RequireMention:   true, // Requer menção em grupos
		CustomPrompt:     "",
		ResponseCooldown: 30,                           // 30 segundos entre respostas
		LastResponse:     time.Now().Add(-time.Minute), // Permitir resposta imediata
	}

	gmp.groupRules[groupJID] = defaultRules
	return defaultRules
}

// isUserAllowed verifica se um usuário tem permissão para interagir
func (gmp *GroupMessageProcessor) isUserAllowed(userJID string, rules *GroupRules) bool {
	// Verificar se está na lista de bloqueados
	for _, blocked := range rules.BlockedUsers {
		if blocked == userJID {
			return false
		}
	}

	// Se lista de permitidos estiver vazia, todos são permitidos
	if len(rules.AllowedUsers) == 0 {
		return true
	}

	// Verificar se está na lista de permitidos
	for _, allowed := range rules.AllowedUsers {
		if allowed == userJID {
			return true
		}
	}

	return false
}

// canRespond verifica se pode responder baseado no cooldown
func (gmp *GroupMessageProcessor) canRespond(rules *GroupRules) bool {
	return time.Since(rules.LastResponse) > time.Duration(rules.ResponseCooldown)*time.Second
}

// isMentioned verifica se o bot foi mencionado na mensagem
func (gmp *GroupMessageProcessor) isMentioned(evt *events.Message, msgText string) bool {
	botJID := gmp.bot.WAClient.Store.ID.ToNonAD().String()

	// Verificar menções diretas (@bot) em ExtendedTextMessage
	if extended := evt.Message.GetExtendedTextMessage(); extended != nil && extended.ContextInfo != nil {
		mentionedJIDs := extended.ContextInfo.GetMentionedJID()
		for _, mentioned := range mentionedJIDs {
			if mentioned == botJID {
				return true
			}
		}
	}

	// Verificar menções em ImageMessage
	if imageMsg := evt.Message.GetImageMessage(); imageMsg != nil && imageMsg.ContextInfo != nil {
		mentionedJIDs := imageMsg.ContextInfo.GetMentionedJID()
		for _, mentioned := range mentionedJIDs {
			if mentioned == botJID {
				return true
			}
		}
	}

	// Verificar menções em VideoMessage
	if videoMsg := evt.Message.GetVideoMessage(); videoMsg != nil && videoMsg.ContextInfo != nil {
		mentionedJIDs := videoMsg.ContextInfo.GetMentionedJID()
		for _, mentioned := range mentionedJIDs {
			if mentioned == botJID {
				return true
			}
		}
	}

	// Verificar menção por nome no texto (fallback)
	botNames := []string{"ducker", "duckeria", "botia", "bot"}
	msgTextLower := strings.ToLower(msgText)
	for _, botName := range botNames {
		if strings.Contains(msgTextLower, "@"+botName) || 
		   (strings.Contains(msgTextLower, botName) && len(msgText) < 100) { // Evitar falsos positivos em textos longos
			return true
		}
	}

	return false
}

// isBotQuoted verifica se a mensagem é uma resposta/citação ao bot
func (gmp *GroupMessageProcessor) isBotQuoted(evt *events.Message) bool {
	// Verificar se há mensagem citada e se é do bot
	if extended := evt.Message.GetExtendedTextMessage(); extended != nil {
		if extended.ContextInfo != nil {
			// Verificar se há mensagem citada
			if extended.ContextInfo.QuotedMessage != nil {
				// Verificar se a mensagem citada foi enviada pelo bot
				// O ContextInfo contém informações sobre quem enviou a mensagem citada
				if extended.ContextInfo.Participant != nil {
					quotedSenderJID := *extended.ContextInfo.Participant
					botJID := gmp.bot.WAClient.Store.ID.ToNonAD().String()
					if quotedSenderJID == botJID {
						return true
					}
				}
				// Alternativa: verificar pelo StanzaID se disponível
				// Mas a forma mais confiável é pelo Participant
			}
		}
	}

	// Verificar também em outros tipos de mensagem (ImageMessage, VideoMessage, etc.)
	// que podem ter ContextInfo com mensagem citada
	if imageMsg := evt.Message.GetImageMessage(); imageMsg != nil && imageMsg.ContextInfo != nil {
		if imageMsg.ContextInfo.Participant != nil {
			quotedSenderJID := *imageMsg.ContextInfo.Participant
			botJID := gmp.bot.WAClient.Store.ID.ToNonAD().String()
			if quotedSenderJID == botJID {
				return true
			}
		}
	}

	if videoMsg := evt.Message.GetVideoMessage(); videoMsg != nil && videoMsg.ContextInfo != nil {
		if videoMsg.ContextInfo.Participant != nil {
			quotedSenderJID := *videoMsg.ContextInfo.Participant
			botJID := gmp.bot.WAClient.Store.ID.ToNonAD().String()
			if quotedSenderJID == botJID {
				return true
			}
		}
	}

	return false
}

// processWithAI processa a mensagem usando IA
func (gmp *GroupMessageProcessor) processWithAI(ctx context.Context, evt *events.Message, msgText string, rules *GroupRules) error {
	// Verificar se o cliente Gemini está configurado
	if gmp.bot.geminiClient == nil {
		log.Warn().Msg("Gemini client não configurado, ignorando mensagem de grupo")
		return nil
	}

	// Enviar evento de "digitando"
	errTyping := gmp.bot.WAClient.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	if errTyping != nil {
		log.Warn().Err(errTyping).Msg("Erro ao enviar status de digitando em grupo")
	}

	// Carregar histórico do grupo (limitado)
	groupHistory, err := gmp.bot.chatContext.LoadGroupMessages(ctx, rules.GroupJID, rules.MaxMessages)
	if err != nil {
		log.Error().Err(err).Str("group", rules.GroupJID).Msg("Erro ao carregar histórico do grupo")
		groupHistory = []ChatMessage{}
	}

	// Salvar mensagem do usuário
	err = gmp.bot.chatContext.SaveMessage(ctx, rules.GroupJID, "user", fmt.Sprintf("%s: %s", evt.Info.Sender.User, msgText))
	if err != nil {
		log.Error().Err(err).Str("group", rules.GroupJID).Msg("Erro ao salvar mensagem do grupo")
	}

	// Criar prompt para grupo
	prompt := gmp.createGroupPrompt(rules, groupHistory, msgText, evt.Info.Sender.User)

	// Gerar resposta com Gemini
	response, err := gmp.bot.geminiClient.GenerateContent(ctx, prompt)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao gerar resposta para grupo")

		errorMsg := "❌ Erro ao processar solicitação no grupo."
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		gmp.bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		return err
	}

	// Limitar tamanho da resposta
	if len(response) > 2000 { // Respostas menores em grupos
		response = response[:2000] + "\n\n... (resposta truncada)"
	}

	// Salvar resposta da IA
	err = gmp.bot.chatContext.SaveMessage(ctx, rules.GroupJID, "assistant", response)
	if err != nil {
		log.Error().Err(err).Str("group", rules.GroupJID).Msg("Erro ao salvar resposta da IA no grupo")
	}

	// Atualizar timestamp da última resposta
	rules.LastResponse = time.Now()

	// Enviar resposta
	responseMsg := fmt.Sprintf("🤖 %s", response)
	msg := &waProto.Message{
		Conversation: &responseMsg,
	}
	_, err = gmp.bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao enviar resposta para grupo")
		return err
	}

	// Encerrar status de digitando
	gmp.bot.WAClient.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

	log.Info().
		Str("group", rules.GroupJID).
		Str("user", evt.Info.Sender.String()).
		Int("contextSize", len(groupHistory)).
		Int("responseLength", len(response)).
		Msg("Resposta enviada para grupo")

	return nil
}

// createGroupPrompt cria o prompt personalizado para mensagens de grupo
func (gmp *GroupMessageProcessor) createGroupPrompt(rules *GroupRules, history []ChatMessage, userMessage, userName string) string {
	systemPrompt := rules.CustomPrompt
	if systemPrompt == "" {
		// Prompt padrão para grupos
		systemPrompt = `Você é o DuckerIA, assistente virtual da Hyper Ducker em um grupo de WhatsApp.

## Sua Identidade em Grupos
- Você está participando de uma conversa em grupo
- Seja mais conciso e direto que em chats privados
- Responda apenas quando for relevante ou quando mencionado
- Mantenha um tom amigável mas profissional

## Comportamento em Grupos
- Responda de forma objetiva e útil
- Evite respostas muito longas (máximo 2000 caracteres)
- Seja respeitoso com todos os membros
- Não faça spam ou respostas desnecessárias
- Foque em ajudar com dúvidas sobre desenvolvimento de apps

## Regras Importantes
- NÃO use emojis
- Seja direto ao ponto
- Responda apenas se for mencionado ou se a pergunta for claramente direcionada a você
- Mantenha a conversa produtiva

## Contexto da Conversa
A conversa atual do grupo está abaixo:`
	}

	// Formatar histórico do grupo
	conversationHistory := FormatConversationHistory(history)

	return fmt.Sprintf("%s\n\n%s\n\n**%s:** %s\n\nResponda de forma útil e concisa, considerando o contexto do grupo.",
		systemPrompt, conversationHistory, userName, userMessage)
}

// SetGroupRules define regras específicas para um grupo
func (gmp *GroupMessageProcessor) SetGroupRules(groupJID string, rules *GroupRules) {
	rules.GroupJID = groupJID
	gmp.groupRules[groupJID] = rules
}

// GetGroupRules obtém as regras atuais de um grupo
func (gmp *GroupMessageProcessor) GetGroupRules(groupJID string) *GroupRules {
	return gmp.getGroupRules(groupJID)
}

// AddAllowedUser adiciona um usuário à lista de permitidos
func (gmp *GroupMessageProcessor) AddAllowedUser(groupJID, userJID string) {
	rules := gmp.getGroupRules(groupJID)
	rules.AllowedUsers = append(rules.AllowedUsers, userJID)
}

// RemoveAllowedUser remove um usuário da lista de permitidos
func (gmp *GroupMessageProcessor) RemoveAllowedUser(groupJID, userJID string) {
	rules := gmp.getGroupRules(groupJID)
	for i, user := range rules.AllowedUsers {
		if user == userJID {
			rules.AllowedUsers = append(rules.AllowedUsers[:i], rules.AllowedUsers[i+1:]...)
			break
		}
	}
}

// BlockUser adiciona um usuário à lista de bloqueados
func (gmp *GroupMessageProcessor) BlockUser(groupJID, userJID string) {
	rules := gmp.getGroupRules(groupJID)
	rules.BlockedUsers = append(rules.BlockedUsers, userJID)
}

// UnblockUser remove um usuário da lista de bloqueados
func (gmp *GroupMessageProcessor) UnblockUser(groupJID, userJID string) {
	rules := gmp.getGroupRules(groupJID)
	for i, user := range rules.BlockedUsers {
		if user == userJID {
			rules.BlockedUsers = append(rules.BlockedUsers[:i], rules.BlockedUsers[i+1:]...)
			break
		}
	}
}

// EnableAI habilita a IA para um grupo
func (gmp *GroupMessageProcessor) EnableAI(groupJID string) {
	rules := gmp.getGroupRules(groupJID)
	rules.EnableAI = true
}

// DisableAI desabilita a IA para um grupo
func (gmp *GroupMessageProcessor) DisableAI(groupJID string) {
	rules := gmp.getGroupRules(groupJID)
	rules.EnableAI = false
}

// SetCustomPrompt define um prompt personalizado para o grupo
func (gmp *GroupMessageProcessor) SetCustomPrompt(groupJID, prompt string) {
	rules := gmp.getGroupRules(groupJID)
	rules.CustomPrompt = prompt
}
