/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export const $model_Node = {
  properties: {
    _order: {
  type: 'number',
},
    bot: {
  type: 'string',
},
    connections: {
  type: 'array',
  contains: {
  type: 'string',
},
},
    describe: {
  type: 'string',
},
    group: {
  type: 'string',
},
    height: {
  type: 'number',
},
    id: {
  type: 'string',
},
    isGroup: {
  type: 'boolean',
},
    label: {
  type: 'string',
},
    parameters: {
  properties: {
  },
},
    parentId: {
  type: 'string',
},
    ports: {
  type: 'array',
  contains: {
  properties: {
    connected: {
  type: 'boolean',
},
    group: {
  type: 'string',
},
    id: {
  type: 'string',
},
    tooltip: {
  type: 'string',
},
    type: {
  type: 'string',
},
  },
},
},
    renderKey: {
  type: 'string',
},
    rule_id: {
  type: 'string',
},
    status: {
  type: 'model_NodeStatus',
},
    variables: {
  type: 'array',
  contains: {
  type: 'string',
},
},
    width: {
  type: 'number',
},
    'x': {
  type: 'number',
},
    'y': {
  type: 'number',
},
  },
} as const;
