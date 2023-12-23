/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

import type { model_JSON } from './model_JSON';
import type { model_StepState } from './model_StepState';

export type model_Step = {
  action?: model_JSON;
  created_at?: string;
  depend?: Array<string>;
  describe?: string;
  ended_at?: string;
  error?: string;
  id?: number;
  input?: model_JSON;
  job_id?: number;
  name?: string;
  node_id?: string;
  output?: model_JSON;
  started_at?: string;
  state?: model_StepState;
  topic?: string;
  uid?: string;
  updated_at?: string;
};

