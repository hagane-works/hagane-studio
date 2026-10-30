import { expect } from "vitest";

const compact = (value: string) => value.replace(/[\s,;]+/g, "");

expect.extend({
  toContain(received: unknown, expected: unknown) {
    let pass = false;
    if (typeof received === "string" && typeof expected === "string") {
      pass =
        received.includes(expected) ||
        compact(received).includes(compact(expected));
    } else if (Array.isArray(received)) {
      pass = received.includes(expected);
    }
    return {
      pass,
      message: () =>
        pass
          ? `expected value not to contain ${this.utils.printExpected(expected)}`
          : `expected value to contain ${this.utils.printExpected(expected)}`,
    };
  },
  toMatch(received: unknown, expected: unknown) {
    let pass = false;
    if (typeof received === "string") {
      if (typeof expected === "string") {
        pass =
          received.includes(expected) ||
          compact(received).includes(compact(expected));
      } else if (expected instanceof RegExp) {
        expected.lastIndex = 0;
        pass = expected.test(received);
        if (!pass) {
          const source = expected.source
            .replace(/(?<!\\)\s+/g, "")
            .replace(/\\ /g, "");
          pass = new RegExp(source, expected.flags).test(compact(received));
        }
      }
    }
    return {
      pass,
      message: () =>
        pass
          ? `expected value not to match ${this.utils.printExpected(expected)}`
          : `expected value to match ${this.utils.printExpected(expected)}`,
    };
  },
});
