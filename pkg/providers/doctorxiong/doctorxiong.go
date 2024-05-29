package doctorxiong

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	ID    = "doctorxiong"
	Token = "token"
)

type Doctorxiong struct {
	token string
}

func NewDoctorxiong(token string) *Doctorxiong {
	return &Doctorxiong{token: token}
}

func (v *Doctorxiong) GetFundDetail(ctx context.Context, code, startDate, endDate string) (*FundDetailResponse, error) {
	c := resty.New()
	resp, err := c.R().
		SetContext(ctx).
		//SetHeader("token", v.token).
		SetQueryParam("code", code).
		SetQueryParam("startDate", startDate).
		SetQueryParam("endDate", endDate).
		SetResult(&FundDetailResponse{}).
		Get("https://api.doctorxiong.club/v1/fund/detail")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		result := resp.Result().(*FundDetailResponse)
		return result, nil
	}
	return nil, nil
}

func (v *Doctorxiong) GetStockDetail(ctx context.Context, code string) (*StockDetailResponse, error) {
	c := resty.New()
	resp, err := c.R().
		SetContext(ctx).
		//SetHeader("token", v.token).
		SetResult(&StockDetailResponse{}).
		SetQueryParam("code", code).
		Get("https://api.doctorxiong.club/v1/stock")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		result := resp.Result().(*StockDetailResponse)
		return result, nil
	}
	return nil, nil
}

func GetFund(ctx context.Context, code string) (*FundReply, error) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -90).Format("2006-01-02")
	endDate := now.Format("2006-01-02")
	dx := NewDoctorxiong("")
	resp, err := dx.GetFundDetail(ctx, code, startDate, endDate)
	if err != nil {
		return nil, err
	}
	if resp.Code != http.StatusOK {
		return nil, errors.New(resp.Message)
	}
	fund := resp.Data
	var netWorthDataDate []string
	var netWorthDataUnit []float64
	var netWorthDataIncrease []float64
	for _, item := range fund.NetWorthData {
		netWorthDataDate = append(netWorthDataDate, item[0].(string))
		f1, _ := strconv.ParseFloat(item[1].(string), 64)
		netWorthDataUnit = append(netWorthDataUnit, f1)
		f2, _ := strconv.ParseFloat(item[2].(string), 64)
		netWorthDataIncrease = append(netWorthDataIncrease, f2)
	}
	var millionCopiesIncomeDataDate []string
	var millionCopiesIncomeDataIncome []float64
	for _, item := range fund.MillionCopiesIncomeData {
		millionCopiesIncomeDataDate = append(millionCopiesIncomeDataDate, item[0].(string))
		f1, _ := strconv.ParseFloat(item[1].(string), 64)
		millionCopiesIncomeDataIncome = append(millionCopiesIncomeDataIncome, f1)
	}

	return &FundReply{
		Code:                          fund.Code,
		Name:                          fund.Name,
		Type:                          fund.Type,
		NetWorth:                      fund.NetWorth,
		ExpectWorth:                   fund.ExpectWorth,
		TotalWorth:                    fund.TotalWorth,
		ExpectGrowth:                  fund.ExpectGrowth,
		DayGrowth:                     fund.DayGrowth,
		LastWeekGrowth:                fund.LastWeekGrowth,
		LastMonthGrowth:               fund.LastMonthGrowth,
		LastThreeMonthsGrowth:         fund.LastThreeMonthsGrowth,
		LastSixMonthsGrowth:           fund.LastSixMonthsGrowth,
		LastYearGrowth:                fund.LastYearGrowth,
		BuyMin:                        fund.BuyMin,
		BuySourceRate:                 fund.BuySourceRate,
		BuyRate:                       fund.BuyRate,
		Manager:                       fund.Manager,
		FundScale:                     fund.FundScale,
		NetWorthDate:                  fund.NetWorthDate,
		ExpectWorthDate:               fund.ExpectWorthDate,
		NetWorthDataDate:              netWorthDataDate,
		NetWorthDataUnit:              netWorthDataUnit,
		NetWorthDataIncrease:          netWorthDataIncrease,
		MillionCopiesIncomeDate:       "",
		SevenDaysYearIncome:           0,
		MillionCopiesIncomeDataDate:   millionCopiesIncomeDataDate,
		MillionCopiesIncomeDataIncome: millionCopiesIncomeDataIncome,
	}, nil
}

func GetStock(ctx context.Context, code string) (*StockReply, error) {
	dx := NewDoctorxiong("")
	resp, err := dx.GetStockDetail(ctx, code)
	if err != nil {
		return nil, err
	}
	if resp.Code != http.StatusOK {
		return nil, errors.New(resp.Message)
	}
	stock := resp.Data
	if len(resp.Data) <= 0 {
		return &StockReply{}, nil
	}
	return &StockReply{
		Code:             stock[0].Code,
		Name:             stock[0].Name,
		Type:             stock[0].Type,
		PriceChange:      stock[0].PriceChange,
		ChangePercent:    stock[0].ChangePercent,
		Open:             stock[0].Open,
		Close:            stock[0].Close,
		Price:            stock[0].Price,
		High:             stock[0].High,
		Low:              stock[0].Low,
		Volume:           stock[0].Volume,
		Turnover:         stock[0].Turnover,
		TurnoverRate:     stock[0].TurnoverRate,
		TotalWorth:       stock[0].TotalWorth,
		CirculationWorth: stock[0].CirculationWorth,
		Date:             stock[0].Date,
		Buy:              nil,
		Sell:             nil,
		Pb:               stock[0].Pb,
		Spe:              stock[0].Spe,
		Pe:               stock[0].Pe,
	}, nil
}
