/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

import type { model_NodeStatus } from './model_NodeStatus';

export type model_Node = {
  _order?: number;
  bot?: string;
  connections?: Array<string>;
  group?: string;
  height?: number;
  id?: string;
  isGroup?: boolean;
  label?: string;
  parameters?: any;
  parentId?: string;
  ports?: Array<{
    connected?: boolean;
    group?: string;
    id?: string;
    tooltip?: string;
    type?: string;
  }>;
  renderKey?: string;
  rule_id?: string;
  status?: model_NodeStatus;
  variables?: Array<string>;
  width?: number;
  'x'?: number;
  'y'?: number;
};

