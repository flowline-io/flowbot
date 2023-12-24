/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { model_Dag } from '../models/model_Dag';
import type { model_Job } from '../models/model_Job';
import type { model_Workflow } from '../models/model_Workflow';
import type { model_WorkflowScript } from '../models/model_WorkflowScript';
import type { model_WorkflowTrigger } from '../models/model_WorkflowTrigger';
import type { protocol_Response } from '../models/protocol_Response';
import type { workflow_rule } from '../models/workflow_rule';

import type { CancelablePromise } from '../core/CancelablePromise';
import type { BaseHttpRequest } from '../core/BaseHttpRequest';

export class WorkflowService {

  constructor(public readonly httpRequest: BaseHttpRequest) {}

  /**
   * get chatbot actions
   * @returns any OK
   * @throws ApiError
   */
  public getWorkflowActions(): CancelablePromise<(protocol_Response & {
data?: Record<string, Array<workflow_rule>>;
})> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/workflow/actions',
    });
  }

  /**
   * workflow job detail
   * @param id Job ID
   * @returns any OK
   * @throws ApiError
   */
  public getWorkflowJob(
id: number,
): CancelablePromise<(protocol_Response & {
data?: model_Job;
})> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/workflow/job/{id}',
      path: {
        'id': id,
      },
    });
  }

  /**
   * workflow job rerun
   * @param id Job ID
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public postWorkflowJobRerun(
id: number,
): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'POST',
      url: '/workflow/job/{id}/rerun',
      path: {
        'id': id,
      },
    });
  }

  /**
   * workflow trigger update
   * @param id Trigger ID
   * @param trigger workflow trigger data
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public putWorkflowTrigger(
id: number,
trigger: model_WorkflowTrigger,
): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'PUT',
      url: '/workflow/trigger/{id}',
      path: {
        'id': id,
      },
      body: trigger,
    });
  }

  /**
   * workflow trigger delete
   * @param id Trigger ID
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public deleteWorkflowTrigger(
id: number,
): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'DELETE',
      url: '/workflow/trigger/{id}',
      path: {
        'id': id,
      },
    });
  }

  /**
   * workflow create
   * @param script workflow script data
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public postWorkflowWorkflow(
script: model_WorkflowScript,
): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'POST',
      url: '/workflow/workflow',
      body: script,
    });
  }

  /**
   * workflow detail
   * @param id ID
   * @returns any OK
   * @throws ApiError
   */
  public getWorkflowWorkflow(
id: number,
): CancelablePromise<(protocol_Response & {
data?: model_Workflow;
})> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/workflow/workflow/{id}',
      path: {
        'id': id,
      },
    });
  }

  /**
   * workflow update
   * @param id ID
   * @param script workflow script data
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public putWorkflowWorkflow(
id: number,
script: model_WorkflowScript,
): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'PUT',
      url: '/workflow/workflow/{id}',
      path: {
        'id': id,
      },
      body: script,
    });
  }

  /**
   * workflow delete
   * @param id ID
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public deleteWorkflowWorkflow(
id: number,
): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'DELETE',
      url: '/workflow/workflow/{id}',
      path: {
        'id': id,
      },
    });
  }

  /**
   * workflow dag detail
   * @param id Workflow ID
   * @returns any OK
   * @throws ApiError
   */
  public getWorkflowWorkflowDag(
id: number,
): CancelablePromise<(protocol_Response & {
data?: model_Dag;
})> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/workflow/workflow/{id}/dag',
      path: {
        'id': id,
      },
    });
  }

  /**
   * workflow dag update
   * @param id Workflow ID
   * @param trigger workflow dag data
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public putWorkflowWorkflowDag(
id: number,
trigger: model_Dag,
): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'PUT',
      url: '/workflow/workflow/{id}/dag',
      path: {
        'id': id,
      },
      body: trigger,
    });
  }

  /**
   * workflow job list
   * @param id Workflow ID
   * @returns any OK
   * @throws ApiError
   */
  public getWorkflowWorkflowJobs(
id: number,
): CancelablePromise<(protocol_Response & {
data?: Array<model_Job>;
})> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/workflow/workflow/{id}/jobs',
      path: {
        'id': id,
      },
    });
  }

  /**
   * workflow script detail
   * @param id Workflow ID
   * @returns any OK
   * @throws ApiError
   */
  public getWorkflowWorkflowScript(
id: number,
): CancelablePromise<(protocol_Response & {
data?: model_WorkflowScript;
})> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/workflow/workflow/{id}/script',
      path: {
        'id': id,
      },
    });
  }

  /**
   * workflow trigger create
   * @param id Workflow ID
   * @param trigger workflow trigger data
   * @returns protocol_Response OK
   * @throws ApiError
   */
  public postWorkflowWorkflowTrigger(
id: number,
trigger: model_WorkflowTrigger,
): CancelablePromise<protocol_Response> {
    return this.httpRequest.request({
      method: 'POST',
      url: '/workflow/workflow/{id}/trigger',
      path: {
        'id': id,
      },
      body: trigger,
    });
  }

  /**
   * workflow trigger list
   * @param id Workflow ID
   * @returns any OK
   * @throws ApiError
   */
  public getWorkflowWorkflowTriggers(
id: number,
): CancelablePromise<(protocol_Response & {
data?: Array<model_WorkflowTrigger>;
})> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/workflow/workflow/{id}/triggers',
      path: {
        'id': id,
      },
    });
  }

  /**
   * workflow list
   * @returns any OK
   * @throws ApiError
   */
  public getWorkflowWorkflows(): CancelablePromise<(protocol_Response & {
data?: Array<model_Workflow>;
})> {
    return this.httpRequest.request({
      method: 'GET',
      url: '/workflow/workflows',
    });
  }

}
