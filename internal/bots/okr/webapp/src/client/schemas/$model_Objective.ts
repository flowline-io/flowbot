/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export const $model_Objective = {
  properties: {
    created_data: {
      type: 'string',
    },
    current_value: {
      type: 'number',
    },
    feasibility: {
      type: 'string',
    },
    id: {
      type: 'number',
    },
    is_plan: {
      type: 'number',
    },
    key_results: {
      type: 'array',
      contains: {
        type: 'model_KeyResult',
      },
    },
    memo: {
      type: 'string',
    },
    motive: {
      type: 'string',
    },
    plan_end: {
      type: 'number',
    },
    plan_start: {
      type: 'number',
    },
    reviews: {
      type: 'array',
      contains: {
        type: 'model_Review',
      },
    },
    sequence: {
      type: 'number',
    },
    tag: {
      type: 'string',
    },
    title: {
      type: 'string',
    },
    topic: {
      type: 'string',
    },
    total_value: {
      type: 'number',
    },
    uid: {
      type: 'string',
    },
    updated_date: {
      type: 'string',
    },
  },
} as const;
