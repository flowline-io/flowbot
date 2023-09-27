/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { model_KeyResult } from '../models/model_KeyResult';
import type { model_KeyResultValue } from '../models/model_KeyResultValue';
import type { model_Objective } from '../models/model_Objective';
import type { protocol_Response } from '../models/protocol_Response';

import type { CancelablePromise } from '../core/CancelablePromise';
import type { BaseHttpRequest } from '../core/BaseHttpRequest';

export class OkrService {

  constructor(public readonly httpRequest: BaseHttpRequest) {}

  /**
   * KeyResult create
   * @param keyResult KeyResult data
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public postOkrKeyResult(
keyResult: model_KeyResult,
): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'POST',
      url: '/okr/key_result',
      body: keyResult,
    });
  }

  /**
   * KeyResult value create
   * @param id key result id
   * @param keyResultValue KeyResultValue data
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public postOkrKeyResultValue(
id: number,
keyResultValue: model_KeyResultValue,
): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'POST',
      url: '/okr/key_result/{id}/value',
      path: {
        'id': id,
      },
      body: keyResultValue,
    });
  }

  /**
   * key result value list
   * @param id key result id
   * @returns any OK
   * @throws ApiError
   */
  public getOkrKeyResultValues(
id: number,
): CancelablePromise<(protocol_Response & {
data?: Array<model_KeyResultValue>;
})> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/okr/key_result/{id}/values',
      path: {
        'id': id,
      },
    });
  }

  /**
   * KeyResult update
   * @param sequence Sequence
   * @param objective KeyResult data
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public putOkrKeyResult(
sequence: number,
objective: model_KeyResult,
): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'PUT',
      url: '/okr/key_result/{sequence}',
      path: {
        'sequence': sequence,
      },
      body: objective,
    });
  }

  /**
   * KeyResult delete
   * @param sequence Sequence
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public deleteOkrKeyResult(
sequence: number,
): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'DELETE',
      url: '/okr/key_result/{sequence}',
      path: {
        'sequence': sequence,
      },
    });
  }

  /**
   * KeyResult value detail
   * @param id key result id
   * @returns any OK
   * @throws ApiError
   */
  public deleteOkrKeyResultValue(
id: number,
): CancelablePromise<(protocol_Response & {
data?: model_KeyResultValue;
})> {
    return this.httpRequest.request({
      method: 'DELETE',
      url: '/okr/key_result_value/{id}',
      path: {
        'id': id,
      },
    });
  }

  /**
   * objective create
   * @param objective objective data
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public postOkrObjective(
objective: model_Objective,
): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'POST',
      url: '/okr/objective',
      body: objective,
    });
  }

  /**
   * objective detail
   * @param sequence Sequence
   * @returns any OK
   * @throws ApiError
   */
  public getOkrObjective(
sequence: number,
): CancelablePromise<(protocol_Response & {
data?: model_Objective;
})> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/okr/objective/{sequence}',
      path: {
        'sequence': sequence,
      },
    });
  }

  /**
   * objective update
   * @param sequence Sequence
   * @param objective objective data
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public putOkrObjective(
sequence: number,
objective: model_Objective,
): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'PUT',
      url: '/okr/objective/{sequence}',
      path: {
        'sequence': sequence,
      },
      body: objective,
    });
  }

  /**
   * objective delete
   * @param sequence Sequence
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public deleteOkrObjective(
sequence: number,
): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'DELETE',
      url: '/okr/objective/{sequence}',
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
  public getOkrObjectives(): CancelablePromise<(protocol_Response & {
data?: Array<model_Objective>;
})> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/okr/objectives',
    });
  }

}
