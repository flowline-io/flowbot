/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export const $model_Workflow = {
  properties: {
    canceled_count: {
  type: 'number',
},
    created_at: {
  type: 'string',
},
    dag: {
  type: 'array',
  contains: {
    type: 'model_Dag',
  },
},
    describe: {
  type: 'string',
},
    failed_count: {
  type: 'number',
},
    flag: {
  type: 'string',
},
    id: {
  type: 'number',
},
    name: {
  type: 'string',
},
    running_count: {
  type: 'number',
},
    state: {
  type: 'model_WorkflowState',
},
    successful_count: {
  type: 'number',
},
    topic: {
  type: 'string',
},
    triggers: {
  type: 'array',
  contains: {
    type: 'model_WorkflowTrigger',
  },
},
    uid: {
  type: 'string',
},
    updated_at: {
  type: 'string',
},
  },
} as const;
