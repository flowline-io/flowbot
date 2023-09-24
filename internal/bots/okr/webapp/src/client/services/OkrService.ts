/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { model_Objective } from '../models/model_Objective';
import type { protocol_Response } from '../models/protocol_Response';

import type { CancelablePromise } from '../core/CancelablePromise';
import type { BaseHttpRequest } from '../core/BaseHttpRequest';

export class OkrService {

  constructor(public readonly httpRequest: BaseHttpRequest) {}

  /**
   * objective create
   * objective create
   * @param objective objective data
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public postOkrV1Objective(
    objective: model_Objective,
  ): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'POST',
      url: '/okr/v1/objective',
      body: objective,
    });
  }

  /**
   * objective detail
   * objective detail
   * @param sequence Sequence
   * @returns any OK
   * @throws ApiError
   */
  public getOkrV1Objective(
    sequence: number,
  ): CancelablePromise<(protocol_Response & {
    data?: model_Objective;
  })> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/okr/v1/objective/{sequence}',
      path: {
        'sequence': sequence,
      },
    });
  }

  /**
   * objective update
   * objective update
   * @param sequence Sequence
   * @param objective objective data
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public putOkrV1Objective(
    sequence: number,
    objective: model_Objective,
  ): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'PUT',
      url: '/okr/v1/objective/{sequence}',
      path: {
        'sequence': sequence,
      },
      body: objective,
    });
  }

  /**
   * objective delete
   * objective delete
   * @param sequence Sequence
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public deleteOkrV1Objective(
    sequence: number,
  ): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'DELETE',
      url: '/okr/v1/objective/{sequence}',
      path: {
        'sequence': sequence,
      },
    });
  }

  /**
   * objective list
   * objective list
   * @returns any OK
   * @throws ApiError
   */
  public getOkrV1Objectives(): CancelablePromise<(protocol_Response & {
    data?: Array<model_Objective>;
  })> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/okr/v1/objectives',
    });
  }

}
