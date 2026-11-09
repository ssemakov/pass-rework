export interface Status {
  generation: string;
  locked: boolean;
  label: string;
  idleMinutes: number;
}
export interface Entry {
  id: string;
  name: string | null;
  login: string | null;
  password?: string | null;
  url: string | null;
  createdAt: string | null;
  updatedAt: string | null;
  revision: string;
  hasPassword: boolean;
}
export interface Selection {
  token: string;
  label: string;
}
export type Input = {
  name: string | null;
  login: string | null;
  password: string | null;
  url: string | null;
};
export type Patch = Input & { replacePassword: boolean; revision: string };
export async function call<T>(method: string, ...args: unknown[]): Promise<T> {
  if (import.meta.env.MODE === "e2e") {
    const r = await fetch("/test-api/" + method, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(args),
    });
    const value = await r.json();
    if (!r.ok) throw new Error(value.error);
    return value;
  }
  const runtime = await nativeRuntime();
  return runtime.Call.ByName(
    "github.com/ssemakov/pass-rework/internal/service.Facade." + method,
    ...args,
  ) as Promise<T>;
}
export async function onLock(
  callback: (generation: string) => void,
): Promise<() => void> {
  if (import.meta.env.MODE === "e2e") {
    const listener = (event: Event) =>
      callback((event as CustomEvent<string>).detail);
    window.addEventListener("pass-test-lock", listener);
    return () => window.removeEventListener("pass-test-lock", listener);
  }
  const { Events } = await import("@wailsio/runtime");
  return Events.On("vault-locked", (event) => {
    if (typeof event.data === "string") callback(event.data);
  });
}
export function message(e: unknown) {
  if (e instanceof Error) return e.message;
  return "That operation could not be completed. Your original vault is preserved.";
}

let native: Promise<typeof import("@wailsio/runtime")> | null = null;
async function nativeRuntime() {
  if (!native)
    native = import("@wailsio/runtime")
      .then(async (runtime) => {
        const capability = await waitForCapability(() => {
          try {
            return runtime.Flags.GetFlag("vaultCapability");
          } catch {
            return undefined;
          }
        });
        if (typeof capability !== "string" || !capability)
          throw new Error("Desktop authorization unavailable");
        runtime.setTransport({
          async call(object, method, _windowName, args) {
            const response = await fetch("/wails/runtime", {
              method: "POST",
              headers: {
                "Content-Type": "application/json",
                "X-Pass-Capability": capability,
              },
              body: JSON.stringify({ object, method, args }),
            });
            if (!response.ok) {
              let reason = "Operation failed";
              try {
                const body = await response.json();
                reason =
                  typeof body.cause === "string"
                    ? body.cause
                    : (body.message ?? reason);
              } catch {}
              throw new Error(reason);
            }
            return response.headers
              .get("Content-Type")
              ?.includes("application/json")
              ? response.json()
              : response.text();
          },
        });
        return runtime;
      })
      .catch((error) => {
        native = null;
        throw error;
      });
  return native;
}

export async function waitForCapability(
  read: () => unknown,
  timeout = 5000,
): Promise<string> {
  const start = Date.now();
  while (Date.now() - start < timeout) {
    const value = read();
    if (typeof value === "string" && value) return value;
    await new Promise((resolve) => setTimeout(resolve, 20));
  }
  throw new Error("Desktop authorization unavailable");
}
