const SESSION_KEY = "store.session";

export function getSession() {
  try {
    const raw = localStorage.getItem(SESSION_KEY);
    if (!raw) return null;
    const session = JSON.parse(raw);
    if (!session?.accessToken || !session?.user?.id) return null;
    return session;
  } catch {
    return null;
  }
}

export function setSession(auth) {
  const session = {
    accessToken: auth.accessToken,
    user: auth.user,
  };
  localStorage.setItem(SESSION_KEY, JSON.stringify(session));
  return session;
}

export function clearSession() {
  localStorage.removeItem(SESSION_KEY);
}
