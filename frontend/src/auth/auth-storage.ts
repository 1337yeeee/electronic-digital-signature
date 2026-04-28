import type { User } from "../types/auth";

const TOKEN_KEY = "eds_lab_access_token";
const USER_KEY = "eds_lab_user";

export function loadStoredToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function loadStoredUser(): User | null {
  const raw = localStorage.getItem(USER_KEY);
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw) as User;
  } catch {
    localStorage.removeItem(USER_KEY);
    return null;
  }
}

export function persistSession(token: string, user: User) {
  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function clearStoredSession() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
}
