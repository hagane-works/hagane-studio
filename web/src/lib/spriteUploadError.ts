export class SpriteUploadError extends Error {
  constructor(
    readonly status: number,
    readonly detail: string,
  ) {
    super(detail);
    this.name = "SpriteUploadError";
  }
}
