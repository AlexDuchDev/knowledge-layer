"use client";

import { useRef, useState } from "react";
import { apiBase, apiHeaders, formatApiClientError, isDevPrincipalHeader, principalUserId } from "@/lib/api";

type Tab = "file" | "text" | "url" | "youtube";

const MAX_BYTES = 50 * 1024 * 1024;

export function UploadModal({
  feedId,
  onClose,
  onUploaded,
}: {
  feedId: string;
  onClose: () => void;
  onUploaded: () => void;
}) {
  const [tab, setTab] = useState<Tab>("file");

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-xl rounded-lg bg-white shadow-xl">
        <div className="flex items-center justify-between border-b px-5 py-3">
          <h2 className="text-base font-semibold">Add content</h2>
          <button
            type="button"
            onClick={onClose}
            className="text-gray-500 hover:text-gray-800"
            aria-label="Close"
          >
            ×
          </button>
        </div>
        <div className="border-b">
          <nav className="flex">
            {(["file", "text", "url", "youtube"] as Tab[]).map((t) => (
              <button
                key={t}
                type="button"
                onClick={() => setTab(t)}
                className={`flex-1 px-3 py-2 text-sm font-medium ${
                  tab === t ? "border-b-2 border-blue-700 text-blue-700" : "text-gray-600 hover:text-gray-900"
                }`}
              >
                {t === "file" ? "File" : t === "text" ? "Paste text" : t === "url" ? "Web URL" : "YouTube"}
              </button>
            ))}
          </nav>
        </div>
        <div className="px-5 py-4">
          {tab === "file" ? <FileTab feedId={feedId} onUploaded={onUploaded} /> : null}
          {tab === "text" ? <TextTab feedId={feedId} onUploaded={onUploaded} /> : null}
          {tab === "url" ? <UrlTab feedId={feedId} onUploaded={onUploaded} /> : null}
          {tab === "youtube" ? <YouTubeTab feedId={feedId} onUploaded={onUploaded} /> : null}
        </div>
      </div>
    </div>
  );
}

function StatusBox({ status }: { status: { kind: "ok" | "err" | "info"; message: string } | null }) {
  if (!status) return null;
  const cls =
    status.kind === "err"
      ? "border-red-200 bg-red-50 text-red-900"
      : status.kind === "ok"
        ? "border-green-200 bg-green-50 text-green-900"
        : "border-blue-200 bg-blue-50 text-blue-900";
  return <div className={`mt-3 rounded-md border px-3 py-2 text-sm ${cls}`}>{status.message}</div>;
}

async function postJSON(path: string, body: unknown): Promise<unknown> {
  const res = await fetch(`${apiBase()}${path}`, {
    method: "POST",
    credentials: "include",
    headers: apiHeaders(),
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status} ${text.slice(0, 400)}`);
  }
  return res.json();
}

function FileTab({ feedId, onUploaded }: { feedId: string; onUploaded: () => void }) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [files, setFiles] = useState<File[]>([]);
  const [progress, setProgress] = useState<{ name: string; pct: number } | null>(null);
  const [status, setStatus] = useState<{ kind: "ok" | "err" | "info"; message: string } | null>(null);

  function selectFiles(list: FileList | null) {
    if (!list) return;
    const arr = Array.from(list);
    const oversize = arr.find((f) => f.size > MAX_BYTES);
    if (oversize) {
      setStatus({ kind: "err", message: `${oversize.name} exceeds 50 MB limit.` });
      return;
    }
    setFiles(arr);
    setStatus(null);
  }

  function uploadOne(file: File): Promise<{ deduped: boolean; raw_artifact: { id: string } } | null> {
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      const url = `${apiBase()}/api/manual/collections/${encodeURIComponent(feedId)}/file`;
      xhr.open("POST", url, true);
      xhr.withCredentials = true;
      // apiHeaders sets Content-Type: application/json which kills multipart;
      // copy only the dev principal header when present.
      if (isDevPrincipalHeader()) {
        xhr.setRequestHeader("X-Principal-User-ID", principalUserId());
      }
      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable) {
          setProgress({ name: file.name, pct: Math.round((e.loaded / e.total) * 100) });
        }
      };
      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          try {
            resolve(JSON.parse(xhr.responseText));
          } catch {
            resolve(null);
          }
        } else {
          reject(new Error(`${xhr.status} ${xhr.responseText.slice(0, 400)}`));
        }
      };
      xhr.onerror = () => reject(new Error("network error"));
      const fd = new FormData();
      fd.append("file", file, file.name);
      xhr.send(fd);
    });
  }

  async function uploadAll() {
    if (files.length === 0) return;
    setStatus({ kind: "info", message: `Uploading ${files.length} file(s)…` });
    let ok = 0;
    let dup = 0;
    let fail = 0;
    for (const f of files) {
      try {
        const r = await uploadOne(f);
        if (r && (r as { deduped?: boolean }).deduped) {
          dup++;
        } else {
          ok++;
        }
      } catch (e) {
        fail++;
        console.error("upload failed", f.name, e);
      }
    }
    setProgress(null);
    setFiles([]);
    if (inputRef.current) inputRef.current.value = "";
    const parts: string[] = [];
    if (ok) parts.push(`${ok} added`);
    if (dup) parts.push(`${dup} skipped (already in collection)`);
    if (fail) parts.push(`${fail} failed`);
    setStatus({ kind: fail ? "err" : "ok", message: parts.join(" · ") });
    onUploaded();
  }

  return (
    <div>
      <div
        className="rounded-md border-2 border-dashed border-gray-300 px-4 py-8 text-center hover:border-blue-400"
        onDragOver={(e) => {
          e.preventDefault();
        }}
        onDrop={(e) => {
          e.preventDefault();
          selectFiles(e.dataTransfer.files);
        }}
      >
        <p className="text-sm text-gray-600">
          Drag & drop files here, or{" "}
          <button
            type="button"
            onClick={() => inputRef.current?.click()}
            className="text-blue-700 underline hover:text-blue-900"
          >
            browse
          </button>
          .
        </p>
        <p className="mt-1 text-xs text-gray-400">PDF, DOCX, TXT, MD, CSV, JSON, HTML — up to 50 MB each.</p>
        <input
          ref={inputRef}
          type="file"
          multiple
          className="hidden"
          accept=".pdf,.docx,.txt,.md,.markdown,.csv,.json,.html,.htm"
          onChange={(e) => selectFiles(e.target.files)}
        />
      </div>

      {files.length > 0 ? (
        <ul className="mt-3 space-y-1 text-sm">
          {files.map((f) => (
            <li key={f.name} className="text-gray-700">
              {f.name} <span className="text-gray-400">({Math.round(f.size / 1024)} KB)</span>
            </li>
          ))}
        </ul>
      ) : null}

      {progress ? (
        <p className="mt-2 text-xs text-gray-500">
          Uploading {progress.name} — {progress.pct}%
        </p>
      ) : null}

      <div className="mt-4 flex gap-2">
        <button
          type="button"
          disabled={files.length === 0}
          onClick={() => void uploadAll()}
          className="rounded-md bg-blue-700 px-3 py-1.5 text-sm text-white hover:bg-blue-800 disabled:bg-gray-400"
        >
          Upload {files.length > 0 ? `(${files.length})` : ""}
        </button>
      </div>
      <StatusBox status={status} />
    </div>
  );
}

function TextTab({ feedId, onUploaded }: { feedId: string; onUploaded: () => void }) {
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [source, setSource] = useState("");
  const [status, setStatus] = useState<{ kind: "ok" | "err" | "info"; message: string } | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(ev: React.FormEvent) {
    ev.preventDefault();
    if (!body.trim()) {
      setStatus({ kind: "err", message: "Body is required." });
      return;
    }
    setBusy(true);
    setStatus({ kind: "info", message: "Saving…" });
    try {
      const r = (await postJSON(`/api/manual/collections/${encodeURIComponent(feedId)}/text`, {
        title,
        body,
        source_attribution: source,
      })) as { deduped?: boolean };
      setStatus({ kind: "ok", message: r.deduped ? "Already in collection." : "Text added." });
      setTitle("");
      setBody("");
      setSource("");
      onUploaded();
    } catch (e) {
      setStatus({ kind: "err", message: formatApiClientError(e) });
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={submit} className="space-y-3">
      <div>
        <label className="block text-xs font-medium text-gray-700">Title (optional)</label>
        <input
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Auto-generated from first line if blank"
          className="mt-1 w-full rounded border border-gray-300 px-2 py-1 text-sm"
        />
      </div>
      <div>
        <label className="block text-xs font-medium text-gray-700">Body</label>
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          rows={10}
          placeholder="Paste content here"
          className="mt-1 w-full rounded border border-gray-300 px-2 py-1 font-mono text-xs"
          required
        />
      </div>
      <div>
        <label className="block text-xs font-medium text-gray-700">Source attribution (optional)</label>
        <input
          type="text"
          value={source}
          onChange={(e) => setSource(e.target.value)}
          placeholder="e.g. customer interview 2025-04-12"
          className="mt-1 w-full rounded border border-gray-300 px-2 py-1 text-sm"
        />
      </div>
      <button
        type="submit"
        disabled={busy}
        className="rounded-md bg-blue-700 px-3 py-1.5 text-sm text-white hover:bg-blue-800 disabled:bg-gray-400"
      >
        {busy ? "Saving…" : "Add text"}
      </button>
      <StatusBox status={status} />
    </form>
  );
}

function UrlTab({ feedId, onUploaded }: { feedId: string; onUploaded: () => void }) {
  const [url, setUrl] = useState("");
  const [status, setStatus] = useState<{ kind: "ok" | "err" | "info"; message: string } | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(ev: React.FormEvent) {
    ev.preventDefault();
    if (!url.trim()) return;
    setBusy(true);
    setStatus({ kind: "info", message: "Fetching…" });
    try {
      const r = (await postJSON(`/api/manual/collections/${encodeURIComponent(feedId)}/url`, { url })) as {
        deduped?: boolean;
      };
      setStatus({ kind: "ok", message: r.deduped ? "Already in collection." : "Page added." });
      setUrl("");
      onUploaded();
    } catch (e) {
      setStatus({ kind: "err", message: formatApiClientError(e) });
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={submit} className="space-y-3">
      <div>
        <label className="block text-xs font-medium text-gray-700">URL</label>
        <input
          type="url"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://example.com/article"
          className="mt-1 w-full rounded border border-gray-300 px-2 py-1 text-sm"
          required
        />
        <p className="mt-1 text-xs text-gray-500">
          The server fetches the page, extracts the title and readable text, and stores it. Pages behind login are not supported.
        </p>
      </div>
      <button
        type="submit"
        disabled={busy}
        className="rounded-md bg-blue-700 px-3 py-1.5 text-sm text-white hover:bg-blue-800 disabled:bg-gray-400"
      >
        {busy ? "Fetching…" : "Add page"}
      </button>
      <StatusBox status={status} />
    </form>
  );
}

function YouTubeTab({ feedId, onUploaded }: { feedId: string; onUploaded: () => void }) {
  const [url, setUrl] = useState("");
  const [status, setStatus] = useState<{ kind: "ok" | "err" | "info"; message: string } | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(ev: React.FormEvent) {
    ev.preventDefault();
    if (!url.trim()) return;
    setBusy(true);
    setStatus({ kind: "info", message: "Fetching transcript…" });
    try {
      const r = (await postJSON(`/api/manual/collections/${encodeURIComponent(feedId)}/youtube`, { url })) as {
        deduped?: boolean;
      };
      setStatus({ kind: "ok", message: r.deduped ? "Already in collection." : "Video transcript added." });
      setUrl("");
      onUploaded();
    } catch (e) {
      setStatus({ kind: "err", message: formatApiClientError(e) });
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={submit} className="space-y-3">
      <div>
        <label className="block text-xs font-medium text-gray-700">YouTube URL or video ID</label>
        <input
          type="text"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://youtube.com/watch?v=… or 11-char video ID"
          className="mt-1 w-full rounded border border-gray-300 px-2 py-1 text-sm"
          required
        />
        <p className="mt-1 text-xs text-gray-500">
          Uses existing captions (manual or auto-generated). Videos without captions are stored with metadata only — no transcript.
        </p>
      </div>
      <button
        type="submit"
        disabled={busy}
        className="rounded-md bg-blue-700 px-3 py-1.5 text-sm text-white hover:bg-blue-800 disabled:bg-gray-400"
      >
        {busy ? "Fetching…" : "Add video"}
      </button>
      <StatusBox status={status} />
    </form>
  );
}
