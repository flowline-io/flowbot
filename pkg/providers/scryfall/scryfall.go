package scryfall

import (
	"context"
	"github.com/go-resty/resty/v2"
	"net/http"
)

const (
	ID = "scryfall"
)

type Scryfall struct{}

func NewScryfall() *Scryfall {
	return &Scryfall{}
}

func (s *Scryfall) CardsSearch(ctx context.Context, q string) ([]Card, error) {
	c := resty.New()
	resp, err := c.R().
		SetContext(ctx).
		SetQueryParam("q", q).
		SetResult(&SearchResult{}).
		Get("https://api.scryfall.com/cards/search")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		result := resp.Result().(*SearchResult)
		if result != nil {
			return result.Data, nil
		}
	}
	return nil, nil
}

type SearchResult struct {
	Object     string `json:"object"`
	TotalCards int    `json:"total_cards"`
	HasMore    bool   `json:"has_more"`
	Data       []Card `json:"data"`
}

type Card struct {
	Object        string `json:"object"`
	ID            string `json:"id"`
	OracleID      string `json:"oracle_id"`
	MultiverseIds []int  `json:"multiverse_ids"`
	Name          string `json:"name"`
	PrintedName   string `json:"printed_name"`
	Lang          string `json:"lang"`
	ReleasedAt    string `json:"released_at"`
	URI           string `json:"uri"`
	ScryfallURI   string `json:"scryfall_uri"`
	Layout        string `json:"layout"`
	HighresImage  bool   `json:"highres_image"`
	ImageStatus   string `json:"image_status"`
	ImageUris     struct {
		Small      string `json:"small"`
		Normal     string `json:"normal"`
		Large      string `json:"large"`
		Png        string `json:"png"`
		ArtCrop    string `json:"art_crop"`
		BorderCrop string `json:"border_crop"`
	} `json:"image_uris"`
	ManaCost        string   `json:"mana_cost"`
	Cmc             float64  `json:"cmc"`
	TypeLine        string   `json:"type_line"`
	PrintedTypeLine string   `json:"printed_type_line"`
	OracleText      string   `json:"oracle_text"`
	PrintedText     string   `json:"printed_text"`
	Power           string   `json:"power"`
	Toughness       string   `json:"toughness"`
	Colors          []string `json:"colors"`
	ColorIdentity   []string `json:"color_identity"`
	Keywords        []string `json:"keywords"`
	Legalities      struct {
		Standard        string `json:"standard"`
		Future          string `json:"future"`
		Historic        string `json:"historic"`
		Gladiator       string `json:"gladiator"`
		Pioneer         string `json:"pioneer"`
		Explorer        string `json:"explorer"`
		Modern          string `json:"modern"`
		Legacy          string `json:"legacy"`
		Pauper          string `json:"pauper"`
		Vintage         string `json:"vintage"`
		Penny           string `json:"penny"`
		Commander       string `json:"commander"`
		Brawl           string `json:"brawl"`
		Historicbrawl   string `json:"historicbrawl"`
		Alchemy         string `json:"alchemy"`
		Paupercommander string `json:"paupercommander"`
		Duel            string `json:"duel"`
		Oldschool       string `json:"oldschool"`
		Premodern       string `json:"premodern"`
	} `json:"legalities"`
	Games           []string `json:"games"`
	Reserved        bool     `json:"reserved"`
	Foil            bool     `json:"foil"`
	Nonfoil         bool     `json:"nonfoil"`
	Finishes        []string `json:"finishes"`
	Oversized       bool     `json:"oversized"`
	Promo           bool     `json:"promo"`
	Reprint         bool     `json:"reprint"`
	Variation       bool     `json:"variation"`
	SetID           string   `json:"set_id"`
	Set             string   `json:"set"`
	SetName         string   `json:"set_name"`
	SetType         string   `json:"set_type"`
	SetURI          string   `json:"set_uri"`
	SetSearchURI    string   `json:"set_search_uri"`
	ScryfallSetURI  string   `json:"scryfall_set_uri"`
	RulingsURI      string   `json:"rulings_uri"`
	PrintsSearchURI string   `json:"prints_search_uri"`
	CollectorNumber string   `json:"collector_number"`
	Digital         bool     `json:"digital"`
	Rarity          string   `json:"rarity"`
	Watermark       string   `json:"watermark,omitempty"`
	FlavorText      string   `json:"flavor_text,omitempty"`
	CardBackID      string   `json:"card_back_id"`
	Artist          string   `json:"artist"`
	ArtistIds       []string `json:"artist_ids"`
	IllustrationID  string   `json:"illustration_id"`
	BorderColor     string   `json:"border_color"`
	Frame           string   `json:"frame"`
	FullArt         bool     `json:"full_art"`
	Textless        bool     `json:"textless"`
	Booster         bool     `json:"booster"`
	StorySpotlight  bool     `json:"story_spotlight"`
	EdhrecRank      int      `json:"edhrec_rank"`
	PennyRank       int      `json:"penny_rank,omitempty"`
	Prices          struct {
		Usd       interface{} `json:"usd"`
		UsdFoil   interface{} `json:"usd_foil"`
		UsdEtched interface{} `json:"usd_etched"`
		Eur       interface{} `json:"eur"`
		EurFoil   interface{} `json:"eur_foil"`
		Tix       interface{} `json:"tix"`
	} `json:"prices"`
	RelatedUris struct {
		Gatherer                  string `json:"gatherer"`
		TcgplayerInfiniteArticles string `json:"tcgplayer_infinite_articles"`
		TcgplayerInfiniteDecks    string `json:"tcgplayer_infinite_decks"`
		Edhrec                    string `json:"edhrec"`
	} `json:"related_uris"`
	PurchaseUris struct {
		Tcgplayer   string `json:"tcgplayer"`
		Cardmarket  string `json:"cardmarket"`
		Cardhoarder string `json:"cardhoarder"`
	} `json:"purchase_uris"`
	SecurityStamp string `json:"security_stamp,omitempty"`
}
