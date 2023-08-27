/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { CancelablePromise } from '../core/CancelablePromise';
import type { BaseHttpRequest } from '../core/BaseHttpRequest';

export class OkrService {

  constructor(public readonly httpRequest: BaseHttpRequest) {}

  /**
   * objective create
   * @returns any OK
   * @throws ApiError
   */
  public objectiveCreate(): CancelablePromise<any> {
    return this.httpRequest.request({
      method: 'POST',
      url: '/bot/okr/v1/objective',
    });
  }

  /**
   * objective detail
   * @param sequence id
   * @returns any OK
   * @throws ApiError
   */
  public objectiveDetail(
sequence: number,
): CancelablePromise<any> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/bot/okr/v1/objective/{sequence}',
      path: {
        'sequence': sequence,
      },
    });
  }

  /**
   * objective update
   * @param sequence id
   * @returns any OK
   * @throws ApiError
   */
  public objectiveUpdate(
sequence: number,
): CancelablePromise<any> {
    return this.httpRequest.request({
      method: 'PUT',
      url: '/bot/okr/v1/objective/{sequence}',
      path: {
        'sequence': sequence,
      },
    });
  }

  /**
   * objective delete
   * @param sequence id
   * @returns any OK
   * @throws ApiError
   */
  public objectiveDelete(
sequence: number,
): CancelablePromise<any> {
    return this.httpRequest.request({
      method: 'DELETE',
      url: '/bot/okr/v1/objective/{sequence}',
      path: {
        'sequence': sequence,
      },
    });
  }

  /**
   * objective list
   * @returns any OK
   * @throws ApiError
   */
  public objectiveList(): CancelablePromise<any> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/bot/okr/v1/objectives',
    });
  }

}
