package rules

import (
	"github.com/bytedance/sonic"
	"github.com/rulego/rulego/api/types"
)

// JsonParser Json
type JsonParser struct{}

func (p *JsonParser) DecodeRuleChain(rootRuleChain []byte) (types.RuleChain, error) {
	var def types.RuleChain
	err := sonic.Unmarshal(rootRuleChain, &def)
	return def, err
}

func (p *JsonParser) DecodeRuleNode(rootRuleChain []byte) (types.RuleNode, error) {
	var def types.RuleNode
	err := sonic.Unmarshal(rootRuleChain, &def)
	return def, err
}

func (p *JsonParser) EncodeRuleChain(def interface{}) ([]byte, error) {
	return sonic.MarshalIndent(def, "", "  ")
}

func (p *JsonParser) EncodeRuleNode(def interface{}) ([]byte, error) {
	return sonic.MarshalIndent(def, "", "  ")
}
