/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { BaseHttpRequest } from './core/BaseHttpRequest';
import type { OpenAPIConfig } from './core/OpenAPI';
import { AxiosHttpRequest } from './core/AxiosHttpRequest';

import { DevService } from './services/DevService';
import { MarkdownService } from './services/MarkdownService';
import { OkrService } from './services/OkrService';
import { WebhookService } from './services/WebhookService';
import { WorkflowService } from './services/WorkflowService';

type HttpRequestConstructor = new (config: OpenAPIConfig) => BaseHttpRequest;

export class AppClient {

  public readonly dev: DevService;
  public readonly markdown: MarkdownService;
  public readonly okr: OkrService;
  public readonly webhook: WebhookService;
  public readonly workflow: WorkflowService;

  public readonly request: BaseHttpRequest;

  constructor(config?: Partial<OpenAPIConfig>, HttpRequest: HttpRequestConstructor = AxiosHttpRequest) {
    this.request = new HttpRequest({
      BASE: config?.BASE ?? 'http://localhost:6060/bot',
      VERSION: config?.VERSION ?? '1.0',
      WITH_CREDENTIALS: config?.WITH_CREDENTIALS ?? false,
      CREDENTIALS: config?.CREDENTIALS ?? 'include',
      TOKEN: config?.TOKEN,
      USERNAME: config?.USERNAME,
      PASSWORD: config?.PASSWORD,
      HEADERS: config?.HEADERS,
      ENCODE_PATH: config?.ENCODE_PATH,
    });

    this.dev = new DevService(this.request);
    this.markdown = new MarkdownService(this.request);
    this.okr = new OkrService(this.request);
    this.webhook = new WebhookService(this.request);
    this.workflow = new WorkflowService(this.request);
  }
}

