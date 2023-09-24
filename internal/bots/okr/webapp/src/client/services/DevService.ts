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
   * get example data
   * @returns any OK
   * @throws ApiError
   */
  public getDevV1Example(): CancelablePromise<(protocol_Response & {
    data?: types_KV;
  })> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/dev/v1/example',
    });
  }

}
