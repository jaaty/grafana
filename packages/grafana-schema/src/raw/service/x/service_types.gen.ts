// THIS FILE IS GENERATED. EDITING IS FUTILE.
//
// Generated by:
//     kinds/gen.go
// Using jennies:
//     TSTypesJenny
//     LatestMajorsOrXJenny
//
// Run 'make gen-cue' from repository root to regenerate.

export interface Endpoint {
  path: string;
  type: ('http' | 'ping' | 'dns' | 'tcp');
}

export interface Service {
  endpoints: Array<Endpoint>;
  /**
   * name of the service
   */
  name: string;
  uid: string;
}

export const defaultService: Partial<Service> = {
  endpoints: [],
};
