package notion

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"net/http"
	"time"
)

const (
	ID = "notion"
)

type ListResult struct {
	Object         string          `json:"object"`
	Results        json.RawMessage `json:"results"`
	NextCursor     interface{}     `json:"next_cursor"`
	HasMore        bool            `json:"has_more"`
	Type           string          `json:"type"`
	PageOrDatabase interface{}     `json:"page_or_database"`
}

type Page struct {
	Object         string    `json:"object"`
	ID             string    `json:"id"`
	CreatedTime    time.Time `json:"created_time"`
	LastEditedTime time.Time `json:"last_edited_time"`
	CreatedBy      struct {
		Object string `json:"object"`
		ID     string `json:"id"`
	} `json:"created_by"`
	LastEditedBy struct {
		Object string `json:"object"`
		ID     string `json:"id"`
	} `json:"last_edited_by"`
	Cover struct {
		Type     string `json:"type"`
		External struct {
			URL string `json:"url"`
		} `json:"external"`
	} `json:"cover"`
	Icon struct {
		Type  string `json:"type"`
		Emoji string `json:"emoji"`
	} `json:"icon"`
	Parent struct {
		Type       string `json:"type"`
		DatabaseID string `json:"database_id"`
	} `json:"parent"`
	Archived   bool `json:"archived"`
	Properties struct {
		StoreAvailability struct {
			ID          string        `json:"id"`
			Type        string        `json:"type"`
			MultiSelect []interface{} `json:"multi_select"`
		} `json:"Store availability"`
		FoodGroup struct {
			ID     string `json:"id"`
			Type   string `json:"type"`
			Select struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Color string `json:"color"`
			} `json:"select"`
		} `json:"Food group"`
		Price struct {
			ID     string      `json:"id"`
			Type   string      `json:"type"`
			Number interface{} `json:"number"`
		} `json:"Price"`
		ResponsiblePerson struct {
			ID     string        `json:"id"`
			Type   string        `json:"type"`
			People []interface{} `json:"people"`
		} `json:"Responsible Person"`
		LastOrdered struct {
			ID   string      `json:"id"`
			Type string      `json:"type"`
			Date interface{} `json:"date"`
		} `json:"Last ordered"`
		CostOfNextTrip struct {
			ID      string `json:"id"`
			Type    string `json:"type"`
			Formula struct {
				Type   string      `json:"type"`
				Number interface{} `json:"number"`
			} `json:"formula"`
		} `json:"Cost of next trip"`
		Recipes struct {
			ID       string        `json:"id"`
			Type     string        `json:"type"`
			Relation []interface{} `json:"relation"`
		} `json:"Recipes"`
		Description struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			RichText []struct {
				Type string `json:"type"`
				Text struct {
					Content string      `json:"content"`
					Link    interface{} `json:"link"`
				} `json:"text"`
				Annotations struct {
					Bold          bool   `json:"bold"`
					Italic        bool   `json:"italic"`
					Strikethrough bool   `json:"strikethrough"`
					Underline     bool   `json:"underline"`
					Code          bool   `json:"code"`
					Color         string `json:"color"`
				} `json:"annotations"`
				PlainText string      `json:"plain_text"`
				Href      interface{} `json:"href"`
			} `json:"rich_text"`
		} `json:"Description"`
		InStock struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Checkbox bool   `json:"checkbox"`
		} `json:"In stock"`
		NumberOfMeals struct {
			ID     string `json:"id"`
			Type   string `json:"type"`
			Rollup struct {
				Type     string `json:"type"`
				Number   int    `json:"number"`
				Function string `json:"function"`
			} `json:"rollup"`
		} `json:"Number of meals"`
		Photo struct {
			ID   string      `json:"id"`
			Type string      `json:"type"`
			URL  interface{} `json:"url"`
		} `json:"Photo"`
		Name struct {
			ID    string `json:"id"`
			Type  string `json:"type"`
			Title []struct {
				Type string `json:"type"`
				Text struct {
					Content string      `json:"content"`
					Link    interface{} `json:"link"`
				} `json:"text"`
				Annotations struct {
					Bold          bool   `json:"bold"`
					Italic        bool   `json:"italic"`
					Strikethrough bool   `json:"strikethrough"`
					Underline     bool   `json:"underline"`
					Code          bool   `json:"code"`
					Color         string `json:"color"`
				} `json:"annotations"`
				PlainText string      `json:"plain_text"`
				Href      interface{} `json:"href"`
			} `json:"title"`
		} `json:"Name"`
	} `json:"properties"`
	URL string `json:"url"`
}

type Notion struct {
	c     *resty.Client
	token string
}

func NewNotion(token string) *Notion {
	v := &Notion{token: token}

	v.c = resty.New()
	v.c.SetBaseURL("https://api.notion.com/v1")
	v.c.SetTimeout(time.Minute)
	v.c.SetHeaders(map[string]string{
		"Content-Type":   "application/json",
		"Notion-Version": "2022-06-28",
	})

	return v
}

func (v *Notion) Search(query string) ([]*Page, error) {
	resp, err := v.c.R().
		SetAuthScheme("Bearer").
		SetAuthToken(v.token).
		SetResult(&ListResult{}).
		SetBody(map[string]interface{}{
			"query": query,
			"sort": map[string]interface{}{
				"direction": "descending",
				"timestamp": "last_edited_time",
			},
			"page_size": 10,
		}).Post("/search")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		list := resp.Result().(*ListResult)

		var results []*Page
		_ = json.Unmarshal(list.Results, &results)
		return results, nil
	} else {
		return nil, fmt.Errorf("%d", resp.StatusCode())
	}
}

func (v *Notion) AppendBlock(blockId string, text string) error {
	resp, err := v.c.R().
		SetAuthScheme("Bearer").
		SetAuthToken(v.token).
		SetResult(&ListResult{}).
		SetBody(map[string]interface{}{
			"children": []map[string]interface{}{
				{
					"object": "block",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{
								"type": "text",
								"text": map[string]interface{}{
									"content": text,
								},
							},
						},
					},
				},
			},
		}).Patch(fmt.Sprintf("/blocks/%s/children", blockId))
	if err != nil {
		return err
	}

	if resp.StatusCode() == http.StatusOK {
		return nil
	} else {
		return fmt.Errorf("%d", resp.StatusCode())
	}
}
