/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { model_Message } from '../models/model_Message';

import type { CancelablePromise } from '../core/CancelablePromise';
import type { BaseHttpRequest } from '../core/BaseHttpRequest';

export class DevService {

  constructor(public readonly httpRequest: BaseHttpRequest) {}

  /**
   * get example data
   * @returns model_Message OK
   * @throws ApiError
   */
  public example(): CancelablePromise<model_Message> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/bot/dev/v1/example',
    });
  }

}
