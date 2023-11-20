/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export const $model_Todo = {
  properties: {
    category: {
  type: 'string',
},
    complete: {
  type: 'number',
},
    content: {
  type: 'string',
},
    created_at: {
  type: 'string',
},
    id: {
  type: 'number',
},
    is_remind_at_time: {
  type: 'number',
},
    key_result_id: {
  type: 'number',
},
    parent_id: {
  type: 'number',
},
    priority: {
  type: 'number',
},
    remark: {
  type: 'string',
},
    remind_at: {
  type: 'number',
},
    repeat_end_at: {
  type: 'number',
},
    repeat_method: {
  type: 'string',
},
    repeat_rule: {
  type: 'string',
},
    sequence: {
  type: 'number',
},
    sub_todos: {
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
  },
} as const;
