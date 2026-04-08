package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"stock-monitor/internal/types"
	"sync"
)

// PortfolioRepository 投资组合仓储
type PortfolioRepository struct {
	filePath string
	mu       sync.RWMutex
	cache    map[string]*types.Stock // 内存缓存，加速查询
}

// NewPortfolioRepository 创建投资组合仓储
func NewPortfolioRepository(dataDir string) *PortfolioRepository {
	return &PortfolioRepository{
		filePath: filepath.Join(dataDir, "portfolio.json"),
		cache:    make(map[string]*types.Stock),
	}
}

// Load 加载投资组合
func (r *PortfolioRepository) Load() ([]*types.Stock, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，返回空列表
			return []*types.Stock{}, nil
		}
		return nil, fmt.Errorf("failed to read portfolio file: %w", err)
	}

	var portfolio types.Portfolio
	if err := json.Unmarshal(data, &portfolio); err != nil {
		return nil, fmt.Errorf("failed to parse portfolio: %w", err)
	}

	// 更新缓存
	r.cache = make(map[string]*types.Stock)
	stocks := make([]*types.Stock, len(portfolio.Stocks))
	for i := range portfolio.Stocks {
		stock := &portfolio.Stocks[i]
		stocks[i] = stock
		r.cache[stock.Code] = stock
	}

	return stocks, nil
}

// Save 保存投资组合
func (r *PortfolioRepository) Save(stocks []*types.Stock) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 转换为 Portfolio 结构
	portfolio := types.Portfolio{
		Stocks: make([]types.Stock, len(stocks)),
	}
	for i, stock := range stocks {
		portfolio.Stocks[i] = *stock
	}

	// 序列化
	data, err := json.MarshalIndent(portfolio, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal portfolio: %w", err)
	}

	// 原子写入（temp → sync → close → rename）
	dir := filepath.Dir(r.filePath)
	f, err := os.CreateTemp(dir, filepath.Base(r.filePath)+".tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temp portfolio file: %w", err)
	}
	tmpPath := f.Name()
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write portfolio file: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to sync portfolio file: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close portfolio file: %w", err)
	}
	if err := os.Rename(tmpPath, r.filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename portfolio file: %w", err)
	}

	// 更新缓存
	r.cache = make(map[string]*types.Stock)
	for _, stock := range stocks {
		r.cache[stock.Code] = stock
	}

	return nil
}

// GetStock 获取单个股票
func (r *PortfolioRepository) GetStock(code string) (*types.Stock, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stock, ok := r.cache[code]
	if !ok {
		return nil, fmt.Errorf("stock %s not found", code)
	}

	return stock, nil
}

// AddStock 添加股票
func (r *PortfolioRepository) AddStock(stock *types.Stock) error {
	stocks, err := r.Load()
	if err != nil {
		return err
	}

	// 检查是否已存在
	for _, s := range stocks {
		if s.Code == stock.Code {
			return fmt.Errorf("stock %s already exists", stock.Code)
		}
	}

	// 添加并保存
	stocks = append(stocks, stock)
	return r.Save(stocks)
}

// RemoveStock 移除股票
func (r *PortfolioRepository) RemoveStock(code string) error {
	stocks, err := r.Load()
	if err != nil {
		return err
	}

	// 查找并移除
	found := false
	newStocks := make([]*types.Stock, 0, len(stocks))
	for _, stock := range stocks {
		if stock.Code != code {
			newStocks = append(newStocks, stock)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("stock %s not found", code)
	}

	return r.Save(newStocks)
}

// UpdateStock 更新股票
func (r *PortfolioRepository) UpdateStock(stock *types.Stock) error {
	stocks, err := r.Load()
	if err != nil {
		return err
	}

	// 查找并更新
	found := false
	for i, s := range stocks {
		if s.Code == stock.Code {
			stocks[i] = stock
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("stock %s not found", stock.Code)
	}

	return r.Save(stocks)
}

// GetAll 获取所有股票
func (r *PortfolioRepository) GetAll() ([]*types.Stock, error) {
	return r.Load()
}

// Count 获取股票数量
func (r *PortfolioRepository) Count() (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.cache), nil
}

// Exists 检查股票是否存在
func (r *PortfolioRepository) Exists(code string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.cache[code]
	return ok
}

// Clear 清空投资组合
func (r *PortfolioRepository) Clear() error {
	return r.Save([]*types.Stock{})
}
