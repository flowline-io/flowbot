/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

import type { model_Dag } from './model_Dag';
import type { model_WorkflowState } from './model_WorkflowState';
import type { model_WorkflowTrigger } from './model_WorkflowTrigger';

export type model_Workflow = {
  canceled_count?: number;
  created_at?: string;
  dag?: Array<model_Dag>;
  describe?: string;
  failed_count?: number;
  flag?: string;
  id?: number;
  name?: string;
  running_count?: number;
  state?: model_WorkflowState;
  successful_count?: number;
  topic?: string;
  triggers?: Array<model_WorkflowTrigger>;
  uid?: string;
  updated_at?: string;
};

