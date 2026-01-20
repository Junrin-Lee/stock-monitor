package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"stock-monitor/internal/consts"
)

// StateHandler 定义状态处理器函数签名
type StateHandler func(tea.Msg) (tea.Model, tea.Cmd)

// Router 状态路由器
type Router struct {
	handlers map[consts.AppState]StateHandler
}

// NewRouter 创建新的路由器
func NewRouter() *Router {
	return &Router{
		handlers: make(map[consts.AppState]StateHandler),
	}
}

// Register 注册状态处理器
func (r *Router) Register(state consts.AppState, handler StateHandler) {
	r.handlers[state] = handler
}

// Route 路由到对应的处理器
func (r *Router) Route(state consts.AppState, msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	if handler, ok := r.handlers[state]; ok {
		model, cmd := handler(msg)
		return model, cmd, true
	}
	return nil, nil, false
}

// HasHandler 检查是否存在处理器
func (r *Router) HasHandler(state consts.AppState) bool {
	_, ok := r.handlers[state]
	return ok
}
