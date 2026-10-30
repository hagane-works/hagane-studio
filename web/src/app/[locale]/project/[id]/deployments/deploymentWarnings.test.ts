import { readFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { createTranslator, type IntlError } from "use-intl";
import { describe, expect, it } from "vitest";

import { localizeDeploymentWarning } from "./deploymentWarnings";

const here = dirname(fileURLToPath(import.meta.url));
const messagesRoot = resolve(here, "../../../../../../messages");
const LOCALES = ["de", "en", "es", "fr", "ja", "ko", "ru", "zh"] as const;

type LooseTranslator = (
  key: string,
  values?: Record<string, string | number>,
) => string;

function bundle(locale: string): Record<string, string> {
  return JSON.parse(
    readFileSync(join(messagesRoot, locale, "divisionDeployment.json"), "utf8"),
  );
}

function translator(
  locale: string,
  onError: (error: IntlError) => void = () => {},
): LooseTranslator {
  return createTranslator({
    locale,
    messages: { divisionDeployment: bundle(locale) },
    namespace: "divisionDeployment",
    onError,
  }) as unknown as LooseTranslator;
}

describe("localizeDeploymentWarning · actionable:unknown_tag", () => {
  it("localizes the new code instead of falling back to the machine string", () => {
    const errors: IntlError[] = [];
    const t = translator("en", (e) => errors.push(e));

    const rendered = localizeDeploymentWarning(
      "actionable:unknown_tag|tag=NEW",
      t,
    );

    expect(rendered).not.toBe("unknown_tag|tag=NEW");
    expect(rendered).not.toContain("unknown_tag");
    expect(rendered).not.toBe("divisionDeployment.warnUnknownTag");
    expect(errors).toHaveLength(0);

    expect(rendered).toContain("NEW");
    expect(rendered).toBe(bundle("en").warnUnknownTag.replace("{tag}", "NEW"));
  });

  it("renders in every one of the 8 UI locales with the tag interpolated", () => {
    for (const locale of LOCALES) {
      const errors: IntlError[] = [];
      const t = translator(locale, (e) => errors.push(e));
      const rendered = localizeDeploymentWarning(
        "actionable:unknown_tag|tag=NEW",
        t,
      );
      expect(errors, locale).toHaveLength(0);
      expect(rendered, locale).toContain("NEW");
      expect(rendered, locale).not.toContain("{tag}");
      expect(rendered, locale).not.toContain("unknown_tag");
      expect(rendered.length, locale).toBeGreaterThan(20);
    }
  });

  it("degrades to '?' rather than blanking when the backend omits the tag param", () => {
    const t = translator("en");
    expect(localizeDeploymentWarning("actionable:unknown_tag", t)).toContain(
      "?",
    );
  });

  it("consumes the code string EXACTLY as the backend spells it", () => {
    const t = translator("en");
    const rendered = localizeDeploymentWarning(
      "actionable:unknown_tag|tag=?",
      t,
    );
    expect(rendered).toBe(bundle("en").warnUnknownTag.replace("{tag}", "?"));
  });
});

describe("localizeDeploymentWarning · the pre-existing codes still map", () => {
  const t = translator("en");
  const en = bundle("en");

  it("no_location / no_template", () => {
    expect(localizeDeploymentWarning("actionable:no_location", t)).toBe(
      en.warnNoLocation,
    );
    expect(localizeDeploymentWarning("info:no_template", t)).toBe(
      en.warnNoTemplate,
    );
  });

  it("unowned_province interpolates both params", () => {
    const rendered = localizeDeploymentWarning(
      "actionable:unowned_province|province=11033|tag=GUN",
      t,
    );
    expect(rendered).toContain("11033");
    expect(rendered).toContain("GUN");
  });

  it("missing_template keeps a template name containing separators", () => {
    const rendered = localizeDeploymentWarning(
      "info:missing_template|template=Panzer|Elite",
      t,
    );
    expect(rendered).toContain("Panzer|Elite");
  });

  it("an unknown future code renders its body instead of blanking", () => {
    expect(
      localizeDeploymentWarning("actionable:some_future_code|a=1", t),
    ).toBe("some_future_code|a=1");
  });
});

describe("the deployments workbench routes its warnings through this module", () => {
  const editor = readFileSync(join(here, "DeploymentEditor.tsx"), "utf8");
  const page = readFileSync(join(here, "page.tsx"), "utf8");

  it("DeploymentEditor.tsx delegates to localizeDeploymentWarning", () => {
    expect(editor).toContain("localizeDeploymentWarning");

    expect(editor).not.toContain('case "no_location"');
    expect(page).not.toContain('case "no_location"');
  });
});
