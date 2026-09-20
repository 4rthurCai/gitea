# Gitea fork 升级记录

目标：官方稳定版 **1.27.3**（2026-08-29 发布）。

- 上游标签：`v1.27.3`，提交 `146cc3eec57174711eac0e0a0c7b38670c6e3922`。
- 原 fork：`origin/release/v1.22.0`，提交 `29f023ef40b7f24c7b9a61ab31f47eef336b6d46`。
- 原上游基线：`v1.22.0`，提交 `803b0c9ab43f809e47f42fad951e7f5fb6652e42`。
- 升级分支：`upgrade/gitea-1.27.3`。原 fork 分支保留。
- **2026-09-17 已在生产环境完成升级**（数据库版本 299 → 343），部署产物为提交 `f22b18461f` 构建的 linux/amd64 程序。

## 必须保留的定制

| 定制                    | 迁移检查点                                                            |
| ----------------------- | --------------------------------------------------------------------- |
| jAccount OAuth          | provider、图标、账号/邮箱/姓名/学号映射及首次注册自动提交             |
| 隐藏 HTTP clone         | 保留 SSH clone 按钮，适配新版 DOM                                     |
| 用户资料限制            | 禁止修改姓名、邮箱隐私和位置；禁止自行删除账户                        |
| Wiki 活动               | 独立 Wiki 统计、作者头像和图表；适配新版仓库存储接口及 Vue/TypeScript |
| CompanyStaff pretend    | 保留公司团队选择和原有身份切换逻辑、`COMPANY_TEAM_NAME` 配置          |
| 屏蔽用户限制            | 保留 fork 对普通用户的限制和原有组织规则                              |
| 通知                    | 订阅页排除归档仓库的问题，并保留额外索引                              |
| Issue/PR 排序           | 默认按最近更新；显式选择其他排序仍可用                                |
| Actions                 | 保留额外索引、tag badge、badge message API 和 `no status` 响应        |
| Actions 共享仓库        | task token 可只读访问 `actions` 组织（不区分大小写）下的仓库          |
| API 对象格式            | 原 fork 修复已由新版的 `ObjectFormatName` 转换覆盖                    |

跨版本迁移需要调整内部接口和模板，不能直接保持旧源代码逐字不变；定制业务行为是保留目标。

`actions` 组织共享仓库的功能对应 focs-gitea 提交 `7066e1d9c1`。原提交分别修改了 API、git HTTP、LFS 三处；1.27.3 中这三处都统一调用 `GetActionsUserRepoPermission`，因此改为在该函数中统一实现（`models/perm/access/repo_permission.go`）。原提交中的 `EnvActionPerm` 在 1.27.3 已被移除，由 pre-receive hook 直接查询权限代替。

## 构建

使用项目 `go.mod` 和 `package.json` 指定的版本：Go 1.26.4+、Node 22.18+、pnpm 11.9.0。

```sh
pnpm install --frozen-lockfile
GITEA_VERSION=1.27.3 TAGS='bindata timetzdata' make build
./gitea --version
```

在 macOS 上为 Linux 服务器交叉编译（数据库为 MySQL/PostgreSQL 时不需要 CGO）：

```sh
GOOS=linux GOARCH=amd64 GITEA_VERSION=1.27.3 TAGS='bindata timetzdata' make build
file gitea                 # ELF 64-bit LSB executable, x86-64, statically linked
go version -m gitea | grep -E 'vcs.revision|vcs.modified'
```

2026-09-17 部署的程序：提交 `f22b18461f`，`vcs.modified=false`，go1.26.4，SHA256 为 `2e991efc56e9645680f2233a6d1d34eb6756e0b19150b5140f19d0957e3ffdae`。

也可从**此 fork 工作树**构建自定义镜像（尚未在目标环境验证）：

```sh
docker build --build-arg GITEA_VERSION=1.27.3 -t gitea-fork:1.27.3 .
```

必须使用包含定制的自建产物；官方现成二进制或镜像不包含本 fork 功能。

### 构建旧版 1.22.0（演练或回滚用）

1.22 分支使用 npm 和 Go 1.22，**必须用 Go 1.22 工具链编译**。用 Go 1.24 及以上版本编译出的程序，启动时会在 `modules/log` 中崩溃（`fatal error: concurrent map read and map write`）：1.22 的日志代码通过 `go:linkname` 读取 Go 运行时的内部结构，而这个结构在新版 Go 中已经改变。

```sh
git worktree add --detach ../gitea-1.22.0 origin/release/v1.22.0
cd ../gitea-1.22.0
npm ci
GOTOOLCHAIN=go1.22.12 GOOS=linux GOARCH=amd64 GITEA_VERSION=1.22.0 TAGS='bindata timetzdata' make build
go version gitea           # gitea: go1.22.12
```

## 生产升级结果（2026-09-17）

| 项目                   | 结果                                                                              |
| ---------------------- | --------------------------------------------------------------------------------- |
| 环境                   | Debian 13 LXC，x86_64，git 2.47.3，MySQL 9.7.2（本机），systemd，系统 OpenSSH     |
| 数据量                 | 数据库约 16 GB，`action` 表约 1100 万行，约 7000 个仓库，约 2000 个用户           |
| 备份                   | 管理员先停 Gitea 和 MySQL，再做 ZFS 快照；另外手动备份了程序、配置和 `~git/.ssh`  |
| 数据库迁移             | 299 → 343，耗时 **41 分钟**，没有 `[E]`/`[F]`                                     |
| doctor                 | `check-db-consistency`、`hooks` 通过；只报了升级前就存在的孤立数据（见待办）      |
| 数据                   | 用户、仓库、Issue、评论、PR、Release、LFS、令牌的行数与升级前一致                 |
| 功能                   | jAccount 登录、`/git/` 子路径链接、SSH clone/push 正常                            |
| Runner                 | 唯一在线的 runner 升级后能正常领取任务；课程 runner 从 2026-08-19 起离线，与升级无关 |

## 生产升级步骤

以下是实际执行过的步骤，敏感信息已替换为占位符。所有命令都在**服务器上的 tmux 里**、**以 root 身份**、**在 bash 中**执行。

- 如果服务器的默认 shell 不是 bash，要先输入 `bash`。
- 新开 tmux 窗口后，要重新执行 P2 定义变量和函数。
- 迁移耗时很长，必须在 tmux 里跑，防止 SSH 断线导致迁移中断。
- 测试机和生产机的窗口很容易混淆，建议在生产机上设置醒目的提示符：
  `PS1='\[\e[41;97m\] PROD \[\e[0m\] \u@\h:\w\$ '`

### P0. 摸底和预检（只读，不需要停服）

```sh
systemctl cat gitea --no-pager | grep -vE '^\s*#|^\s*$'
systemctl show gitea -p MainPID -p User -p ExecStart -p WorkingDirectory --no-pager
grep -nE '^\[|^ *(RUN_USER|WORK_PATH|APP_DATA_PATH|ROOT|PATH|DB_TYPE|HOST|NAME|USER|HTTP_PORT|ROOT_URL|DOMAIN|DISABLE_SSH|START_SSH_SERVER|SSH_PORT|ALLOWED_HOST_LIST|LOG_SQL|THEMES) *=' /etc/gitea/app.ini
. /etc/os-release; echo "$PRETTY_NAME"; uname -m; git --version; mysql --version
find /var/lib/gitea/custom -maxdepth 4 | head -50
grep -o 'command="[^"]*"' ~git/.ssh/authorized_keys | sed 's/key-[0-9]*/key-N/' | sort | uniq -c
```

需要确认：

- **数据库版本号必须是 299。** 不是的话要停下来，先查清楚 fork 和官方的迁移版本号是否一致。
- MySQL ≥ 8.0。git ≥ 2.42 时才支持 SHA-256 仓库，低于这个版本只是不能用 SHA-256 仓库。
- CPU 架构是 x86_64，与构建产物一致。
- **`custom/templates` 应该不存在。** 1.23 起模板结构改了很多次，旧模板大概率会让页面报错。
- `authorized_keys` 里记录的程序路径。升级后程序必须放在同一路径，否则所有用户的 SSH 访问都会失效。
- 数据库排序规则最好是区分大小写的，比如 `utf8mb4_0900_as_cs`。
- 最大的几张表和它们的索引。缺少标准索引的大表，迁移时会补建索引，耗时很长。

数据库连接：从 `app.ini` 读出账号密码，生成一个只有 root 能读的临时配置文件，避免密码出现在命令行和 shell 历史里。**升级全部完成后要删除这个文件。**

```sh
ini() { awk -v k="$1" '/^\[database\]/{s=1;next} /^\[/{s=0} s && $0 ~ "^ *"k" *=" {sub(/^[^=]*= */,""); gsub(/^[`"]|[`"] *$/,""); print; exit}' /etc/gitea/app.ini; }
DBHOST=$(ini HOST); H=${DBHOST%:*}; P=${DBHOST##*:}; [ "$P" = "$DBHOST" ] && P=3306; [ "$H" = localhost ] && H=127.0.0.1
( umask 077
  printf '[client]\nhost=%s\nport=%s\nuser=%s\npassword="%s"\n[mysql]\ndatabase=%s\n' "$H" "$P" "$(ini USER)" "$(ini PASSWD)" "$(ini NAME)" > /root/.gitea-my.cnf )
M() { mysql --defaults-extra-file=/root/.gitea-my.cnf "$@"; }
M -N -e "SELECT @@version, @@datadir, @@innodb_buffer_pool_size"
M -N -e "SELECT version FROM version"
M -e "SELECT table_name, ROUND((data_length+index_length)/1024/1024) mb, table_rows FROM information_schema.tables WHERE table_schema=DATABASE() ORDER BY mb DESC LIMIT 10"
M -e "SELECT status, COUNT(*) FROM action_run GROUP BY status"    # 6 = running
```

`database=` 必须放在 `[mysql]` 段里，因为 `mysqldump` 不认识这个选项。

### P1. 上传程序

```sh
scp gitea <gitea-host>:/root/gitea-1.27.3
```

上传后在服务器上校验：

```sh
chmod 755 /root/gitea-1.27.3
sha256sum /root/gitea-1.27.3
/root/gitea-1.27.3 --version
```

1.27 不允许以 root 身份运行（包括 `--help`），只有 `--version` 可以。

### P2. 定义变量和函数

```sh
export GITEA_BIN=/usr/local/bin/gitea GITEA_CONF=/etc/gitea/app.ini GITEA_WORK=/var/lib/gitea GITEA_CUSTOM=/var/lib/gitea/custom/
G() { runuser -u git -- env HOME=/home/git USER=git GITEA_WORK_DIR="$GITEA_WORK" "$GITEA_BIN" --custom-path "$GITEA_CUSTOM" -c "$GITEA_CONF" -w "$GITEA_WORK" "$@"; }
M() { mysql --defaults-extra-file=/root/.gitea-my.cnf "$@"; }
B=$(cat /root/last-backup 2>/dev/null); echo "$B"
```

- `G` 的参数要和 systemd 里 `ExecStart` 的参数保持一致，比如 `--custom-path`。
- 用 `runuser` 而不是 `sudo`，因为精简的容器里可能没有安装 sudo。

### P3. 停服和备份

Gitea 停服前先确认没有正在运行的 Actions 任务（`status = 6`）。

```sh
systemctl stop gitea
B=/root/backup-gitea-$(date +%Y%m%d-%H%M); mkdir -p "$B"; echo "$B" > /root/last-backup
cp -a "$GITEA_BIN" "$B/gitea-1.22.0"
cp -a "$GITEA_CONF" /etc/systemd/system/gitea.service "$B/"
cp -a /home/git/.ssh "$B/git-ssh"
for t in user repository issue comment pull_request release notification action lfs_meta_object access_token action_run action_runner login_source; do
  echo "$t: $(M -N -e "SELECT COUNT(*) FROM \`$t\`")"
done | tee "$B/counts-before.txt"
M -e "SELECT table_name, index_name, GROUP_CONCAT(column_name ORDER BY seq_in_index) cols FROM information_schema.statistics WHERE table_schema=DATABASE() GROUP BY table_name, index_name" > "$B/indexes-before.txt"
```

数据目录和数据库的完整备份，这次用的是管理员做的 ZFS 快照：**先停 Gitea 和 MySQL，再做快照**。

没有快照的话，还要另外执行：

```sh
mysqldump --defaults-extra-file=/root/.gitea-my.cnf --single-transaction --no-tablespaces --skip-routines <db-name> | gzip -1 > "$B/db.sql.gz"
zcat "$B/db.sql.gz" | tail -1                  # must be: -- Dump completed on ...
tar -C /var/lib -cf "$B/gitea-data.tar" gitea
```

### P4. 准备新配置

```sh
cp -a "$GITEA_CONF" /root/app.ini.1.27.3
sed -i 's/^\[server\]$/[server]\nPUBLIC_URL_DETECTION = legacy/' /root/app.ini.1.27.3
diff "$GITEA_CONF" /root/app.ini.1.27.3
```

需要处理的配置项：

- **`[webhook] ALLOWED_HOST_LIST` 挪到 `[security]` 下。** 1.27 会在日志里提示这个旧写法已废弃，v28 起将不再支持。但 **1.22 只认 `[webhook]` 下的写法**，所以如果回滚到 1.22 时没有用快照恢复配置，要手动把这一项挪回去。
- **加上 `PUBLIC_URL_DETECTION = legacy`。** 1.26 起默认值变成了 `auto`，会总是根据请求的 `Host` 头生成页面上的完整链接。站点挂在反向代理后面的子路径下时，设成 `legacy` 可以保持 1.22 的行为。
- `[camo] Allways` 改名为 `Always`。设置过 `[ui] THEMES` 的话要删掉，因为 1.23 起主题名字改了。

改完后 `diff` 应该只显示预期的几处改动。

### P5. 替换程序和配置

```sh
install -m 755 /root/gitea-1.27.3 "$GITEA_BIN"
cp /root/app.ini.1.27.3 "$GITEA_CONF"
"$GITEA_BIN" --version
```

服务器上 root 的 `cp` 如果带了 `-i` 别名，覆盖文件前会询问，输入 `y` 即可。

### P6. 迁移数据库

```sh
time G migrate 2>&1 | tee "$B/migrate.log"
M -N -e "SELECT version FROM version"                          # 343
grep -E "\[E\]|\[F\]" "$B/migrate.log" || echo "no errors"
grep "\[W\]" "$B/migrate.log"
```

- **迁移过程中不要按 Ctrl-C。** 日志里的 SQL 是执行完才打印的，所以建大索引时，屏幕可能好几分钟没有新输出，这是正常的。
- 想看进度的话，另开一个 tmux 窗口：

```sh
M -e "SELECT id, time, state, LEFT(info, 100) q FROM information_schema.processlist WHERE command <> 'Sleep' AND info NOT LIKE '%processlist%'"
```

- 生产环境耗时最长的是给 `action` 表建索引：v308、v317、v339 这三个迁移都要在这张表上建索引。其次是 v309（notification 表）和 v321（把列改成 LONGTEXT，要重建表）。

**版本号不是 343，或者出现 `[E]`/`[F]`：不要启动服务，直接回滚。**

### P7. doctor 检查

```sh
time G doctor check --run check-db-consistency --run hooks 2>&1 | tee "$B/doctor.log"
```

- 1.27.3 里**没有任何检查项被标记为默认**，所以不带参数的 `doctor check` 会显示 `checks: 0`，什么都不检查。
- `--all` 会扫描所有存储，仓库很多时非常慢，生产环境不建议用。
- 不加 `--fix` 时只报告不修改，Gitea 运行期间也可以跑。这次在生产环境跑了 16 分钟。

### P8. 启动

```sh
systemctl start gitea
sleep 10
systemctl is-active gitea
journalctl -u gitea --since "-2min" --no-pager | grep -E "\[E\]|\[F\]|eprecat|Listen|bleve" | cut -c1-200
curl -s http://127.0.0.1:<http-port>/api/healthz
echo
```

- 开启了 `REQUIRE_SIGNIN_VIEW` 时，1.27 的 `/api/v1/version` 也要求登录才能访问，所以改用 `/api/healthz` 检查服务状态。
- 日志里的 `Found older bleve index with version 4, Gitea will remove it and rebuild` 是正常的：Issue 搜索索引会在后台重建，期间搜索结果可能不完整。
- 启动时 Gitea 还会校对所有仓库的统计数字，数据量大时会持续几分钟，**这段时间页面会比较慢**。
- `Found legacy public asset "css" in CustomPath`：`custom/public/css` 是 1.21 之前放主题文件的旧位置，早就不生效了。把这个目录移走（比如移到 `$B`）即可。

### P9. 验证

```sh
for t in user repository issue comment pull_request release lfs_meta_object access_token login_source; do
  echo "$t: $(M -N -e "SELECT COUNT(*) FROM \`$t\`")"
done
M -e "SELECT t.id, rr.name AS runner, FROM_UNIXTIME(t.started) AS started, FROM_UNIXTIME(t.stopped) AS stopped, t.status FROM action_task t LEFT JOIN action_runner rr ON rr.id = t.runner_id ORDER BY t.id DESC LIMIT 10"
M -e "SELECT name, agent_labels, FROM_UNIXTIME(last_online) AS last_online FROM action_runner WHERE deleted IS NULL ORDER BY last_online DESC LIMIT 10"
```

- 和 `$B/counts-before.txt` 对比。服务启动后已经有用户访问，所以 `notification`、`action`、`action_run` 这几张表的行数会变，不用对比。
- **不要对 `action_run.commit_sha` 这类没有索引的列做 `LIKE` 查询**，在大表上会一直跑不完。按 Ctrl-C 可以终止查询。

浏览器验证清单：

- [ ] 页面底部显示 1.27.3，logo 等自定义资源正常显示
- [ ] jAccount 登录成功，登录后跳回的地址带有正确的子路径
- [ ] 页面上的链接和克隆地址都带有正确的子路径；只显示 SSH 克隆按钮，端口正确
- [ ] SSH clone 和 push 正常
- [ ] 个人设置里姓名、位置、邮箱隐私都不能修改，也没有删除账户的入口
- [ ] 仓库「活动」页有 Wiki 统计；Issue 列表默认按最近更新排序；订阅页不显示归档仓库的 Issue
- [ ] `.ipynb` 文件正常渲染；新建仓库时可以选择自定义的标签模板
- [ ] 推送后 workflow 能被在线的 runner 领取并跑完

**注意：服务一启动，用户就能访问了**（反向代理没有拦截）。启动之后如果再用快照回滚，这段时间里用户产生的数据会丢失，所以要尽快做完验证。

### P10. 收尾

```sh
cp -a "$GITEA_CONF" "$B/app.ini.before-p10"
sed -i 's/^ *LOG_SQL *= *.*/LOG_SQL = false/' "$GITEA_CONF"
chown root:git "$GITEA_CONF" && chmod 640 "$GITEA_CONF"
systemctl restart gitea
sleep 10
systemctl is-active gitea
journalctl -u gitea --since "-1min" --no-pager | grep -E "\[E\]|\[F\]" | cut -c1-200
```

- 生产配置里原来写的是 `LOG_SQL = true`（默认值是 `false`），会把每条 SQL 都写进日志。
- `app.ini` 原来的权限是 777，改成 `root:git 640` 后 Gitea 仍能正常启动。
- 重启会再次触发启动时的仓库统计校对，建议在访问少的时候做。
- 所有工作结束后执行 `rm -f /root/.gitea-my.cnf`，删掉临时的数据库连接配置文件。

### 回滚

1. 执行 `systemctl stop gitea`。
2. **有快照的话**，让管理员停掉容器、回滚到快照、再启动容器。
3. **没有快照的话**，按文件回滚：

```sh
install -m 755 "$B/gitea-1.22.0" "$GITEA_BIN"
cp -a "$B/app.ini" "$GITEA_CONF"            # must be the 1.22-compatible config
mysql -e "DROP DATABASE <db-name>"
zcat "$B/db.sql.gz" | mysql <db-name>       # recreate the database first if the dump has no CREATE DATABASE
mv /var/lib/gitea /var/lib/gitea.after-1.27.3
tar -C /var/lib -xf "$B/gitea-data.tar"
systemctl start gitea
```

几点说明：

- 恢复数据库前**必须先删库**。备份的 SQL 只会删除并重建备份里有的表，1.27 迁移新建的表会残留下来，导致下次升级失败。
- **迁移过的数据库不能给 1.22 使用。**
- 回滚会丢失从备份时刻到回滚之间的所有写入。

## 迁移日志中可以忽略的警告

以下警告都是 xorm 同步表结构时的提示。xorm 不会修改已有列的类型、是否可空或默认值，用官方 1.27.3 升级也会出现同样的提示：

- `action_artifact column status db type is BIGINT(20), struct type is INT`：迁移 v331 中临时定义的结构体用了 `int`，正式模型用的是 `int64`，与数据库一致。
- 各种 `db type is MEDIUMTEXT, struct type is TEXT`：数据库里的列比代码定义的更大。
- `notice.description`、`review_state.commit_sha` 是否可空不一致；`system_setting.version`、`label.archived_unix` 默认值不一致；`project.type` 有无符号不一致。
- `hook_task has column repo_id but struct has not related field`：老版本留下的列，允许为空，1.22 起就是这样。
- `sha256 hash support is disabled - requires Git >= 2.42`：只影响 SHA-256 仓库。

`u_s_uu` 索引不会出现：v309 迁移想在 notification 表的 `(user_id, status, updated_unix)` 上建这个索引，而 fork 已经在这三列上建过 `idx_notification_user_status_updated`。xorm 按「类型 + 列集合」判断索引是否已存在，不看名字，所以跳过了。查询性能不受影响。生产库 `action` 表上手工建的 `idx_action_order` 也是同样的情况，被保留了下来。

## 踩过的坑

- **配置被管理员提前改过。** 生产环境的 `ALLOWED_HOST_LIST` 在升级前已经由管理员挪到了 `[security]` 下，所以按 `[webhook]` 去查找时结果为空。做类似修改前，要先检查提取到的值不为空，并先确认配置有没有被人改过。
- **命令粘贴到了非 bash 的 shell。** 在 zsh/fish 中粘贴时，`;` 前面可能被加上反斜杠，导致请求的地址变了。所以要确认提示符，先进入 bash。
- **`timeout` 不加 `--foreground` 时收不到 Ctrl-C。** 命令卡住期间按下的方向键会积攒在输入缓冲里，命令结束后，bash 可能调出历史命令。所以提示符出来后要先按 Ctrl-C 清空当前行。
- **本机 Go 模块缓存解压不完整。** 缓存里留下了 `.incomplete` 标记，导致 `go build` 报 `no required module provides package`。删除对应的缓存目录后，还要执行 `go clean -cache`，清掉已缓存的模块索引。

## 待办

- **调大 MySQL 缓冲池。** 当前 `innodb_buffer_pool_size` 只有默认的 128 MB，而数据库有 16 GB，页面加载约需 4.5 秒。机器有 31 GB 内存，可用 13 GB，已经在使用交换空间，建议先设为 6 GB。在访问少的时候执行：

```sh
readlink -f /etc/mysql/my.cnf
grep -n includedir "$(readlink -f /etc/mysql/my.cnf)"     # must include mysql.conf.d
cat > /etc/mysql/mysql.conf.d/zz-gitea-tuning.cnf <<'EOF'
[mysqld]
innodb_buffer_pool_size = 6G
EOF
mysqld --validate-config --user=mysql && echo CONFIG_OK
systemctl stop gitea
systemctl restart mysql
M -N -e "SELECT ROUND(@@innodb_buffer_pool_size/1024/1024/1024, 1)"
systemctl start gitea
free -h
```

  出现问题时，删掉 `zz-gitea-tuning.cnf` 后重启 MySQL 即可恢复。

- **删除临时数据库连接配置。** 调完缓冲池后执行 `rm -f /root/.gitea-my.cnf`。
- **启动课程 runner。** `AR-*` 这些 runner（标签 `focs-latest-slim`，自编译版本 `v0.2.11+focs-*`）从 2026-08-19 起一直离线，需要启动它们，并确认它们与 1.27.3 兼容，比如 1.27 新增的「取消中」状态和复用 workflow 相关的改动。
- **清理孤立数据（可选）。** doctor 报告了约 60 万条用户已删除的 Action 记录、少量孤立的 OAuth2 授权码和重定向记录，以及 Topic 计数问题。可以在低峰时段执行 `G doctor check --run check-db-consistency --fix`，删除大量数据时会给数据库带来较大压力。
- **限制 MySQL 的监听地址。** MySQL 当前监听 `*:3306`，不需要对外提供服务的话，改为只监听 `127.0.0.1`。
- **清理重复的 runner 记录。** `action_runner` 表中有大量重复注册、长期离线的记录，可在管理后台清理。
- **清理旧的公钥备份。** `~git/.ssh/` 下有 2024 年留下的 `authorized_keys_*.gitea_bak` 备份文件，可以删除。
- **保留快照。** 观察数日，确认稳定后再让管理员删除快照。

## 测试机演练记录（2026-09-17）

演练环境：一台共享的 LXC 测试机（Debian 12，git 2.39.5）。

- 部署方式：fork 1.22.0（用 go1.22.12 编译），MySQL 8.0 运行在 Docker 中，只监听 `127.0.0.1`。Gitea 的 Web 和内置 SSH 服务也只监听本机，通过 SSH 隧道访问。
- 测试数据：通过脚本造了 alice/bob 两个用户，`course` 组织（含 CompanyStaff 团队），`actions` 组织，以及 Issue、PR、Release、两位作者的 Wiki、LFS 文件、Actions 运行记录和归档仓库。
- 检查方式：升级前后各跑一次检查脚本，对比结果。检查内容包括行数、API、LFS、Wiki 作者、克隆按钮、资料限制、Wiki 活动、Issue 默认排序、订阅页隐藏归档仓库、badge 和 pretend。
- 升级后：除版本号和数据库版本外，唯一的差异是非管理员访问 `/-/admin` 时返回码从 404 变成了 403。这是上游改的（`routers/web/web.go`），仍然是拒绝访问。
- 回滚演练：删库、导入备份、恢复数据目录并换回旧程序后，1.22.0 正常启动，与升级前的基线一致。
- 1.22.0 的 Wiki API 把分支写死为 `master`，在默认分支为 `main` 的 Wiki 上会返回 404。这是上游 1.22.0 的 bug，网页端不受影响。

## 本地验证结果

- `make frontend`：通过。
- `pnpm exec vue-tsc --noEmit`：通过。
- 修改的前端文件 ESLint：通过。
- `GITEA_VERSION=1.27.3 TAGS='bindata timetzdata' make build`：通过，生成本地 macOS/arm64 程序 `gitea`，包含前端资源。
- jAccount、用户权限/资料、活动统计、组织、Issue 索引、模板、Actions 的相关 Go 测试：通过（DB 索引子包无测试）。
- `make generate-swagger`：通过，包含 badge message API 的 Swagger/OpenAPI 定义。
- `GITEA_TEST_DATABASE=sqlite go test ./tests/integration -run '^TestForkBadgeMessage$' -count=1`：通过；验证令牌要求和默认分支、branch、tag 的无运行记录响应。
- Actions 共享仓库：`models/perm/access`、`services/lfs` 的单元测试，以及 `TestForkActionsSharedRepositoryHTTP`、`TestActions*` 和 AGit 相关集成测试：通过。
- `git diff --check v1.27.3`：通过。
- Elasticsearch / Meilisearch 测试因本机未启动相应外部服务失败；未据此改动产品行为。

构建期间本机 Go 模块缓存曾出现解压不完整的问题；修复后使用独立 `GOCACHE=/tmp/gitea-1.27.3-go-cache` 完成验证。项目 `go.mod`、`go.sum` 保持官方 1.27.3 的版本。

升级与恢复依据：

- https://blog.gitea.com/release-of-1.27.3/
- https://docs.gitea.com/installation/upgrade-from-gitea/
- https://docs.gitea.com/administration/backup-and-restore/
