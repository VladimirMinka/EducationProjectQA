import { computed, ref } from "vue";
import { getSession as readSession, setSession as writeSession, clearSession as wipeSession } from "../lib/auth";

const sessionRef = ref(readSession());

export function useSession() {
  const session = computed(() => sessionRef.value);

  function setSession(auth) {
    sessionRef.value = writeSession(auth);
    return sessionRef.value;
  }

  function clearSession() {
    wipeSession();
    sessionRef.value = null;
  }

  function refreshSession() {
    sessionRef.value = readSession();
    return sessionRef.value;
  }

  return { session, setSession, clearSession, refreshSession };
}
