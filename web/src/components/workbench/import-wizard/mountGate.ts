export function shouldMountModImport({
  countryCount,
  hadPendingFolder,
  isLoadExistingMod,
}: {
  countryCount: number;

  hadPendingFolder: boolean;

  isLoadExistingMod: boolean;
}): boolean {
  return countryCount > 0 || hadPendingFolder || isLoadExistingMod;
}
