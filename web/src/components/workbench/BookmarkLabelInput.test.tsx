import { cleanup, fireEvent, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BookmarkLabelInput } from "./BookmarkLabelInput";

afterEach(cleanup);

describe("BookmarkLabelInput", () => {
  it("preserves raw spacing when the normalized parent value is unchanged", () => {
    const onTokensChange = vi.fn();
    const { container, rerender } = render(
      <BookmarkLabelInput
        aria-label="Country labels"
        value="major european"
        onTokensChange={onTokensChange}
      />,
    );

    const input = container.querySelector("input") as HTMLInputElement;
    fireEvent.change(input, { target: { value: "major european " } });

    expect(onTokensChange).toHaveBeenLastCalledWith(["major", "european"]);
    rerender(
      <BookmarkLabelInput
        aria-label="Country labels"
        value="major european"
        onTokensChange={onTokensChange}
      />,
    );
    expect(input.value).toBe("major european ");
  });

  it("replaces the local draft when labels change externally", () => {
    const onTokensChange = vi.fn();
    const { container, rerender } = render(
      <BookmarkLabelInput
        aria-label="Country labels"
        value="major"
        onTokensChange={onTokensChange}
      />,
    );

    const input = container.querySelector("input") as HTMLInputElement;
    fireEvent.change(input, { target: { value: "major, european " } });
    expect(onTokensChange).toHaveBeenLastCalledWith(["major", "european"]);

    rerender(
      <BookmarkLabelInput
        aria-label="Country labels"
        value="ai_generated"
        onTokensChange={onTokensChange}
      />,
    );
    expect(input.value).toBe("ai_generated");
  });
});
