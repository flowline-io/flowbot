/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export const $model_Job = {
  properties: {
    created_at: {
  type: 'string',
},
    dag_id: {
  type: 'number',
},
    ended_at: {
  type: 'string',
},
    id: {
  type: 'number',
},
    script_version: {
  type: 'number',
},
    started_at: {
  type: 'string',
},
    state: {
  type: 'model_JobState',
},
    steps: {
  type: 'array',
  contains: {
    type: 'model_Step',
  },
},
    topic: {
  type: 'string',
},
    trigger_id: {
  type: 'number',
},
    uid: {
  type: 'string',
},
    updated_at: {
  type: 'string',
},
    workflow_id: {
  type: 'number',
},
  },
} as const;
