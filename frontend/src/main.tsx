import React, {
  createContext,
  useContext,
  useCallback,
  useEffect,
  useRef,
  useMemo,
  useState,
} from "react";
import { createRoot } from "react-dom/client";
import {
  call,
  onLock,
  message,
  type Entry,
  type Status,
  type Selection,
  type Input,
  type Patch,
} from "./api";
import { SessionFence } from "./fence";
import { Field, Icon, IconButton } from "./legacy";
import "./style.css";
const Feedback = createContext({ error: "", notice: "" });
const empty: Input = { name: "", login: "", password: "", url: "" };
function App() {
  const [status, setStatus] = useState<Status>({
    locked: true,
    generation: "",
    label: "",
    idleMinutes: 5,
  });
  const [entries, setEntries] = useState<Entry[]>([]),
    [search, setSearch] = useState("");
  const [selected, setSelected] = useState<string | null>(null),
    [revealed, setRevealed] = useState<string | null>(null);
  const [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [busy, setBusyValue] = useState(false);
  const busyRef = useRef(false);
  const setBusy = useCallback((value: boolean) => {
    busyRef.current = value;
    setBusyValue(value);
  }, []);
  const [panel, setPanel] = useState<
    "unlock" | "create" | "import" | "rotate" | "settings" | null
  >(null);
  const [showConfirmation, setShowConfirmation] = useState(false);
  const [passwordFocus, setPasswordFocus] = useState("password");
  const [selection, setSelection] = useState<Selection | null>(null),
    [source, setSource] = useState<Selection | null>(null),
    [destination, setDestination] = useState<Selection | null>(null);
  const [password, setPassword] = useState(""),
    [nextPassword, setNextPassword] = useState(""),
    [confirmation, setConfirmation] = useState("");
  const [editing, setEditing] = useState<Entry | null | undefined>(undefined),
    [draft, setDraft] = useState<Input>(empty),
    [replacePassword, setReplacePassword] = useState(false),
    [deleting, setDeleting] = useState<Entry | null>(null);
  const fence = useRef(new SessionFence()),
    searchSerial = useRef(0),
    revealSerial = useRef(0),
    statusRef = useRef(status);
  const closing = useRef<Promise<boolean> | null>(null);
  const transitionSerial = useRef(0),
    plannedLockGeneration = useRef<string | null>(null);
  statusRef.current = status;
  const conceal = useCallback(() => {
    revealSerial.current++;
    setRevealed(null);
  }, []);
  const scrub = useCallback(() => {
    fence.current.invalidate();
    setEntries([]);
    conceal();
    setSelected(null);
    setEditing(undefined);
    setDraft(empty);
    setPassword("");
    setNextPassword("");
    setConfirmation("");
    setShowConfirmation(false);
    setSelection(null);
    setSource(null);
    setDestination(null);
    setPanel(null);
    setDeleting(null);
    setBusy(false);
    setError("");
    setSearch("");
    statusRef.current = { ...statusRef.current, locked: true };
    setStatus(statusRef.current);
  }, [conceal]);
  const updateState = useCallback(async () => {
    const g = fence.current.capture();
    const current = await call<Status>("State");
    if (
      !fence.current.accepts(g) ||
      BigInt(current.generation) < BigInt(statusRef.current.generation || "0")
    )
      return;
    if (
      (current.locked || current.generation !== statusRef.current.generation) &&
      !(
        statusRef.current.locked &&
        !statusRef.current.generation &&
        current.locked
      )
    ) {
      scrub();
    }
    statusRef.current = current;
    setStatus(current);
  }, [scrub]);
  useEffect(() => {
    let active = true,
      polling = false;
    let dispose = () => {};
    void onLock((generation) => {
      if (
        active &&
        (BigInt(generation) > BigInt(statusRef.current.generation || "0") ||
          (generation === statusRef.current.generation &&
            !statusRef.current.locked))
      ) {
        if (generation !== plannedLockGeneration.current)
          transitionSerial.current++;
        scrub();
        if (closing.current) setBusy(true);
        statusRef.current = { ...statusRef.current, generation };
        setStatus(statusRef.current);
      }
    }).then((d) => {
      if (active) dispose = d;
      else d();
    });
    void updateState().catch(() =>
      setError("Unable to connect to the vault service."),
    );
    const t = setInterval(() => {
      if (polling || busyRef.current) return;
      polling = true;
      const g = fence.current.capture();
      void call<Status>("State")
        .then((s) => {
          if (
            !active ||
            !fence.current.accepts(g) ||
            BigInt(s.generation) < BigInt(statusRef.current.generation || "0")
          )
            return;
          if (busyRef.current && !s.locked) return;
          if (
            (s.locked && !statusRef.current.locked) ||
            s.generation !== statusRef.current.generation
          ) {
            scrub();
            statusRef.current = s;
            setStatus(s);
          }
        })
        .catch(() => {})
        .finally(() => {
          polling = false;
        });
    }, 1000);
    return () => {
      active = false;
      dispose();
      clearInterval(t);
    };
  }, [scrub, updateState]);
  const run = async <T,>(
    task: (check: () => void) => Promise<T>,
    success?: (value: T) => void,
  ) => {
    const g = fence.current.capture();
    setBusy(true);
    setError("");
    setNotice("");
    try {
      const check = () => {
        if (!fence.current.accepts(g)) throw new Error("Session changed");
      };
      const result = await task(check);
      if (fence.current.accepts(g)) success?.(result);
    } catch (e) {
      if (fence.current.accepts(g)) setError(message(e));
    } finally {
      if (fence.current.accepts(g)) setBusy(false);
    }
  };
  const refresh = useCallback(async () => {
    const s = statusRef.current;
    if (s.locked) return;
    const n = ++searchSerial.current,
      g = fence.current.capture();
    try {
      const list = await call<Entry[]>("List", s.generation, "");
      if (n === searchSerial.current && fence.current.accepts(g))
        setEntries(list);
    } catch (e) {
      if (fence.current.accepts(g)) setError(message(e));
    }
  }, []);
  useEffect(() => {
    if (!status.locked) void refresh();
  }, [refresh, status.locked, status.generation]);
  useEffect(() => {
    if (status.locked) return;
    let last = 0;
    const activity = () => {
      if (Date.now() - last > 10000) {
        last = Date.now();
        void call("Activity", statusRef.current.generation).catch(() => {});
      }
    };
    window.addEventListener("keydown", activity);
    window.addEventListener("pointerdown", activity);
    return () => {
      window.removeEventListener("keydown", activity);
      window.removeEventListener("pointerdown", activity);
    };
  }, [status.locked]);
  const lock = (cancelTransition = true): Promise<boolean> => {
    if (cancelTransition) transitionSerial.current++;
    if (closing.current) return closing.current;
    const expected = (
      BigInt(statusRef.current.generation || "0") + 1n
    ).toString();
    plannedLockGeneration.current = expected;
    scrub();
    setBusy(true);
    const task = (async () => {
      try {
        await call("Lock");
        const state = await call<Status>("State");
        // New storage forms wait for this handshake. Never scrub a later form
        // from an old lock continuation, and never apply an unlocked result.
        if (
          !state.locked ||
          BigInt(state.generation) < BigInt(statusRef.current.generation || "0")
        )
          return false;
        statusRef.current = state;
        setStatus(state);
        if (state.generation !== expected) {
          // Another native lock superseded this transition even if its event
          // has not reached the WebView yet. Retain status, discard intent.
          transitionSerial.current++;
          return false;
        }
        return true;
      } catch {
        setError("Could not lock storage. Quit Pass to close it.");
        return false;
      } finally {
        closing.current = null;
        plannedLockGeneration.current = null;
        setBusy(false);
      }
    })();
    closing.current = task;
    return task;
  };
  useEffect(() => {
    const keyboard = (e: KeyboardEvent) => {
      if (
        (e.metaKey || e.ctrlKey) &&
        (e.key === "~" || e.code === "Backquote")
      ) {
        e.preventDefault();
        void lock();
      }
      if (e.key === "Escape") transitionSerial.current++;
      if (e.key === "Escape" && !busy) {
        conceal();
        setEditing(undefined);
        setDraft(empty);
        setPanel(null);
        setPassword("");
        setNextPassword("");
        setConfirmation("");
        setDeleting(null);
      }
    };
    window.addEventListener("keydown", keyboard);
    return () => window.removeEventListener("keydown", keyboard);
  });
  const choose = async (purpose: string, set: (s: Selection) => void) => {
    await run(() => call<Selection>("Choose", purpose), set);
  };
  const openPanel = (p: typeof panel) => {
    setPanel(p);
    setShowConfirmation(false);
    setSelection(null);
    setSource(null);
    setDestination(null);
    setPassword("");
    setNextPassword("");
    setConfirmation("");
    setError("");
    setNotice("");
    conceal();
  };
  const authenticate = (e: React.FormEvent) => {
    e.preventDefault();
    const creating = panel === "create";
    if (!password) return;
    transitionSerial.current++;
    if (creating && !showConfirmation) {
      setShowConfirmation(true);
      return;
    }
    if (creating && password !== confirmation) {
      setError("Passwords don't match");
      return;
    }
    void run(
      async (check) => {
        const picked =
          selection ??
          (await call<Selection>("Choose", creating ? "create" : "open"));
        check();
        setSelection(picked);
        try {
          await call(creating ? "Create" : "Unlock", picked.token, password);
        } catch (e) {
          if (!creating && message(e).includes("cannot unlock"))
            throw new Error("That didn't work");
          throw e;
        }
      },
      () => {
        fence.current.invalidate();
        setBusy(false);
        setPassword("");
        setConfirmation("");
        setShowConfirmation(false);
        setSelection(null);
        setPanel(null);
        void updateState();
      },
    );
  };
  const visible = useMemo(
    () =>
      search.length < 3
        ? []
        : entries.filter((entry) =>
            [entry.name, entry.url].some((value) =>
              value?.toLocaleLowerCase().includes(search.toLocaleLowerCase()),
            ),
          ),
    [entries, search],
  );
  const startEdit = (entry: Entry | null) => {
    conceal();
    if (entry === null) setSearch("");
    setEditing(entry);
    setDraft(
      entry
        ? {
            name: entry.name,
            login: entry.login,
            url: entry.url,
            password: null,
          }
        : { ...empty },
    );
    setReplacePassword(!entry);
    setError("");
  };
  const save = (e: React.FormEvent) => {
    e.preventDefault();
    const original = editing;
    void run(
      () =>
        original
          ? call<Entry>("Update", status.generation, original.id, {
              ...draft,
              replacePassword,
              revision: original.revision,
            } satisfies Patch)
          : call<Entry>("CreateEntry", status.generation, draft),
      (result) => {
        setEditing(undefined);
        setDraft(empty);
        setSelected(result.id);
        setNotice("");
        void refresh();
      },
    );
  };
  const copy = (entry: Entry, field: string) => {
    void run(() => call("Copy", statusRef.current.generation, entry.id, field));
  };
  const reveal = (entry: Entry) => {
    if (selected === entry.id && revealed !== null) {
      conceal();
      return;
    }
    conceal();
    setSelected(entry.id);
    const nonce = ++revealSerial.current;
    void run(
      () => call<Entry>("Get", statusRef.current.generation, entry.id),
      (result) => {
        if (nonce === revealSerial.current) setRevealed(result.password ?? "");
      },
    );
  };
  const backup = () =>
    void run(
      async (check) => {
        const picked = await call<Selection>("Choose", "backup");
        check();
        return call("Backup", status.generation, picked.token);
      },
      () => setNotice("Backup saved and verified"),
    );
  const rotate = (e: React.FormEvent) => {
    e.preventDefault();
    if (nextPassword !== confirmation) {
      setError("Passwords don't match");
      return;
    }
    void run(
      () => call<string>("Rotate", status.generation, password, nextPassword),
      (rollback) => {
        fence.current.invalidate();
        setBusy(false);
        setPassword("");
        setNextPassword("");
        setConfirmation("");
        setPanel(null);
        void updateState().then(() =>
          setNotice("Master Password changed. Backup: " + rollback),
        );
      },
    );
  };
  const importBackup = (e: React.FormEvent) => {
    e.preventDefault();
    if (!source || !destination) return;
    if (nextPassword !== confirmation) {
      setError("Passwords don't match");
      return;
    }
    void run(
      () =>
        call<number>(
          "Import",
          source.token,
          destination.token,
          password,
          nextPassword,
        ),
      (count) => {
        setPassword("");
        setNextPassword("");
        setConfirmation("");
        setSource(null);
        setDestination(null);
        setNotice(`Verified ${count} secrets`);
        setPanel(null);
      },
    );
  };
  const preserve = () => {
    if (!password) {
      setError("Enter the original master password first.");
      return;
    }
    void run(
      async (check) => {
        const src = await call<Selection>("Choose", "source");
        check();
        const dst = await call<Selection>("Choose", "backup");
        check();
        return call("PreserveBackup", src.token, dst.token, password);
      },
      () => {
        setPassword("");
        setSource(null);
        setNotice("Backup saved and verified");
      },
    );
  };
  const begin = async (next: "unlock" | "create" | "import") => {
    if (busyRef.current && !closing.current) return;
    const transition = ++transitionSerial.current;
    const pending = closing.current;
    if (
      pending
        ? !(await pending)
        : !statusRef.current.locked && !(await lock(false))
    )
      return;
    if (transition !== transitionSerial.current) return;
    openPanel(next);
    if (next === "unlock") await choose("open", setSelection);
  };
  useEffect(() => {
    const command = (e: Event) => {
      const name = (e as CustomEvent<string>).detail;
      if (name === "lock") {
        void lock();
        return;
      }
      if (name === "open") {
        void begin("unlock");
        return;
      }
      if (name === "create" || name === "import") {
        void begin(name);
        return;
      }
      if (["new", "backup", "rotate", "settings"].includes(name))
        transitionSerial.current++;
      if (busyRef.current) return;
      if (!statusRef.current.locked) {
        if (name === "new") startEdit(null);
        else if (name === "backup") backup();
        else if (name === "rotate" || name === "settings") openPanel(name);
      }
    };
    window.addEventListener("pass-command", command);
    return () => window.removeEventListener("pass-command", command);
  });
  useEffect(() => {
    if (
      status.locked &&
      status.generation &&
      !busy &&
      !showConfirmation &&
      panel !== "import"
    )
      document.getElementById("master-password")?.focus();
  }, [
    panel,
    selection,
    status.locked,
    status.generation,
    busy,
    showConfirmation,
  ]);
  const field = (key: keyof Input, label: string, type = "text") => (
    <Field
      dense
      label={label}
      type={type}
      autoComplete="off"
      disabled={busy}
      value={draft[key] ?? ""}
      onChange={(e) => setDraft({ ...draft, [key]: e.target.value })}
    />
  );
  const messages = (
    <div className="messages">
      {error && (
        <div role="alert" className="error">
          {error}
        </div>
      )}
      {notice && (
        <div role="status" className="notice">
          {notice}
        </div>
      )}
    </div>
  );
  const returnButton = (label: string) => (
    <IconButton
      name="KeyboardReturn"
      label={label}
      tooltip="enter"
      className="icon-button primary-icon"
      type="submit"
      disabled={busy}
      tabIndex={-1}
    />
  );
  const authPassword = (creating: boolean) => (
    <>
      <Field
        id="master-password"
        aria-label="Master password"
        type="password"
        autoComplete="off"
        disabled={busy}
        placeholder={
          creating
            ? "Enter your storage Master Password"
            : "Enter your Master Password"
        }
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        onFocus={() => setPasswordFocus("password")}
        end={
          !creating || !showConfirmation || passwordFocus === "password"
            ? returnButton(creating ? "Continue" : "Unlock storage")
            : undefined
        }
      />
      {creating && showConfirmation && (
        <div className="confirmation">
          <Field
            autoFocus
            aria-label="Confirm Master Password"
            type="password"
            autoComplete="off"
            disabled={busy}
            placeholder="Please confirm the Master Password"
            value={confirmation}
            onChange={(e) => setConfirmation(e.target.value)}
            onFocus={() => setPasswordFocus("confirmation")}
            end={
              passwordFocus !== "password"
                ? returnButton("Create Storage")
                : undefined
            }
          />
        </div>
      )}
    </>
  );
  const rowField = (entry: Entry, label: "login" | "password" | "url") => {
    const hidden = label === "password",
      shown = hidden && selected === entry.id && revealed !== null;
    const value = hidden
      ? shown
        ? revealed
        : "*****"
      : entry[label === "login" ? "login" : "url"];
    return (
      <div className="item-row">
        <div className="item-label">
          <span>{label}</span>
        </div>
        <div className="item-value">
          {((hidden && entry.hasPassword) || (!hidden && value)) && (
            <div className="show-field">
              <span
                className="field-text"
                aria-label={
                  hidden
                    ? shown
                      ? "Revealed password"
                      : "Password hidden"
                    : undefined
                }
              >
                {value}
              </span>
              <span className="field-controls">
                <IconButton
                  small
                  name="FileCopy"
                  label={
                    hidden
                      ? "Copy password"
                      : label === "login"
                        ? "Copy username"
                        : "Copy URL"
                  }
                  tooltip={"copy " + label}
                  disabled={busy}
                  onClick={() =>
                    copy(
                      entry,
                      hidden ? "password" : label === "login" ? "login" : "url",
                    )
                  }
                />
                {hidden && (
                  <IconButton
                    small
                    name={shown ? "VisibilityOff" : "Visibility"}
                    label={shown ? "Hide password" : "Reveal password"}
                    tooltip={(shown ? "hide " : "reveal ") + label}
                    disabled={busy}
                    onClick={() => reveal(entry)}
                  />
                )}
              </span>
            </div>
          )}
        </div>
      </div>
    );
  };
  return (
    <Feedback.Provider value={{ error, notice }}>
      <div className="app" data-ready={status.generation !== ""}>
        {status.locked && panel !== "import" ? (
          <main className="auth-container">
            <form onSubmit={authenticate} className="auth-stack">
              <div className="auth-group">
                {panel === "create" && (
                  <div className="storage-location">
                    <span>{selection?.label ?? "storage.pass"}</span>
                    <IconButton
                      name="FolderOpen"
                      label="Select storage path"
                      disabled={busy}
                      onClick={() => choose("create", setSelection)}
                    />
                  </div>
                )}
                {authPassword(panel === "create")}
                <div className="auth-feedback">{messages}</div>
              </div>
            </form>
          </main>
        ) : editing !== undefined ? (
          <main className="fixed-container entry-form">
            <form
              onSubmit={save}
              aria-label={editing ? "Edit secret" : "Add new secret"}
            >
              {field("name", "Item name")}
              {field("login", "Username")}
              {editing && (
                <label className="replace-password">
                  <input
                    type="checkbox"
                    checked={replacePassword}
                    disabled={busy}
                    onChange={(e) => {
                      setReplacePassword(e.target.checked);
                      setDraft({
                        ...draft,
                        password: e.target.checked ? "" : null,
                      });
                    }}
                  />
                  Replace password
                </label>
              )}
              {(!editing || replacePassword) &&
                field("password", "Password", "password")}
              {field("url", "Website URL")}
              {messages}
              <div className="form-actions">
                <button
                  type="button"
                  className="button outlined"
                  disabled={busy}
                  onClick={() => {
                    setEditing(undefined);
                    setDraft(empty);
                    setError("");
                  }}
                >
                  Cancel
                </button>
                <button className="button contained" disabled={busy}>
                  Save
                </button>
              </div>
            </form>
          </main>
        ) : panel === "import" ? (
          <main className="fixed-container transfer-form">
            <form onSubmit={importBackup} aria-label="Import Storage">
              <Field
                dense
                label="Backup Master Password"
                type="password"
                required
                autoComplete="off"
                disabled={busy}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
              <div className="storage-location">
                <span>{source?.label ?? "Select encrypted backup"}</span>
                <IconButton
                  name="FolderOpen"
                  label="Select encrypted backup"
                  disabled={busy}
                  onClick={() => choose("source", setSource)}
                />
              </div>
              <div className="storage-location">
                <span>{destination?.label ?? "Select storage path"}</span>
                <IconButton
                  name="FolderOpen"
                  label="Select fresh destination"
                  disabled={busy}
                  onClick={() => choose("destination", setDestination)}
                />
              </div>
              <Field
                dense
                label="New Master Password"
                type="password"
                required
                autoComplete="off"
                disabled={busy}
                value={nextPassword}
                onChange={(e) => setNextPassword(e.target.value)}
              />
              <Field
                dense
                label="Confirm Master Password"
                type="password"
                required
                autoComplete="off"
                disabled={busy}
                value={confirmation}
                onChange={(e) => setConfirmation(e.target.value)}
              />
              <button
                type="button"
                className="button text-button"
                disabled={busy}
                onClick={preserve}
              >
                Preserve original backup
              </button>
              {messages}
              <div className="form-actions">
                <button
                  type="button"
                  className="button outlined"
                  disabled={busy}
                  onClick={() => openPanel(null)}
                >
                  Cancel
                </button>
                <button
                  className="button contained"
                  disabled={busy || !source || !destination}
                >
                  Import
                </button>
              </div>
            </form>
          </main>
        ) : (
          <main className="fixed-container home">
            <div className="home-heading">
              <span className="tooltip-anchor">
                <button
                  className="fab"
                  aria-label="Add new secret"
                  disabled={busy}
                  onClick={() => startEdit(null)}
                >
                  <Icon name="Add" />
                </button>
                <span role="tooltip" className="tooltip bottom">
                  Add new secret
                </span>
              </span>
              <span className="secret-count">
                {entries.length} secrets stored
              </span>
            </div>
            <div className="search">
              <Field
                autoFocus
                label="Search"
                aria-label="Search stored secrets"
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  conceal();
                }}
              />
            </div>
            {messages}
            <div className="items-list" aria-label="Stored secrets">
              {visible.map((entry) => (
                <article
                  className="item-card"
                  key={entry.id}
                  aria-label={entry.name ?? ""}
                >
                  <div className="card-content">
                    <div className="item-name">{entry.name}</div>
                    {rowField(entry, "login")}
                    {rowField(entry, "password")}
                    {rowField(entry, "url")}
                    <div className="item-actions">
                      <IconButton
                        small
                        name="Edit"
                        label="Edit secret"
                        tooltip="edit secret"
                        disabled={busy}
                        onClick={() => startEdit(entry)}
                      />
                      <IconButton
                        small
                        name="Delete"
                        label="Delete secret"
                        tooltip="delete secret"
                        disabled={busy}
                        onClick={() => {
                          conceal();
                          setDeleting(entry);
                        }}
                      />
                    </div>
                  </div>
                </article>
              ))}
            </div>
          </main>
        )}
        {!status.locked && panel === "settings" && (
          <Modal
            title="Storage settings"
            close={() => openPanel(null)}
            busy={busy}
          >
            <label className="setting-label">
              Lock after inactivity
              <select
                aria-label="Lock after inactivity"
                value={status.idleMinutes}
                disabled={busy}
                onChange={(e) =>
                  void run(
                    () =>
                      call(
                        "SetIdle",
                        status.generation,
                        Number(e.target.value),
                      ),
                    () => void updateState(),
                  )
                }
              >
                <option value={1}>1 minute</option>
                <option value={5}>5 minutes</option>
                <option value={15}>15 minutes</option>
              </select>
            </label>
            <div className="form-actions">
              <button
                className="button outlined"
                onClick={() => openPanel(null)}
              >
                Close
              </button>
            </div>
          </Modal>
        )}
        {!status.locked && panel === "rotate" && (
          <Modal
            title="Change Master Password"
            close={() => openPanel(null)}
            busy={busy}
          >
            <form onSubmit={rotate}>
              <Field
                dense
                label="Current Master Password"
                type="password"
                required
                autoComplete="off"
                disabled={busy}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
              <Field
                dense
                label="New Master Password"
                type="password"
                required
                autoComplete="off"
                disabled={busy}
                value={nextPassword}
                onChange={(e) => setNextPassword(e.target.value)}
              />
              <Field
                dense
                label="Confirm Master Password"
                type="password"
                required
                autoComplete="off"
                disabled={busy}
                value={confirmation}
                onChange={(e) => setConfirmation(e.target.value)}
              />
              <p className="caption">
                The encrypted backup uses the old Master Password.
              </p>
              <div className="form-actions">
                <button
                  type="button"
                  className="button outlined"
                  disabled={busy}
                  onClick={() => openPanel(null)}
                >
                  Cancel
                </button>
                <button className="button contained" disabled={busy}>
                  Save
                </button>
              </div>
            </form>
          </Modal>
        )}
        {deleting && (
          <Modal
            title="Delete secret?"
            close={() => setDeleting(null)}
            busy={busy}
          >
            <p>{deleting.name}</p>
            <p className="caption">
              Encrypted backups may still contain this secret.
            </p>
            <div className="form-actions">
              <button
                className="button outlined"
                disabled={busy}
                onClick={() => setDeleting(null)}
              >
                Cancel
              </button>
              <button
                className="button contained"
                disabled={busy}
                onClick={() =>
                  void run(
                    () =>
                      call(
                        "Delete",
                        status.generation,
                        deleting.id,
                        deleting.revision,
                      ),
                    () => {
                      setDeleting(null);
                      setSelected(null);
                      conceal();
                      setNotice("");
                      void refresh();
                    },
                  )
                }
              >
                Delete
              </button>
            </div>
          </Modal>
        )}
      </div>
    </Feedback.Provider>
  );
}

function Modal({
  title,
  children,
  close,
  busy,
}: {
  title: string;
  children: React.ReactNode;
  close: () => void;
  busy: boolean;
}) {
  const feedback = useContext(Feedback);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    const node = ref.current;
    const first = node?.querySelector<HTMLElement>("input,button,select");
    first?.focus();
    const trap = (e: KeyboardEvent) => {
      if (e.key !== "Tab") return;
      const all = [
        ...node!.querySelectorAll<HTMLElement>(
          "button:not([disabled]),input:not([disabled]),select:not([disabled])",
        ),
      ];
      if (!all.length) return;
      const first = all[0],
        last = all[all.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    };
    node?.addEventListener("keydown", trap);
    return () => {
      node?.removeEventListener("keydown", trap);
      previous?.focus();
    };
  }, []);
  return (
    <div className="overlay">
      <div
        ref={ref}
        className="modal"
        role="dialog"
        aria-modal="true"
        aria-label={title}
      >
        <div className="modal-heading">
          <h2>{title}</h2>
          <IconButton
            name="Close"
            label="Close dialog"
            disabled={busy}
            onClick={close}
          />
        </div>
        {feedback.error && (
          <div role="alert" className="error">
            {feedback.error}
          </div>
        )}
        {feedback.notice && (
          <div role="status" className="notice">
            {feedback.notice}
          </div>
        )}
        {children}
      </div>
    </div>
  );
}
createRoot(document.getElementById("root")!).render(<App />);
