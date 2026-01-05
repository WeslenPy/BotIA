package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

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
	case "cantada":
		return ch.handleCantadaCommand(ctx, args, evt, bot)
	case "historia", "história":
		return ch.handleHistoriaCommand(ctx, args, evt, bot)
	case "autodestruicao", "autodestruição":
		return ch.handleAutodestruicaoCommand(ctx, args, evt, bot)
	case "roletacasais", "roleta", "casais":
		return ch.handleRoletaCasaisCommand(ctx, evt, bot)
	case "help", "ajuda", "menu":
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

// handleCantadaCommand processa o comando !cantada
func (ch *CommandHandler) handleCantadaCommand(ctx context.Context, args []string, evt *events.Message, bot *BotClient) error {
	// Verificar se o cliente Gemini está configurado
	if bot.geminiClient == nil {
		errorMsg := "❌ Gemini não está configurado. Configure a API key para usar este comando."
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		return err
	}

	// Verificar se há um usuário mencionado
	if len(args) == 0 {
		errorMsg := "❌ Use: !cantada @usuario\nExemplo: !cantada @johndoe"
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		return err
	}

	// Extrair informações da menção
	targetMention := args[0]
	var targetJID string
	var targetName string

	if strings.HasPrefix(targetMention, "@") {
		targetJID, targetName = ch.extractMentionInfo(targetMention, evt)
	} else {
		targetName = targetMention
	}

	// Enviar evento de "digitando"
	errTyping := bot.WAClient.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	if errTyping != nil {
		log.Warn().Err(errTyping).Msg("Erro ao enviar status de digitando")
	}

	// Criar prompt para gerar cantada
	prompt := `Você é um especialista em criar cantadas criativas e engraçadas em português brasileiro.

Crie uma cantada 

Requisitos:
- A cantada deve ser criativa e engraçada
- Deve ser adequada para todos os públicos (sem conteúdo ofensivo ou inapropriado)
- Use linguagem natural e descontraída
- Pode ser romântica, engraçada ou criativa
- Máximo de 3-4 frases
- NÃO use emojis
- Responda APENAS com a cantada, sem explicações ou comentários adicionais
- A cantada deve ser direcionada à pessoa mencionada

Crie a cantada agora:`

	log.Info().
		Str("target", targetName).
		Str("targetJID", targetJID).
		Msg("Gerando cantada com Gemini")

	// Gerar cantada usando a API do Gemini
	cantada, err := bot.geminiClient.GenerateContent(ctx, prompt)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao gerar cantada com Gemini")

		// Encerrar status de digitando
		bot.WAClient.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

		// Informar erro ao usuário
		errorMsg := "❌ Erro ao gerar cantada. Tente novamente mais tarde."
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		return err
	}

	// Limitar tamanho da cantada
	if len(cantada) > 500 {
		cantada = cantada[:500] + "..."
	}

	// Encerrar status de digitando
	bot.WAClient.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

	// Enviar cantada gerada com menção ao usuário
	cantadaMsg := fmt.Sprintf("💕 *Cantada para @%s:*\n\n%s", targetName, cantada)

	// Se temos o JID, enviar com menção clicável
	if targetJID != "" {
		err = ch.sendMentionMessage(ctx, cantadaMsg, targetJID, evt, bot)
	} else {
		// Fallback: enviar sem menção clicável
		msg := &waProto.Message{
			Conversation: &cantadaMsg,
		}
		_, err = bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
	}

	if err != nil {
		log.Error().Err(err).Msg("Erro ao enviar cantada")
		return err
	}

	log.Info().
		Int("length", len(cantada)).
		Str("target", targetName).
		Msg("Cantada enviada com sucesso")

	return nil
}

// handleHistoriaCommand processa o comando !historia
func (ch *CommandHandler) handleHistoriaCommand(ctx context.Context, args []string, evt *events.Message, bot *BotClient) error {
	// Verificar se o cliente Gemini está configurado
	if bot.geminiClient == nil {
		errorMsg := "❌ Gemini não está configurado. Configure a API key para usar este comando."
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		return err
	}

	// Extrair tipo da história dos argumentos
	historiaTipo := "aventura" // Tipo padrão
	if len(args) > 0 {
		historiaTipo = strings.ToLower(strings.Join(args, " "))
	}

	// Enviar evento de "digitando"
	errTyping := bot.WAClient.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	if errTyping != nil {
		log.Warn().Err(errTyping).Msg("Erro ao enviar status de digitando")
	}

	// Criar prompt para gerar história
	prompt := fmt.Sprintf(`Você é um contador de histórias criativo e envolvente em português brasileiro.

Crie uma história do gênero: %s

Requisitos:
- A história deve ser do gênero %s
- Deve ser envolvente e interessante
- Deve ter começo, meio e fim
- Use linguagem natural e fluida
- Seja criativo e original
- A história deve ter entre 5 e 10 parágrafos
- NÃO use emojis
- Responda APENAS com a história, sem explicações ou comentários adicionais
- Se for terror, mantenha o suspense mas seja adequado para todos os públicos
- Se for comédia, seja engraçada mas respeitosa
- Se for romance, seja romântica mas discreta
- Se for aventura, seja emocionante e dinâmica
- Se for ficção científica, seja criativa e interessante

Crie a história agora:`, historiaTipo, historiaTipo)

	log.Info().
		Str("tipo", historiaTipo).
		Msg("Gerando história com Gemini")

	// Gerar história usando a API do Gemini
	historia, err := bot.geminiClient.GenerateContent(ctx, prompt)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao gerar história com Gemini")

		// Encerrar status de digitando
		bot.WAClient.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

		// Informar erro ao usuário
		errorMsg := "❌ Erro ao gerar história. Tente novamente mais tarde."
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		return err
	}

	// Limitar tamanho da história (histórias podem ser mais longas)
	if len(historia) > 3000 {
		historia = historia[:3000] + "\n\n... (história truncada)"
	}

	// Encerrar status de digitando
	bot.WAClient.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

	// Enviar história gerada
	historiaMsg := fmt.Sprintf("📖 *História de %s:*\n\n%s", strings.Title(historiaTipo), historia)
	msg := &waProto.Message{
		Conversation: &historiaMsg,
	}
	_, err = bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao enviar história")
		return err
	}

	log.Info().
		Int("length", len(historia)).
		Str("tipo", historiaTipo).
		Msg("História enviada com sucesso")

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

// handleAutodestruicaoCommand processa o comando de auto-destruição
func (ch *CommandHandler) handleAutodestruicaoCommand(ctx context.Context, args []string, evt *events.Message, bot *BotClient) error {
	// Verificar se é um grupo
	if evt.Info.Chat.Server != "g.us" {
		errorMsg := "❌ Este comando só funciona em grupos!"
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		return err
	}

	// Parsear minutos (padrão: 5 minutos)
	minutes := 5
	if len(args) > 0 {
		parsedMinutes, err := strconv.Atoi(args[0])
		if err == nil && parsedMinutes > 0 {
			minutes = parsedMinutes
		}
	}

	// Validar limites (1 a 60 minutos)
	if minutes < 1 {
		minutes = 1
	}
	if minutes > 60 {
		minutes = 60
	}

	groupJID := evt.Info.Chat.String()
	groupProcessor := bot.groupProcessor

	// Verificar se já está pausado
	rules := groupProcessor.GetGroupRules(groupJID)
	if rules.IsPaused && time.Now().Before(rules.PausedUntil) {
		remaining := time.Until(rules.PausedUntil)
		remainingMinutes := int(remaining.Minutes())
		if remainingMinutes < 1 {
			remainingMinutes = 1
		}
		errorMsg := fmt.Sprintf("⚠️ Bot já está pausado! Reativa em %d minuto(s).", remainingMinutes)
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		return err
	}

	// Mensagem inicial
	initMsg := fmt.Sprintf("⚠️ *AUTO-DESTRUIÇÃO ATIVADA*\n\nBot será pausado por *%d minuto(s)*.\n\nIniciando countdown de 5 segundos...", minutes)
	msg := &waProto.Message{
		Conversation: &initMsg,
	}
	_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao enviar mensagem inicial de auto-destruição")
	}

	// Iniciar countdown de 5 segundos em goroutine
	go func() {
		ctx := context.Background()

		// Countdown de 5 segundos com emoji de explosão
		for i := 5; i > 0; i-- {
			time.Sleep(1 * time.Second)
			countdownMsg := fmt.Sprintf("💥 %d", i)
			msg := &waProto.Message{
				Conversation: &countdownMsg,
			}
			_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
			if err != nil {
				log.Error().Err(err).Int("countdown", i).Msg("Erro ao enviar mensagem de countdown")
			}
		}

		// Pausar o bot após o countdown
		duration := time.Duration(minutes) * time.Minute
		groupProcessor.PauseGroup(groupJID, duration)

		// Mensagem de pausa ativada
		pauseMsg := "💥 *Bot pausado!*\n\nBot ficará inativo por um período."
		msg = &waProto.Message{
			Conversation: &pauseMsg,
		}
		_, err = bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		if err != nil {
			log.Error().Err(err).Msg("Erro ao enviar mensagem de pausa")
		}

		// Aguardar o tempo de pausa
		time.Sleep(duration)

		// Verificar se ainda está pausado antes de reativar
		rules := groupProcessor.GetGroupRules(groupJID)
		if rules.IsPaused && time.Now().After(rules.PausedUntil) {
			// Reativar o bot
			groupProcessor.UnpauseGroup(groupJID)

			// Mensagem final
			finalMsg := "✅ *Bot reativado!*\n\nAuto-destruição concluída. Bot está funcionando normalmente novamente."
			msg = &waProto.Message{
				Conversation: &finalMsg,
			}
			_, err = bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
			if err != nil {
				log.Error().Err(err).Msg("Erro ao enviar mensagem final de reativação")
			}

			log.Info().Str("group", groupJID).Msg("Auto-destruição concluída, bot reativado")
		}
	}()

	return nil
}

// getParticipantName tenta obter o nome do participante de todas as formas possíveis
func (ch *CommandHandler) getParticipantName(ctx context.Context, jid types.JID, bot *BotClient) string {
	// Tentar obter o nome do contato do store
	contact, err := bot.WAClient.Store.Contacts.GetContact(ctx, jid)
	if err == nil {
		// Priorizar FullName
		if contact.FullName != "" {
			return contact.FullName
		}
		// Se não tiver FullName, usar PushName
		if contact.PushName != "" {
			return contact.PushName
		}
	}

	// Se não conseguiu obter do store, retornar string vazia
	// (não usar JID.User para evitar mostrar números)
	return ""
}

// isOnlyNumber verifica se a string contém apenas números
func (ch *CommandHandler) isOnlyNumber(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

// removeSpecialChars remove emojis e caracteres especiais, mantendo apenas letras, números, espaços e acentos
func (ch *CommandHandler) removeSpecialChars(s string) string {
	var result strings.Builder
	for _, r := range s {
		// Manter letras (incluindo acentos), números, espaços e alguns caracteres básicos
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) ||
			r == '-' || r == '_' || r == '.' || r == ',' || r == '!' || r == '?' {
			result.WriteRune(r)
		}
		// Ignorar emojis e outros caracteres especiais
	}
	return strings.TrimSpace(result.String())
}

// handleRoletaCasaisCommand processa o comando de roleta dos casais
func (ch *CommandHandler) handleRoletaCasaisCommand(ctx context.Context, evt *events.Message, bot *BotClient) error {
	// Verificar se é um grupo
	if evt.Info.Chat.Server != "g.us" {
		errorMsg := "❌ Este comando só funciona em grupos!"
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		return err
	}

	// Obter informações do grupo
	groupJID := evt.Info.Chat
	groupInfo, err := bot.WAClient.GetGroupInfo(ctx, groupJID)
	if err != nil {
		log.Error().Err(err).Str("group", groupJID.String()).Msg("Erro ao obter informações do grupo")
		errorMsg := "❌ Erro ao obter informações do grupo."
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		return err
	}

	// Obter lista de participantes (excluindo o bot)
	participants := []string{}
	botJID := bot.WAClient.Store.ID.ToNonAD().String()

	for _, participant := range groupInfo.Participants {
		participantJID := participant.JID.ToNonAD().String()
		// Excluir o bot da lista
		if participantJID != botJID {
			// Obter nome do participante - sempre tentar usar o nome, nunca o JID
			name := ch.getParticipantName(ctx, participant.JID, bot)
			// Só adicionar se tiver um nome válido (não vazio e não é apenas número)
			if name != "" && !ch.isOnlyNumber(name) {
				participants = append(participants, name)
			}
		}
	}

	// Verificar se há participantes suficientes
	if len(participants) < 2 {
		errorMsg := "❌ É necessário pelo menos 2 membros no grupo para formar um casal!"
		msg := &waProto.Message{
			Conversation: &errorMsg,
		}
		_, err := bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
		return err
	}

	// Selecionar 2 membros aleatórios para formar um casal
	rand.Seed(time.Now().UnixNano())

	// Selecionar primeiro membro aleatório
	index1 := rand.Intn(len(participants))
	membro1 := ch.removeSpecialChars(participants[index1])

	// Selecionar segundo membro aleatório (diferente do primeiro)
	index2 := rand.Intn(len(participants))
	for index2 == index1 {
		index2 = rand.Intn(len(participants))
	}
	membro2 := ch.removeSpecialChars(participants[index2])

	// Formar mensagem com o casal
	resultMsg := fmt.Sprintf("💕 *ROleta DOS CASAIS*\n\n💑 *%s* e *%s*", membro1, membro2)

	msg := &waProto.Message{
		Conversation: &resultMsg,
	}
	_, err = bot.WAClient.SendMessage(ctx, evt.Info.Chat, msg)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao enviar resultado da roleta dos casais")
		return err
	}

	log.Info().
		Str("group", groupJID.String()).
		Int("participants", len(participants)).
		Str("casal", fmt.Sprintf("%s e %s", membro1, membro2)).
		Msg("Roleta dos casais executada com sucesso")

	return nil
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
• *!cantada @usuario* - Gerar uma cantada para alguém usando IA
• *!historia [tipo]* - Gerar uma história usando IA (ex: !historia terror, !historia comedia)
• *!explique* - Explicar uma mensagem marcada (marque uma mensagem e digite !explique)
• *!autodestruicao [minutos]* - Pausar o bot por X minutos com countdown (padrão: 5 min, máximo: 60 min)
• *!roletacasais* ou *!roleta* - Formar casais aleatórios com os membros do grupo
• *!help* ou *!ajuda* - Mostrar esta lista de comandos

_Exemplos:_
• !tapa @amigo
• !chute @amigo
• !beijo @amigo
• !abraco @amigo
• !piada
• !cantada @amigo
• !historia terror
• !historia comedia
• Marque uma mensagem e digite: !explique
• !autodestruicao 10 (pausa por 10 minutos)
• !roletacasais (forma casais aleatórios)
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
	IsPaused         bool      `json:"is_paused"`         // Se o bot está pausado no grupo
	PausedUntil      time.Time `json:"paused_until"`      // Quando a pausa termina
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

	// Verificar se o bot está pausado (ignora TODAS as funções, incluindo comandos)
	if rules.IsPaused {
		// Verificar se a pausa já expirou
		if time.Now().After(rules.PausedUntil) {
			rules.IsPaused = false
			log.Info().Str("group", groupJID).Msg("Pausa expirada, bot reativado")
		} else {
			log.Info().
				Str("group", groupJID).
				Time("paused_until", rules.PausedUntil).
				Str("message", msgText).
				Msg("Bot pausado, ignorando todas as funções (comandos e mensagens)")
			return nil
		}
	}

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

	// Limitar tamanho da resposta (respostas curtas e diretas)
	if len(response) > 500 {
		response = response[:500] + "..."
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
		// Prompt padrão para grupos - direto, curto e natural
		systemPrompt = `Você é o DuckerIA, participando de um grupo de WhatsApp.

## Sua Personalidade
- Você é descontraído, amigável e natural
- Mantém um tom leve e acessível
- Você é parte do grupo, não apenas um assistente
- Use linguagem natural e coloquial

## Como Responder
- Seja DIRETO e OBJETIVO - vá direto ao ponto
- Respostas CURTAS (máximo 3-4 frases, idealmente 1-2)
- Responda apenas o que foi perguntado, sem enrolação
- Não force assuntos ou tente mudar o tema da conversa
- Se alguém perguntar sobre tecnologia, responda. Se não perguntar, não mencione
- Não fale sobre desenvolvimento, apps ou tecnologia a menos que seja o assunto da conversa
- Seja natural e participe da conversa como qualquer membro do grupo

## Estilo de Comunicação
- Respostas MUITO curtas e diretas (máximo 100 caracteres)
- Linguagem natural e conversacional
- Pode usar expressões maranhenses ocasionalmente (visse, rapaz/moça, tranquilo, beleza)
- Seja empático mas objetivo
- Quando apropriado, faça comentários leves e descontraídos
- NÃO use emojis

## Regras Importantes
- Seja respeitoso com todos
- Não seja formal ou robótico
- NÃO force assuntos de tecnologia
- NÃO tente vender ou promover nada
- Se não souber algo, seja honesto e direto
- Responda de forma natural, como se fosse um amigo no grupo

## Contexto da Conversa
A conversa atual do grupo está abaixo. Use apenas para entender o contexto, mas responda de forma DIRETA e CURTA:`
	}

	// Formatar histórico do grupo
	conversationHistory := FormatConversationHistory(history)

	return fmt.Sprintf("%s\n\n%s\n\n**%s:** %s\n\nResponda de forma DIRETA, CURTA e NATURAL. Vá direto ao ponto, sem enrolação. Não force assuntos de tecnologia.",
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

// PauseGroup pausa o bot em um grupo por um período determinado
func (gmp *GroupMessageProcessor) PauseGroup(groupJID string, duration time.Duration) {
	rules := gmp.getGroupRules(groupJID)
	rules.IsPaused = true
	rules.PausedUntil = time.Now().Add(duration)
	log.Info().
		Str("group", groupJID).
		Dur("duration", duration).
		Time("paused_until", rules.PausedUntil).
		Msg("Bot pausado no grupo")
}

// UnpauseGroup remove a pausa do bot em um grupo
func (gmp *GroupMessageProcessor) UnpauseGroup(groupJID string) {
	rules := gmp.getGroupRules(groupJID)
	rules.IsPaused = false
	rules.PausedUntil = time.Time{}
	log.Info().Str("group", groupJID).Msg("Bot despausado no grupo")
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
