/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

import type { model_KeyResult } from './model_KeyResult';
import type { model_Review } from './model_Review';

export type model_Objective = {
  created_data?: string;
  current_value?: number;
  feasibility?: string;
  id?: number;
  is_plan?: number;
  key_results?: Array<model_KeyResult>;
  memo?: string;
  motive?: string;
  plan_end?: number;
  plan_start?: number;
  reviews?: Array<model_Review>;
  sequence?: number;
  tag?: string;
  title?: string;
  topic?: string;
  total_value?: number;
  uid?: string;
  updated_date?: string;
};
