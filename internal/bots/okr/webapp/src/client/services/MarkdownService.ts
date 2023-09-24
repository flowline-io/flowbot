/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { protocol_Response } from '../models/protocol_Response';

import type { CancelablePromise } from '../core/CancelablePromise';
import type { BaseHttpRequest } from '../core/BaseHttpRequest';

export class MarkdownService {

  constructor(public readonly httpRequest: BaseHttpRequest) {}

  /**
   * markdown editor page
   * markdown editor page
   * @param flag Flag
   * @returns void
   * @throws ApiError
   */
  public getMarkdownV1Editor(
    flag: string,
  ): CancelablePromise<void> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/markdown/v1/editor/{flag}',
      path: {
        'flag': flag,
      },
    });
  }

  /**
   * save markdown data
   * save markdown data
   * @param data Data
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public postMarkdownV1Markdown(
    data: Record<string, string>,
  ): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'POST',
      url: '/markdown/v1/markdown',
      body: data,
    });
  }

}
