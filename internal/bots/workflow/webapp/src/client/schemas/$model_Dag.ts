/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export const $model_Dag = {
  properties: {
    created_at: {
      type: 'string',
    },
    edges: {
      type: 'array',
      contains: {
        type: 'model_Edge',
      },
    },
    id: {
      type: 'number',
    },
    nodes: {
      type: 'array',
      contains: {
        type: 'model_Node',
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
    workflow_id: {
      type: 'number',
    },
  },
} as const;
