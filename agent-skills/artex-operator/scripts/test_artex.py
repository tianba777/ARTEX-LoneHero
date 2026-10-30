import base64
import contextlib
import io
import json
import os
from pathlib import Path
import tempfile
import threading
import unittest
from unittest.mock import patch
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import urllib.parse

import artex


class ClientTests(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.TemporaryDirectory()
        self.addCleanup(self.dir.cleanup)
        self.requests = []
        self.gate = False
        self.pending = []
        self.other_hits = []
        outer = self

        class Destination(BaseHTTPRequestHandler):
            def do_GET(self):
                outer.other_hits.append(self.path)
                self.send_response(200)
                self.end_headers()
            def log_message(self, *args):
                pass

        self.other = ThreadingHTTPServer(('127.0.0.1', 0), Destination)
        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass
            def answer(self, status, value, headers=None):
                self.send_response(status)
                self.send_header('Content-Type', 'application/json')
                for k, v in (headers or {}).items():
                    self.send_header(k, v)
                self.end_headers()
                self.wfile.write(json.dumps(value).encode())
            def do_GET(self):
                outer.requests.append((self.command, self.path, dict(self.headers)))
                path = urllib.parse.urlsplit(self.path).path
                if path == '/api/redirect':
                    return self.answer(302, {}, {'Location':f'http://127.0.0.1:{outer.other.server_port}/stolen'})
                if path == '/api/basic-auth/check':
                    expected = 'Basic ' + base64.b64encode(b'gate-user:gate-password').decode()
                    if self.headers.get('Authorization') != expected:
                        return self.answer(401, {})
                    return self.answer(200, {}, {'Set-Cookie':'artex_basic_auth=gate-ok; HttpOnly; Path=/'})
                if outer.gate and 'artex_basic_auth=gate-ok' not in self.headers.get('Cookie',''):
                    return self.answer(401, {})
                if self.headers.get('Authorization') != 'Bearer opaque-private-token':
                    return self.answer(401, {})
                if path == '/api/tasks/7':
                    return self.answer(200, {'id':'7','status':'running'})
                if path == '/api/exploration/intents':
                    return self.answer(200, [])
                if path == '/api/intercept/task/7':
                    return self.answer(200, {'items':outer.pending,'total':len(outer.pending)})
                if path == '/api/report':
                    self.send_response(200)
                    self.send_header('Content-Type','text/markdown')
                    self.end_headers()
                    self.wfile.write(b'# private report\n')
                    return
                self.answer(200, {'echo':'Bearer opaque-private-token','token':'opaque-private-token'})
            def do_POST(self):
                body = self.rfile.read(int(self.headers.get('Content-Length', 0)))
                outer.requests.append((self.command, self.path, dict(self.headers), body))
                if self.path == '/api/auth/login':
                    if outer.gate and 'artex_basic_auth=gate-ok' not in self.headers.get('Cookie',''):
                        return self.answer(401, {})
                    if json.loads(body) != {'username':'ARTEX','password':'test-password'}:
                        return self.answer(401, {})
                    return self.answer(200, {'token':'opaque-private-token'})
                if self.path == '/api/unavailable':
                    return self.answer(503, {'error':'private backend body secret'})
                return self.answer(201, {'accepted': True})
        self.server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
        for server in [self.server, self.other]:
            thread = threading.Thread(target=server.serve_forever, daemon=True)
            thread.start()
            self.addCleanup(server.server_close)
            self.addCleanup(server.shutdown)
        self.base = f'http://127.0.0.1:{self.server.server_port}'
        self.session = Path(self.dir.name) / 'session.json'
        self.env = patch.dict(os.environ, {'ARTEX_TOKEN':'','ARTEX_PASSWORD':'','ARTEX_BASIC_USER':'',
                                           'ARTEX_BASIC_PASSWORD':'','ARTEX_SESSION_FILE':str(self.session)})
        self.env.start()
        self.addCleanup(self.env.stop)
        artex.KNOWN_SECRETS.clear()
        self.client = artex.Client(self.base, self.session)

    def test_login_private_storage_origin_binding_and_redaction(self):
        with contextlib.redirect_stdout(io.StringIO()) as out:
            artex.emit(self.client.login('test-password'))
            artex.emit(self.client.request('GET', '/api/tasks'))
        self.assertNotIn('opaque-private-token', out.getvalue())
        self.assertNotIn('test-password', out.getvalue())
        if os.name != 'nt':
            self.assertEqual(self.session.stat().st_mode & 0o777, 0o600)
        wrong = artex.Client(self.base + '/other-instance', self.session)
        with self.assertRaises(artex.ClientError):
            wrong.token()
        if os.name != 'nt':
            self.session.chmod(0o644)
            with self.assertRaises(artex.ClientError):
                self.client.token()

    def test_redirect_never_sends_credentials_to_destination(self):
        self.client.login('test-password')
        with self.assertRaises(artex.ClientError):
            self.client.request('GET', '/api/redirect')
        self.assertEqual(self.other_hits, [])

    def test_mutation_guard_and_no_automatic_retry(self):
        with self.assertRaises(artex.ClientError):
            self.client.request('POST', '/api/unavailable', {})
        self.assertEqual(self.requests, [])
        with self.assertRaises(artex.ClientError) as error:
            self.client.request('POST', '/api/unavailable', {}, write=True)
        self.assertEqual(len(self.requests), 1)
        self.assertNotIn('backend body secret', str(error.exception))

    def test_native_basic_gate_coexists_with_jwt(self):
        self.gate = True
        with patch.dict(os.environ, {'ARTEX_BASIC_USER':'gate-user','ARTEX_BASIC_PASSWORD':'gate-password'}):
            self.client.basic_gate()
            self.client.login('test-password')
            self.client.request('GET', '/api/tasks')
        h = self.requests[-1][2]
        self.assertEqual(h['Authorization'],'Bearer opaque-private-token')
        self.assertIn('artex_basic_auth=gate-ok',h['Cookie'])

    def test_watch_only_reads_explicit_task_and_stops_on_approval(self):
        self.client.login('test-password')
        self.requests.clear()
        self.pending = [{'id':19,'task_id':'7','status':'pending'}]
        with patch('artex.time.sleep') as sleep, contextlib.redirect_stdout(io.StringIO()) as out:
            artex.main(['--base-url',self.base,'watch','7','--seconds','60'])
        self.assertEqual(len(self.requests), 3)
        self.assertTrue(all(r[0]=='GET' for r in self.requests))
        self.assertIn('task=7',self.requests[1][1])
        self.assertIn('/api/intercept/task/7',self.requests[2][1])
        self.assertTrue(json.loads(out.getvalue())['pending_approvals']['items'])
        sleep.assert_not_called()

    def test_upload_and_private_non_json_export(self):
        self.client.login('test-password')
        file = Path(self.dir.name) / 'notes.txt'
        file.write_text('fixture data')
        body, headers = artex.multipart([file], 'file')
        self.client.request('POST','/api/chat/upload?scope=task&id=7',body,headers,write=True)
        self.assertIn(b'name="file"; filename="notes.txt"',self.requests[-1][3])
        self.assertIn(b'fixture data',self.requests[-1][3])
        dest = Path(self.dir.name) / 'report.md'
        result = self.client.request('GET','/api/report?task=7',output=dest)
        self.assertEqual(dest.read_bytes(), b'# private report\n')
        self.assertNotIn('private report',str(result))
        if os.name != 'nt':
            self.assertEqual(dest.stat().st_mode & 0o777,0o600)

    def test_rejects_foreign_paths_traversal_and_query_credentials(self):
        for path in ['https://elsewhere/api/tasks','//elsewhere/api/tasks','/api/../admin',
                     '/api/%252e%252e/admin','/api/tasks?token=secret','/api/tasks#fragment']:
            with self.subTest(path=path), self.assertRaises(artex.ClientError):
                self.client.request('GET',path)
        self.assertEqual(self.requests,[])


if __name__ == '__main__':
    unittest.main()
