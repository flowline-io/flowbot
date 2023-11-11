/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

export type model_Todo = {
  category?: string;
  complete?: number;
  content?: string;
  created_at?: string;
  id?: number;
  is_remind_at_time?: number;
  key_result_id?: number;
  parent_id?: number;
  priority?: number;
  remark?: string;
  remind_at?: number;
  repeat_end_at?: number;
  repeat_method?: string;
  repeat_rule?: string;
  sequence?: number;
  sub_todos?: Array<model_Todo>;
  topic?: string;
  uid?: string;
  updated_at?: string;
};

