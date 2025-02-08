package wallos

type GetSubscriptionsResponse struct {
	Success       bool           `json:"success"`
	Title         string         `json:"title"`
	Subscriptions []Subscription `json:"subscriptions"`
	Notes         []interface{}  `json:"notes"`
}

type Subscription struct {
	Id                        int         `json:"id"`
	Name                      string      `json:"name"`
	Logo                      string      `json:"logo"`
	Price                     float64     `json:"price"`
	CurrencyId                int         `json:"currency_id"`
	StartDate                 string      `json:"start_date"`
	NextPayment               string      `json:"next_payment"`
	Cycle                     int         `json:"cycle"`
	Frequency                 int         `json:"frequency"`
	AutoRenew                 int         `json:"auto_renew"`
	Notes                     string      `json:"notes"`
	PaymentMethodId           int         `json:"payment_method_id"`
	PayerUserId               int         `json:"payer_user_id"`
	CategoryId                int         `json:"category_id"`
	Notify                    int         `json:"notify"`
	Url                       string      `json:"url"`
	Inactive                  int         `json:"inactive"`
	NotifyDaysBefore          *int        `json:"notify_days_before"`
	UserId                    int         `json:"user_id"`
	CancelationDate           interface{} `json:"cancelation_date"`
	CancellationDate          string      `json:"cancellation_date"`
	CategoryName              string      `json:"category_name"`
	PayerUserName             string      `json:"payer_user_name"`
	PaymentMethodName         string      `json:"payment_method_name"`
	ReplacementSubscriptionId int         `json:"replacement_subscription_id,omitempty"`
}

type GetMonthlyCostResponse struct {
	Success              bool     `json:"success"`
	Title                string   `json:"title"`
	MonthlyCost          string   `json:"monthly_cost"`
	LocalizedMonthlyCost string   `json:"localized_monthly_cost"`
	CurrencyCode         string   `json:"currency_code"`
	CurrencySymbol       string   `json:"currency_symbol"`
	Notes                []string `json:"notes"`
}
