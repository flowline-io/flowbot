/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export const $model_Review = {
  properties: {
    created_at: {
      type: 'string',
    },
    evaluations: {
      type: 'array',
      contains: {
        type: 'model_ReviewEvaluation',
      },
    },
    id: {
      type: 'number',
    },
    objective_id: {
      type: 'number',
    },
    rating: {
      type: 'number',
    },
    topic: {
      type: 'string',
    },
    type: {
      type: 'model_ReviewType',
    },
    uid: {
      type: 'string',
    },
    updated_at: {
      type: 'string',
    },
  },
} as const;
