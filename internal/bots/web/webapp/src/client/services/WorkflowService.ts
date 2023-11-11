/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { protocol_Response } from '../models/protocol_Response';
import type { workflow_rule } from '../models/workflow_rule';

import type { CancelablePromise } from '../core/CancelablePromise';
import type { BaseHttpRequest } from '../core/BaseHttpRequest';

export class WorkflowService {

  constructor(public readonly httpRequest: BaseHttpRequest) {}

  /**
   * get chatbot actions
   * @returns any OK
   * @throws ApiError
   */
  public getWorkflowActions(): CancelablePromise<(protocol_Response & {
    data?: Record<string, Array<workflow_rule>>;
  })> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/workflow/actions',
    });
  }

}
