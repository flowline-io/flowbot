/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

import type { model_KeyResultValue } from './model_KeyResultValue';
import type { model_Todo } from './model_Todo';
import type { model_ValueModeType } from './model_ValueModeType';

export type model_KeyResult = {
  created_at?: string;
  current_value?: number;
  id?: number;
  initial_value?: number;
  key_result_values?: Array<model_KeyResultValue>;
  memo?: string;
  objective_id?: number;
  sequence?: number;
  tag?: string;
  target_value?: number;
  title?: string;
  todos?: Array<model_Todo>;
  topic?: string;
  uid?: string;
  updated_at?: string;
  value_mode?: model_ValueModeType;
};
