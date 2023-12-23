/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

import type { model_Edge } from './model_Edge';
import type { model_Node } from './model_Node';

export type model_Dag = {
  created_at?: string;
  edges?: Array<model_Edge>;
  id?: number;
  nodes?: Array<model_Node>;
  topic?: string;
  uid?: string;
  updated_at?: string;
  workflow_id?: number;
};

