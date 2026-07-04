import { describe, expect, it } from "vitest";
import { en } from "./en";
import { fa } from "./fa";

describe("i18n dictionaries", () => {
  it("fa covers every en key", () => {
    const enKeys = Object.keys(en).sort();
    const faKeys = Object.keys(fa).sort();
    expect(faKeys).toEqual(enKeys);
  });

  it("has no empty translations", () => {
    for (const [key, value] of Object.entries(en)) {
      expect(value, `en.${key}`).not.toBe("");
    }
    for (const [key, value] of Object.entries(fa)) {
      expect(value, `fa.${key}`).not.toBe("");
    }
  });

  it("keeps interpolation placeholders consistent between languages", () => {
    for (const key of Object.keys(en) as Array<keyof typeof en>) {
      const enParams = (en[key].match(/\{[a-z]+\}/gi) ?? []).sort();
      const faParams = (fa[key].match(/\{[a-z]+\}/gi) ?? []).sort();
      expect(faParams, `placeholders for ${key}`).toEqual(enParams);
    }
  });
});
