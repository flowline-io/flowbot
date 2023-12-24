/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

import type { protocol_ResponseStatus } from './protocol_ResponseStatus';

export type protocol_Response = {
  /**
   * Response data
   */
  data?: any;
  /**
   * Error message, it is recommended to fill in a human-readable error message when the action fails to execute,
 * or an empty string when it succeeds.
   */
  message?: string;
  /**
   * The return code, which must conform to the return code rules defined later on this page
   */
  retcode?: number;
  /**
   * Execution status (success or failure), must be one of ok and failed,
 * indicating successful and unsuccessful execution, respectively.
   */
  status?: protocol_ResponseStatus;
};
