import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { BookmarkReplacePathNotice } from "./BookmarkReplacePathNotice";

afterEach(cleanup);

const copy = {
  title: "Your scenarios become the game's starting list",
  description:
    "The standard 1936 and 1939 entries are replaced by this project's scenarios.",
  singleBookmarkWarning:
    "With one scenario, HOI4 skips the scenario list. After country selection, players cannot return to the main menu.",
  singleBookmarkAction:
    "Add a second scenario to keep the scenario list available.",
  addSecondBookmarkLabel: "Add second scenario",
  secondBookmarkDraftMessage: "The second scenario is not saved yet.",
  navigationReadyMessage:
    "Players can now use and return from scenario selection.",
  isCreatingSecond: false,
  showNavigationReady: false,
  onAddSecond: () => undefined,
};

function renderNotice(bookmarkCount: number) {
  return render(
    <BookmarkReplacePathNotice bookmarkCount={bookmarkCount} {...copy} />,
  );
}

describe("BookmarkReplacePathNotice", () => {
  it("renders nothing when no Bookmark will be exported", () => {
    const view = renderNotice(0);
    expect(view.container.firstChild).toBeNull();
    expect(screen.queryByRole("note")).toBeNull();
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("shows the replacement result and actionable risk for one Bookmark", () => {
    renderNotice(1);
    expect(
      screen.getByRole("note", { name: copy.title }).textContent,
    ).toContain(copy.description);
    expect(screen.getByRole("alert").textContent).toContain(
      copy.singleBookmarkWarning,
    );
    expect(screen.getByRole("alert").textContent).toContain(
      copy.singleBookmarkAction,
    );
    expect(
      screen.getByRole("button", { name: copy.addSecondBookmarkLabel }),
    ).not.toBeNull();
  });

  it("replaces the repeated action with in-progress copy while the second draft is open", () => {
    render(
      <BookmarkReplacePathNotice
        bookmarkCount={1}
        {...copy}
        isCreatingSecond
      />,
    );
    expect(screen.getByRole("alert").textContent).toContain(
      copy.secondBookmarkDraftMessage,
    );
    expect(
      screen.queryByRole("button", { name: copy.addSecondBookmarkLabel }),
    ).toBeNull();
  });

  it("shows only the neutral replacement result for two Bookmarks", () => {
    renderNotice(2);
    expect(
      screen.getByRole("note", { name: copy.title }).textContent,
    ).toContain(copy.description);
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("closes the loop after the second Bookmark is saved", () => {
    render(
      <BookmarkReplacePathNotice
        bookmarkCount={2}
        {...copy}
        showNavigationReady
      />,
    );
    expect(screen.getByRole("status").textContent).toContain(
      copy.navigationReadyMessage,
    );
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("never offers a checkbox because Bookmark export controls this behavior", () => {
    renderNotice(1);
    expect(screen.queryByRole("checkbox")).toBeNull();
  });
});
