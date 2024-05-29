package doctorxiong

type FundDetailResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Code                    string          `json:"code"`
		Name                    string          `json:"name"`
		Type                    string          `json:"type"`
		NetWorth                float64         `json:"netWorth"`
		ExpectWorth             float64         `json:"expectWorth"`
		TotalWorth              float64         `json:"totalWorth"`
		ExpectGrowth            string          `json:"expectGrowth"`
		DayGrowth               string          `json:"dayGrowth"`
		LastWeekGrowth          string          `json:"lastWeekGrowth"`
		LastMonthGrowth         string          `json:"lastMonthGrowth"`
		LastThreeMonthsGrowth   string          `json:"lastThreeMonthsGrowth"`
		LastSixMonthsGrowth     string          `json:"lastSixMonthsGrowth"`
		LastYearGrowth          string          `json:"lastYearGrowth"`
		BuyMin                  string          `json:"buyMin"`
		BuySourceRate           string          `json:"buySourceRate"`
		BuyRate                 string          `json:"buyRate"`
		Manager                 string          `json:"manager"`
		FundScale               string          `json:"fundScale"`
		NetWorthDate            string          `json:"netWorthDate"`
		ExpectWorthDate         string          `json:"expectWorthDate"`
		NetWorthData            [][]interface{} `json:"netWorthData"`
		MillionCopiesIncomeData [][]interface{} `json:"millionCopiesIncomeData"`
		MillionCopiesIncomeDate string          `json:"millionCopiesIncomeDate"`
		SevenDaysYearIncome     float64         `json:"sevenDaysYearIncome"`
		SevenDaysYearIncomeDate [][]interface{} `json:"sevenDaysYearIncomeDate"`
	} `json:"data"`
	Meta string `json:"meta"`
}

type StockDetailResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    []struct {
		Code             string   `json:"code"`
		Name             string   `json:"name"`
		Type             string   `json:"type"`
		PriceChange      string   `json:"priceChange"`
		ChangePercent    string   `json:"changePercent"`
		Open             string   `json:"open"`
		Close            string   `json:"close"`
		Price            string   `json:"price"`
		High             string   `json:"high"`
		Low              string   `json:"low"`
		Volume           string   `json:"volume"`
		Turnover         string   `json:"turnover"`
		TurnoverRate     string   `json:"turnoverRate"`
		TotalWorth       string   `json:"totalWorth"`
		CirculationWorth string   `json:"circulationWorth"`
		Date             string   `json:"date"`
		Buy              []string `json:"buy"`
		Sell             []string `json:"sell"`
		Pb               string   `json:"pb"`
		Spe              string   `json:"spe"`
		Pe               string   `json:"pe"`
	} `json:"data"`
	Meta interface{} `json:"meta"`
}

type FundReply struct {
	Code                          string    `json:"code,omitempty"`
	Name                          string    `json:"name,omitempty"`
	Type                          string    `json:"type,omitempty"`
	NetWorth                      float64   `json:"net_worth,omitempty"`
	ExpectWorth                   float64   `json:"expect_worth,omitempty"`
	TotalWorth                    float64   `json:"total_worth,omitempty"`
	ExpectGrowth                  string    `json:"expect_growth,omitempty"`
	DayGrowth                     string    `json:"day_growth,omitempty"`
	LastWeekGrowth                string    `json:"last_week_growth,omitempty"`
	LastMonthGrowth               string    `json:"last_month_growth,omitempty"`
	LastThreeMonthsGrowth         string    `json:"last_three_months_growth,omitempty"`
	LastSixMonthsGrowth           string    `json:"last_six_months_growth,omitempty"`
	LastYearGrowth                string    `json:"last_year_growth,omitempty"`
	BuyMin                        string    `json:"buy_min,omitempty"`
	BuySourceRate                 string    `json:"buy_source_rate,omitempty"`
	BuyRate                       string    `json:"buy_rate,omitempty"`
	Manager                       string    `json:"manager,omitempty"`
	FundScale                     string    `json:"fund_scale,omitempty"`
	NetWorthDate                  string    `json:"net_worth_date,omitempty"`
	ExpectWorthDate               string    `json:"expect_worth_date,omitempty"`
	NetWorthDataDate              []string  `json:"net_worth_data_date,omitempty"`
	NetWorthDataUnit              []float64 `json:"net_worth_data_unit,omitempty"`
	NetWorthDataIncrease          []float64 `json:"net_worth_data_increase,omitempty"`
	MillionCopiesIncomeDataDate   []string  `json:"million_copies_income_data_date,omitempty"`
	MillionCopiesIncomeDataIncome []float64 `json:"million_copies_income_data_income,omitempty"`
	MillionCopiesIncomeDate       string    `json:"million_copies_income_date,omitempty"`
	SevenDaysYearIncome           float64   `json:"seven_days_year_income,omitempty"`
}

type StockReply struct {
	Code             string   `json:"code,omitempty"`
	Name             string   `json:"name,omitempty"`
	Type             string   `json:"type,omitempty"`
	PriceChange      string   `json:"price_change,omitempty"`
	ChangePercent    string   `json:"change_percent,omitempty"`
	Open             string   `json:"open,omitempty"`
	Close            string   `json:"close,omitempty"`
	Price            string   `json:"price,omitempty"`
	High             string   `json:"high,omitempty"`
	Low              string   `json:"low,omitempty"`
	Volume           string   `json:"volume,omitempty"`
	Turnover         string   `json:"turnover,omitempty"`
	TurnoverRate     string   `json:"turnover_rate,omitempty"`
	TotalWorth       string   `json:"total_worth,omitempty"`
	CirculationWorth string   `json:"circulation_worth,omitempty"`
	Date             string   `json:"date,omitempty"`
	Buy              []string `json:"buy,omitempty"`
	Sell             []string `json:"sell,omitempty"`
	Pb               string   `json:"pb,omitempty"`
	Spe              string   `json:"spe,omitempty"`
	Pe               string   `json:"pe,omitempty"`
}
