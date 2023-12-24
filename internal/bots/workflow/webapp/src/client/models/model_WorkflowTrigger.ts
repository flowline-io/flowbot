/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

import type { model_JSON } from './model_JSON';
import type { model_TriggerType } from './model_TriggerType';
import type { model_WorkflowTriggerState } from './model_WorkflowTriggerState';

export type model_WorkflowTrigger = {
  count?: number;
  created_at?: string;
  id?: number;
  rule?: model_JSON;
  state?: model_WorkflowTriggerState;
  topic?: string;
  type?: model_TriggerType;
  uid?: string;
  updated_at?: string;
  workflow_id?: number;
};
