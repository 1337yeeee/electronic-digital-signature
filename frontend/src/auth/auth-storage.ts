import type { User } from "../types/auth";

const TOKEN_KEY = "eds_lab_access_token";
const USER_KEY = "eds_lab_user";
const storage = window.sessionStorage;

function clearLegacyLocalStorage() {
  window.localStorage.removeItem(TOKEN_KEY);
  window.localStorage.removeItem(USER_KEY);
}

export function loadStoredToken(): string | null {
  clearLegacyLocalStorage();
  return storage.getItem(TOKEN_KEY);
}

export function loadStoredUser(): User | null {
  clearLegacyLocalStorage();
  const raw = storage.getItem(USER_KEY);
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw) as User;
  } catch {
    storage.removeItem(USER_KEY);
    return null;
  }
}

export function persistSession(token: string, user: User) {
  storage.setItem(TOKEN_KEY, token);
  storage.setItem(USER_KEY, JSON.stringify(user));
}

export function clearStoredSession() {
  storage.removeItem(TOKEN_KEY);
  storage.removeItem(USER_KEY);
}
