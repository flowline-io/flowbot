package types

type TheCard struct {
	Fn    string `json:"fn,omitempty"`
	Phone struct {
		Type   string `json:"type,omitempty"`
		Data   string `json:"data,omitempty"`
		Ref    string `json:"ref,omitempty"`
		Width  int    `json:"width,omitempty"`
		Height int    `json:"height,omitempty"`
		Size   int    `json:"size,omitempty"`
	} `json:"phone"`
	Note string `json:"note,omitempty"`
	N    struct {
		Surname    string `json:"surname,omitempty"`
		Given      string `json:"given,omitempty"`
		Additional string `json:"additional,omitempty"`
		Prefix     string `json:"prefix,omitempty"`
		Suffix     string `json:"suffix,omitempty"`
	} `json:"n"`
	Org struct {
		Fn    string `json:"fn,omitempty"`
		Title string `json:"title,omitempty"`
	} `json:"org"`
	Tel []struct {
		Type string `json:"type,omitempty"`
		Uri  string `json:"uri,omitempty"`
	} `json:"tel,omitempty"`
	Email []struct {
		Type string `json:"type,omitempty"`
		Uri  string `json:"uri,omitempty"`
	} `json:"email,omitempty"`
	Comm []struct {
		Type string `json:"type,omitempty"`
		Name string `json:"name,omitempty"`
		Uri  string `json:"uri,omitempty"`
	} `json:"comm,omitempty"`
	Bday struct {
		Y int `json:"y,omitempty"`
		M int `json:"m,omitempty"`
		D int `json:"d,omitempty"`
	} `json:"bday"`
}
