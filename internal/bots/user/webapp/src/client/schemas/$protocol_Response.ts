/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export const $protocol_Response = {
  properties: {
    data: {
  type: 'any',
  description: `Response data`,
},
    message: {
  type: 'string',
  description: `Error message, it is recommended to fill in a human-readable error message when the action fails to execute,
or an empty string when it succeeds.`,
},
    retcode: {
  type: 'number',
  description: `The return code, which must conform to the return code rules defined later on this page`,
},
    status: {
  type: 'string',
  description: `Execution status (success or failure), must be one of ok and failed,
indicating successful and unsuccessful execution, respectively.`,
},
  },
} as const;
