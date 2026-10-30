import React, { act } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRoot, type Root } from "react-dom/client";

import { RadioCardGroup, type RadioCardOption } from "./RadioCardGroup";

let container: HTMLDivElement;
let root: Root;

const options = (overrides: Record<string, unknown> = {}) => [
  { value: "skin", title: "只换配色涂装", consequence: "形状不变 · 最快最省" },
  {
    value: "advanced",
    title: "换个样子",
    consequence: "形状会变 · 沿用原来的动作",
  },
  {
    value: "model",
    title: "完全换一个",
    consequence: "动作要重做",
    ...overrides,
  },
];

function mount(node: React.ReactElement) {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  act(() => {
    root.render(node);
  });
}

afterEach(() => {
  act(() => {
    root.unmount();
  });
  container.remove();
});

beforeEach(() => {
  vi.clearAllMocks();
});

describe("RadioCardGroup", () => {
  it("renders a vertical radiogroup with one radio per option", () => {
    mount(
      <RadioCardGroup
        label="改多狠"
        value="advanced"
        onChange={() => {}}
        options={options()}
      />,
    );
    const group = container.querySelector('[role="radiogroup"]')!;
    expect(group.getAttribute("aria-orientation")).toBe("vertical");
    expect(group.getAttribute("aria-label")).toBe("改多狠");
    expect(container.querySelectorAll('[role="radio"]')).toHaveLength(3);
    expect(
      container.querySelector('[role="radio"][aria-checked="true"]')!
        .textContent,
    ).toContain("换个样子");
  });

  it("keeps every consequence line on screen so the three rungs can be compared", () => {
    mount(
      <RadioCardGroup
        label="改多狠"
        value="advanced"
        onChange={() => {}}
        options={options()}
      />,
    );
    for (const line of [
      "形状不变 · 最快最省",
      "形状会变 · 沿用原来的动作",
      "动作要重做",
    ]) {
      expect(container.textContent).toContain(line);
    }
  });

  it("shows a disabled rung's reason as visible text, not only as a title", () => {
    mount(
      <RadioCardGroup
        label="改多狠"
        value="advanced"
        onChange={() => {}}
        options={options({ disabled: true, reason: "这类模型必须保留原骨架" })}
      />,
    );
    const disabled = container.querySelector(
      '[role="radio"][aria-disabled="true"]',
    )!;
    expect(disabled.textContent).toContain("这类模型必须保留原骨架");
    const describedBy = disabled.getAttribute("aria-describedby")!;
    expect(container.querySelector(`#${describedBy}`)!.textContent).toContain(
      "这类模型必须保留原骨架",
    );
  });

  it("never selects a disabled rung on click", () => {
    const onChange = vi.fn();
    mount(
      <RadioCardGroup
        label="改多狠"
        value="advanced"
        onChange={onChange}
        options={options({ disabled: true, reason: "这类模型必须保留原骨架" })}
      />,
    );
    const disabled = container.querySelector(
      '[role="radio"][aria-disabled="true"]',
    ) as HTMLElement;
    act(() => {
      disabled.click();
    });
    expect(onChange).not.toHaveBeenCalled();
  });

  it("moves and selects with the arrow keys, skipping disabled rungs", () => {
    const onChange = vi.fn();
    mount(
      <RadioCardGroup
        label="改多狠"
        value="skin"
        onChange={onChange}
        options={options({ disabled: true, reason: "不可用" })}
      />,
    );
    const group = container.querySelector('[role="radiogroup"]') as HTMLElement;
    act(() => {
      group.dispatchEvent(
        new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true }),
      );
    });
    expect(onChange).toHaveBeenCalledWith("advanced");
    onChange.mockClear();
    act(() => {
      group.dispatchEvent(
        new KeyboardEvent("keydown", { key: "ArrowUp", bubbles: true }),
      );
    });

    expect(onChange).toHaveBeenCalledWith("advanced");
  });

  it("keeps the selected card unbolded and never paints the whole card amber", () => {
    mount(
      <RadioCardGroup
        label="改多狠"
        value="advanced"
        onChange={() => {}}
        options={options()}
      />,
    );
    const checked = container.querySelector(
      '[role="radio"][aria-checked="true"]',
    ) as HTMLElement;
    expect(checked.className).not.toMatch(/font-(semibold|bold|medium)/);
    expect(checked.className).not.toMatch(/bg-accent(?!\/)/);
  });

  it("reaches the adjacent option in both directions when the selected option is itself disabled (3 options)", () => {
    const onChange = vi.fn();
    const threeOptions: RadioCardOption<string>[] = [
      { value: "a", title: "a", consequence: "ca" },
      {
        value: "b",
        title: "b",
        consequence: "cb",
        disabled: true,
        reason: "不可用",
      },
      { value: "c", title: "c", consequence: "cc" },
    ];
    mount(
      <RadioCardGroup
        label="改多狠"
        value="b"
        onChange={onChange}
        options={threeOptions}
      />,
    );
    const group = container.querySelector('[role="radiogroup"]') as HTMLElement;

    act(() => {
      group.dispatchEvent(
        new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true }),
      );
    });
    expect(onChange).toHaveBeenCalledWith("c");

    onChange.mockClear();
    act(() => {
      group.dispatchEvent(
        new KeyboardEvent("keydown", { key: "ArrowUp", bubbles: true }),
      );
    });
    expect(onChange).toHaveBeenCalledWith("a");
  });

  it("does not skip over the option immediately below a disabled selected option (4 options)", () => {
    const onChange = vi.fn();
    const fourOptions: RadioCardOption<string>[] = [
      { value: "a", title: "a", consequence: "ca" },
      { value: "b", title: "b", consequence: "cb" },
      {
        value: "c",
        title: "c",
        consequence: "cc",
        disabled: true,
        reason: "不可用",
      },
      { value: "d", title: "d", consequence: "cd" },
    ];
    mount(
      <RadioCardGroup
        label="改多狠"
        value="c"
        onChange={onChange}
        options={fourOptions}
      />,
    );
    const group = container.querySelector('[role="radiogroup"]') as HTMLElement;

    act(() => {
      group.dispatchEvent(
        new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true }),
      );
    });

    expect(onChange).toHaveBeenCalledWith("d");
    expect(onChange).not.toHaveBeenCalledWith("a");
  });

  it("wraps forward from the last usable option back to the first usable one", () => {
    const onChange = vi.fn();
    mount(
      <RadioCardGroup
        label="改多狠"
        value="model"
        onChange={onChange}
        options={options()}
      />,
    );
    const group = container.querySelector('[role="radiogroup"]') as HTMLElement;
    act(() => {
      group.dispatchEvent(
        new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true }),
      );
    });
    expect(onChange).toHaveBeenCalledWith("skin");
  });

  it("puts the sole reachable tab stop on the first enabled option when the selected one is disabled", () => {
    const threeOptions: RadioCardOption<string>[] = [
      { value: "a", title: "a", consequence: "ca" },
      {
        value: "b",
        title: "b",
        consequence: "cb",
        disabled: true,
        reason: "不可用",
      },
      { value: "c", title: "c", consequence: "cc" },
    ];
    mount(
      <RadioCardGroup
        label="改多狠"
        value="b"
        onChange={() => {}}
        options={threeOptions}
      />,
    );
    const radios = Array.from(
      container.querySelectorAll<HTMLButtonElement>('[role="radio"]'),
    );
    const reachable = radios.filter((radio) => radio.tabIndex === 0);
    expect(reachable).toHaveLength(1);
    expect(reachable[0].dataset.rung).toBe("a");
  });

  it("never truncates the title, consequence, or reason text (DESIGN §4.13)", () => {
    mount(
      <RadioCardGroup
        label="改多狠"
        value="advanced"
        onChange={() => {}}
        options={options({ disabled: true, reason: "这类模型必须保留原骨架" })}
      />,
    );
    const spans = Array.from(container.querySelectorAll("span"));
    const withText = spans.filter(
      (span) => (span.textContent ?? "").trim().length > 0,
    );
    for (const span of withText) {
      expect(span.className).not.toMatch(/\btruncate\b/);
    }
  });

  it("defaults aria-orientation to vertical when orientation is not passed", () => {
    mount(
      <RadioCardGroup
        label="改多狠"
        value="advanced"
        onChange={() => {}}
        options={options()}
      />,
    );
    const group = container.querySelector('[role="radiogroup"]')!;
    expect(group.getAttribute("aria-orientation")).toBe("vertical");
  });

  it("follows the orientation prop for aria-orientation (DESIGN §4.13 horizontal)", () => {
    mount(
      <RadioCardGroup
        label="改多狠"
        value="advanced"
        onChange={() => {}}
        orientation="horizontal"
        options={options()}
      />,
    );
    const group = container.querySelector('[role="radiogroup"]')!;
    expect(group.getAttribute("aria-orientation")).toBe("horizontal");
  });

  it("still exposes every consequence line simultaneously when laid out horizontally", () => {
    mount(
      <RadioCardGroup
        label="改多狠"
        value="advanced"
        onChange={() => {}}
        orientation="horizontal"
        options={options({ disabled: true, reason: "这类模型必须保留原骨架" })}
      />,
    );
    for (const line of [
      "形状不变 · 最快最省",
      "形状会变 · 沿用原来的动作",
      "动作要重做",
      "这类模型必须保留原骨架",
    ]) {
      expect(container.textContent).toContain(line);
    }
  });
});
