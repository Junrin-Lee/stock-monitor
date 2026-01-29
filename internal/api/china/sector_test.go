package china

import (
	"testing"

	"stock-monitor/internal/types"
)

// TestFetchIndustrySectorList 测试获取行业板块列表
func TestFetchIndustrySectorList(t *testing.T) {
	sectors, err := FetchSectorList(types.SectorTypeIndustry)
	if err != nil {
		t.Fatalf("获取行业板块失败: %v", err)
	}

	if len(sectors) == 0 {
		t.Fatal("行业板块列表为空")
	}

	t.Logf("获取到 %d 个行业板块", len(sectors))

	// 验证第一个板块的数据结构
	first := sectors[0]
	if first.Code == "" {
		t.Error("板块代码为空")
	}
	if first.Name == "" {
		t.Error("板块名称为空")
	}
	if first.Type != types.SectorTypeIndustry {
		t.Errorf("板块类型错误: 期望 %s, 实际 %s", types.SectorTypeIndustry, first.Type)
	}

	t.Logf("示例板块: %s (%s), 涨跌幅: %.2f%%, 成交额: %.2f亿",
		first.Name, first.Code, first.ChangePercent, first.Turnover/100000000)
}

// TestFetchConceptSectorList 测试获取概念板块列表
func TestFetchConceptSectorList(t *testing.T) {
	sectors, err := FetchSectorList(types.SectorTypeConcept)
	if err != nil {
		t.Fatalf("获取概念板块失败: %v", err)
	}

	if len(sectors) == 0 {
		t.Fatal("概念板块列表为空")
	}

	t.Logf("获取到 %d 个概念板块", len(sectors))

	// 验证数据结构
	first := sectors[0]
	if first.Type != types.SectorTypeConcept {
		t.Errorf("板块类型错误: 期望 %s, 实际 %s", types.SectorTypeConcept, first.Type)
	}

	t.Logf("示例概念: %s (%s), 涨跌幅: %.2f%%",
		first.Name, first.Code, first.ChangePercent)
}

// TestFetchSectorStocks 测试获取板块成分股
func TestFetchSectorStocks(t *testing.T) {
	// 先获取一个板块代码
	sectors, err := FetchSectorList(types.SectorTypeIndustry)
	if err != nil {
		t.Fatalf("获取板块列表失败: %v", err)
	}

	if len(sectors) == 0 {
		t.Fatal("板块列表为空")
	}

	// 使用第一个板块测试
	sectorCode := sectors[0].Code
	t.Logf("测试板块: %s (%s)", sectors[0].Name, sectorCode)

	stocks, err := FetchSectorStocks(sectorCode)
	if err != nil {
		t.Fatalf("获取成分股失败: %v", err)
	}

	if len(stocks) == 0 {
		t.Fatal("成分股列表为空")
	}

	t.Logf("获取到 %d 只成分股", len(stocks))

	// 验证数据结构
	first := stocks[0]
	if first.Code == "" {
		t.Error("股票代码为空")
	}
	if first.Name == "" {
		t.Error("股票名称为空")
	}

	t.Logf("示例成分股: %s (%s), 现价: %.2f, 涨跌幅: %.2f%%",
		first.Name, first.Code, first.Price, first.ChangePercent)
}
