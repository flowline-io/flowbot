/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { protocol_Response } from '../models/protocol_Response';

import type { CancelablePromise } from '../core/CancelablePromise';
import type { BaseHttpRequest } from '../core/BaseHttpRequest';

export class WebhookService {

  constructor(public readonly httpRequest: BaseHttpRequest) {}

  /**
   * trigger webhook
   * trigger webhook
   * @param flag Flag
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public postWebhookTrigger(
flag: string,
): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'POST',
      url: '/webhook/trigger/{flag}',
      path: {
        'flag': flag,
      },
    });
  }

}
