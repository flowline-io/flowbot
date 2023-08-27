/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { CancelablePromise } from '../core/CancelablePromise';
import type { BaseHttpRequest } from '../core/BaseHttpRequest';

export class MarkdownService {

  constructor(public readonly httpRequest: BaseHttpRequest) {}

  /**
   * get markdown editor
   * @param flag flag param
   * @returns any OK
   * @throws ApiError
   */
  public editor(
flag: string,
): CancelablePromise<any> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/bot/markdown/v1/editor/{flag}',
      path: {
        'flag': flag,
      },
    });
  }

  /**
   * create markdown page
   * @returns any OK
   * @throws ApiError
   */
  public saveMarkdown(): CancelablePromise<any> {
    return this.httpRequest.request({
      method: 'POST',
      url: '/bot/markdown/v1/markdown',
    });
  }

}
