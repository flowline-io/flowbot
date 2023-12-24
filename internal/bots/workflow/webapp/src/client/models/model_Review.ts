/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

import type { model_ReviewEvaluation } from './model_ReviewEvaluation';
import type { model_ReviewType } from './model_ReviewType';

export type model_Review = {
  created_at?: string;
  evaluations?: Array<model_ReviewEvaluation>;
  id?: number;
  objective_id?: number;
  rating?: number;
  topic?: string;
  type?: model_ReviewType;
  uid?: string;
  updated_at?: string;
};
