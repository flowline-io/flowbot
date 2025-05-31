package rules

var testRuleFile = `
{
  "ruleChain": {
    "id":"chain_call_rest_api",
    "name": "测试规则链",
    "root": true,
	"debugMode": true
  },
  "metadata": {
    "nodes": [
      {
        "id": "s1",
        "type": "jsFilter",
        "name": "过滤",
        "configuration": {
          "jsScript": "return msg.deviceId=='aa';"
        }
      },
      {
        "id": "s2",
        "type": "jsTransform",
        "name": "转换",
        "configuration": {
          "jsScript": "msg.temperature=msg.temperature/10; return {'msg':msg,'metadata':metadata,'msgType':msgType};"
        }
      },
      {
        "id": "s3",
        "type": "log",
        "name": "记录日志",
        "configuration": {
			"jsScript": "return 'Incoming message:\\n' + JSON.stringify(msg) + '\\nIncoming metadata:\\n' + JSON.stringify(metadata);"
        }
      }
    ],
    "connections": [
      {
        "fromId": "s1",
        "toId": "s2",
        "type": "True"
      },
      {
        "fromId": "s2",
        "toId": "s3",
        "type": "Success"
      }
    ]
  }
}
`
