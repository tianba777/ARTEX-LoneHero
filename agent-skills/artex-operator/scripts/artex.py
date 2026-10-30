#!/usr/bin/env python3
"""Portable, single-origin ARTEX API client. Python standard library only."""
import argparse
import base64
import getpass
import hashlib
import http.cookiejar
import json
import mimetypes
import os
from pathlib import Path
import re
import secrets
import socket
import stat
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request


class ClientError(Exception):
    pass


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        fp.close()
        raise ClientError("Redirect refused; configure the final ARTEX base URL. A write may already have applied.")


SECRET_KEYS = {"token", "access_token", "api_key", "password", "old_password", "new_password",
               "authorization", "cookie", "set-cookie", "secret", "password_hash"}
PUBLIC = {"/api/health", "/api/auth/status", "/api/auth/login", "/api/auth/init", "/api/basic-auth/check"}
TERMINAL = {"done", "failed", "timeout"}
KNOWN_SECRETS = set()


def redact(value):
    if isinstance(value, dict):
        return {k: "[redacted]" if k.lower().replace("-", "_") in {s.replace("-", "_") for s in SECRET_KEYS}
                else redact(v) for k, v in value.items()}
    if isinstance(value, list):
        return [redact(v) for v in value]
    if isinstance(value, str):
        for secret in KNOWN_SECRETS:
            if secret:
                value = value.replace(secret, "[redacted]")
        for name in ("ARTEX_TOKEN", "ARTEX_PASSWORD", "ARTEX_BASIC_PASSWORD"):
            secret = os.environ.get(name)
            if secret:
                value = value.replace(secret, "[redacted]")
        return re.sub(r"eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+", "[redacted-jwt]", value)
    return value


def emit(value):
    print(json.dumps(redact(value), ensure_ascii=False), flush=True)


def private_write(path, content):
    path = Path(path).expanduser()
    path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    fd, temporary = tempfile.mkstemp(prefix=".artex-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as f:
            f.write(content)
        os.replace(temporary, path)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


def normalize_base(value):
    u = urllib.parse.urlsplit(value)
    if u.scheme not in ("http", "https") or not u.hostname or u.username or u.password or u.query or u.fragment:
        raise ClientError("ARTEX_URL must be an http(s) origin, optionally with a path prefix, without credentials/query.")
    if any(c.isspace() for c in value) or "\\" in value or any(s in (".", "..") for s in u.path.split("/")):
        raise ClientError("Invalid ARTEX_URL path.")
    return urllib.parse.urlunsplit((u.scheme.lower(), u.netloc.lower(), u.path.rstrip("/"), "", ""))


def validate_path(path):
    u = urllib.parse.urlsplit(path)
    decoded = urllib.parse.unquote(urllib.parse.unquote(u.path))
    if u.scheme or u.netloc or u.fragment or not path.startswith("/api/") or "\\" in decoded:
        raise ClientError("Use a relative /api/... path on the configured ARTEX instance.")
    if any(c.isspace() for c in path) or any(p in (".", "..") for p in decoded.split("/")):
        raise ClientError("Invalid API path.")
    if any(k.lower() in SECRET_KEYS for k, _ in urllib.parse.parse_qsl(u.query)):
        raise ClientError("Credentials belong in headers/session storage, not query strings.")
    return path


class Client:
    def __init__(self, base, session=None, timeout=30):
        self.base = normalize_base(base)
        self.timeout = timeout
        name = hashlib.sha256(self.base.encode()).hexdigest()[:24] + ".json"
        self.session = Path(session or os.environ.get("ARTEX_SESSION_FILE") or
                            Path.home() / ".config" / "artex-operator" / name).expanduser()
        self.cookies = http.cookiejar.CookieJar()
        self.opener = urllib.request.build_opener(NoRedirect(), urllib.request.HTTPCookieProcessor(self.cookies))

    def token(self):
        if os.environ.get("ARTEX_TOKEN"):
            KNOWN_SECRETS.add(os.environ["ARTEX_TOKEN"])
            return os.environ["ARTEX_TOKEN"]
        if not self.session.exists():
            return ""
        info = self.session.lstat()
        if not stat.S_ISREG(info.st_mode) or (os.name != "nt" and info.st_mode & 0o077):
            raise ClientError("Session must be a regular private file (POSIX mode 600).")
        data = json.loads(self.session.read_text())
        if data.get("base_url") != self.base:
            raise ClientError("Session belongs to a different ARTEX instance; login for this base URL.")
        token = data.get("token", "")
        KNOWN_SECRETS.add(token)
        return token

    def basic_gate(self):
        user, password = os.environ.get("ARTEX_BASIC_USER"), os.environ.get("ARTEX_BASIC_PASSWORD")
        if bool(user) != bool(password):
            raise ClientError("Set both ARTEX_BASIC_USER and ARTEX_BASIC_PASSWORD for the optional ARTEX Basic gate.")
        if user:
            auth = base64.b64encode((user + ":" + password).encode()).decode()
            self.request("GET", "/api/basic-auth/check", headers={"Authorization": "Basic " + auth})

    def request(self, method, path, body=None, headers=None, write=False, output=None, max_bytes=8 << 20, timeout=None):
        method = method.upper()
        validate_path(path)
        if method not in {"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"}:
            raise ClientError("Unsupported HTTP method.")
        if method not in {"GET", "HEAD"} and not write:
            raise ClientError("State-changing requests require --write; use only within the user's authorization.")
        h = {"Accept": "application/json", "User-Agent": "artex-operator/1"}
        token = self.token()
        if token and urllib.parse.urlsplit(path).path not in PUBLIC:
            h["Authorization"] = "Bearer " + token
        if isinstance(body, (dict, list)):
            body = json.dumps(body, ensure_ascii=False).encode()
            h["Content-Type"] = "application/json"
        h.update(headers or {})
        req = urllib.request.Request(self.base + path, data=body, headers=h, method=method)
        try:
            with self.opener.open(req, timeout=timeout or self.timeout) as response:
                if response.headers.get_content_type() == "text/event-stream":
                    raise ClientError("Use bounded snapshot/watch for monitoring; use a bounded SSE client for event streams.")
                data = response.read(max_bytes + 1)
                if len(data) > max_bytes:
                    raise ClientError("Response exceeded --max-bytes; use pagination or an explicitly larger bound.")
                if output:
                    private_write(output, data)
                    return {"output": str(Path(output).expanduser()), "bytes": len(data), "status": response.status}
                if not data:
                    return {"status": response.status}
                try:
                    return json.loads(data)
                except (ValueError, UnicodeError):
                    return {"status": response.status, "content_type": response.headers.get_content_type(),
                            "bytes": len(data), "hint": "Use --output to save non-JSON content privately."}
        except urllib.error.HTTPError as e:
            # Response bodies can echo credentials. Never print them or URL queries.
            e.close()
            raise ClientError(f"HTTP {e.code} for {method} {urllib.parse.urlsplit(path).path}; "
                              "do not blindly retry writes; inspect current state first.") from None
        except (urllib.error.URLError, TimeoutError, socket.timeout):
            raise ClientError("Connection failed or timed out. A write may already have applied; reconcile before retrying.") from None

    def login(self, password):
        KNOWN_SECRETS.add(password)
        value = self.request("POST", "/api/auth/login", {"username": "ARTEX", "password": password}, write=True)
        if not value.get("token"):
            raise ClientError("Login response did not contain a token.")
        KNOWN_SECRETS.add(value["token"])
        private_write(self.session, json.dumps({"base_url": self.base, "token": value["token"]}).encode())
        return {"authenticated": True, "session_file": str(self.session), "base_url": self.base}

    def snapshot(self, task, timeout=None):
        if not str(task).isdigit() or int(task) <= 0:
            raise ClientError("Task ID must be a positive integer.")
        tid = str(int(task))
        # Explicit task IDs avoid the server's mutable global active-task fallback.
        end = time.monotonic() + (timeout or self.timeout * 3)
        def read(path):
            remaining = end - time.monotonic()
            if remaining <= 0:
                raise ClientError("Snapshot time window ended; task state is not inferred.")
            return self.request("GET", path, timeout=min(self.timeout, remaining))
        detail = read(f"/api/tasks/{tid}")
        if str(detail.get("id")) != tid:
            raise ClientError("Unexpected task response; verify the ARTEX URL and API version.")
        intents = read(f"/api/exploration/intents?task={tid}&limit=100")
        approvals = read(f"/api/intercept/task/{tid}?status=pending&page=1&size=100")
        rows = intents if isinstance(intents, list) else intents.get("items", [])
        paused = [i for i in rows if i.get("state") in {"paused", "blocked", "exhausted", "stopped"}]
        return {"task": detail, "attention_intents": paused, "intents_returned": len(rows),
                "pending_approvals": approvals,
                "note": "Intent list is bounded; page the API if full history is needed."}


def multipart(files, field):
    if not re.fullmatch(r"[A-Za-z0-9_-]+", field):
        raise ClientError("Invalid multipart field name.")
    boundary = "artex-" + secrets.token_hex(16)
    chunks = []
    total = 0
    for name in files:
        p = Path(name)
        if not p.is_file() or p.stat().st_size > 32 << 20:
            raise ClientError("Upload must be a regular file no larger than 32 MiB.")
        content = p.read_bytes()
        total += len(content)
        if total > 32 << 20 or any(c in p.name for c in '\r\n"'):
            raise ClientError("Upload batch too large or filename unsuitable for multipart.")
        kind = mimetypes.guess_type(p.name)[0] or "application/octet-stream"
        chunks.append((f'--{boundary}\r\nContent-Disposition: form-data; name="{field}"; filename="{p.name}"\r\n'
                       f'Content-Type: {kind}\r\n\r\n').encode() + content + b"\r\n")
    return b"".join(chunks) + f"--{boundary}--\r\n".encode(), {"Content-Type": "multipart/form-data; boundary=" + boundary}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--base-url", default=os.environ.get("ARTEX_URL", "http://127.0.0.1:8787"))
    p.add_argument("--session-file")
    p.add_argument("--timeout", type=float, default=30)
    sub = p.add_subparsers(dest="command", required=True)
    sub.add_parser("login")
    r = sub.add_parser("request")
    r.add_argument("method")
    r.add_argument("path")
    r.add_argument("--write", action="store_true")
    content = r.add_mutually_exclusive_group()
    content.add_argument("--json-file", help="JSON file or - for stdin; avoids secrets in command arguments")
    content.add_argument("--upload", nargs="+", help="multipart upload; defaults to field file")
    r.add_argument("--file-field", default="file")
    r.add_argument("--output")
    r.add_argument("--max-bytes", type=int, default=64 << 20)
    sub.add_parser("snapshot").add_argument("task_id")
    w = sub.add_parser("watch")
    w.add_argument("task_id")
    w.add_argument("--seconds", type=float, default=60)
    w.add_argument("--interval", type=float, default=15)
    args = p.parse_args(argv)
    if not 0 < args.timeout <= 600:
        raise ClientError("Timeout must be in (0, 600] seconds.")
    if args.command == "watch" and not (0 < args.seconds <= 3600 and args.interval >= 2):
        raise ClientError("Watch needs 0 < seconds <= 3600 and interval >= 2.")
    if args.command == "request":
        validate_path(args.path)
        if args.method.upper() not in {"GET", "HEAD"} and not args.write:
            raise ClientError("State-changing requests require --write.")
        if args.max_bytes < 1:
            raise ClientError("--max-bytes must be positive.")
    client = Client(args.base_url, args.session_file, args.timeout)
    client.basic_gate()
    if args.command == "login":
        password = os.environ.get("ARTEX_PASSWORD") or getpass.getpass("ARTEX password: ")
        emit(client.login(password))
    elif args.command == "request":
        body, headers = None, None
        if args.json_file:
            body = json.loads(sys.stdin.read() if args.json_file == "-" else Path(args.json_file).read_text())
        if args.upload:
            body, headers = multipart(args.upload, args.file_field)
        emit(client.request(args.method, args.path, body, headers, args.write, args.output, args.max_bytes))
    elif args.command == "snapshot":
        emit(client.snapshot(args.task_id))
    else:
        end = time.monotonic() + args.seconds
        while time.monotonic() < end:
            value = client.snapshot(args.task_id, timeout=min(client.timeout, max(0.1, end-time.monotonic())))
            emit(value)
            t = value["task"]
            if (t.get("status") in TERMINAL or t.get("paused") or t.get("status") == "paused"
                or t.get("llm_failover_state") == "chain_exhausted" or value["attention_intents"]
                or value["pending_approvals"].get("items")):
                return
            time.sleep(max(0, min(args.interval, end-time.monotonic())))


if __name__ == "__main__":
    try:
        main()
    except (ClientError, ValueError, OSError) as e:
        # File errors contain paths, not HTTP credentials or bodies.
        print(json.dumps({"error": str(e)}, ensure_ascii=False), file=sys.stderr)
        sys.exit(1)
