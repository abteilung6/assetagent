export type UserDisplayInput = {
  givenName?: string | null;
  displayName: string;
  email?: string | null;
};

/** Prefer given name, else first token of display name, else email local-part. */
export function primaryDisplayName(user: UserDisplayInput): string {
  const given = user.givenName?.trim();
  if (given) {
    return given;
  }
  const display = user.displayName.trim();
  if (display) {
    return display.split(/\s+/)[0] ?? display;
  }
  const email = user.email?.trim() ?? "";
  if (email.includes("@")) {
    return email.split("@")[0] ?? email;
  }
  return email || "User";
}

export function initialsFromName(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return "?";
  }
  if (parts.length === 1) {
    return parts[0]!.slice(0, 2).toUpperCase();
  }
  return `${parts[0]![0] ?? ""}${parts[1]![0] ?? ""}`.toUpperCase();
}
