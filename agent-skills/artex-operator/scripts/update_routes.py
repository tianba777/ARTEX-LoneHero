#!/usr/bin/env python3
"""Regenerate the literal server route index after an upstream sync."""
import argparse
from pathlib import Path
import re
import sys


def render(repo):
    entries = []
    for p in sorted((repo / 'server').rglob('*.go')):
        if p.name.endswith('_test.go'):
            continue
        for n, line in enumerate(p.read_text().splitlines(), 1):
            m = re.search(r'\.HandleFunc\("([A-Z]+) (/api/[^" ]*)",\s*(.*)\)', line)
            if m:
                entries.append((m[2], m[1], m[3].split(')')[0].strip(), f'{p.relative_to(repo)}:{n}'))
    if not entries:
        raise ValueError('No ARTEX routes found: supply the matching checkout with --repo.')
    out = ['# 源码接口目录', '', '按路径关键词搜索，只加载需要的部分。由本项目源码中的字面量 HandleFunc 注册生成；参数必须再查 handler/前端类型。此表不是 OpenAPI schema，也不代表接口已被实际调用测试。', '',
           '动态拼接的 side-questions 路由见 operations.md；它们的父路径/行为在 server/side_questions.go。', '',
           '| Method | Path | Handler | Source |', '| --- | --- | --- | --- |']
    for path, method, handler, source in sorted(set(entries)):
        out.append(f'| {method} | `{path}` | `{handler}` | `{source}` |')
    return '\n'.join(out) + '\n'


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--repo', type=Path, default=Path(__file__).resolve().parents[3])
    p.add_argument('--check', action='store_true')
    a = p.parse_args()
    dest = Path(__file__).resolve().parents[1] / 'references' / 'routes.md'
    content = render(a.repo.resolve())
    if a.check:
        if not dest.exists() or dest.read_text() != content:
            print('Route index differs from source; run update_routes.py after verifying handlers.', file=sys.stderr)
            return 1
    else:
        dest.write_text(content)
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
