package rules

var testRule = `
{
  "ruleChain": {
    "id":"chain_call_rest_api",
    "name": "test rule chain",
    "root": true,
	"debugMode": true
  },
  "metadata": {
    "nodes": [
      {
        "id": "s1",
        "type": "jsFilter",
        "name": "filter",
        "configuration": {
          "jsScript": "return msg.deviceId=='aa';"
        }
      },
      {
        "id": "s2",
        "type": "jsTransform",
        "name": "transform",
        "configuration": {
          "jsScript": "msg.temperature=msg.temperature/10; return {'msg':msg,'metadata':metadata,'msgType':msgType};"
        }
      },
      {
        "id": "s3",
        "type": "log",
        "name": "logging",
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

var testYamlRule = `
ruleChain:
  id: chain_call_rest_api
  name: test rule chain
  root: true
  debugMode: true
metadata:
  nodes:
    - id: s1
      type: jsFilter
      name: filter
      configuration:
        jsScript: return msg.deviceId=='aa';
    - id: s2
      type: jsTransform
      name: transform
      configuration:
        jsScript: >-
          msg.temperature=msg.temperature/10; return {'msg':msg,'metadata':metadata,'msgType':msgType};
    - id: s3
      type: log
      name: logging
      configuration:
        jsScript: >-
          return 'Incoming message:\n' + JSON.stringify(msg) + '\nIncoming metadata:\n' + JSON.stringify(metadata);
  connections:
    - fromId: s1
      toId: s2
      type: 'True'
    - fromId: s2
      toId: s3
      type: Success
`

var testCustomDslYamlRule = `
id: chain_call_rest_api
name: test rule chain
root: true
debugMode: true

nodes:
- id: s1
  type: jsFilter
  name: filter
  configuration:
    jsScript: return msg.deviceId=='aa';
- id: s2
  type: jsTransform
  name: transform
  configuration:
    jsScript: msg.temperature=msg.temperature/10; return {'msg':msg,'metadata':metadata,'msgType':msgType};
- id: s3
  type: log
  name: logging
  configuration:
    jsScript: return 'Incoming message:\n' + JSON.stringify(msg) + '\nIncoming metadata:\n' + JSON.stringify(metadata);

pipelines:
- s1 --True--> s2 --Success--> s3
`
