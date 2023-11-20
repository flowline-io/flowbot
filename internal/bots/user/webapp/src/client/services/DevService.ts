/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { protocol_Response } from '../models/protocol_Response';
import type { types_KV } from '../models/types_KV';

import type { CancelablePromise } from '../core/CancelablePromise';
import type { BaseHttpRequest } from '../core/BaseHttpRequest';

export class DevService {

  constructor(public readonly httpRequest: BaseHttpRequest) {}

  /**
   * Show example
   * @returns any OK
   * @throws ApiError
   */
  public getDevExample(): CancelablePromise<(protocol_Response & {
data?: types_KV;
})> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/dev/example',
    });
  }

}
