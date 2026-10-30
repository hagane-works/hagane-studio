import { describe, expect, it } from "vitest";
import { avatarSource, parseAvatarSelection } from "./avatar";

describe("avatar selection", () => {
  it("uses the engineer dog when saved data is absent or invalid", () => {
    expect(parseAvatarSelection(null)).toEqual({
      kind: "pet",
      id: "field_engineer_dog",
    });
    expect(parseAvatarSelection('{"kind":"pet","id":"unknown"}')).toEqual({
      kind: "pet",
      id: "field_engineer_dog",
    });
  });

  it("accepts a built-in pet and a local custom image", () => {
    const pet = parseAvatarSelection(
      '{"kind":"pet","id":"admiral_penguin"}',
    );
    const custom = parseAvatarSelection(
      '{"kind":"custom","dataUrl":"data:image/webp;base64,AAAA"}',
    );
    expect(avatarSource(pet)).toBe("/avatars/pets/admiral_penguin.png");
    expect(avatarSource(custom)).toBe("data:image/webp;base64,AAAA");
  });
});
