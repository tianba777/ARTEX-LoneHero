#!/usr/bin/env bash
# Merge RuoJi6 on a new branch. Never reset, rebase, force-push or deploy.
set -euo pipefail
cd "$(dirname "$0")/.."
if [ -n "$(git status --porcelain)" ]; then
  echo "工作区有未提交改动，请先提交或保存后再同步。" >&2
  exit 1
fi
if ! git remote get-url upstream >/dev/null 2>&1; then
  git remote add upstream https://github.com/RuoJi6/ARTEX.git
fi
case "$(git remote get-url upstream)" in
  https://github.com/RuoJi6/ARTEX.git|git@github.com:RuoJi6/ARTEX.git) ;;
  *) echo "upstream 地址与记录不符，停止同步。" >&2; exit 1 ;;
esac
git fetch upstream main
if git merge-base --is-ancestor upstream/main HEAD; then
  echo "当前分支已包含上游最新提交。"
  exit 0
fi
branch="sync/ruoji6-$(date -u +%Y%m%dT%H%M%SZ)-$(git rev-parse --short upstream/main)"
git switch -c "$branch"
if ! git merge --no-edit upstream/main; then
  echo "同步有冲突，已保留在 $branch。解决冲突并通过检查后再提交；撤销本次合并可用 git merge --abort。" >&2
  exit 1
fi
echo "上游已合并到 $branch，请完成检查后提交 PR。此脚本不会推送或部署。"
