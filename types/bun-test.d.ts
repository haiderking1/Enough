declare module "bun:test" {
  export const describe: (name: string, fn: () => void) => void;
  export const it: (name: string, fn: () => void | Promise<void>) => void;
  export const expect: <T>(actual: T) => any;
  export const beforeEach: (fn: () => void | Promise<void>) => void;
  export const afterEach: (fn: () => void | Promise<void>) => void;
  export const beforeAll: (fn: () => void | Promise<void>) => void;
  export const afterAll: (fn: () => void | Promise<void>) => void;
  export const mock: {
    module(path: string, factory: () => Record<string, unknown>): void;
    restore(): void;
    fn<T>(impl?: T): T & { mock: { calls: unknown[][] } };
  };
}
