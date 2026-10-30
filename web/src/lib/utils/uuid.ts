type UUIDCrypto = {
  randomUUID?: () => string;
  getRandomValues?: (values: Uint8Array) => Uint8Array;
};

let fallbackSequence = 0;

function fillFallbackRandom(values: Uint8Array): void {
  fallbackSequence = (fallbackSequence + 1) >>> 0;
  const now = Date.now();
  for (let index = 0; index < values.length; index += 1) {
    const timeByte = Math.floor(now / 2 ** ((index % 6) * 8)) & 0xff;
    const sequenceByte = (fallbackSequence >>> ((index % 4) * 8)) & 0xff;
    values[index] = Math.floor(Math.random() * 256) ^ timeByte ^ sequenceByte;
  }
}

export function createUUID(
  provider: UUIDCrypto | undefined = typeof globalThis === "undefined"
    ? undefined
    : (globalThis.crypto as UUIDCrypto | undefined),
): string {
  if (typeof provider?.randomUUID === "function") {
    try {
      return provider.randomUUID.call(provider);
    } catch {}
  }

  const bytes = new Uint8Array(16);
  if (typeof provider?.getRandomValues === "function") {
    try {
      provider.getRandomValues.call(provider, bytes);
    } catch {
      fillFallbackRandom(bytes);
    }
  } else {
    fillFallbackRandom(bytes);
  }

  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = Array.from(bytes, (value) => value.toString(16).padStart(2, "0"));
  return `${hex.slice(0, 4).join("")}-${hex.slice(4, 6).join("")}-${hex
    .slice(6, 8)
    .join("")}-${hex.slice(8, 10).join("")}-${hex.slice(10).join("")}`;
}
