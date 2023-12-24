/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export const $model_KeyResult = {
  properties: {
    created_at: {
  type: 'string',
},
    current_value: {
  type: 'number',
},
    id: {
  type: 'number',
},
    initial_value: {
  type: 'number',
},
    key_result_values: {
  type: 'array',
  contains: {
    type: 'model_KeyResultValue',
  },
},
    memo: {
  type: 'string',
},
    objective_id: {
  type: 'number',
},
    sequence: {
  type: 'number',
},
    tag: {
  type: 'string',
},
    target_value: {
  type: 'number',
},
    title: {
  type: 'string',
},
    todos: {
  type: 'array',
  contains: {
    type: 'model_Todo',
  },
},
    topic: {
  type: 'string',
},
    uid: {
  type: 'string',
},
    updated_at: {
  type: 'string',
},
    value_mode: {
  type: 'model_ValueModeType',
},
  },
} as const;
