package bot

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Command interface {
	isValid(tgbotapi.Update, State) bool
}

type Handler func(*Manager, tgbotapi.Update)

type HandlerInfo struct {
	filter  interface{}
	state   State
	handler Handler
}

func (h HandlerInfo) isValid(update tgbotapi.Update, state State) bool {
	switch filter := h.filter.(type) {
	case string:
		return (h.state == state || h.state == AnyState) &&
			(strings.HasPrefix(update.Message.Text, (h.filter).(string)) || h.filter == "*")
	case ContentType:
		switch filter {
		case OnText:
			return (h.state == state || h.state == AnyState) && update.Message.Text != ""
		case OnPhoto:
			return (h.state == state || h.state == AnyState) && update.Message.Photo != nil
		case OnVideo:
			return (h.state == state || h.state == AnyState) && update.Message.Video != nil
		}
	}

	return false
}

type Binder func(*Manager, tgbotapi.Update) State

type BinderInfo struct {
	state  State
	binder Binder
}

func (b BinderInfo) isValid(update tgbotapi.Update, state State) bool {
	return b.state == state || b.state == AnyState
}

type processFunc func(tgbotapi.Update)
type middlewareFunc func(*Manager, tgbotapi.Update, processFunc) processFunc

type Manager struct {
	Bot
	State    State
	Data     Context
	commands []Command
}

func NewManager(bot Bot) *Manager {
	return &Manager{
		Bot:   bot,
		State: DefaultState,
		Data:  NewContext(),
	}
}

func (m *Manager) Run(middlewares ...middlewareFunc) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := m.GetUpdatesChan(u)

	for update := range updates {
		if middlewares != nil {
			process := m.process
			for _, middleware := range middlewares {
				process = middleware(m, update, process)
			}
			if process != nil {
				process(update)
			} else {
				continue
			}
		} else {
			m.process(update)
		}
	}
}

func (m *Manager) process(update tgbotapi.Update) {
	skipCommands := false

	for _, command := range m.commands {
		if !command.isValid(update, m.State) {
			continue
		}

		switch c := command.(type) {
		case HandlerInfo:
			if skipCommands {
				continue
			}

			c.handler(m, update)
			skipCommands = true
		case BinderInfo:
			m.SetState(c.binder(m, update))
		}
	}
	skipCommands = false
}

func (m *Manager) Handle(filter interface{}, state State, h Handler) {
	m.commands = append(m.commands, HandlerInfo{
		filter:  filter,
		state:   state,
		handler: h,
	})
}

func (m *Manager) Bind(state State, b Binder) {
	m.commands = append(m.commands, BinderInfo{
		state:  state,
		binder: b,
	})
}

func (m *Manager) SetState(state State) {
	m.State = state
}
