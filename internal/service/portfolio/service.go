package portfolio

import (
	"fmt"
	"stock-monitor/internal/api"
	"stock-monitor/internal/data"
	apperrors "stock-monitor/internal/errors"
	"stock-monitor/internal/types"
)

// APIClient 接口定义（用于解耦）
type APIClient interface {
	GetStockInfo(code string) *types.StockData
	GetStockPrice(code string) *types.StockData
}

// apiClientAdapter 适配器，将包级函数包装为接口实现
type apiClientAdapter struct{}

func (a *apiClientAdapter) GetStockInfo(code string) *types.StockData {
	return api.GetStockInfo(code)
}

func (a *apiClientAdapter) GetStockPrice(code string) *types.StockData {
	return api.GetStockPrice(code)
}

// Service 投资组合服务
type Service struct {
	repo      *data.PortfolioRepository
	apiClient APIClient
	cache     *data.StockPriceCache
}

// NewService 创建投资组合服务
func NewService(repo *data.PortfolioRepository, apiClient APIClient, cache *data.StockPriceCache) *Service {
	if apiClient == nil {
		apiClient = &apiClientAdapter{}
	}
	return &Service{
		repo:      repo,
		apiClient: apiClient,
		cache:     cache,
	}
}

// AddStock 添加股票到投资组合
func (s *Service) AddStock(code string, costPrice float64, quantity int) (*types.Stock, error) {
	// 1. 验证输入
	if code == "" {
		return nil, apperrors.NewValidationError("stock code cannot be empty")
	}
	if costPrice <= 0 {
		return nil, apperrors.NewValidationError("cost price must be positive")
	}
	if quantity <= 0 {
		return nil, apperrors.NewValidationError("quantity must be positive")
	}

	// 2. 获取股票信息
	stockData := s.apiClient.GetStockInfo(code)
	if stockData == nil {
		return nil, apperrors.NewAPIError("failed to fetch stock info", nil).
			WithContext("code", code)
	}

	// 3. 创建股票对象
	stock := &types.Stock{
		Code:      stockData.Symbol,
		Name:      stockData.Name,
		Price:     stockData.Price,
		CostPrice: costPrice,
		Quantity:  quantity,
		PrevClose: stockData.PrevClose,
	}

	// 4. 保存到仓储
	if err := s.repo.AddStock(stock); err != nil {
		return nil, apperrors.NewDataError("failed to save stock", err)
	}

	return stock, nil
}

// RemoveStock 从投资组合移除股票
func (s *Service) RemoveStock(code string) error {
	if code == "" {
		return apperrors.NewValidationError("stock code cannot be empty")
	}

	if err := s.repo.RemoveStock(code); err != nil {
		return apperrors.NewDataError("failed to remove stock", err)
	}

	return nil
}

// UpdateStock 更新股票信息
func (s *Service) UpdateStock(code string, costPrice float64, quantity int) (*types.Stock, error) {
	// 验证
	if code == "" {
		return nil, apperrors.NewValidationError("stock code cannot be empty")
	}

	// 查找股票
	stock, err := s.repo.GetStock(code)
	if err != nil {
		return nil, apperrors.NewNotFoundError("stock")
	}

	// 更新字段
	if costPrice > 0 {
		stock.CostPrice = costPrice
	}
	if quantity > 0 {
		stock.Quantity = quantity
	}

	// 保存
	if err := s.repo.UpdateStock(stock); err != nil {
		return nil, apperrors.NewDataError("failed to update stock", err)
	}

	return stock, nil
}

// GetAllStocks 获取所有股票
func (s *Service) GetAllStocks() ([]*types.Stock, error) {
	stocks, err := s.repo.GetAll()
	if err != nil {
		return nil, apperrors.NewDataError("failed to load stocks", err)
	}
	return stocks, nil
}

// RefreshPrices 刷新所有股票价格
func (s *Service) RefreshPrices(stocks []*types.Stock) error {
	for _, stock := range stocks {
		// 先查缓存
		if cachedData := s.cache.Get(stock.Code); cachedData != nil {
			s.updateStockPrice(stock, cachedData)
			continue
		}

		// 缓存未命中，从 API 获取
		stockData := s.apiClient.GetStockPrice(stock.Code)
		if stockData != nil && stockData.Price > 0 {
			s.updateStockPrice(stock, stockData)
			s.cache.Set(stock.Code, stockData)
		}
	}

	return nil
}

// updateStockPrice 更新股票价格（内部方法）
func (s *Service) updateStockPrice(stock *types.Stock, data *types.StockData) {
	stock.Price = data.Price
	stock.Change = data.Change
	stock.ChangePercent = data.ChangePercent
	stock.StartPrice = data.StartPrice
	stock.MaxPrice = data.MaxPrice
	stock.MinPrice = data.MinPrice
	stock.PrevClose = data.PrevClose
}

// CalculateTotalProfit 计算总盈亏
func (s *Service) CalculateTotalProfit(stocks []*types.Stock) float64 {
	totalProfit := 0.0
	for _, stock := range stocks {
		if stock.Price > 0 {
			profit := (stock.Price - stock.CostPrice) * float64(stock.Quantity)
			totalProfit += profit
		}
	}
	return totalProfit
}

// CalculateTotalMarketValue 计算总市值
func (s *Service) CalculateTotalMarketValue(stocks []*types.Stock) float64 {
	totalValue := 0.0
	for _, stock := range stocks {
		totalValue += stock.Price * float64(stock.Quantity)
	}
	return totalValue
}

// ValidateStockCode 验证股票代码格式
func (s *Service) ValidateStockCode(code string) error {
	if code == "" {
		return apperrors.NewValidationError("stock code cannot be empty")
	}

	// 基本格式检查
	if len(code) < 4 {
		return apperrors.NewValidationError("stock code too short")
	}

	return nil
}

// IsStockInPortfolio 检查股票是否在投资组合中
func (s *Service) IsStockInPortfolio(code string) bool {
	_, err := s.repo.GetStock(code)
	return err == nil
}

// GetStock 获取单个股票
func (s *Service) GetStock(code string) (*types.Stock, error) {
	stock, err := s.repo.GetStock(code)
	if err != nil {
		return nil, apperrors.NewNotFoundError(fmt.Sprintf("stock %s", code))
	}
	return stock, nil
}
