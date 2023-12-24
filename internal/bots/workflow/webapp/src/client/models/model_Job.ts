/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

import type { model_JobState } from './model_JobState';
import type { model_Step } from './model_Step';

export type model_Job = {
  created_at?: string;
  dag_id?: number;
  ended_at?: string;
  id?: number;
  script_version?: number;
  started_at?: string;
  state?: model_JobState;
  steps?: Array<model_Step>;
  topic?: string;
  trigger_id?: number;
  uid?: string;
  updated_at?: string;
  workflow_id?: number;
};
