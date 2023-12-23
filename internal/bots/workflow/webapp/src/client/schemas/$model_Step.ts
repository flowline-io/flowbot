/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export const $model_Step = {
  properties: {
    action: {
      type: 'model_JSON',
    },
    created_at: {
      type: 'string',
    },
    depend: {
      type: 'array',
      contains: {
        type: 'string',
      },
    },
    describe: {
      type: 'string',
    },
    ended_at: {
      type: 'string',
    },
    error: {
      type: 'string',
    },
    id: {
      type: 'number',
    },
    input: {
      type: 'model_JSON',
    },
    job_id: {
      type: 'number',
    },
    name: {
      type: 'string',
    },
    node_id: {
      type: 'string',
    },
    output: {
      type: 'model_JSON',
    },
    started_at: {
      type: 'string',
    },
    state: {
      type: 'model_StepState',
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
