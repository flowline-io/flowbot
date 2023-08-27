/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { CancelablePromise } from '../core/CancelablePromise';
import type { BaseHttpRequest } from '../core/BaseHttpRequest';

export class WebhookService {

  constructor(public readonly httpRequest: BaseHttpRequest) {}

  /**
   * trigger webhook
   * @param flag flag param
   * @returns any OK
   * @throws ApiError
   */
  public webhook(
flag: string,
): CancelablePromise<any> {
    return this.httpRequest.request({
      method: 'POST',
      url: '/bot/webhook/v1/webhook/{flag}',
      path: {
        'flag': flag,
      },
    });
  }

}
