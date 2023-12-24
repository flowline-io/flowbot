/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export const $protocol_Response = {
  properties: {
    data: {
  description: `Response data`,
  properties: {
  },
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
  type: 'all-of',
  description: `Execution status (success or failure), must be one of ok and failed,
indicating successful and unsuccessful execution, respectively.`,
  contains: [{
  type: 'protocol_ResponseStatus',
}],
},
  },
} as const;
