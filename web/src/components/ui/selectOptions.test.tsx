import { describe, expect, it } from "vitest";
import * as React from "react";

import { collectSections } from "./selectOptions";

describe("collectSections — 既有调用点(只有散装 option)不得有任何行为变化", () => {
  it("一串 option → 单个无标题段,顺序与取值原样", () => {
    const sections = collectSections([
      <option key="a" value="political_advisor">
        政治顾问
      </option>,
      <option key="b" value="army_chief">
        陆军总长
      </option>,
    ]);
    expect(sections).toHaveLength(1);
    expect(sections[0].label).toBeNull();
    expect(sections[0].options.map((o) => o.value)).toEqual([
      "political_advisor",
      "army_chief",
    ]);
  });

  it("value 缺省 → 空串(旧行为)", () => {
    const sections = collectSections([<option key="x">占位</option>]);
    expect(sections[0].options[0].value).toBe("");
  });

  it("数字 value 转成字符串(旧行为)", () => {
    const sections = collectSections([
      <option key="1" value={7}>
        七
      </option>,
    ]);
    expect(sections[0].options[0].value).toBe("7");
  });

  it("disabled 透传(旧行为)", () => {
    const sections = collectSections([
      <option key="a" value="a" disabled>
        不可选
      </option>,
    ]);
    expect(sections[0].options[0].disabled).toBe(true);
  });

  it("Fragment 包着的 option 会被摊平(旧行为)", () => {
    const sections = collectSections(
      <>
        <option value="a">A</option>
        <option value="b">B</option>
      </>,
    );
    expect(sections).toHaveLength(1);
    expect(sections[0].options.map((o) => o.value)).toEqual(["a", "b"]);
  });

  it("`{cond && <option/>}` 产生的 false / null 被忽略(旧行为)", () => {
    const show = false;
    const sections = collectSections([
      <option key="a" value="a">
        A
      </option>,
      show ? (
        <option key="b" value="b">
          B
        </option>
      ) : null,
      false,
    ] as React.ReactNode);
    expect(sections[0].options.map((o) => o.value)).toEqual(["a"]);
  });

  it("完全没有 option → 空数组,不产出空段", () => {
    expect(collectSections(null)).toEqual([]);
    expect(collectSections([])).toEqual([]);
  });
});

describe("collectSections — optgroup", () => {
  it("原版散装 + 一组基 mod → 两段,标题只在第二段", () => {
    const sections = collectSections([
      <option key="v1" value="political_advisor">
        政治顾问
      </option>,
      <option key="v2" value="theorist">
        军事理论家
      </option>,
      <optgroup key="g" label="来自 Kaiserreich">
        <option value="economy_advisor">economy_advisor</option>
        <option value="foreign_advisor">foreign_advisor</option>
      </optgroup>,
    ]);
    expect(sections).toHaveLength(2);
    expect(sections[0].label).toBeNull();
    expect(sections[0].options.map((o) => o.value)).toEqual([
      "political_advisor",
      "theorist",
    ]);
    expect(sections[1].label).toBe("来自 Kaiserreich");
    expect(sections[1].options.map((o) => o.value)).toEqual([
      "economy_advisor",
      "foreign_advisor",
    ]);
  });

  it("🔴 空 optgroup 不产出段 —— 基 mod 词表为空时调用方可以无脑传", () => {
    const baseModValues: string[] = [];
    const sections = collectSections([
      <option key="v" value="political_advisor">
        政治顾问
      </option>,
      <optgroup key="g" label="来自 Kaiserreich">
        {baseModValues.map((v) => (
          <option key={v} value={v}>
            {v}
          </option>
        ))}
      </optgroup>,
    ]);
    expect(sections).toHaveLength(1);
    expect(sections[0].label).toBeNull();
  });

  it("optgroup 之后的散装 option 另起一段,不回填到前面(保持源码顺序)", () => {
    const sections = collectSections([
      <option key="v" value="a">
        A
      </option>,
      <optgroup key="g" label="组">
        <option value="b">B</option>
      </optgroup>,
      <option key="c" value="__custom__">
        自定义…
      </option>,
    ]);
    expect(sections.map((s) => s.label)).toEqual([null, "组", null]);
    expect(sections[2].options.map((o) => o.value)).toEqual(["__custom__"]);
  });

  it("optgroup 内部的 .map() 与 Fragment 同样摊平", () => {
    const sections = collectSections([
      <optgroup key="g" label="组">
        <>
          {["x", "y"].map((v) => (
            <option key={v} value={v}>
              {v}
            </option>
          ))}
        </>
      </optgroup>,
    ]);
    expect(sections[0].options.map((o) => o.value)).toEqual(["x", "y"]);
  });

  it("多个 optgroup 各自成段", () => {
    const sections = collectSections([
      <optgroup key="g1" label="一">
        <option value="a">A</option>
      </optgroup>,
      <optgroup key="g2" label="二">
        <option value="b">B</option>
      </optgroup>,
    ]);
    expect(sections.map((s) => s.label)).toEqual(["一", "二"]);
  });

  it("optgroup 没给 label → 当作无标题段(不崩)", () => {
    const sections = collectSections([
      <optgroup key="g">
        <option value="a">A</option>
      </optgroup>,
    ]);
    expect(sections).toHaveLength(1);
    expect(sections[0].label).toBeNull();
  });
});
