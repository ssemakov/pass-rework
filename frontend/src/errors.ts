import { message } from "./api";
export type ErrorKind = "password" | "legacy" | "storage" | "other";
export function presentError(error: unknown): {
  message: string;
  kind: ErrorKind;
} {
  const text = message(error);
  if (text.includes("verified SQLCipher"))
    return {
      message:
        "Storage encryption could not be verified. Reinstall PassV2 before opening this file.",
      kind: "storage",
    };
  if (text.includes("every-entry verification failed"))
    return {
      message:
        "Could not verify the copied storage. The original file was kept unchanged.",
      kind: "storage",
    };
  if (text.startsWith("lock the current vault"))
    return {
      message: "Lock storage before opening or importing another file.",
      kind: "storage",
    };
  if (text === "file selection failed")
    return {
      message: "The file could not be selected. Try again.",
      kind: "storage",
    };
  if (text.includes("close the old app"))
    return {
      message:
        "Close the original app before importing, or choose a clean backup.",
      kind: "storage",
    };
  if (text === "another storage operation is in progress")
    return {
      message: "Another operation is running. Wait for it to finish.",
      kind: "other",
    };
  if (text === "invalid entry or password")
    return { message: "Check the entry and master password.", kind: "other" };
  if (text === "Session changed")
    return {
      message: "Storage changed. Try the operation again.",
      kind: "other",
    };
  if (text === "clipboard unavailable")
    return {
      message: "Could not copy to the clipboard. Try again.",
      kind: "other",
    };
  if (text.includes("cannot unlock") || text === "Wrong password")
    return { message: "Wrong password", kind: "password" };
  if (text.startsWith("This is a V1 storage"))
    return { message: text, kind: "legacy" };
  if (text.includes("Storage file not found"))
    return { message: "Storage file not found", kind: "storage" };
  if (text.includes("permissions"))
    return {
      message:
        "Storage file permissions prevent opening it. Open another file or import this storage.",
      kind: "storage",
    };
  if (/Storage|storage format|storage location|storage selection/i.test(text))
    return { message: text, kind: "storage" };
  if (text.includes("database operation failed"))
    return {
      message:
        "Storage could not be read. Open another file or import a backup.",
      kind: "storage",
    };
  if (text.includes("vault locked"))
    return {
      message: "Storage is locked. Enter your password again.",
      kind: "storage",
    };
  if (text === "request not authorized")
    return {
      message: "This file selection expired. Select the storage file again.",
      kind: "storage",
    };
  if (text.includes("entry changed"))
    return {
      message: "This secret changed. Reload it before saving.",
      kind: "other",
    };
  if (text.includes("reserved format"))
    return { message: "Choose a different master password.", kind: "storage" };
  return { message: text, kind: "other" };
}
