import { fireEvent, render, screen } from "@testing-library/react";
import type { ComponentProps, ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { WorkbenchLoadError } from "./WorkbenchLoadError";

vi.mock("use-intl", () => ({
  useTranslations: () => (key: string) =>
    ({
      title: "Could not load",
      description: "Saved data was not changed.",
      retry: "Try again",
    })[key] ?? key,
}));

vi.mock("@/components/ui/button", () => ({
  Button: (props: ComponentProps<"button">) => <button {...props} />,
}));

vi.mock("./EditorEmptyState", () => ({
  EditorEmptyState: ({
    title,
    description,
    actions,
  }: {
    title: ReactNode;
    description: ReactNode;
    actions: ReactNode;
  }) => (
    <div>
      <h2>{title}</h2>
      <p>{description}</p>
      {actions}
    </div>
  ),
}));

describe("WorkbenchLoadError", () => {
  it("shows a safe retry state without exposing backend details", () => {
    const retry = vi.fn();
    render(<WorkbenchLoadError onRetry={retry} />);

    expect(screen.getByRole("alert")).toBeTruthy();
    expect(screen.getByText("Could not load")).toBeTruthy();
    expect(screen.getByText("Saved data was not changed.")).toBeTruthy();
    expect(screen.queryByText(/503|database|unavailable/i)).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(retry).toHaveBeenCalledTimes(1);
  });
});
